package fsgenerator

import (
	"fmt"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

func toBytesFunction(fields []*protoast.Field) fsast.Func {
	body := []fsast.Node{rawLine("\tvar result: PackedByteArray = PackedByteArray()")}
	for _, field := range fields {
		body = append(body, serializeField(field)...)
	}
	body = append(body, fsast.Return{Value: "result"})
	return fsast.Func{
		Doc:        toBytesDoc(),
		Name:       "to_bytes",
		ReturnType: fstypes.Named("PackedByteArray"),
		Body:       body,
	}
}

func serializeField(field *protoast.Field) []fsast.Node {
	tag := field.Number<<3 | wireTypeForField(field.FieldType)

	backing := "_" + field.Name
	switch field.FieldType {
	case "bool":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\tif %s:", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(1 if %s else 0))", backing)),
		}
	case "string":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\tif %s != \"\":", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)),
			rawLine(fmt.Sprintf("\t\tvar %s_data: PackedByteArray = foundry.proto.Wire.encode_string(%s)", field.Name, backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s_data.size()))", field.Name)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(%s_data)", field.Name)),
		}
	case "bytes":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\tif %s.size() > 0:", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s.size()))", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(%s)", backing)),
		}
	default:
		return []fsast.Node{
			rawLine(fmt.Sprintf("\tif %s != 0:", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s))", backing)),
		}
	}
}

func wireTypeForField(fieldType string) int {
	switch fieldType {
	case "string", "bytes":
		return 2
	default:
		return 0
	}
}

func rawLine(code string) fsast.Raw {
	return fsast.Raw{Code: code + "\n"}
}
