package fsgenerator

import (
	"fmt"
	"strconv"
	"strings"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// FileEntry represents an imported proto file available to the generator.
type FileEntry struct {
	File     *protoast.ProtoFile
	Filename string
}

// Generate renders top-level message and enum skeletons for a proto file.
func Generate(file *protoast.ProtoFile, sourceName string, imports []FileEntry) (GeneratedFiles, error) {
	localNamer, err := newTypeNamer(file, sourceName)
	if err != nil {
		return nil, err
	}

	namespace := NamespaceFor(file)
	if err := ValidateNamespace(namespace); err != nil {
		return nil, err
	}

	files := GeneratedFiles{}
	if file == nil {
		return files, nil
	}

	resolve := newResolver(file, sourceName, imports, localNamer)

	enumPlans := make([]enumPlan, 0, len(file.Enums))
	for _, enum := range file.Enums {
		enumPlans = append(enumPlans, enumPlan{
			Name: localNamer.Name(enum.Name),
			Enum: enum,
		})
	}
	messagePlans := make([]messagePlan, 0, len(file.Messages))
	for _, message := range file.Messages {
		if err := validateWireFields(message); err != nil {
			return nil, err
		}
		plan, err := planMessage(
			message,
			"",
			"",
			qualifiedProtoName(file.Package, message.Name),
			resolve,
		)
		if err != nil {
			return nil, err
		}
		messagePlans = append(messagePlans, plan)
	}

	if err := resolve.collisions.Err(localNamer.prefix); err != nil {
		return nil, err
	}

	for i := range enumPlans {
		files[outputPath(namespace, enumPlans[i].Name)] = renderEnum(namespace, enumPlans[i].Name, enumPlans[i].Enum)
	}
	for i := range messagePlans {
		plan := &messagePlans[i]
		source := renderMessage(namespace, plan)
		if err := CheckPublicAPI(source); err != nil {
			return nil, err
		}
		files[outputPath(namespace, plan.Name)] = source

		unions := collectOneofs(plan)
		for i := range unions {
			files[outputPath(namespace, unions[i].Type)] = renderOneofUnion(namespace, &unions[i])
		}
	}

	return files, nil
}

// collectOneofs gathers the unions of a message and every message nested in it,
// each of which is emitted as its own file-level enum.
func collectOneofs(plan *messagePlan) []oneofPlan {
	oneofs := append([]oneofPlan(nil), plan.Oneofs...)
	for i := range plan.Nested {
		oneofs = append(oneofs, collectOneofs(&plan.Nested[i])...)
	}
	return oneofs
}

func renderOneofUnion(namespace string, oneof *oneofPlan) string {
	return fsast.File{
		Namespace:    namespace,
		Imports:      oneof.Namespaces(),
		Declarations: []fsast.Node{oneofUnion(oneof)},
	}.Render()
}

// validateWireFields refuses constructs the emitter cannot frame on the wire,
// so an unsupported schema fails loudly instead of producing a binding that
// silently drops the field. It recurses, since a nested message is emitted
// with the same machinery as its parent.
func validateWireFields(message *protoast.Message) error {
	for _, field := range message.Fields {
		if err := validateWireType(field.FieldType); err != nil {
			return err
		}
	}
	for _, oneof := range message.Oneofs {
		for _, field := range oneof.Fields {
			if err := validateWireType(field.FieldType); err != nil {
				return err
			}
			if field.Repeated {
				return fmt.Errorf("oneof %s field %s cannot be repeated", oneof.Name, field.Name)
			}
		}
	}
	for _, mapField := range message.Maps {
		if err := validateWireType(mapField.KeyType); err != nil {
			return err
		}
		if err := validateWireType(mapField.ValueType); err != nil {
			return err
		}
	}
	for _, nested := range message.NestedMessages {
		if err := validateWireFields(nested); err != nil {
			return err
		}
	}
	return nil
}

// validateWireType rejects the scalars whose wire encoding is not implemented.
// These need zig-zag or fixed-width framing rather than a plain varint, so
// emitting them as varints would produce silently wrong bytes.
func validateWireType(protoType string) error {
	switch protoType {
	case "float", "double", "fixed32", "fixed64", "sfixed32", "sfixed64", "sint32", "sint64":
		return fmt.Errorf("unsupported scalar type %s for wire generation", protoType)
	}
	return nil
}

func messageDoc(typeName string, schemaDoc []string) []string {
	return docOrFallback(schemaDoc, []string{"Generated protobuf message binding for " + typeName + "."})
}

func enumDoc(typeName string, schemaDoc []string) []string {
	return docOrFallback(schemaDoc, []string{"Generated protobuf enum binding for " + typeName + "."})
}

func fieldDoc(fieldName string) []string {
	return []string{"The " + fieldName + " protobuf field."}
}

func oneofDoc(oneofName string) []string {
	return []string{"The " + oneofName + " protobuf oneof; null when no case is set."}
}

func oneofUnionDoc(oneofName string) []string {
	return []string{"Cases of the " + oneofName + " protobuf oneof."}
}

func toWireDoc() []string {
	return []string{"Returns the protobuf wire value for this case."}
}

func fromWireDoc() []string {
	return []string{"Returns the case for a protobuf wire value, or null if it names none."}
}

func fromBytesDoc(typeName string) []string {
	return []string{"Decodes protobuf wire data into a new " + typeName + " message."}
}

func toBytesDoc() []string {
	return []string{"Serializes this message to protobuf wire data."}
}

func mergeFromBytesDoc() []string {
	return []string{"Merges protobuf wire data into this message."}
}

func docOrFallback(schemaDoc, fallback []string) []string {
	if len(schemaDoc) == 0 {
		return fallback
	}
	out := make([]string, 0, len(schemaDoc))
	for _, line := range schemaDoc {
		if strings.TrimSpace(line) == "" && len(out) == 0 {
			continue
		}
		out = append(out, strings.TrimSpace(line))
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func renderEnum(namespace, typeName string, enum *protoast.Enum) string {
	return fsast.File{
		Namespace: namespace,
		Declarations: []fsast.Node{
			enumDeclaration(typeName, enum, false),
		},
	}.Render()
}

// enumDeclaration builds an enum and hosts its wire conversion on it, so the
// raw int never leaks into the message bindings that reference the enum and
// the unknown-value policy proto3 requires is defined once.
func enumDeclaration(typeName string, enum *protoast.Enum, inner bool) fsast.Enum {
	values := make([]fsast.EnumValue, 0, len(enum.Values))
	for _, value := range enum.Values {
		values = append(values, fsast.EnumValue{
			Doc:    docOrFallback(value.Doc, nil),
			Name:   value.Name,
			Number: value.Number,
		})
	}
	return fsast.Enum{
		Doc:     enumDoc(typeName, enum.Doc),
		Inner:   inner,
		Name:    typeName,
		Values:  values,
		Members: enumWireFunctions(typeName, enum),
	}
}

func enumWireFunctions(typeName string, enum *protoast.Enum) []fsast.Node {
	if zeroValueName(enum) == "" {
		return nil
	}

	// Each case is declared with its own wire number, so the conversion out is
	// the cast and a match table could only ever drift from the constants.
	toWire := []fsast.Node{line(0, "return self as int")}

	fromWire := []fsast.Node{line(0, "match value:")}
	emitted := map[int]bool{}
	for _, value := range enum.Values {
		// An allow_alias enum declares the same number twice; the second arm
		// would be dead, so the first declared spelling wins.
		if emitted[value.Number] {
			continue
		}
		emitted[value.Number] = true
		fromWire = append(fromWire,
			line(1, strconv.Itoa(value.Number)+":"),
			line(2, "return "+typeName+"."+value.Name),
		)
	}
	fromWire = append(fromWire, line(1, "_:"), line(2, "return null"))

	return []fsast.Node{
		fsast.Func{
			Doc:        toWireDoc(),
			Name:       "to_wire",
			ReturnType: fstypes.Named("int"),
			Body:       toWire,
		},
		fsast.Func{
			Doc:        fromWireDoc(),
			Static:     true,
			Name:       "from_wire",
			Parameters: []fsast.Parameter{{Name: "value", Type: fstypes.Named("int")}},
			// Self is the spelling that resolves to the namespaced enum from
			// inside its own body; the bare name does not unify with it in
			// every position, notably a tagged-union case argument.
			ReturnType: fstypes.Nullable(fstypes.Named("Self")),
			Body:       fromWire,
		},
	}
}

// unknownFieldsMemberDeclaration is the buffer every message keeps unrecognized
// fields in.
func unknownFieldsMemberDeclaration() fsast.Node {
	return fsast.Doc{
		Lines: []string{"Fields this schema does not recognize, kept verbatim so a re-encode is lossless."},
		Node: fsast.Var{
			Name:  unknownFieldsMember,
			Type:  fstypes.Named("PackedByteArray"),
			Value: "PackedByteArray()",
		},
	}
}

func zeroValueName(enum *protoast.Enum) string {
	return zeroValueNameOf(enum.Values)
}

// zeroValueNameOf is the case a proto3 enum defaults to: the one numbered zero,
// which proto3 requires to be declared first.
func zeroValueNameOf(values []*protoast.EnumValue) string {
	for _, value := range values {
		if value.Number == 0 {
			return value.Name
		}
	}
	if len(values) > 0 {
		return values[0].Name
	}
	return ""
}

func renderMessage(namespace string, plan *messagePlan) string {
	return fsast.File{
		Namespace:    namespace,
		Imports:      append([]string{"foundry.proto"}, plan.Namespaces()...),
		Declarations: []fsast.Node{messageClass(plan, false)},
	}.Render()
}

func messageClass(plan *messagePlan, inner bool) fsast.Class {
	members := make([]fsast.Node, 0, len(plan.Fields)+len(plan.Oneofs)*2+4)

	// Nested types are declared before the members that reference them.
	for _, nested := range plan.Enums {
		members = append(members, enumDeclaration(nested.Name, nested.Enum, true))
	}
	for i := range plan.Nested {
		members = append(members, messageClass(&plan.Nested[i], true))
	}
	for i := range plan.Fields {
		if plan.Fields[i].OneofCase != "" {
			continue
		}
		members = append(members, fieldMember(&plan.Fields[i]))
	}
	for i := range plan.Oneofs {
		members = append(members, oneofMember(&plan.Oneofs[i]))
	}
	// The retained-value companions follow the public members they belong to;
	// an initializer does not run a setter, so nothing depends on the order.
	for i := range plan.Fields {
		if plan.Fields[i].RetainsUnknownEnum() {
			members = append(members, unknownEnumMember(&plan.Fields[i]))
		}
	}
	members = append(members,
		unknownFieldsMemberDeclaration(),
		fromBytesFactory(plan.Name),
		toBytesFunction(plan.Fields, plan.Oneofs),
		mergeFromBytesFunction(plan.Fields),
	)

	return fsast.Class{
		Doc:   messageDoc(plan.Name, plan.Doc),
		Inner: inner,
		// A nested binding is as much a leaf as a top-level one.
		Final:   true,
		Name:    plan.Name,
		Extends: "RefCounted",
		Uses:    []string{"Message"},
		Members: members,
	}
}
