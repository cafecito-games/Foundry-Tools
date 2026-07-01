package fsgenerator

import (
	"fmt"
	"strconv"
	"strings"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
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
		case "float", "double", "fixed32", "fixed64", "sfixed32", "sfixed64", "sint32", "sint64":
			return fmt.Errorf("unsupported scalar type %s for wire generation", field.FieldType)
		}
	}
	return nil
}

func messageDoc(typeName string, schemaDoc []string) []string {
	return docOrFallback(schemaDoc, []string{"Generated protobuf message binding for " + typeName + "."})
}

func enumDoc(typeName string, schemaDoc []string) []string {
	return docOrFallback(schemaDoc, []string{"Generated protobuf enum binding for " + typeName + "."})
}

func setterDoc(fieldName string) []string {
	return []string{"Sets the " + fieldName + " protobuf field."}
}

func getterDoc(fieldName string) []string {
	return []string{"Returns the " + fieldName + " protobuf field."}
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

func writeDoc(builder *strings.Builder, indent string, lines []string) {
	for _, line := range lines {
		builder.WriteString(indent)
		builder.WriteString("##")
		if line != "" {
			builder.WriteByte(' ')
			builder.WriteString(line)
		}
		builder.WriteByte('\n')
	}
}

func renderEnum(namespace, typeName string, enum *protoast.Enum) string {
	var builder strings.Builder
	builder.WriteString("enum_name ")
	builder.WriteString(typeName)
	builder.WriteString(" {\n")
	for _, value := range enum.Values {
		writeDoc(&builder, "\t", docOrFallback(value.Doc, nil))
		builder.WriteByte('\t')
		builder.WriteString(value.Name)
		builder.WriteString(" = ")
		builder.WriteString(strconv.Itoa(value.Number))
		builder.WriteString(",\n")
	}
	builder.WriteString("}\n")

	return fsast.File{
		Namespace: namespace,
		Declarations: []fsast.Node{
			fsast.Doc{
				Lines: enumDoc(typeName, enum.Doc),
				Node:  fsast.Raw{Code: builder.String()},
			},
		},
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
				Doc:     messageDoc(typeName, message.Doc),
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
