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
	wire64Bit           = 1
	wireLengthDelimited = 2
	wire32Bit           = 5
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
}

// fieldPlan is a fully resolved message field.
type fieldPlan struct {
	Doc      []string
	Position protoast.Position
	Kind     string
	// Name is the emitted member name, which is the proto field name unless it
	// collided with a keyword, generated member, or engine type.
	Name string
	// RawName is the exact protobuf field name, preserved for fallback docs and
	// collision diagnostics.
	RawName string
	// LocalStem disambiguates emitter-owned locals for a oneof alternative.
	// Ordinary fields and maps leave it empty and use RawName.
	LocalStem   string
	Escape      memberEscape
	Number      int
	Cardinality cardinality
	Value       valuePlan
	Key         valuePlan
	// Packed marks a repeated field this schema asks to be written as a single
	// length-delimited run. It follows the value's type unless the field
	// carries an explicit `[packed = ...]`, and it governs the encoder only.
	Packed bool
	// Packable marks a repeated field whose value type has a packed form at
	// all. The decoder works from this rather than from Packed: protobuf
	// requires both encodings to be accepted whatever the schema declared.
	Packable bool
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
	stem := p.LocalStem
	if stem == "" {
		stem = p.RawName
	}
	return localName(append([]string{stem}, parts...)...)
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
	// ProtoReference is the unprefixed reference used for protobuf lexical
	// lookup, such as `Player.Badge`.
	ProtoReference string
	// Reference is the fully scoped Foundry Script reference, such as
	// `GamePlayer.GameBadge`, which is what a declaration outside the class
	// must use.
	Reference string
	IsEnum    bool
	// ZeroCase is the enum's zero-valued case name; empty for a message.
	ZeroCase string
	// Namespace is the Foundry Script namespace this type is generated into
	// when it comes from an imported proto file; empty for local types.
	Namespace string
	// TopLevel is the outermost declaration this type is nested in.
	TopLevel string
	// Declaration identifies the schema declaration that produced this type.
	// Imported declarations are reported only when a field resolves to them.
	Declaration declarationInfo
}

// typeRegistry maps a scoped reference to its declaration. Membership is also
// how the generator tells an enum-typed field from a message-typed one: the
// parser only populates Field.IsEnum for references it had to resolve across
// files.
type typeRegistry map[string]typeInfo

// declarationIndex preserves exact raw protobuf paths independently of the
// normalized registry used to resolve emitted Foundry names.
type declarationIndex struct {
	byFullName     map[string]declarationInfo
	byRelativeName map[string]declarationInfo
}

func newDeclarationIndex(protoPackage string, declarations []declarationInfo) declarationIndex {
	index := declarationIndex{
		byFullName:     map[string]declarationInfo{},
		byRelativeName: map[string]declarationInfo{},
	}
	packagePrefix := strings.TrimPrefix(protoPackage, ".")
	if packagePrefix != "" {
		packagePrefix += "."
	}
	for _, declaration := range declarations {
		fullName := strings.TrimPrefix(declaration.ProtoName, ".")
		index.byFullName[fullName] = declaration
		if packagePrefix != "" && strings.HasPrefix(fullName, packagePrefix) {
			index.byRelativeName[strings.TrimPrefix(fullName, packagePrefix)] = declaration
		} else {
			index.byRelativeName[fullName] = declaration
		}
	}
	return index
}

func (i declarationIndex) resolve(fullPath, relativeReference string) (declarationInfo, bool) {
	if fullPath != "" {
		declaration, found := i.byFullName[strings.TrimPrefix(fullPath, ".")]
		return declaration, found
	}
	if strings.HasPrefix(relativeReference, ".") {
		declaration, found := i.byFullName[strings.TrimPrefix(relativeReference, ".")]
		return declaration, found
	}
	declaration, found := i.byRelativeName[relativeReference]
	return declaration, found
}

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
	local            typeRegistry
	localNamer       typeNamer
	sourceName       string
	protoPackage     string
	collisions       *collisionCollector
	memberCollisions *memberCollisionCollector
	// imported holds each dependency's declarations, keyed by its source file.
	imported map[string]typeRegistry
	// importedDeclarations preserves exact raw protobuf declaration identities
	// separately from the normalized registry used for emitted-name resolution.
	importedDeclarations map[string]declarationIndex
	// namespaces maps a proto source filename to the namespace its types are
	// generated into.
	namespaces map[string]string
	// dependencyNamers maps a proto source filename to the naming policy of the
	// file that declares its types.
	dependencyNamers map[string]typeNamer
	// dependencyErrors records invalid dependency prefixes without failing the
	// local generation unless a field actually references that dependency.
	dependencyErrors map[string]error
	// unnamespaced records dependencies with no usable namespace, whether they
	// declare none at all or one that would not parse. Their types cannot be
	// named from another file, so a reference is reported rather than emitted
	// as an import that breaks the generated file.
	unnamespaced map[string]bool
}

func newResolver(file *protoast.ProtoFile, sourceName string, imports []FileEntry, localNamer typeNamer) *resolver {
	resolve := &resolver{
		local:                typeRegistry{},
		localNamer:           localNamer,
		sourceName:           sourceName,
		protoPackage:         file.Package,
		collisions:           newCollisionCollector(),
		memberCollisions:     newMemberCollisionCollector(),
		imported:             map[string]typeRegistry{},
		importedDeclarations: map[string]declarationIndex{},
		namespaces:           map[string]string{},
		dependencyNamers:     map[string]typeNamer{},
		dependencyErrors:     map[string]error{},
		unnamespaced:         map[string]bool{},
	}
	for i := range imports {
		namer, err := newTypeNamer(imports[i].File, imports[i].Filename)
		if err != nil {
			resolve.dependencyErrors[imports[i].Filename] = err
			continue
		}
		resolve.dependencyNamers[imports[i].Filename] = namer

		namespace := NamespaceFor(imports[i].File)
		// A dependency's namespace is emitted as an import statement, so a
		// malformed one is a parse error in a file the user did not write.
		// Treat it as unusable rather than passing it through.
		if ValidateNamespace(namespace) != nil {
			resolve.unnamespaced[imports[i].Filename] = true
			continue
		}
		resolve.namespaces[imports[i].Filename] = namespace
		registry := typeRegistry{}
		declarations := registry.registerFile(imports[i].File, imports[i].Filename, namespace, namer)
		resolve.imported[imports[i].Filename] = registry
		resolve.importedDeclarations[imports[i].Filename] = newDeclarationIndex(
			imports[i].File.Package,
			declarations,
		)
	}
	declarations := resolve.local.registerFile(file, sourceName, "", localNamer)
	for _, declaration := range declarations {
		resolve.collisions.AddLocal(declaration)
	}
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
	if r.local.declaresInScope(scope, head) {
		return true
	}
	declaring := 0
	for _, registry := range r.imported {
		for key := range registry {
			info := registry[key]
			if strings.Contains(info.ProtoReference, ".") || info.TopLevel != head {
				continue
			}
			declaring++
		}
	}
	return declaring > 1
}

func (r typeRegistry) declaresInScope(scope, emittedName string) bool {
	for prefix := scope; ; {
		for key := range r {
			info := r[key]
			parent := ""
			if cut := strings.LastIndex(info.ProtoReference, "."); cut >= 0 {
				parent = info.ProtoReference[:cut]
			}
			emittedSegment := info.Reference
			if cut := strings.LastIndex(emittedSegment, "."); cut >= 0 {
				emittedSegment = emittedSegment[cut+1:]
			}
			if parent == prefix && emittedSegment == emittedName {
				return true
			}
		}
		if prefix == "" {
			return false
		}
		if cut := strings.LastIndex(prefix, "."); cut >= 0 {
			prefix = prefix[:cut]
		} else {
			prefix = ""
		}
	}
}

func (r typeRegistry) registerFile(
	file *protoast.ProtoFile,
	sourceName, namespace string,
	namer typeNamer,
) []declarationInfo {
	if file == nil {
		return nil
	}
	declarations := make([]declarationInfo, 0, len(file.Enums)+len(file.Messages))
	for _, enum := range file.Enums {
		protoReference := TypeName(enum.Name)
		reference := namer.Name(enum.Name)
		declaration := declarationInfo{
			SourceName:    sourceName,
			Position:      enum.Position,
			Kind:          "enum",
			ProtoName:     qualifiedProtoName(file.Package, enum.Name),
			GeneratedName: reference,
		}
		declarations = append(declarations, declaration)
		r[protoReference] = typeInfo{
			ProtoReference: protoReference,
			Reference:      reference,
			IsEnum:         true,
			ZeroCase:       zeroValueName(enum),
			Namespace:      namespace,
			TopLevel:       reference,
			Declaration:    declaration,
		}
	}
	for _, message := range file.Messages {
		r.registerMessage(
			message,
			TypeName(message.Name),
			namer.Name(message.Name),
			message.Name,
			file.Package,
			sourceName,
			namespace,
			namer,
			&declarations,
		)
	}
	return declarations
}

// registerMessage records a message and every type nested in it. Nested types
// are keyed by their scoped `Outer.Inner` reference, which is the spelling the
// parser rewrote cross-file field types to and the one proto scoping implies
// for local ones.
func (r typeRegistry) registerMessage(
	message *protoast.Message,
	protoReference, reference, protoPath, protoPackage, sourceName, namespace string,
	namer typeNamer,
	declarations *[]declarationInfo,
) {
	topLevel := reference
	if cut := strings.Index(topLevel, "."); cut >= 0 {
		topLevel = topLevel[:cut]
	}
	declaration := declarationInfo{
		SourceName:    sourceName,
		Position:      message.Position,
		Kind:          "message",
		ProtoName:     qualifiedProtoName(protoPackage, protoPath),
		GeneratedName: namer.Name(message.Name),
	}
	*declarations = append(*declarations, declaration)
	r[protoReference] = typeInfo{
		ProtoReference: protoReference,
		Reference:      reference,
		Namespace:      namespace,
		TopLevel:       topLevel,
		Declaration:    declaration,
	}
	for _, enum := range message.NestedEnums {
		nestedProtoReference := protoReference + "." + TypeName(enum.Name)
		nestedReference := reference + "." + namer.Name(enum.Name)
		declaration := declarationInfo{
			SourceName:    sourceName,
			Position:      enum.Position,
			Kind:          "enum",
			ProtoName:     qualifiedProtoName(protoPackage, protoPath+"."+enum.Name),
			GeneratedName: namer.Name(enum.Name),
		}
		*declarations = append(*declarations, declaration)
		r[nestedProtoReference] = typeInfo{
			ProtoReference: nestedProtoReference,
			Reference:      nestedReference,
			IsEnum:         true,
			ZeroCase:       zeroValueName(enum),
			Namespace:      namespace,
			TopLevel:       topLevel,
			Declaration:    declaration,
		}
	}
	for _, nested := range message.NestedMessages {
		r.registerMessage(
			nested,
			protoReference+"."+TypeName(nested.Name),
			reference+"."+namer.Name(nested.Name),
			protoPath+"."+nested.Name,
			protoPackage,
			sourceName,
			namespace,
			namer,
			declarations,
		)
	}
}

func qualifiedProtoName(protoPackage, protoPath string) string {
	if protoPackage == "" {
		return protoPath
	}
	return protoPackage + "." + protoPath
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

// scalarWireType reports the wire type for a proto scalar. The zig-zag types
// stay varints — zig-zag changes the value, not the framing.
func scalarWireType(protoType string) int {
	switch protoType {
	case "string", "bytes":
		return wireLengthDelimited
	case "fixed32", "sfixed32", "float":
		return wire32Bit
	case "fixed64", "sfixed64", "double":
		return wire64Bit
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
// default. Every numeric scalar packs, fixed-width ones included; only the
// length-delimited types are always written one record per element.
func (v valuePlan) isPackable() bool {
	return v.Kind != kindMessage && v.WireType != wireLengthDelimited
}

// encodeCall renders the runtime call that frames one value of this type.
// Each scalar names its own codec, so the generated source states the framing
// the schema asked for instead of leaving it implied by the tag.
func (v valuePlan) encodeCall(expression string) string {
	switch v.ProtoType {
	case "float":
		return "Wire.encode_float(" + expression + ")"
	case "double":
		return "Wire.encode_double(" + expression + ")"
	case "fixed32", "sfixed32":
		return "Wire.encode_fixed32(" + expression + ")"
	case "fixed64", "sfixed64":
		return "Wire.encode_fixed64(" + expression + ")"
	case "sint32":
		return "Wire.encode_sint32(" + expression + ")"
	case "sint64":
		return "Wire.encode_sint64(" + expression + ")"
	default:
		return "Wire.encode_varint(" + varintExpression(v, expression) + ")"
	}
}

// readCarrier is the tuple a read of this type returns. They differ because
// the value they carry does: an int for the integral types, a float for the
// two IEEE-754 ones.
func (v valuePlan) readCarrier() string {
	switch v.ProtoType {
	case "float", "double":
		return "FloatRead"
	case "fixed32", "sfixed32", "fixed64", "sfixed64":
		return "FixedRead"
	default:
		return "VarintRead"
	}
}

// readFunction is the runtime call that reads one untagged value of this type.
//
// fixed64 and sfixed64 share a reader: Foundry's int is signed 64-bit, so both
// keep every bit and differ only in what the number means. The 32-bit pair
// cannot share, since fixed32 spans a range that does not fit the sign
// convention of sfixed32.
func (v valuePlan) readFunction() string {
	switch v.ProtoType {
	case "float":
		return "Wire.read_float"
	case "double":
		return "Wire.read_double"
	case "fixed32":
		return "Wire.read_fixed32"
	case "sfixed32":
		return "Wire.read_sfixed32"
	case "fixed64", "sfixed64":
		return "Wire.read_fixed64"
	case "sint32":
		return "Wire.read_sint32"
	case "sint64":
		return "Wire.read_sint64"
	default:
		return "Wire.decode_varint"
	}
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
	ProtoType     string
	FullProtoPath string
	IsEnum        bool
	// EnumValues is the referenced enum's values when it was declared in
	// another file; the parser fills this in as part of resolving the import.
	EnumValues []*protoast.EnumValue
	// SourceFile is the proto file the type was declared in. Descriptor-driven
	// generation also sets it for same-file references.
	SourceFile string
}

type typeResolution struct {
	Info           typeInfo
	Reference      string
	Found          bool
	CanonicalLocal bool
}

func (r *resolver) isDependencySource(sourceFile string) bool {
	return sourceFile != "" && sourceFile != r.sourceName
}

// localReference canonicalizes an absolute or current-package-qualified
// protobuf name to the registry-relative spelling used for local declarations.
//
// A package-looking relative name may instead name an ordinary nested
// declaration. Resolve that spelling first, matching protobuf's lexical rules,
// and only treat it as package-qualified when no declaration captures it.
func (r *resolver) localReference(scope, protoType string) (string, bool) {
	normalized := strings.TrimLeft(protoType, ".")
	absolute := normalized != protoType
	reference := TypeReference(normalized)
	if !absolute {
		if _, found := r.local.resolve(scope, reference); found {
			return reference, false
		}
	}

	protoPackage := strings.TrimLeft(r.protoPackage, ".")
	if protoPackage != "" {
		packagePrefix := protoPackage + "."
		if strings.HasPrefix(normalized, packagePrefix) {
			return TypeReference(strings.TrimPrefix(normalized, packagePrefix)), true
		}
	}
	return reference, absolute
}

// resolve looks a reference up in the source that declared it. A field the
// parser resolved across files is looked up in that file's namespace only;
// anything else resolves lexically from scope outward, as proto does.
func (r *resolver) resolve(use typeUse, scope, reference string) typeResolution {
	if r.isDependencySource(use.SourceFile) {
		info, found := r.imported[use.SourceFile][reference]
		return typeResolution{Info: info, Reference: reference, Found: found}
	}
	localReference, canonical := r.localReference(scope, use.ProtoType)
	if canonical {
		info, found := r.local.resolve("", localReference)
		return typeResolution{
			Info:           info,
			Reference:      localReference,
			Found:          found,
			CanonicalLocal: true,
		}
	}
	info, found := r.local.resolve(scope, localReference)
	return typeResolution{Info: info, Reference: localReference, Found: found}
}

func (r *resolver) namedValuePlan(use typeUse, scope string) (valuePlan, error) {
	isDependency := r.isDependencySource(use.SourceFile)
	// Inside the declaring class the reference is emitted as the schema wrote
	// it: Foundry resolves inner type names lexically, exactly as proto does.
	// Outside it, the registry's scoped reference is what resolves.
	if isDependency {
		if err := r.dependencyErrors[use.SourceFile]; err != nil {
			return valuePlan{}, err
		}
		if r.unnamespaced[use.SourceFile] {
			return valuePlan{}, fmt.Errorf(
				"%s is declared in %s, which has no usable namespace: give it a package or a valid (foundrytools.namespace) option",
				use.ProtoType, use.SourceFile)
		}
	}
	protoReference := TypeReference(use.ProtoType)
	namer := r.localNamer
	if isDependency {
		namer = r.dependencyNamers[use.SourceFile]
	}
	emittedReference := namer.Reference(use.ProtoType)
	resolution := r.resolve(use, scope, protoReference)
	info, found := resolution.Info, resolution.Found
	if found && resolution.CanonicalLocal {
		protoReference = resolution.Reference
		emittedReference = info.Reference
	}
	if !found {
		// The descriptor-driven plugin path can hand over a reference whose
		// declaration is not in the request. The parser still told us whether
		// it is an enum and what its values are, which is everything the wire
		// framing needs; only the scoped reference degrades to the lexical one.
		namespace := ""
		if isDependency {
			namespace = r.namespaces[use.SourceFile]
		}
		info = typeInfo{
			ProtoReference: protoReference,
			Reference:      emittedReference,
			IsEnum:         use.IsEnum,
			ZeroCase:       zeroValueNameOf(use.EnumValues),
			Namespace:      namespace,
		}
	} else if isDependency {
		if declaration, declared := r.importedDeclarations[use.SourceFile].resolve(
			use.FullProtoPath,
			use.ProtoType,
		); declared {
			r.collisions.AddDependency(declaration)
		}
	}
	// An imported type is named by the short reference the import makes
	// available, unless something else would answer to that name too, in which
	// case only the namespace-qualified spelling picks out the declaration.
	lexical, qualified := emittedReference, info.Reference
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
	Doc      []string
	Position protoast.Position
	Field    string
	RawField string
	Escape   memberEscape
	Type     string
	Members  []fieldPlan
}

type enumPlan struct {
	Name string
	Enum *protoast.Enum
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
	Enums  []enumPlan
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
func planMessage(
	message *protoast.Message,
	protoParentScope, generatedParentScope string,
	protoOwnerIdentity string,
	resolve *resolver,
) (messagePlan, error) {
	protoScope := TypeName(message.Name)
	if protoParentScope != "" {
		protoScope = protoParentScope + "." + protoScope
	}
	name := resolve.localNamer.Name(message.Name)
	generatedScope := name
	if generatedParentScope != "" {
		generatedScope = generatedParentScope + "." + generatedScope
	}
	plans := make([]fieldPlan, 0, len(message.Fields)+len(message.Maps))

	for _, field := range message.Fields {
		plan, err := planField(field, message.Name, protoScope, resolve)
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
		caseType := oneofTypeName(generatedScope, oneof)
		resolve.collisions.AddLocal(declarationInfo{
			SourceName:    resolve.sourceName,
			Position:      oneof.Position,
			Kind:          "oneof enum",
			ProtoName:     protoOwnerIdentity + "." + oneof.Name,
			GeneratedName: caseType,
		})
		groupName := planMemberName(oneof.Name)
		members := make([]fieldPlan, 0, len(oneof.Fields))
		for _, field := range oneof.Fields {
			plan, err := planField(field, message.Name, protoScope, resolve)
			if err != nil {
				return messagePlan{}, err
			}
			// A oneof member is only ever set through the union, so it has no
			// independent presence of its own.
			alternativeName := planOneofAlternativeName(field.Name)
			plan.Name = alternativeName.Generated
			plan.Escape = alternativeName.Escape
			plan.Cardinality = cardinalitySingular
			plan.OneofCaseName = TypeName(field.Name)
			plan.OneofCase = caseType + "." + plan.OneofCaseName
			plan.OneofField = groupName.Generated
			plan.LocalStem = oneof.Name + "_" + field.Name
			members = append(members, plan)
			plans = append(plans, plan)
		}
		oneofs = append(oneofs, oneofPlan{
			Doc:      oneof.Doc,
			Position: oneof.Position,
			Field:    groupName.Generated,
			RawField: oneof.Name,
			Escape:   groupName.Escape,
			Type:     caseType,
			Members:  members,
		})
	}

	for _, mapField := range message.Maps {
		plan, err := planMapField(mapField, message.Name, protoScope, resolve)
		if err != nil {
			return messagePlan{}, err
		}
		plans = append(plans, plan)
	}

	enums := make([]enumPlan, 0, len(message.NestedEnums))
	for _, enum := range message.NestedEnums {
		enums = append(enums, enumPlan{
			Name: resolve.localNamer.Name(enum.Name),
			Enum: enum,
		})
	}

	nested := make([]messagePlan, 0, len(message.NestedMessages))
	for _, child := range message.NestedMessages {
		childPlan, err := planMessage(
			child,
			protoScope,
			generatedScope,
			protoOwnerIdentity+"."+child.Name,
			resolve,
		)
		if err != nil {
			return messagePlan{}, err
		}
		nested = append(nested, childPlan)
	}

	sortPlansByNumber(plans)
	plan := messagePlan{
		Doc:    message.Doc,
		Name:   name,
		Scope:  protoScope,
		Fields: plans,
		Oneofs: oneofs,
		Enums:  enums,
		Nested: nested,
	}
	resolve.memberCollisions.addMessage(resolve.sourceName, protoOwnerIdentity, plans, oneofs)
	return plan, nil
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
		ProtoType:     field.FieldType,
		FullProtoPath: field.FullTypePath,
		IsEnum:        field.IsEnum,
		EnumValues:    field.EnumValues,
		SourceFile:    field.SourceFile,
	}, scope)
	if err != nil {
		return fieldPlan{}, fmt.Errorf("field %s.%s: %w", messageName, field.Name, err)
	}
	memberName := planMemberName(field.Name)
	plan := fieldPlan{
		Doc:      field.Doc,
		Position: field.Position,
		Kind:     "field",
		Name:     memberName.Generated,
		RawName:  field.Name,
		Escape:   memberName.Escape,
		Number:   field.Number,
		Value:    value,
	}
	switch {
	case field.Repeated:
		plan.Cardinality = cardinalityRepeated
		plan.Packable = value.isPackable()
		plan.Packed, err = packingFor(field, value)
		if err != nil {
			return fieldPlan{}, fmt.Errorf("field %s.%s: %w", messageName, field.Name, err)
		}
	case field.Optional:
		plan.Cardinality = cardinalityOptional
	default:
		plan.Cardinality = cardinalitySingular
	}
	return plan, nil
}

// packingFor decides how a repeated field is written. proto3 packs every
// numeric and enum value by default, and `[packed = ...]` overrides that
// choice for the encoder only — a decoder keeps accepting both encodings.
// `[packed = true]` on a value that has no packed form is rejected rather
// than quietly ignored, matching protoc.
func packingFor(field *protoast.Field, value valuePlan) (bool, error) {
	packable := value.isPackable()
	requested, declared := field.Options["packed"].(bool)
	if !declared {
		return packable, nil
	}
	if requested && !packable {
		return false, fmt.Errorf(
			"[packed = true] is only valid on a repeated numeric or enum field, not %s",
			field.FieldType,
		)
	}
	return requested, nil
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
		ProtoType:     mapField.ValueType,
		FullProtoPath: mapField.FullValueTypePath,
		IsEnum:        mapField.ValueIsEnum,
		SourceFile:    mapField.ValueSourceFile,
	}, scope)
	if err != nil {
		return fieldPlan{}, fmt.Errorf("field %s.%s: %w", messageName, mapField.Name, err)
	}
	memberName := planMemberName(mapField.Name)
	return fieldPlan{
		Doc:         mapField.Doc,
		Position:    mapField.Position,
		Kind:        "map field",
		Name:        memberName.Generated,
		RawName:     mapField.Name,
		Escape:      memberName.Escape,
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
