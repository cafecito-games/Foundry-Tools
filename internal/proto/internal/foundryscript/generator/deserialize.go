package fsgenerator

import (
	"fmt"

	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

func mergeFromBytesFunction(plans []fieldPlan) fsast.Func {
	body := []fsast.Node{
		line(0, "var offset: int = 0"),
		line(0, "while offset < data.size():"),
		line(1, "var tag_read: VarintRead = Wire.decode_varint(data, offset)"),
		line(1, "if tag_read.error != ProtobufError.OK:"),
		line(2, "return tag_read.error"),
		line(1, "offset = tag_read.offset"),
		line(1, "var wire_type: int = Wire.get_wire_type(tag_read.value)"),
		line(1, "match Wire.get_field_number(tag_read.value):"),
	}
	for i := range plans {
		body = append(body, line(2, fmt.Sprintf("%d:", plans[i].Number)))
		body = append(body, deserializeField(&plans[i])...)
	}
	// Unknown fields must be tolerated; the skip policy lives in the runtime.
	body = append(body,
		line(2, "_:"),
		line(3, "var skipped: SkipRead = Wire.skip_field(data, offset, wire_type)"),
		line(3, "if skipped.error != ProtobufError.OK:"),
		line(4, "return skipped.error"),
		line(3, "offset = skipped.offset"),
		fsast.Return{Value: "ProtobufError.OK"},
	)
	return fsast.Func{
		Doc:        mergeFromBytesDoc(),
		Name:       "merge_from_bytes",
		Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Named("ProtobufError"),
		Body:       body,
	}
}

func deserializeField(plan *fieldPlan) []fsast.Node {
	switch {
	case plan.Cardinality == cardinalityMap:
		return deserializeMap(plan)
	case plan.Cardinality == cardinalityRepeated && plan.Packed:
		return deserializePackedRepeated(plan)
	default:
		return decodeValue(3, plan.Value, plan.Name, assignmentFor(plan))
	}
}

// assignmentFor builds the statement that stores one decoded value, which is
// the only thing that differs between a scalar field, a repeated element, and
// a oneof case.
func assignmentFor(plan *fieldPlan) func(expression string) string {
	switch {
	case plan.OneofCase != "":
		return func(expression string) string {
			return fmt.Sprintf("%s = %s(%s)", plan.OneofField, plan.OneofCase, expression)
		}
	case plan.Cardinality == cardinalityRepeated:
		return func(expression string) string {
			return fmt.Sprintf("%s.append(%s)", plan.Name, expression)
		}
	default:
		return func(expression string) string {
			return fmt.Sprintf("%s = %s", plan.Name, expression)
		}
	}
}

// decodeValue emits the wire-type guard, the read, and the store for one value.
func decodeValue(depth int, value valuePlan, name string, assign func(string) string) []fsast.Node {
	nodes := []fsast.Node{
		line(depth, fmt.Sprintf("if wire_type != %s:", wireTypeConstant(value.WireType))),
		line(depth+1, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	}
	return append(nodes, readValue(depth, value, name, assign, "wire_type")...)
}

// readValue emits the read and store for one value, without the wire-type
// guard, so map entries and repeated elements can reuse it.
func readValue(depth int, value valuePlan, name string, assign func(string) string, _ string) []fsast.Node {
	switch {
	case value.Kind == kindMessage:
		length := name + "_length"
		message := name + "_message"
		errorName := name + "_error"
		nodes := lengthPrefix(depth, length)
		return append(nodes,
			line(depth, fmt.Sprintf("var %s: %s = %s.new()", message, value.Type.Render(), value.Type.Render())),
			line(depth, fmt.Sprintf("var %s: ProtobufError = %s.merge_from_bytes(data.slice(offset, offset + %s.value))", errorName, message, length)),
			line(depth, fmt.Sprintf("if %s != ProtobufError.OK:", errorName)),
			line(depth+1, "return "+errorName),
			line(depth, assign(message)),
			line(depth, fmt.Sprintf("offset += %s.value", length)),
		)
	case value.ProtoType == "string", value.ProtoType == "bytes":
		length := name + "_length"
		read := name + "_read"
		carrier, decoder := "StringRead", "decode_string"
		if value.ProtoType == "bytes" {
			carrier, decoder = "BytesRead", "decode_bytes"
		}
		nodes := lengthPrefix(depth, length)
		return append(nodes,
			line(depth, fmt.Sprintf("var %s: %s = Wire.%s(data, offset, %s.value)", read, carrier, decoder, length)),
			line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", read)),
			line(depth+1, "return "+read+".error"),
			line(depth, assign(read+".value")),
			line(depth, fmt.Sprintf("offset = %s.offset", read)),
		)
	default:
		read := name + "_read"
		return []fsast.Node{
			line(depth, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(data, offset)", read)),
			line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", read)),
			line(depth+1, "return "+read+".error"),
			line(depth, assign(varintResultExpression(value, read+".value"))),
			line(depth, fmt.Sprintf("offset = %s.offset", read)),
		}
	}
}

// lengthPrefix reads and bounds-checks the length of a length-delimited field.
func lengthPrefix(depth int, length string) []fsast.Node {
	return []fsast.Node{
		line(depth, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(data, offset)", length)),
		line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", length)),
		line(depth+1, "return "+length+".error"),
		line(depth, fmt.Sprintf("offset = %s.offset", length)),
		line(depth, fmt.Sprintf("if %s.value < 0 or offset + %s.value > data.size():", length, length)),
		line(depth+1, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
	}
}

// deserializePackedRepeated accepts both encodings: a packed run and the
// unpacked one-record-per-element form, which protobuf parsers must tolerate
// regardless of how the sender chose to write the field.
func deserializePackedRepeated(plan *fieldPlan) []fsast.Node {
	length := plan.Name + "_length"
	end := plan.Name + "_end"
	packed := plan.Name + "_packed"
	assign := assignmentFor(plan)

	nodes := []fsast.Node{line(3, "if wire_type == Wire.WIRE_LENGTH_DELIMITED:")}
	nodes = append(nodes, lengthPrefix(4, length)...)
	nodes = append(nodes,
		line(4, fmt.Sprintf("var %s: int = offset + %s.value", end, length)),
		line(4, fmt.Sprintf("while offset < %s:", end)),
		line(5, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(data, offset)", packed)),
		line(5, fmt.Sprintf("if %s.error != ProtobufError.OK:", packed)),
		line(6, "return "+packed+".error"),
		line(5, assign(varintResultExpression(plan.Value, packed+".value"))),
		line(5, fmt.Sprintf("offset = %s.offset", packed)),
		line(3, fmt.Sprintf("elif wire_type == %s:", wireTypeConstant(plan.Value.WireType))),
	)
	nodes = append(nodes, readValue(4, plan.Value, plan.Name, assign, "wire_type")...)
	return append(nodes,
		line(3, "else:"),
		line(4, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	)
}

func deserializeMap(plan *fieldPlan) []fsast.Node {
	length := plan.Name + "_length"
	end := plan.Name + "_end"
	key := plan.Name + "_key"
	value := plan.Name + "_value"
	entryTag := plan.Name + "_entry_tag"
	entryWireType := plan.Name + "_entry_wire_type"

	nodes := []fsast.Node{
		line(3, "if wire_type != Wire.WIRE_LENGTH_DELIMITED:"),
		line(4, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	}
	nodes = append(nodes, lengthPrefix(3, length)...)
	nodes = append(nodes,
		line(3, fmt.Sprintf("var %s: int = offset + %s.value", end, length)),
		line(3, fmt.Sprintf("var %s: %s = %s", key, plan.Key.Type.Render(), plan.Key.ZeroValue)),
		line(3, fmt.Sprintf("var %s: %s = %s", value, mapValueType(plan.Value).Render(), mapValueZero(plan.Value))),
		line(3, fmt.Sprintf("while offset < %s:", end)),
		line(4, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(data, offset)", entryTag)),
		line(4, fmt.Sprintf("if %s.error != ProtobufError.OK:", entryTag)),
		line(5, "return "+entryTag+".error"),
		line(4, fmt.Sprintf("offset = %s.offset", entryTag)),
		line(4, fmt.Sprintf("var %s: int = Wire.get_wire_type(%s.value)", entryWireType, entryTag)),
		line(4, fmt.Sprintf("match Wire.get_field_number(%s.value):", entryTag)),
		line(5, "1:"),
		line(6, fmt.Sprintf("if %s != %s:", entryWireType, wireTypeConstant(plan.Key.WireType))),
		line(7, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	)
	nodes = append(nodes, readValue(6, plan.Key, key, func(expression string) string {
		return key + " = " + expression
	}, entryWireType)...)
	nodes = append(nodes,
		line(5, "2:"),
		line(6, fmt.Sprintf("if %s != %s:", entryWireType, wireTypeConstant(plan.Value.WireType))),
		line(7, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	)
	nodes = append(nodes, readValue(6, plan.Value, value, func(expression string) string {
		return value + " = " + expression
	}, entryWireType)...)
	return append(nodes,
		line(5, "_:"),
		line(6, fmt.Sprintf("var %s_skip: SkipRead = Wire.skip_field(data, offset, %s)", plan.Name, entryWireType)),
		line(6, fmt.Sprintf("if %s_skip.error != ProtobufError.OK:", plan.Name)),
		line(7, fmt.Sprintf("return %s_skip.error", plan.Name)),
		line(6, fmt.Sprintf("offset = %s_skip.offset", plan.Name)),
		line(3, fmt.Sprintf("%s[%s] = %s", plan.Name, key, value)),
	)
}

// mapValueZero is the placeholder a map entry starts from. A message-valued
// entry has no zero instance, so it starts null and is replaced by the read.
func mapValueZero(value valuePlan) string {
	if value.Kind == kindMessage {
		return "null"
	}
	return value.ZeroValue
}

func mapValueType(value valuePlan) fstypes.Type {
	if value.Kind == kindMessage {
		return fstypes.Nullable(value.Type)
	}
	return value.Type
}

// varintResultExpression converts a decoded varint back into the field type.
func varintResultExpression(value valuePlan, expression string) string {
	switch {
	case value.Kind == kindEnum:
		return value.Type.Render() + ".from_wire(" + expression + ")"
	case value.ProtoType == "bool":
		return expression + " != 0"
	default:
		return expression
	}
}

func wireTypeConstant(wireType int) string {
	if wireType == wireLengthDelimited {
		return "Wire.WIRE_LENGTH_DELIMITED"
	}
	return "Wire.WIRE_VARINT"
}
