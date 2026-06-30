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
		rawLine("\t\tvar wire_type: int = foundry.proto.Wire.get_wire_type(tag_read.value)"),
		rawLine("\t\tvar field_number: int = foundry.proto.Wire.get_field_number(tag_read.value)"),
		rawLine("\t\tmatch field_number:"),
	}
	for _, field := range fields {
		body = append(body, deserializeField(field)...)
	}
	body = append(body,
		rawLine("\t\t\t_:"),
		rawLine("\t\t\t\tmatch wire_type:"),
		rawLine("\t\t\t\t\tfoundry.proto.Wire.WIRE_VARINT:"),
		rawLine("\t\t\t\t\t\tvar skipped_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
		rawLine("\t\t\t\t\t\tif skipped_read.error != foundry.proto.ProtobufError.OK:"),
		rawLine("\t\t\t\t\t\t\treturn skipped_read.error"),
		rawLine("\t\t\t\t\t\toffset = skipped_read.offset"),
		rawLine("\t\t\t\t\tfoundry.proto.Wire.WIRE_LENGTH_DELIMITED:"),
		rawLine("\t\t\t\t\t\tvar length_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
		rawLine("\t\t\t\t\t\tif length_read.error != foundry.proto.ProtobufError.OK:"),
		rawLine("\t\t\t\t\t\t\treturn length_read.error"),
		rawLine("\t\t\t\t\t\toffset = length_read.offset"),
		rawLine("\t\t\t\t\t\tif offset + length_read.value > data.size():"),
		rawLine("\t\t\t\t\t\t\treturn foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
		rawLine("\t\t\t\t\t\toffset += length_read.value"),
		rawLine("\t\t\t\t\tfoundry.proto.Wire.WIRE_32BIT:"),
		rawLine("\t\t\t\t\t\tif offset + 4 > data.size():"),
		rawLine("\t\t\t\t\t\t\treturn foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
		rawLine("\t\t\t\t\t\toffset += 4"),
		rawLine("\t\t\t\t\tfoundry.proto.Wire.WIRE_64BIT:"),
		rawLine("\t\t\t\t\t\tif offset + 8 > data.size():"),
		rawLine("\t\t\t\t\t\t\treturn foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
		rawLine("\t\t\t\t\t\toffset += 8"),
		rawLine("\t\t\t\t\t_:"),
		rawLine("\t\t\t\t\t\treturn foundry.proto.ProtobufError.WIRE_TYPE_MISMATCH"),
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
	case "bool":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\t\t\t%d:", field.Number)),
			rawLine("\t\t\t\tvar value_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
			rawLine("\t\t\t\tif value_read.error != foundry.proto.ProtobufError.OK:"),
			rawLine("\t\t\t\t\treturn value_read.error"),
			rawLine(fmt.Sprintf("\t\t\t\t_%s = value_read.value != 0", field.Name)),
			rawLine("\t\t\t\toffset = value_read.offset"),
		}
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
	case "bytes":
		return []fsast.Node{
			rawLine(fmt.Sprintf("\t\t\t%d:", field.Number)),
			rawLine("\t\t\t\tvar length_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"),
			rawLine("\t\t\t\tif length_read.error != foundry.proto.ProtobufError.OK:"),
			rawLine("\t\t\t\t\treturn length_read.error"),
			rawLine("\t\t\t\toffset = length_read.offset"),
			rawLine("\t\t\t\tvar bytes_read: FieldRead[PackedByteArray] = foundry.proto.Wire.decode_bytes(data, offset, length_read.value)"),
			rawLine("\t\t\t\tif bytes_read.error != foundry.proto.ProtobufError.OK:"),
			rawLine("\t\t\t\t\treturn bytes_read.error"),
			rawLine(fmt.Sprintf("\t\t\t\t_%s = bytes_read.value", field.Name)),
			rawLine("\t\t\t\toffset = bytes_read.offset"),
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
