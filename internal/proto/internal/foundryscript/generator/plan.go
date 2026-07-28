package fsgenerator

import (
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
	Type      fstypes.Type
	// ZeroValue is the Foundry Script literal for the proto default.
	ZeroValue string
	WireType  int
}

// fieldPlan is a fully resolved message field.
type fieldPlan struct {
	Doc         []string
	Name        string
	Number      int
	Cardinality cardinality
	Value       valuePlan
	Key         valuePlan
	// Packed marks a repeated field that proto3 encodes as a single
	// length-delimited run of varints.
	Packed bool
	// OneofCase is the tagged-union case name when this field is a oneof
	// member; empty otherwise.
	OneofCase string
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
	wireType := p.Value.WireType
	if p.Cardinality == cardinalityMap || p.Packed {
		wireType = wireLengthDelimited
	}
	return p.Number<<3 | wireType
}

// enumRegistry maps a fully qualified enum reference to the name of its
// zero-valued case. Membership is also how the generator tells an enum-typed
// field from a message-typed one: the parser only populates Field.IsEnum for
// references it had to resolve across files.
type enumRegistry map[string]string

func (r enumRegistry) register(reference string, enum *protoast.Enum) {
	r[reference] = zeroValueName(enum)
}

// resolve looks reference up from the innermost scope outward, matching
// proto's own name resolution, and reports its zero-value case if it is an
// enum at all.
func (r enumRegistry) resolve(scope, reference string) (string, bool) {
	for prefix := scope; ; {
		candidate := reference
		if prefix != "" {
			candidate = prefix + "." + reference
		}
		if zero, ok := r[candidate]; ok {
			return zero, true
		}
		if prefix == "" {
			return "", false
		}
		if cut := strings.LastIndex(prefix, "."); cut >= 0 {
			prefix = prefix[:cut]
		} else {
			prefix = ""
		}
	}
}

// collectEnums walks the file, recording every enum reachable by name,
// including nested enums under their qualified `Outer.Kind` reference.
func collectEnums(file *protoast.ProtoFile) enumRegistry {
	registry := enumRegistry{}
	if file == nil {
		return registry
	}
	for _, enum := range file.Enums {
		registry.register(TypeName(enum.Name), enum)
	}
	for _, message := range file.Messages {
		collectMessageEnums(registry, TypeName(message.Name), message)
	}
	return registry
}

func collectMessageEnums(registry enumRegistry, prefix string, message *protoast.Message) {
	for _, enum := range message.NestedEnums {
		registry.register(prefix+"."+TypeName(enum.Name), enum)
	}
	for _, nested := range message.NestedMessages {
		collectMessageEnums(registry, prefix+"."+TypeName(nested.Name), nested)
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
	return valuePlan{
		Kind:      kindScalar,
		ProtoType: protoType,
		Type:      ScalarType(protoType),
		ZeroValue: scalarZeroValue(protoType),
		WireType:  scalarWireType(protoType),
	}
}

func namedValuePlan(protoType string, isEnum bool, scope string, enums enumRegistry) valuePlan {
	// The reference is emitted as the schema wrote it: Foundry resolves inner
	// type names lexically, exactly as proto does.
	reference := TypeReference(protoType)
	zero, found := enums.resolve(scope, reference)
	if !found && !isEnum {
		return valuePlan{
			Kind:      kindMessage,
			ProtoType: protoType,
			Type:      fstypes.Named(reference),
			ZeroValue: "null",
			WireType:  wireLengthDelimited,
		}
	}
	return valuePlan{
		Kind:      kindEnum,
		ProtoType: protoType,
		Type:      fstypes.Named(reference),
		ZeroValue: reference + "." + zero,
		WireType:  wireVarint,
	}
}

func valuePlanFor(protoType string, isEnum bool, scope string, enums enumRegistry) valuePlan {
	switch protoType {
	case "int32", "int64", "uint32", "uint64", "sint32", "sint64",
		"fixed32", "fixed64", "sfixed32", "sfixed64",
		"double", "float", "bool", "string", "bytes":
		return scalarValuePlan(protoType)
	default:
		return namedValuePlan(protoType, isEnum, scope, enums)
	}
}

// oneofPlan is one proto oneof, emitted as a nullable tagged-union member.
type oneofPlan struct {
	Doc     []string
	Field   string
	Type    string
	Members []fieldPlan
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

// planMessage resolves every member of a message into emitter-ready plans. The
// field list is in field-number order so serialization is deterministic and
// the emitted match arms read in wire order.
func planMessage(message *protoast.Message, parentScope string, enums enumRegistry) messagePlan {
	scope := TypeName(message.Name)
	if parentScope != "" {
		scope = parentScope + "." + scope
	}
	plans := make([]fieldPlan, 0, len(message.Fields)+len(message.Maps))

	for _, field := range message.Fields {
		plans = append(plans, planField(field, scope, enums))
	}

	oneofs := make([]oneofPlan, 0, len(message.Oneofs))
	for _, oneof := range message.Oneofs {
		caseType := oneofTypeName(scope, oneof)
		members := make([]fieldPlan, 0, len(oneof.Fields))
		for _, field := range oneof.Fields {
			plan := planField(field, scope, enums)
			// A oneof member is only ever set through the union, so it has no
			// independent presence of its own.
			plan.Cardinality = cardinalitySingular
			plan.OneofCase = caseType + "." + TypeName(field.Name)
			plan.OneofField = oneof.Name
			members = append(members, plan)
			plans = append(plans, plan)
		}
		oneofs = append(oneofs, oneofPlan{
			Doc:     oneof.Doc,
			Field:   oneof.Name,
			Type:    caseType,
			Members: members,
		})
	}

	for _, mapField := range message.Maps {
		plans = append(plans, planMapField(mapField, scope, enums))
	}

	nested := make([]messagePlan, 0, len(message.NestedMessages))
	for _, child := range message.NestedMessages {
		nested = append(nested, planMessage(child, scope, enums))
	}

	sortPlansByNumber(plans)
	return messagePlan{
		Doc:    message.Doc,
		Name:   TypeName(message.Name),
		Scope:  scope,
		Fields: plans,
		Oneofs: oneofs,
		Enums:  message.NestedEnums,
		Nested: nested,
	}
}

func planField(field *protoast.Field, scope string, enums enumRegistry) fieldPlan {
	value := valuePlanFor(field.FieldType, field.IsEnum, scope, enums)
	plan := fieldPlan{
		Doc:    field.Doc,
		Name:   field.Name,
		Number: field.Number,
		Value:  value,
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
	return plan
}

func planMapField(mapField *protoast.MapField, scope string, enums enumRegistry) fieldPlan {
	return fieldPlan{
		Doc:         mapField.Doc,
		Name:        mapField.Name,
		Number:      mapField.Number,
		Cardinality: cardinalityMap,
		Key:         valuePlanFor(mapField.KeyType, false, scope, enums),
		Value:       valuePlanFor(mapField.ValueType, mapField.ValueIsEnum, scope, enums),
	}
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
