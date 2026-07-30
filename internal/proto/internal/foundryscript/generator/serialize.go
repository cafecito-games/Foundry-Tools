package fsgenerator

import (
	"fmt"

	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

const resultBuffer = generatedPrefix + "result"

func toBytesFunction(plans []fieldPlan, oneofs []oneofPlan) fsast.Func {
	body := []fsast.Node{
		line(0, "var "+resultBuffer+": PackedByteArray = PackedByteArray()"),
	}
	// Fields are written in field-number order, oneofs included: a oneof's match
	// is emitted where its lowest-numbered member falls, so the bytes come out
	// in schema order and a hexdump reads against the .proto.
	written := map[string]bool{}
	for i := range plans {
		if plans[i].OneofCase == "" {
			body = append(body, serializeField(&plans[i])...)
			continue
		}
		oneof := findOneof(oneofs, plans[i].OneofField)
		if oneof == nil || written[oneof.Field] {
			continue
		}
		written[oneof.Field] = true
		body = append(body, serializeOneof(oneof)...)
	}
	// Every record in the buffer carries a field number this schema has no
	// member for, so nothing here competes with a live field and the position
	// of the run does not matter. A value that does share a field number with a
	// member is kept by that member instead, in the member's own position.
	body = append(body,
		line(0, resultBuffer+".append_array("+unknownFieldsMember+")"),
		fsast.Return{Value: resultBuffer},
	)
	return fsast.Func{
		Doc:        toBytesDoc(),
		Name:       toBytesMethod,
		ReturnType: fstypes.Named("PackedByteArray"),
		Body:       body,
	}
}

func findOneof(oneofs []oneofPlan, field string) *oneofPlan {
	for i := range oneofs {
		if oneofs[i].Field == field {
			return &oneofs[i]
		}
	}
	return nil
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
	case plan.Value.ProtoType == "double":
		// A float has two zeroes and protobuf only treats one of them as the
		// default, which a plain comparison cannot express.
		return "not Wire.is_default_float(" + plan.Name + ")"
	case plan.Value.ProtoType == "float":
		// Same, on the binary32 value that will be written rather than on the
		// binary64 one the member holds.
		return "not Wire.is_default_float32(" + plan.Name + ")"
	default:
		// The value's own zero, so a float tests against 0.0 rather than
		// leaning on int-to-float comparison to mean the same thing.
		return plan.Name + " != " + plan.Value.ZeroValue
	}
}

func serializeSingular(plan *fieldPlan) []fsast.Node {
	var nodes []fsast.Node
	if plan.RetainsUnknownEnum() {
		// A retained value stands in for the field, in the field's own
		// position, so what a reader takes as the last record for this number
		// is the same value the sender wrote. The member's setter guarantees
		// the two are never both live.
		nodes = append(nodes, retainedRecord(0, "if", plan)...)
		nodes = append(nodes, line(0, "elif "+presenceCondition(plan)+":"))
	} else {
		nodes = append(nodes, line(0, "if "+presenceCondition(plan)+":"))
	}
	return append(nodes, appendValue(1, plan.Value, plan.Name, plan.Local(), plan.TagExpression(), resultBuffer)...)
}

// retainedRecord writes one field's retained enum bytes back with its tag.
func retainedRecord(depth int, keyword string, plan *fieldPlan) []fsast.Node {
	return []fsast.Node{
		line(depth, fmt.Sprintf("%s %s.size() > 0:", keyword, plan.UnknownMember())),
		line(depth+1, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", resultBuffer, plan.TagExpression())),
		line(depth+1, fmt.Sprintf("%s.append_array(%s)", resultBuffer, plan.UnknownMember())),
	}
}

func serializeUnpackedRepeated(plan *fieldPlan) []fsast.Node {
	item := plan.Local("item")
	nodes := []fsast.Node{
		line(0, fmt.Sprintf("for %s: %s in %s:", item, plan.Value.Type.Render(), plan.Name)),
	}
	return append(nodes, appendValue(1, plan.Value, item, plan.Local(), plan.TagExpression(), resultBuffer)...)
}

func serializePackedRepeated(plan *fieldPlan) []fsast.Node {
	item := plan.Local("item")
	buffer := plan.Local("data")
	return []fsast.Node{
		line(0, fmt.Sprintf("if %s.size() > 0:", plan.Name)),
		line(1, fmt.Sprintf("var %s: PackedByteArray = PackedByteArray()", buffer)),
		line(1, fmt.Sprintf("for %s: %s in %s:", item, plan.Value.Type.Render(), plan.Name)),
		line(2, fmt.Sprintf("%s.append_array(%s)", buffer, plan.Value.encodeCall(item))),
		line(1, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", resultBuffer, plan.TagExpression())),
		line(1, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", resultBuffer, buffer)),
		line(1, fmt.Sprintf("%s.append_array(%s)", resultBuffer, buffer)),
	}
}

func serializeMap(plan *fieldPlan) []fsast.Node {
	key := plan.Local("key")
	entry := plan.Local("entry")
	nodes := []fsast.Node{
		line(0, fmt.Sprintf("for %s: %s in %s:", key, plan.Key.Type.Render(), plan.Name)),
		line(1, fmt.Sprintf("var %s: PackedByteArray = PackedByteArray()", entry)),
	}
	// On the wire a map is a repeated message of {key = 1, value = 2} entries.
	nodes = append(nodes, appendValue(1, plan.Key, key, plan.Local("key"), tagExpression(1, plan.Key.WireType), entry)...)
	nodes = append(nodes, appendValue(1, plan.Value, plan.Name+"["+key+"]", plan.Local("value"), tagExpression(2, plan.Value.WireType), entry)...)
	return append(nodes,
		line(1, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", resultBuffer, plan.TagExpression())),
		line(1, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", resultBuffer, entry)),
		line(1, fmt.Sprintf("%s.append_array(%s)", resultBuffer, entry)),
	)
}

func serializeOneof(oneof *oneofPlan) []fsast.Node {
	// A retained case stands in for the whole union. The union's setter clears
	// every retained buffer, and retaining one clears the union, so at most one
	// branch of this chain is ever live.
	retaining := oneof.RetainingMembers()
	if len(retaining) == 0 {
		return oneofMatch(0, oneof)
	}
	var nodes []fsast.Node
	for i := range retaining {
		keyword := "if"
		if i > 0 {
			keyword = "elif"
		}
		nodes = append(nodes, retainedRecord(0, keyword, &retaining[i])...)
	}
	nodes = append(nodes, line(0, "else:"))
	return append(nodes, oneofMatch(1, oneof)...)
}

func oneofMatch(depth int, oneof *oneofPlan) []fsast.Node {
	nodes := []fsast.Node{line(depth, "match "+oneof.Field+":")}
	for i := range oneof.Members {
		member := &oneof.Members[i]
		bound := member.Local()
		nodes = append(nodes, line(depth+1, fmt.Sprintf("%s(var %s):", member.OneofCase, bound)))
		nodes = append(nodes, appendValue(depth+2, member.Value, bound, member.Local(), member.TagExpression(), resultBuffer)...)
	}
	// An unset union writes nothing; proto3 has no tag for an empty oneof.
	return append(nodes, line(depth+1, "_:"), line(depth+2, "pass"))
}

// appendValue appends one tagged value to the named buffer. local is the stem
// for any temporary the encoding needs; tag is the expression that builds the
// field key.
func appendValue(depth int, value valuePlan, expression, local, tag, buffer string) []fsast.Node {
	switch {
	case value.Kind == kindMessage:
		data := local + "_data"
		return []fsast.Node{
			line(depth, fmt.Sprintf("var %s: PackedByteArray = %s.%s()", data, expression, toBytesMethod)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", buffer, tag)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", buffer, data)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, data)),
		}
	case value.ProtoType == "string":
		data := local + "_data"
		return []fsast.Node{
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", buffer, tag)),
			line(depth, fmt.Sprintf("var %s: PackedByteArray = Wire.encode_string(%s)", data, expression)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", buffer, data)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, data)),
		}
	case value.ProtoType == "bytes":
		return []fsast.Node{
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", buffer, tag)),
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s.size()))", buffer, expression)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, expression)),
		}
	default:
		return []fsast.Node{
			line(depth, fmt.Sprintf("%s.append_array(Wire.encode_varint(%s))", buffer, tag)),
			line(depth, fmt.Sprintf("%s.append_array(%s)", buffer, value.encodeCall(expression))),
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
