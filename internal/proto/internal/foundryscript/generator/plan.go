package fsgenerator

import (
	"fmt"
	"sort"
	"strings"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// Protobuf wire types.
const (
	wireVarint          = 0
	wireLengthDelimited = 2
)

type valueKind int

const (
	kindScalar valueKind = iota
	kindEnum
	kindMessage
)

type cardinality int

const (
	cardinalitySingular cardinality = iota
	// cardinalityOptional is proto3 explicit presence, modelled as a nullable.
	cardinalityOptional
	cardinalityRepeated
	cardinalityMap
)

// valuePlan describes one encodable value: a whole field, or a map key or map
// value. It carries everything the emitter needs to frame that value on the
// wire, so no emitter branch has to re-derive it from the proto type string.
type valuePlan struct {
	Kind      valueKind
	ProtoType string
	// Type is the reference as it is written inside the declaring class, where
	// Foundry resolves inner type names lexically exactly as proto does.
	Type fstypes.Type
	// QualifiedType is the same reference scoped from file level, for the
	// hoisted oneof union, which is not inside the declaring class.
	QualifiedType fstypes.Type
	// ZeroValue is the Foundry Script literal for the proto default.
	ZeroValue string
	WireType  int
	// Namespace is the Foundry Script namespace this type is generated into
	// when it comes from an imported proto file; empty for local types.
	Namespace string
	// TopLevel is the outermost declaration the type is nested in. A hoisted
	// union cannot name a type nested inside the class that owns it.
	TopLevel string
}

// fieldPlan is a fully resolved message field.
type fieldPlan struct {
	Doc []string
	// Name is the emitted member name, which is the proto field name unless it
	// collided with a keyword or a generated member.
	Name string
	// RawName is the proto field name, used only to derive local variable names
	// so that escaping a member does not change the code around it.
	RawName     string
	Number      int
	Cardinality cardinality
	Value       valuePlan
	Key         valuePlan
	// Packed marks a repeated field that proto3 encodes as a single
	// length-delimited run of varints.
	Packed bool
	// OneofCase is the qualified tagged-union case this field is constructed
	// through when it is a oneof member; empty otherwise.
	OneofCase string
	// OneofCaseName is that case's own name, as declared in the union.
	OneofCaseName string
	// OneofField is the union-typed member the case is assigned to.
	OneofField string
}

// DeclaredType is the annotation for the generated member.
func (p fieldPlan) DeclaredType() fstypes.Type {
	switch p.Cardinality {
	case cardinalityOptional:
		return fstypes.Nullable(p.Value.Type)
	case cardinalityRepeated:
		return fstypes.Array(p.Value.Type)
	case cardinalityMap:
		return fstypes.Dictionary(p.Key.Type, p.Value.Type)
	default:
		if p.Value.Kind == kindMessage {
			// proto3 message fields have explicit presence; a reference type
			// has no meaningful non-null zero value.
			return fstypes.Nullable(p.Value.Type)
		}
		return p.Value.Type
	}
}

// DeclaredDefault is the initializer for the generated member.
func (p fieldPlan) DeclaredDefault() string {
	switch p.Cardinality {
	case cardinalityOptional:
		return "null"
	case cardinalityRepeated:
		return "[]"
	case cardinalityMap:
		return "{}"
	default:
		if p.Value.Kind == kindMessage {
			return "null"
		}
		return p.Value.ZeroValue
	}
}

// Tag is the encoded field key: field number and wire type.
func (p fieldPlan) Tag() int {
	return p.Number<<3 | p.TagWireType()
}

// TagWireType is the wire type this field's own tag carries, which is not the
// value's wire type for a map or a packed repeated field.
func (p fieldPlan) TagWireType() int {
	if p.Cardinality == cardinalityMap || p.Packed {
		return wireLengthDelimited
	}
	return p.Value.WireType
}

// TagExpression renders the field key as the runtime call that builds it, so
// the emitted source states the field number and framing instead of a literal
// only a protobuf implementer can decode by eye.
func (p fieldPlan) TagExpression() string {
	return tagExpression(p.Number, p.TagWireType())
}

func tagExpression(number, wireType int) string {
	return fmt.Sprintf("Wire.make_tag(%d, %s)", number, wireTypeConstant(wireType))
}

// Local names a variable the emitter introduces for this field.
func (p fieldPlan) Local(parts ...string) string {
	return localName(append([]string{p.RawName}, parts...)...)
}

// RetainsUnknownEnum reports whether this field keeps the raw bytes of an
// unrecognized enum value in a companion member of its own.
//
// proto3 enums are open, so a number this schema has no case for still has to
// survive a decode and re-encode. It cannot be appended to the shared
// unknown-field buffer the way a wholly unknown field can: it carries a field
// number this schema *does* know, and protobuf takes the last record for a
// singular field, so a trailing copy would override the member and a leading
// one would be overridden by it. Keeping it per field lets the value be written
// in the field's own position, and lets the member's setter drop it the moment
// anything assigns a value the schema can represent.
//
// Only a single-valued field can do this. A repeated or map-valued enum has no
// one member to attach the raw bytes to.
func (p fieldPlan) RetainsUnknownEnum() bool {
	if p.Value.Kind != kindEnum {
		return false
	}
	return p.Cardinality == cardinalitySingular || p.Cardinality == cardinalityOptional
}

// UnknownMember is the companion holding those raw bytes.
func (p fieldPlan) UnknownMember() string {
	return p.Local("unknown")
}

// clearedMember resets the member this field is read into, so it reads as unset
// while its retained raw value stands in for it. Going through the assignment
// is deliberate: it runs the setter, which drops any value retained earlier.
func (p fieldPlan) clearedMember() string {
	if p.OneofCase != "" {
		return p.OneofField + " = null"
	}
	return p.Name + " = " + p.DeclaredDefault()
}

// typeInfo is what the emitter needs to know about a named type it references:
// where the declaration lives and, for an enum, what its proto default is.
type typeInfo struct {
	// Reference is the fully scoped Foundry Script reference, such as
	// `Player.Badge`, which is what a declaration outside the class must use.
	Reference string
	IsEnum    bool
	// ZeroCase is the enum's zero-valued case name; empty for a message.
	ZeroCase string
	// Namespace is the Foundry Script namespace this type is generated into
	// when it comes from an imported proto file; empty for local types.
	Namespace string
	// TopLevel is the outermost declaration this type is nested in.
	TopLevel string
}

// typeRegistry maps a scoped reference to its declaration. Membership is also
// how the generator tells an enum-typed field from a message-typed one: the
// parser only populates Field.IsEnum for references it had to resolve across
// files.
type typeRegistry map[string]typeInfo

// resolve looks reference up from the innermost scope outward, matching proto's
// own name resolution.
func (r typeRegistry) resolve(scope, reference string) (typeInfo, bool) {
	for prefix := scope; ; {
		candidate := reference
		if prefix != "" {
			candidate = prefix + "." + reference
		}
		if info, ok := r[candidate]; ok {
			return info, true
		}
		if prefix == "" {
			return typeInfo{}, false
		}
		if cut := strings.LastIndex(prefix, "."); cut >= 0 {
			prefix = prefix[:cut]
		} else {
			prefix = ""
		}
	}
}

// resolver answers the two questions the emitter has about a named type: what
// its declaration looks like, and which namespace it has to be imported from.
//
// Declarations are kept per source rather than merged into one table. The
// parser rewrites a cross-file reference to its short name, so a schema that
// declares `Slot` locally and also imports a `Slot` leaves two declarations
// competing for one spelling; only the field's source file says which was
// meant.
type resolver struct {
	local typeRegistry
	// imported holds each dependency's declarations, keyed by its namespace.
	imported map[string]typeRegistry
	// namespaces maps a proto source filename to the namespace its types are
	// generated into.
	namespaces map[string]string
	// unnamespaced records dependencies with no usable namespace, whether they
	// declare none at all or one that would not parse. Their types cannot be
	// named from another file, so a reference is reported rather than emitted
	// as an import that breaks the generated file.
	unnamespaced map[string]bool
}

func newResolver(file *protoast.ProtoFile, imports []FileEntry) *resolver {
	resolve := &resolver{
		local:        typeRegistry{},
		imported:     map[string]typeRegistry{},
		namespaces:   map[string]string{},
		unnamespaced: map[string]bool{},
	}
	for i := range imports {
		namespace := NamespaceFor(imports[i].File)
		// A dependency's namespace is emitted as an import statement, so a
		// malformed one is a parse error in a file the user did not write.
		// Treat it as unusable rather than passing it through.
		if ValidateNamespace(namespace) != nil {
			resolve.unnamespaced[imports[i].Filename] = true
			continue
		}
		resolve.namespaces[imports[i].Filename] = namespace
		registry, seen := resolve.imported[namespace]
		if !seen {
			registry = typeRegistry{}
			resolve.imported[namespace] = registry
		}
		registry.registerFile(imports[i].File, namespace)
	}
	resolve.local.registerFile(file, "")
	return resolve
}

// ambiguous reports whether the short spelling of an imported reference would
// fail to name the declaration it means. Only the outermost segment matters: a
// competing `Slot` captures `Slot.Detail` just as surely as it captures `Slot`.
//
// Two things make it ambiguous. A local declaration captures the name outright,
// and it is resolved the way Foundry resolves it -- from scope outward -- so a
// type nested in an enclosing message shadows just as a top-level one does. A
// second imported namespace declaring the same name is also ambiguous, and
// Foundry rejects that outright rather than picking one.
func (r *resolver) ambiguous(scope, reference string) bool {
	head, _, _ := strings.Cut(reference, ".")
	if _, shadowed := r.local.resolve(scope, head); shadowed {
		return true
	}
	declaring := 0
	for _, registry := range r.imported {
		if _, declares := registry[head]; declares {
			declaring++
		}
	}
	return declaring > 1
}

func (r typeRegistry) registerFile(file *protoast.ProtoFile, namespace string) {
	if file == nil {
		return
	}
	for _, enum := range file.Enums {
		reference := TypeName(enum.Name)
		r[reference] = typeInfo{
			Reference: reference,
			IsEnum:    true,
			ZeroCase:  zeroValueName(enum),
			Namespace: namespace,
			TopLevel:  reference,
		}
	}
	for _, message := range file.Messages {
		r.registerMessage(message, TypeName(message.Name), namespace)
	}
}

// registerMessage records a message and every type nested in it. Nested types
// are keyed by their scoped `Outer.Inner` reference, which is the spelling the
// parser rewrote cross-file field types to and the one proto scoping implies
// for local ones.
func (r typeRegistry) registerMessage(message *protoast.Message, reference, namespace string) {
	topLevel := reference
	if cut := strings.Index(reference, "."); cut >= 0 {
		topLevel = reference[:cut]
	}
	r[reference] = typeInfo{
		Reference: reference,
		Namespace: namespace,
		TopLevel:  topLevel,
	}
	for _, enum := range message.NestedEnums {
		nested := reference + "." + TypeName(enum.Name)
		r[nested] = typeInfo{
			Reference: nested,
			IsEnum:    true,
			ZeroCase:  zeroValueName(enum),
			Namespace: namespace,
			TopLevel:  topLevel,
		}
	}
	for _, nested := range message.NestedMessages {
		r.registerMessage(nested, reference+"."+TypeName(nested.Name), namespace)
	}
}

// TypeReference converts a possibly-dotted proto type path to its Foundry
// Script reference. Nested types keep their scoping, so `outer.inner_thing`
// becomes `Outer.InnerThing` rather than being flattened.
func TypeReference(protoType string) string {
	parts := strings.Split(strings.TrimPrefix(protoType, "."), ".")
	converted := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		converted = append(converted, TypeName(part))
	}
	return strings.Join(converted, ".")
}

// scalarWireType reports the wire type for a proto scalar. Only the varint and
// length-delimited scalars are supported; validateWireFields rejects the rest
// before any plan is built.
func scalarWireType(protoType string) int {
	switch protoType {
	case "string", "bytes":
		return wireLengthDelimited
	default:
		return wireVarint
	}
}

func scalarZeroValue(protoType string) string {
	switch protoType {
	case "string":
		return `""`
	case "bytes":
		return "PackedByteArray()"
	case "bool":
		return "false"
	case "float", "double":
		return "0.0"
	default:
		return "0"
	}
}

// isPackable reports whether proto3 packs a repeated field of this value by
// default. Only fixed-width and varint scalars pack; length-delimited ones
// are always written one record per element.
func (v valuePlan) isPackable() bool {
	return v.Kind != kindMessage && v.WireType == wireVarint
}

func scalarValuePlan(protoType string) valuePlan {
	scalar := ScalarType(protoType)
	return valuePlan{
		Kind:      kindScalar,
		ProtoType: protoType,
		Type:      scalar,
		// A scalar names the same type from any scope.
		QualifiedType: scalar,
		ZeroValue:     scalarZeroValue(protoType),
		WireType:      scalarWireType(protoType),
	}
}

// typeUse is one reference to a named type, with everything the parser already
// resolved about it.
type typeUse struct {
	ProtoType string
	IsEnum    bool
	// EnumValues is the referenced enum's values when it was declared in
	// another file; the parser fills this in as part of resolving the import.
	EnumValues []*protoast.EnumValue
	// SourceFile is the proto file the type was declared in, when that is not
	// the file being generated.
	SourceFile string
}

// resolve looks a reference up in the source that declared it. A field the
// parser resolved across files is looked up in that file's namespace only;
// anything else resolves lexically from scope outward, as proto does.
func (r *resolver) resolve(use typeUse, scope, reference string) (typeInfo, bool) {
	if namespace := r.namespaces[use.SourceFile]; namespace != "" {
		if info, found := r.imported[namespace][reference]; found {
			return info, true
		}
	}
	return r.local.resolve(scope, reference)
}

func (r *resolver) namedValuePlan(use typeUse, scope string) (valuePlan, error) {
	// Inside the declaring class the reference is emitted as the schema wrote
	// it: Foundry resolves inner type names lexically, exactly as proto does.
	// Outside it, the registry's scoped reference is what resolves.
	if r.unnamespaced[use.SourceFile] {
		return valuePlan{}, fmt.Errorf(
			"%s is declared in %s, which has no usable namespace: give it a package or a valid (foundrytools.namespace) option",
			use.ProtoType, use.SourceFile)
	}
	reference := TypeReference(use.ProtoType)
	info, found := r.resolve(use, scope, reference)
	if !found {
		// The descriptor-driven plugin path can hand over a reference whose
		// declaration is not in the request. The parser still told us whether
		// it is an enum and what its values are, which is everything the wire
		// framing needs; only the scoped reference degrades to the lexical one.
		info = typeInfo{
			Reference: reference,
			IsEnum:    use.IsEnum,
			ZeroCase:  zeroValueNameOf(use.EnumValues),
			Namespace: r.namespaces[use.SourceFile],
			TopLevel:  reference,
		}
	}
	// An imported type is named by the short reference the import makes
	// available, unless something else would answer to that name too, in which
	// case only the namespace-qualified spelling picks out the declaration.
	lexical, qualified := reference, info.Reference
	if info.Namespace != "" {
		lexical = info.Reference
		if r.ambiguous(scope, info.Reference) {
			lexical = info.Namespace + "." + info.Reference
		}
		qualified = lexical
	}
	plan := valuePlan{
		ProtoType:     use.ProtoType,
		Type:          fstypes.Named(lexical),
		QualifiedType: fstypes.Named(qualified),
		Namespace:     info.Namespace,
		TopLevel:      info.TopLevel,
	}
	if !info.IsEnum {
		plan.Kind = kindMessage
		plan.ZeroValue = "null"
		plan.WireType = wireLengthDelimited
		return plan, nil
	}
	if info.ZeroCase == "" {
		// Emitting a default here is what produced an unparseable `Color.`.
		return valuePlan{}, fmt.Errorf("enum %s has no value to default to", use.ProtoType)
	}
	plan.Kind = kindEnum
	plan.ZeroValue = lexical + "." + info.ZeroCase
	plan.WireType = wireVarint
	return plan, nil
}

func (r *resolver) valuePlanFor(use typeUse, scope string) (valuePlan, error) {
	switch use.ProtoType {
	case "int32", "int64", "uint32", "uint64", "sint32", "sint64",
		"fixed32", "fixed64", "sfixed32", "sfixed64",
		"double", "float", "bool", "string", "bytes":
		return scalarValuePlan(use.ProtoType), nil
	default:
		return r.namedValuePlan(use, scope)
	}
}

// oneofPlan is one proto oneof, emitted as a nullable tagged-union member.
type oneofPlan struct {
	Doc     []string
	Field   string
	Type    string
	Members []fieldPlan
}

// RetainingMembers are the oneof's members that keep raw enum bytes.
func (o *oneofPlan) RetainingMembers() []fieldPlan {
	retaining := make([]fieldPlan, 0, len(o.Members))
	for i := range o.Members {
		if o.Members[i].RetainsUnknownEnum() {
			retaining = append(retaining, o.Members[i])
		}
	}
	return retaining
}

// Namespaces are the imported namespaces the hoisted union file has to declare.
func (o *oneofPlan) Namespaces() []string {
	seen := map[string]bool{}
	namespaces := make([]string, 0, len(o.Members))
	for i := range o.Members {
		namespace := o.Members[i].Value.Namespace
		if namespace == "" || seen[namespace] {
			continue
		}
		seen[namespace] = true
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

// messagePlan is a message resolved for emission, including its nested types.
type messagePlan struct {
	Doc  []string
	Name string
	// Scope is the qualified path of this message, used to resolve type
	// references declared in an enclosing message.
	Scope  string
	Fields []fieldPlan
	Oneofs []oneofPlan
	Enums  []*protoast.Enum
	Nested []messagePlan
}

// Namespaces are the imported namespaces this message's file has to declare,
// gathered across its own fields and every message nested in it, since nested
// messages are emitted into the same file.
func (p *messagePlan) Namespaces() []string {
	seen := map[string]bool{}
	p.collectNamespaces(seen)
	namespaces := make([]string, 0, len(seen))
	for namespace := range seen {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

func (p *messagePlan) collectNamespaces(seen map[string]bool) {
	for i := range p.Fields {
		if namespace := p.Fields[i].Key.Namespace; namespace != "" {
			seen[namespace] = true
		}
		if namespace := p.Fields[i].Value.Namespace; namespace != "" {
			seen[namespace] = true
		}
	}
	for i := range p.Nested {
		p.Nested[i].collectNamespaces(seen)
	}
}

// planMessage resolves every member of a message into emitter-ready plans. The
// field list is in field-number order so serialization is deterministic and
// the emitted match arms read in wire order.
func planMessage(message *protoast.Message, parentScope string, resolve *resolver) (messagePlan, error) {
	scope := TypeName(message.Name)
	if parentScope != "" {
		scope = parentScope + "." + scope
	}
	plans := make([]fieldPlan, 0, len(message.Fields)+len(message.Maps))

	for _, field := range message.Fields {
		plan, err := planField(field, message.Name, scope, resolve)
		if err != nil {
			return messagePlan{}, err
		}
		plans = append(plans, plan)
	}

	oneofs := make([]oneofPlan, 0, len(message.Oneofs))
	for _, oneof := range message.Oneofs {
		if err := ValidateMemberName(message.Name, "oneof", oneof.Name); err != nil {
			return messagePlan{}, err
		}
		caseType := oneofTypeName(scope, oneof)
		members := make([]fieldPlan, 0, len(oneof.Fields))
		for _, field := range oneof.Fields {
			plan, err := planField(field, message.Name, scope, resolve)
			if err != nil {
				return messagePlan{}, err
			}
			if err := validateOneofPayload(scope, oneof.Name, field.Name, plan.Value); err != nil {
				return messagePlan{}, err
			}
			// A oneof member is only ever set through the union, so it has no
			// independent presence of its own.
			plan.Cardinality = cardinalitySingular
			plan.OneofCaseName = TypeName(field.Name)
			plan.OneofCase = caseType + "." + plan.OneofCaseName
			plan.OneofField = FieldName(oneof.Name)
			plan.RawName = oneof.Name + "_" + field.Name
			members = append(members, plan)
			plans = append(plans, plan)
		}
		oneofs = append(oneofs, oneofPlan{
			Doc:     oneof.Doc,
			Field:   FieldName(oneof.Name),
			Type:    caseType,
			Members: members,
		})
	}

	for _, mapField := range message.Maps {
		plan, err := planMapField(mapField, message.Name, scope, resolve)
		if err != nil {
			return messagePlan{}, err
		}
		plans = append(plans, plan)
	}

	nested := make([]messagePlan, 0, len(message.NestedMessages))
	for _, child := range message.NestedMessages {
		childPlan, err := planMessage(child, scope, resolve)
		if err != nil {
			return messagePlan{}, err
		}
		nested = append(nested, childPlan)
	}

	sortPlansByNumber(plans)
	if err := validateMemberNames(message.Name, plans, oneofs); err != nil {
		return messagePlan{}, err
	}
	return messagePlan{
		Doc:    message.Doc,
		Name:   TypeName(message.Name),
		Scope:  scope,
		Fields: plans,
		Oneofs: oneofs,
		Enums:  message.NestedEnums,
		Nested: nested,
	}, nil
}

// validateMemberNames refuses a message whose fields do not map onto distinct
// members. Escaping a keyword appends an underscore, so a schema declaring both
// `var` and `var_` would otherwise emit the member twice and conflate two
// distinct protobuf fields.
func validateMemberNames(messageName string, plans []fieldPlan, oneofs []oneofPlan) error {
	declaredBy := map[string]string{unknownFieldsMember: "the generated unknown-field buffer"}
	claim := func(member, source string) error {
		if previous, taken := declaredBy[member]; taken {
			return fmt.Errorf(
				"message %s: %s and %s both map to the member %s; rename one of them",
				messageName, previous, source, member)
		}
		declaredBy[member] = source
		return nil
	}
	for i := range plans {
		// A retained-value companion is a member too, and its name is derived
		// by joining names with underscores, so a field `a_b` and a oneof `a`
		// with a member `b` reach the same spelling from different directions.
		if plans[i].RetainsUnknownEnum() {
			if err := claim(plans[i].UnknownMember(), "the retained value of "+plans[i].RawName); err != nil {
				return err
			}
		}
		if plans[i].OneofCase != "" {
			continue
		}
		if err := claim(plans[i].Name, "field "+plans[i].RawName); err != nil {
			return err
		}
	}
	for i := range oneofs {
		if err := claim(oneofs[i].Field, "oneof "+oneofs[i].Field); err != nil {
			return err
		}
	}
	return nil
}

// validateOneofPayload refuses the one payload shape the hoisted union cannot
// name. The union is emitted at file level, so referring to a type nested in
// the class that owns the oneof closes a resolution cycle -- the class needs
// the union to declare its member, and the union needs the class to reach the
// nested type -- which Foundry cannot break for a class that conforms to a
// trait, and every message binding does.
func validateOneofPayload(scope, oneofName, fieldName string, value valuePlan) error {
	if value.Kind == kindScalar || value.Namespace != "" {
		return nil
	}
	topLevel := scope
	if cut := strings.Index(scope, "."); cut >= 0 {
		topLevel = scope[:cut]
	}
	if value.TopLevel != topLevel || value.Reference() == topLevel {
		return nil
	}
	return fmt.Errorf(
		"oneof %s field %s: %s is nested in %s, and a oneof cannot carry a type nested in the message that declares it; move it out of %s",
		oneofName, fieldName, value.Reference(), topLevel, topLevel)
}

// Reference is the scoped Foundry Script reference for this value's type.
func (v valuePlan) Reference() string {
	return v.QualifiedType.Render()
}

func planField(field *protoast.Field, messageName, scope string, resolve *resolver) (fieldPlan, error) {
	if err := ValidateMemberName(messageName, "field", field.Name); err != nil {
		return fieldPlan{}, err
	}
	value, err := resolve.valuePlanFor(typeUse{
		ProtoType:  field.FieldType,
		IsEnum:     field.IsEnum,
		EnumValues: field.EnumValues,
		SourceFile: field.SourceFile,
	}, scope)
	if err != nil {
		return fieldPlan{}, fmt.Errorf("field %s.%s: %w", messageName, field.Name, err)
	}
	plan := fieldPlan{
		Doc:     field.Doc,
		Name:    FieldName(field.Name),
		RawName: field.Name,
		Number:  field.Number,
		Value:   value,
	}
	switch {
	case field.Repeated:
		plan.Cardinality = cardinalityRepeated
		plan.Packed = value.isPackable()
	case field.Optional:
		plan.Cardinality = cardinalityOptional
	default:
		plan.Cardinality = cardinalitySingular
	}
	return plan, nil
}

func planMapField(mapField *protoast.MapField, messageName, scope string, resolve *resolver) (fieldPlan, error) {
	if err := ValidateMemberName(messageName, "field", mapField.Name); err != nil {
		return fieldPlan{}, err
	}
	key, err := resolve.valuePlanFor(typeUse{ProtoType: mapField.KeyType}, scope)
	if err != nil {
		return fieldPlan{}, fmt.Errorf("field %s.%s: %w", messageName, mapField.Name, err)
	}
	value, err := resolve.valuePlanFor(typeUse{
		ProtoType:  mapField.ValueType,
		IsEnum:     mapField.ValueIsEnum,
		SourceFile: mapField.ValueSourceFile,
	}, scope)
	if err != nil {
		return fieldPlan{}, fmt.Errorf("field %s.%s: %w", messageName, mapField.Name, err)
	}
	return fieldPlan{
		Doc:         mapField.Doc,
		Name:        FieldName(mapField.Name),
		RawName:     mapField.Name,
		Number:      mapField.Number,
		Cardinality: cardinalityMap,
		Key:         key,
		Value:       value,
	}, nil
}

func sortPlansByNumber(plans []fieldPlan) {
	for i := 1; i < len(plans); i++ {
		for j := i; j > 0 && plans[j].Number < plans[j-1].Number; j-- {
			plans[j], plans[j-1] = plans[j-1], plans[j]
		}
	}
}

// oneofTypeName is the tagged-union type generated for a oneof. Enums cannot be
// generic, so the union is emitted per-oneof rather than shared, and it is
// emitted at file level rather than nested inside the message: a tagged-union
// case pattern resolves only one level deep, so a nested union could not be
// matched by any consumer outside the class.
func oneofTypeName(scope string, oneof *protoast.Oneof) string {
	return strings.ReplaceAll(scope, ".", "") + TypeName(oneof.Name) + "Case"
}
