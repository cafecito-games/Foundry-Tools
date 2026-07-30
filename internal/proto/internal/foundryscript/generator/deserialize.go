package fsgenerator

import (
	"fmt"

	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// The cursor and the tag decoded at the top of every merge_from_bytes loop.
// Every name the emitter introduces carries generatedPrefix, which is what
// keeps it clear of a field named `offset`, `data` or `result`.
const (
	cursorLocal   = generatedPrefix + "offset"
	tagLocal      = generatedPrefix + "tag"
	wireTypeLocal = generatedPrefix + "wire_type"
	// dataParameter is the wire payload merge_from_bytes reads from. It is
	// prefixed like every other generated name so a field named `data` is not
	// shadowed by it inside the function body.
	dataParameter = generatedPrefix + "data"
)

// mergeMode says how a submessage acquires the instance it decodes into.
type mergeMode int

const (
	// mergeFresh builds a new instance and hands it to the assignment. This is
	// right for a repeated element, a oneof case, or any value that replaces
	// rather than accumulates.
	mergeFresh mergeMode = iota
	// mergeInto merges into an instance the surrounding code already holds.
	mergeInto
	// mergeMember merges into the message-typed member itself, instantiating it
	// first if it is still null. protobuf requires a singular message field to
	// merge, not replace, when it appears more than once in the same stream.
	mergeMember
)

// readContext is everything a value decode needs beyond the value itself: where
// to put the result, how a submessage gets its instance, and what to do with a
// wire value the schema has no enum case for.
type readContext struct {
	// local is the stem for the temporaries this decode introduces.
	local string
	// assign stores one decoded value.
	assign func(expression string) string
	mode   mergeMode
	// target is the instance to merge into for mergeInto and mergeMember.
	target string
	// retainUnknownEnum emits the statements that preserve an unrecognized enum
	// value spanning data[from:to].
	retainUnknownEnum func(depth int, from, to string) []fsast.Node
	// boundary names the local holding the end of the enclosing length-delimited
	// region, when there is one. The runtime readers bound themselves against
	// the whole buffer, so reading inside a map entry has to be checked against
	// the entry as well or a truncated one consumes the field after it.
	boundary string
}

// boundaryGuard rejects a read that finished past the enclosing region. It
// emits nothing when the read is not inside one.
func (c readContext) boundaryGuard(depth int, read string) []fsast.Node {
	if c.boundary == "" {
		return nil
	}
	return []fsast.Node{
		line(depth, fmt.Sprintf("if %s.offset > %s:", read, c.boundary)),
		line(depth+1, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
	}
}

func mergeFromBytesFunction(plans []fieldPlan) fsast.Func {
	skipped := localName("skipped")
	body := []fsast.Node{
		line(0, "var "+cursorLocal+": int = 0"),
		line(0, "while "+cursorLocal+" < "+dataParameter+".size():"),
		line(1, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(%s, %s)", tagLocal, dataParameter, cursorLocal)),
		line(1, fmt.Sprintf("if %s.error != ProtobufError.OK:", tagLocal)),
		line(2, "return "+tagLocal+".error"),
		line(1, fmt.Sprintf("%s = %s.offset", cursorLocal, tagLocal)),
		line(1, fmt.Sprintf("var %s: int = Wire.get_wire_type(%s.value)", wireTypeLocal, tagLocal)),
		line(1, fmt.Sprintf("match Wire.get_field_number(%s.value):", tagLocal)),
	}
	for i := range plans {
		body = append(body, line(2, fmt.Sprintf("%d:", plans[i].Number)))
		body = append(body, deserializeField(&plans[i])...)
	}
	// An unknown field is kept, not just skipped: proto3 requires it to survive
	// a decode/re-encode round trip so a peer on a newer schema loses nothing
	// passing through this binding.
	body = append(body,
		line(2, "_:"),
		line(3, fmt.Sprintf("var %s: SkipRead = Wire.capture_field(%s, %s, %s.value, %s, %s)",
			skipped, dataParameter, cursorLocal, tagLocal, wireTypeLocal, unknownFieldsMember)),
		line(3, fmt.Sprintf("if %s.error != ProtobufError.OK:", skipped)),
		line(4, "return "+skipped+".error"),
		line(3, cursorLocal+" = "+skipped+".offset"),
		fsast.Return{Value: "ProtobufError.OK"},
	)
	return fsast.Func{
		Doc:        mergeFromBytesDoc(),
		Name:       mergeFromBytesMethod,
		Parameters: []fsast.Parameter{{Name: dataParameter, Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Named("ProtobufError"),
		Body:       body,
	}
}

func deserializeField(plan *fieldPlan) []fsast.Node {
	switch {
	case plan.Cardinality == cardinalityMap:
		return deserializeMap(plan)
	case plan.Cardinality == cardinalityRepeated && plan.Packable:
		return deserializePackableRepeated(plan)
	default:
		return decodeValue(3, plan.Value, fieldContext(plan))
	}
}

// fieldContext is how a whole field stores what it decodes, which is the only
// thing that differs between a singular field, a repeated element, and a oneof
// case.
func fieldContext(plan *fieldPlan) readContext {
	context := readContext{local: plan.Local()}
	switch {
	case plan.OneofCase != "":
		context.assign = func(expression string) string {
			return fmt.Sprintf("%s = %s(%s)", plan.OneofField, plan.OneofCase, expression)
		}
	case plan.Cardinality == cardinalityRepeated:
		context.assign = func(expression string) string {
			return fmt.Sprintf("%s.append(%s)", plan.Name, expression)
		}
	default:
		context.assign = func(expression string) string {
			return fmt.Sprintf("%s = %s", plan.Name, expression)
		}
		if plan.Value.Kind == kindMessage {
			context.mode = mergeMember
			context.target = plan.Name
		}
	}
	if plan.RetainsUnknownEnum() {
		// The raw bytes go in the field's own companion, and the member is
		// cleared first so the field or union reads as unset: that assignment
		// runs the setter, which is what discards a value retained earlier.
		context.retainUnknownEnum = func(depth int, from, to string) []fsast.Node {
			return []fsast.Node{
				line(depth, plan.clearedMember()),
				line(depth, fmt.Sprintf("%s = %s.slice(%s, %s)", plan.UnknownMember(), dataParameter, from, to)),
			}
		}
	} else {
		context.retainUnknownEnum = foldUnrecognizedEnum(plan.Value, context.assign)
	}
	return context
}

// foldUnrecognizedEnum is what a repeated or map-valued enum does with a number
// this schema has no case for: it takes the proto default.
//
// The value cannot be kept. An Array[T] or Dictionary[K, T] of an enum holds
// only declared cases, so the raw number has nowhere to live, and the shared
// unknown-field buffer is not a substitute: it is emitted as one run, and both
// of these carry meaning in their position. Moving a repeated element into it
// reorders the sequence, and moving a map entry into it flips which of two
// records for the same key protobuf takes as the last one. Losing the value is
// visible and bounded; silently reordering a sequence is neither.
func foldUnrecognizedEnum(value valuePlan, assign func(string) string) func(int, string, string) []fsast.Node {
	return func(depth int, _, _ string) []fsast.Node {
		return []fsast.Node{line(depth, assign(value.ZeroValue))}
	}
}

// decodeValue emits the wire-type guard, the read, and the store for one value.
func decodeValue(depth int, value valuePlan, context readContext) []fsast.Node {
	nodes := []fsast.Node{
		line(depth, fmt.Sprintf("if %s != %s:", wireTypeLocal, wireTypeConstant(value.WireType))),
		line(depth+1, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	}
	return append(nodes, readValue(depth, value, context)...)
}

// readValue emits the read and store for one value, without the wire-type
// guard, so map entries and repeated elements can reuse it.
func readValue(depth int, value valuePlan, context readContext) []fsast.Node {
	read := context.local + "_read"
	switch {
	case value.Kind == kindMessage:
		return readMessageValue(depth, value, context, read)
	case value.Kind == kindEnum:
		return readEnumValue(depth, value, context, read)
	case value.ProtoType == "string", value.ProtoType == "bytes":
		carrier, reader := "StringRead", "read_string"
		if value.ProtoType == "bytes" {
			carrier, reader = "BytesRead", "read_bytes"
		}
		nodes := []fsast.Node{
			line(depth, fmt.Sprintf("var %s: %s = Wire.%s(%s, %s)", read, carrier, reader, dataParameter, cursorLocal)),
			line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", read)),
			line(depth+1, "return "+read+".error"),
		}
		nodes = append(nodes, context.boundaryGuard(depth, read)...)
		return append(nodes,
			line(depth, context.assign(read+".value")),
			line(depth, fmt.Sprintf("%s = %s.offset", cursorLocal, read)),
		)
	default:
		nodes := []fsast.Node{
			line(depth, fmt.Sprintf("var %s: %s = %s(%s, %s)", read, value.readCarrier(), value.readFunction(), dataParameter, cursorLocal)),
			line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", read)),
			line(depth+1, "return "+read+".error"),
		}
		nodes = append(nodes, context.boundaryGuard(depth, read)...)
		return append(nodes,
			line(depth, context.assign(varintResultExpression(value, read+".value"))),
			line(depth, fmt.Sprintf("%s = %s.offset", cursorLocal, read)),
		)
	}
}

func readMessageValue(depth int, value valuePlan, context readContext, read string) []fsast.Node {
	typeName := value.Type.Render()
	var nodes []fsast.Node
	target := context.target
	switch context.mode {
	case mergeMember:
		nodes = append(nodes,
			line(depth, fmt.Sprintf("if not (%s is %s):", target, typeName)),
			line(depth+1, fmt.Sprintf("%s = %s.new()", target, typeName)),
		)
	case mergeInto:
		// The caller already holds the instance.
	default:
		target = context.local + "_message"
		nodes = append(nodes, line(depth, fmt.Sprintf("var %s: %s = %s.new()", target, typeName, typeName)))
	}
	nodes = append(nodes,
		line(depth, fmt.Sprintf("var %s: SkipRead = Wire.read_message(%s, %s, %s)", read, dataParameter, cursorLocal, target)),
		line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", read)),
		line(depth+1, "return "+read+".error"),
	)
	nodes = append(nodes, context.boundaryGuard(depth, read)...)
	if context.mode == mergeFresh {
		nodes = append(nodes, line(depth, context.assign(target)))
	}
	return append(nodes, line(depth, fmt.Sprintf("%s = %s.offset", cursorLocal, read)))
}

// readEnumValue keeps a wire value this schema has no case for instead of
// folding it onto the zero case. proto3 enums are open, so collapsing an
// unrecognized value would silently destroy a field written by a newer peer.
func readEnumValue(depth int, value valuePlan, context readContext, read string) []fsast.Node {
	typeName := value.Type.Render()
	decoded := context.local + "_case"
	nodes := []fsast.Node{
		line(depth, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(%s, %s)", read, dataParameter, cursorLocal)),
		line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", read)),
		line(depth+1, "return "+read+".error"),
	}
	nodes = append(nodes, context.boundaryGuard(depth, read)...)
	nodes = append(nodes,
		line(depth, fmt.Sprintf("var %s: %s? = %s.from_wire(%s.value)", decoded, typeName, typeName, read)),
		line(depth, fmt.Sprintf("if %s is %s:", decoded, typeName)),
		line(depth+1, context.assign(decoded)),
		line(depth, "else:"),
	)
	nodes = append(nodes, context.retainUnknownEnum(depth+1, cursorLocal, read+".offset")...)
	return append(nodes, line(depth, fmt.Sprintf("%s = %s.offset", cursorLocal, read)))
}

// deserializePackableRepeated accepts both encodings: a packed run and the
// unpacked one-record-per-element form, which protobuf parsers must tolerate
// regardless of how the sender chose to write the field. That includes a field
// this schema marked `[packed = false]`: the option binds the encoder alone.
func deserializePackableRepeated(plan *fieldPlan) []fsast.Node {
	length := plan.Local("length")
	end := plan.Local("end")
	packed := plan.Local("packed")
	context := fieldContext(plan)

	nodes := []fsast.Node{line(3, "if "+wireTypeLocal+" == Wire.WIRE_LENGTH_DELIMITED:")}
	nodes = append(nodes, readLength(4, length)...)
	nodes = append(nodes,
		line(4, fmt.Sprintf("var %s: int = %s.offset + %s.value", end, length, length)),
		line(4, fmt.Sprintf("%s = %s.offset", cursorLocal, length)),
		line(4, fmt.Sprintf("while %s < %s:", cursorLocal, end)),
	)
	nodes = append(nodes, readPackedElement(5, plan, context, packed, end)...)
	nodes = append(nodes, line(3, fmt.Sprintf("elif %s == %s:", wireTypeLocal, wireTypeConstant(plan.Value.WireType))))
	nodes = append(nodes, readValue(4, plan.Value, context)...)
	return append(nodes,
		line(3, "else:"),
		line(4, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	)
}

// readPackedElement decodes one element of a packed run, which carries no tag
// of its own and so cannot reuse the tagged read path.
//
// The element readers bound themselves against the whole buffer rather than
// against this run, so the read is followed by a check that it stayed inside
// it. Without that, a payload whose length is not a whole number of elements
// would silently consume bytes belonging to the next field and accept a
// message it should reject — deterministically so for the fixed-width types,
// where element size is known in advance.
func readPackedElement(depth int, plan *fieldPlan, context readContext, packed, end string) []fsast.Node {
	nodes := []fsast.Node{
		line(depth, fmt.Sprintf("var %s: %s = %s(%s, %s)", packed, plan.Value.readCarrier(), plan.Value.readFunction(), dataParameter, cursorLocal)),
		line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", packed)),
		line(depth+1, "return "+packed+".error"),
		line(depth, fmt.Sprintf("if %s.offset > %s:", packed, end)),
		line(depth+1, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
	}
	if plan.Value.Kind == kindEnum {
		typeName := plan.Value.Type.Render()
		decoded := plan.Local("packed", "case")
		nodes = append(nodes,
			line(depth, fmt.Sprintf("var %s: %s? = %s.from_wire(%s.value)", decoded, typeName, typeName, packed)),
			line(depth, fmt.Sprintf("if %s is %s:", decoded, typeName)),
			line(depth+1, context.assign(decoded)),
			line(depth, "else:"),
		)
		nodes = append(nodes, context.retainUnknownEnum(depth+1, cursorLocal, packed+".offset")...)
	} else {
		nodes = append(nodes, line(depth, context.assign(varintResultExpression(plan.Value, packed+".value"))))
	}
	return append(nodes, line(depth, fmt.Sprintf("%s = %s.offset", cursorLocal, packed)))
}

func deserializeMap(plan *fieldPlan) []fsast.Node {
	length := plan.Local("length")
	end := plan.Local("end")
	key := plan.Local("key")
	value := plan.Local("value")
	entryTag := plan.Local("entry", "tag")
	entryWireType := plan.Local("entry", "wire", "type")

	nodes := []fsast.Node{
		line(3, fmt.Sprintf("if %s != Wire.WIRE_LENGTH_DELIMITED:", wireTypeLocal)),
		line(4, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	}
	nodes = append(nodes, readLength(3, length)...)
	nodes = append(nodes,
		line(3, fmt.Sprintf("var %s: int = %s.offset + %s.value", end, length, length)),
		line(3, fmt.Sprintf("%s = %s.offset", cursorLocal, length)),
		line(3, fmt.Sprintf("var %s: %s = %s", key, plan.Key.Type.Render(), plan.Key.ZeroValue)),
		line(3, fmt.Sprintf("var %s: %s = %s", value, plan.Value.Type.Render(), mapValueZero(plan.Value))),
		line(3, fmt.Sprintf("while %s < %s:", cursorLocal, end)),
		line(4, fmt.Sprintf("var %s: VarintRead = Wire.decode_varint(%s, %s)", entryTag, dataParameter, cursorLocal)),
		line(4, fmt.Sprintf("if %s.error != ProtobufError.OK:", entryTag)),
		line(5, "return "+entryTag+".error"),
		// The tag is read from the whole buffer, so an unterminated one inside
		// the entry would otherwise carry on into the field after it.
		line(4, fmt.Sprintf("if %s.offset > %s:", entryTag, end)),
		line(5, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
		line(4, fmt.Sprintf("%s = %s.offset", cursorLocal, entryTag)),
		line(4, fmt.Sprintf("var %s: int = Wire.get_wire_type(%s.value)", entryWireType, entryTag)),
		line(4, fmt.Sprintf("match Wire.get_field_number(%s.value):", entryTag)),
		line(5, "1:"),
		line(6, fmt.Sprintf("if %s != %s:", entryWireType, wireTypeConstant(plan.Key.WireType))),
		line(7, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	)
	nodes = append(nodes, readValue(6, plan.Key, readContext{
		local:    key,
		assign:   func(expression string) string { return key + " = " + expression },
		boundary: end,
	})...)
	nodes = append(nodes,
		line(5, "2:"),
		line(6, fmt.Sprintf("if %s != %s:", entryWireType, wireTypeConstant(plan.Value.WireType))),
		line(7, "return ProtobufError.WIRE_TYPE_MISMATCH"),
	)
	valueContext := readContext{
		local:  value,
		assign: func(expression string) string { return value + " = " + expression },
		// The entry's value already holds a fresh instance, so a submessage
		// merges into it rather than replacing it.
		mode:     mergeInto,
		target:   value,
		boundary: end,
	}
	valueContext.retainUnknownEnum = foldUnrecognizedEnum(plan.Value, valueContext.assign)
	nodes = append(nodes, readValue(6, plan.Value, valueContext)...)
	nodes = append(nodes,
		line(5, "_:"),
		line(6, fmt.Sprintf("var %s: SkipRead = Wire.skip_field(%s, %s, %s)", plan.Local("skip"), dataParameter, cursorLocal, entryWireType)),
		line(6, fmt.Sprintf("if %s.error != ProtobufError.OK:", plan.Local("skip"))),
		line(7, fmt.Sprintf("return %s.error", plan.Local("skip"))),
		line(6, fmt.Sprintf("if %s.offset > %s:", plan.Local("skip"), end)),
		line(7, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"),
		line(6, fmt.Sprintf("%s = %s.offset", cursorLocal, plan.Local("skip"))),
	)
	return append(nodes, line(3, fmt.Sprintf("%s[%s] = %s", plan.Name, key, value)))
}

// readLength reads and bounds-checks a length prefix. The runtime owns the
// policy; this is only the error propagation the caller has to do.
func readLength(depth int, length string) []fsast.Node {
	return []fsast.Node{
		line(depth, fmt.Sprintf("var %s: VarintRead = Wire.read_length(%s, %s)", length, dataParameter, cursorLocal)),
		line(depth, fmt.Sprintf("if %s.error != ProtobufError.OK:", length)),
		line(depth+1, "return "+length+".error"),
	}
}

// mapValueZero is the value a map entry starts from. An entry may legally omit
// its value field, in which case protobuf says the field default applies, so a
// message-valued entry starts as an empty message rather than null: the
// dictionary value type is not nullable and must never receive null.
func mapValueZero(value valuePlan) string {
	if value.Kind == kindMessage {
		return value.Type.Render() + ".new()"
	}
	return value.ZeroValue
}

// varintResultExpression converts a decoded varint back into the field type.
func varintResultExpression(value valuePlan, expression string) string {
	if value.ProtoType == "bool" {
		return expression + " != 0"
	}
	return expression
}

func wireTypeConstant(wireType int) string {
	switch wireType {
	case wireLengthDelimited:
		return "Wire.WIRE_LENGTH_DELIMITED"
	case wire32Bit:
		return "Wire.WIRE_32BIT"
	case wire64Bit:
		return "Wire.WIRE_64BIT"
	default:
		return "Wire.WIRE_VARINT"
	}
}
