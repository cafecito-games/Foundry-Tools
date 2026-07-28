package fsgenerator

import (
	"fmt"

	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

func toBytesFunction(plans []fieldPlan, oneofs []oneofPlan) fsast.Func {
	body := []fsast.Node{line(0, "var result: PackedByteArray = PackedByteArray()")}
	for i := range plans {
		if plans[i].OneofCase != "" {
			continue
		}
		body = append(body, serializeField(&plans[i])...)
	}
	for i := range oneofs {
		body = append(body, serializeOneof(&oneofs[i])...)
	}
	body = append(body, fsast.Return{Value: "result"})
	return fsast.Func{
		Doc:        toBytesDoc(),
		Name:       "to_bytes",
		ReturnType: fstypes.Named("PackedByteArray"),
		Body:       body,
	}
}

func serializeField(plan *fieldPlan) []fsast.Node {
	switch plan.Cardinality {
	case cardinalityRepeated:
		if plan.Packed {
			return serializePackedRepeated(plan)
		}
		return serializeUnpackedRepeated(plan)
	case cardinalityMap:
		return serializeMap(plan)
	default:
		return serializeSingular(plan)
	}
}

// presenceCondition is the guard deciding whether a singular field goes on the
// wire. Implicit-presence fields use proto3's zero-value rule; nullable ones
// are tested with `is`, not `!= null`: for a nullable built-in such as String?
// or int?, comparing against null reports the wrong answer, while `is` is
// correct for every kind, including message and tagged-union types.
func presenceCondition(plan *fieldPlan) string {
	if plan.Cardinality == cardinalityOptional || plan.Value.Kind == kindMessage {
		return plan.Name + " is " + plan.Value.Type.Render()
	}
	switch {
	case plan.Value.Kind == kindEnum:
		return plan.Name + " != " + plan.Value.ZeroValue
	case plan.Value.ProtoType == "bool":
		return plan.Name
	case plan.Value.ProtoType == "string":
		return plan.Name + ` != ""`
	case plan.Value.ProtoType == "bytes":
		return plan.Name + ".size() > 0"
	default:
		return plan.Name + " != 0"
	}
}

func serializeSingular(plan *fieldPlan) []fsast.Node {
	nodes := []fsast.Node{line(0, "if "+presenceCondition(plan)+":")}
	return append(nodes, appendValue(1, plan.Value, plan.Name, plan.Name, plan.Tag(), "result")...)
}

func serializeUnpackedRepeated(plan *fieldPlan) []fsast.Node {
	item := plan.Name + "_item"
	nodes := []fsast.Node{
		line(0, fmt.Sprintf("for %s: %s in %s:", item, plan.Value.Type.Render(), plan.Name)),
	}
	return append(nodes, appendValue(1, plan.Value, item, plan.Name, plan.Tag(), "result")...)
}

func serializePackedRepeated(plan *fieldPlan) []fsast.Node {
	item := plan.Name + "_item"
	buffer := plan.Name + "_data"
	return []fsast.Node{
		line(0, fmt.Sprintf("if %s.size() > 0:", plan.Name)),
		line(1, fmt.Sprintf("var %s: PackedByteArray = PackedByteArray()", buffer)),
		line(1, fmt.Sprintf("for %s: %s in %s:", item, plan.Value.Type.Render(), plan.Name)),
		line(2, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", buffer, varintExpression(plan.Value, item))),
		line(1, fmt.Sprintf("result.append_array(Wire.encode_varint(%d))", plan.Tag())),
		line(1, fmt.Sprintf("result.append_array(Wire.encode_varint(%s.size()))", buffer)),
		line(1, fmt.Sprintf("result.append_array(%s)", buffer)),
	}
}

func serializeMap(plan *fieldPlan) []fsast.Node {
	key := plan.Name + "_key"
	entry := plan.Name + "_entry"
	nodes := []fsast.Node{
		line(0, fmt.Sprintf("for %s: %s in %s:", key, plan.Key.Type.Render(), plan.Name)),
		line(1, fmt.Sprintf("var %s: PackedByteArray = PackedByteArray()", entry)),
	}
	// On the wire a map is a repeated message of {key = 1, value = 2} entries.
	nodes = append(nodes, appendValue(1, plan.Key, key, key, 1<<3|plan.Key.WireType, entry)...)
	nodes = append(nodes, appendValue(1, plan.Value, plan.Name+"["+key+"]", plan.Name+"_value", 2<<3|plan.Value.WireType, entry)...)
	return append(nodes,
		line(1, fmt.Sprintf("result.append_array(Wire.encode_varint(%d))", plan.Tag())),
		line(1, fmt.Sprintf("result.append_array(Wire.encode_varint(%s.size()))", entry)),
		line(1, fmt.Sprintf("result.append_array(%s)", entry)),
	)
}

func serializeOneof(oneof *oneofPlan) []fsast.Node {
	nodes := []fsast.Node{line(0, "match "+oneof.Field+":")}
	for i := range oneof.Members {
		member := &oneof.Members[i]
		nodes = append(nodes, line(1, fmt.Sprintf("%s(var %s):", member.OneofCase, member.Name)))
		nodes = append(nodes, appendValue(2, member.Value, member.Name, member.Name, member.Tag(), "result")...)
	}
	// An unset union writes nothing; proto3 has no tag for an empty oneof.
	return append(nodes, line(1, "_:"), line(2, "pass"))
}

// appendValue appends one tagged value to the named buffer.
func appendValue(depth int, value valuePlan, expression, name string, tag int, buffer string) []fsast.Node {
	switch {
	case value.Kind == kindMessage:
		data := name + "_data"
		return []fsast.Node{
			line(depth, fmt.Sprintf("var %s: PackedByteArray = %s.to_bytes()", data, expression)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%d))", buffer, tag)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", buffer, data)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, data)),
		}
	case value.ProtoType == "string":
		data := name + "_data"
		return []fsast.Node{
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%d))", buffer, tag)),
			line(depth, fmt.Sprintf("var %s: PackedByteArray = Wire.encode_string(%s)", data, expression)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", buffer, data)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, data)),
		}
	case value.ProtoType == "bytes":
		return []fsast.Node{
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%d))", buffer, tag)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", buffer, expression)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, expression)),
		}
	default:
		return []fsast.Node{
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%d))", buffer, tag)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", buffer, varintExpression(value, expression))),
		}
	}
}

// varintExpression converts a value to the int a varint encoder accepts.
func varintExpression(value valuePlan, expression string) string {
	switch {
	case value.Kind == kindEnum:
		return expression + ".to_wire()"
	case value.ProtoType == "bool":
		return fmt.Sprintf("1 if %s else 0", expression)
	default:
		return expression
	}
}

func line(depth int, code string) fsast.Line {
	return fsast.Line{Depth: depth, Code: code}
}
