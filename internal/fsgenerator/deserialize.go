package fsgenerator

import (
	"fmt"

	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func mergeFromBytesFunction(fields []*protoast.Field) fsast.Func {
	body := []fsast.Node{
		rawLine("\tvar offset: int = 0"),
		rawLine("\twhile offset < data.size():"),
		rawLine("\t\tvar tag_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
		rawLine("\t\tif tag_read.error != foundry.proto.ProtobufError.OK:"),
		rawLine("\t\t\treturn tag_read.error"),
		rawLine("\t\toffset = tag_read.offset"),
		rawLine("\t\tvar field_number: int = foundry.proto.Wire.get_field_number(tag_read.value)"),
		rawLine("\t\tmatch field_number:"),
	}
	for _, field := range fields {
		body = append(body, deserializeField(field)...)
	}
	body = append(body,
		rawLine("\t\t\t_:"),
		rawLine("\t\t\t\treturn foundry.proto.ProtobufError.UNKNOWN_REQUIRED_FEATURE"),
		fsast.Return{Value: "foundry.proto.ProtobufError.OK"},
	)
	return fsast.Func{
		Name:       "merge_from_bytes",
		Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Named("foundry.proto.ProtobufError"),
		Body:       body,
	}
}

func deserializeField(field *protoast.Field) []fsast.Node {
	switch field.FieldType {
	case "string":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\t\t\t%d:", field.Number)),
			rawLine("\t\t\t\tvar length_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
			rawLine("\t\t\t\tif length_read.error != foundry.proto.ProtobufError.OK:"),
			rawLine("\t\t\t\t\treturn length_read.error"),
			rawLine("\t\t\t\toffset = length_read.offset"),
			rawLine("\t\t\t\tvar string_read: FieldRead[String] = foundry.proto.Wire.decode_string(data, offset, length_read.value)"),
			rawLine("\t\t\t\tif string_read.error != foundry.proto.ProtobufError.OK:"),
			rawLine("\t\t\t\t\treturn string_read.error"),
			rawLine(fmt.Sprintf("\t\t\t\t_%s = string_read.value", field.Name)),
			rawLine("\t\t\t\toffset = string_read.offset"),
		}
	default:
		return []fsast.Node{
			rawLine(fmt.Sprintf("\t\t\t%d:", field.Number)),
			rawLine("\t\t\t\tvar value_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
			rawLine("\t\t\t\tif value_read.error != foundry.proto.ProtobufError.OK:"),
			rawLine("\t\t\t\t\treturn value_read.error"),
			rawLine(fmt.Sprintf("\t\t\t\t_%s = value_read.value", field.Name)),
			rawLine("\t\t\t\toffset = value_read.offset"),
		}
	}
}
