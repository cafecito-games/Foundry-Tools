package fsgenerator

import (
	"fmt"

	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func toBytesFunction(fields []*protoast.Field) fsast.Func {
	body := []fsast.Node{rawLine("\tvar result: PackedByteArray = PackedByteArray()")}
	for _, field := range fields {
		body = append(body, serializeField(field)...)
	}
	body = append(body, fsast.Return{Value: "result"})
	return fsast.Func{
		Name:       "to_bytes",
		ReturnType: fstypes.Named("PackedByteArray"),
		Body:       body,
	}
}

func serializeField(field *protoast.Field) []fsast.Node {
	tag := field.Number << 3
	wireType := 0
	if field.FieldType == "string" {
		wireType = 2
	}
	tag |= wireType

	backing := "_" + field.Name
	switch field.FieldType {
	case "string":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\tif %s != \"\":", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)),
			rawLine(fmt.Sprintf("\t\tvar %s_data: PackedByteArray = foundry.proto.Wire.encode_string(%s)", field.Name, backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s_data.size()))", field.Name)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(%s_data)", field.Name)),
		}
	default:
		return []fsast.Node{
			rawLine(fmt.Sprintf("\tif %s != 0:", backing)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)),
			rawLine(fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s))", backing)),
		}
	}
}

func rawLine(code string) fsast.Raw {
	return fsast.Raw{Code: code + "\n"}
}
