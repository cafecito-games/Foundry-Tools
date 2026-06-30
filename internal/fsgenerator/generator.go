package fsgenerator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

// FileEntry represents an imported proto file available to the generator.
type FileEntry struct {
	File     *protoast.ProtoFile
	Filename string
}

// Generate renders top-level message and enum skeletons for a proto file.
func Generate(file *protoast.ProtoFile, sourceName string, _ []FileEntry) (GeneratedFiles, error) {
	_ = sourceName

	namespace := NamespaceFor(file)
	if err := ValidateNamespace(namespace); err != nil {
		return nil, err
	}

	files := GeneratedFiles{}
	if file == nil {
		return files, nil
	}

	for _, enum := range file.Enums {
		typeName := TypeName(enum.Name)
		files[outputPath(namespace, typeName)] = renderEnum(namespace, typeName, enum)
	}
	for _, message := range file.Messages {
		if err := validateWireFields(message); err != nil {
			return nil, err
		}
		typeName := TypeName(message.Name)
		source := renderMessage(namespace, typeName, message)
		if err := CheckPublicAPI(source); err != nil {
			return nil, err
		}
		files[outputPath(namespace, typeName)] = source
	}

	return files, nil
}

func validateWireFields(message *protoast.Message) error {
	for _, field := range message.Fields {
		switch field.FieldType {
		case "float", "double", "fixed32", "fixed64", "sfixed32", "sfixed64":
			return fmt.Errorf("unsupported scalar type %s for wire generation", field.FieldType)
		}
	}
	return nil
}

func renderEnum(namespace, typeName string, enum *protoast.Enum) string {
	var builder strings.Builder
	builder.WriteString("enum_name ")
	builder.WriteString(typeName)
	builder.WriteString(" {\n")
	for _, value := range enum.Values {
		builder.WriteByte('\t')
		builder.WriteString(value.Name)
		builder.WriteString(" = ")
		builder.WriteString(strconv.Itoa(value.Number))
		builder.WriteString(",\n")
	}
	builder.WriteString("}\n")

	return fsast.File{
		Namespace:    namespace,
		Declarations: []fsast.Node{fsast.Raw{Code: builder.String()}},
	}.Render()
}

func renderMessage(namespace, typeName string, message *protoast.Message) string {
	members := make([]fsast.Node, 0, len(message.Fields)*3+3)
	for _, field := range message.Fields {
		members = append(members, fsast.Var{
			Name:  "_" + field.Name,
			Type:  fieldType(field),
			Value: fieldDefaultValue(field.FieldType),
		})
		members = append(members, fieldMembers(field)...)
	}
	members = append(members,
		fromBytesFactory(typeName),
		toBytesFunction(message.Fields),
		mergeFromBytesFunction(message.Fields),
	)

	return fsast.File{
		Namespace: namespace,
		Imports:   []string{"foundry.proto"},
		Declarations: []fsast.Node{
			// Current Foundry builds cannot resolve/apply imported runtime trait bodies
			// such as foundry.proto.Message[T] here, so conformance is deferred.
			fsast.Class{
				Final:   true,
				Name:    typeName,
				Extends: "RefCounted",
				Members: members,
			},
		},
	}.Render()
}

func fieldDefaultValue(protoType string) string {
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
