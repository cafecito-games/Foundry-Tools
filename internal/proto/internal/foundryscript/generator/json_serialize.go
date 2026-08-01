package fsgenerator

import (
	"fmt"
	"strconv"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// The engine's JSON builtins, named unqualified. They are global script
// classes, so a .fs file in any namespace resolves them with no import; what
// keeps a schema type from shadowing one is engineJSONBuiltinTypeNames.
const (
	jsonNodeType         = "JsonNode"
	jsonSerializableName = "JsonSerializable"
)

// The JSON members a binding declares.
const (
	toJSONMethod     = "to_json"
	fromJSONMethod   = "from_json"
	toJSONNameMethod = "to_json_name"
)

// jsonNodeParameter is the document a decode reads from.
const jsonNodeParameter = generatedPrefix + "node"

const (
	// jsonDocument is the accumulator to_json builds the document in.
	jsonDocument = generatedPrefix + "json"
	// jsonFloatMethod renders one float the way canonical proto3 requires,
	// which is not what the encoder does with a non-finite one.
	jsonFloatMethod = generatedPrefix + "json_float"
	// jsonFloatParameter is that helper's parameter.
	jsonFloatParameter = generatedPrefix + "value"
)

// jsonEmission is what the message and enum emitters need to know about the
// JSON option for one generation run.
type jsonEmission struct {
	// Enabled is Options.JSON. Nothing in this file is emitted without it.
	Enabled bool
	// SourceName is the import path of the file being generated, which is what
	// selects a well-known type's canonical JSON form.
	SourceName string
}

func toJSONDoc() []string {
	return []string{
		"Returns this message as a proto3 canonical JSON document.",
		"",
		"JSON.stringify(message, \"\", false) renders it as text; the third argument",
		"turns off key sorting, which keeps members in field declaration order.",
	}
}

func toJSONNameDoc() []string {
	return []string{"Returns the proto3 JSON name for this case."}
}

func jsonFloatDoc() []string {
	return []string{
		"Returns one float as canonical proto3 JSON.",
		"",
		"A non-finite value never reaches the Float case: the encoder writes NaN as",
		"null and the infinities as ±1e99999, none of which is canonical, so the",
		"three specified string forms are produced here instead.",
	}
}

// jsonMembers are the members Options.JSON adds to a message binding.
func jsonMembers(plan *messagePlan, emission jsonEmission) []fsast.Node {
	form := wellKnownJSONFormFor(emission.SourceName, plan.Scope)
	members := []fsast.Node{toJSONFunction(plan, form), fromJSONSeamFunction(plan)}
	if jsonWritesAFloat(plan) {
		members = append(members, jsonFloatFunction())
	}
	return members
}

// jsonUses is the trait list a message declares. Conforming to the engine's
// trait is what teaches JSON.stringify to lower this message at all, so a
// binding without it has no route to JSON text.
func jsonUses(emission jsonEmission) []string {
	if !emission.Enabled {
		return []string{"Message"}
	}
	return []string{"Message", jsonSerializableName}
}

func toJSONFunction(plan *messagePlan, form wellKnownJSONForm) fsast.Func {
	body := wellKnownToJSONBody(form, plan)
	if body == nil {
		body = messageToJSONBody(plan)
	}
	return fsast.Func{
		Doc:        toJSONDoc(),
		Name:       toJSONMethod,
		ReturnType: fstypes.Named(jsonNodeType),
		Body:       body,
	}
}

// fromJSONSeamFunction is the decode half of the trait, which the emitted
// conformance cannot go without: the analyzer rejects a class that declares
// JsonSerializable and implements only to_json, so the conformance -- and with
// it the whole route from a message to JSON text -- is unavailable until both
// halves exist.
//
// The decoder is generated separately. Until it lands this reports a failure
// rather than returning a partly-decoded message, so a caller learns the
// decode did not happen instead of being handed something that looks decoded.
func fromJSONSeamFunction(plan *messagePlan) fsast.Func {
	return fsast.Func{
		Doc: []string{
			"Decodes a proto3 canonical JSON document into a new " + plan.Name + " message.",
			"",
			"Not generated yet: this reports a failure for every document. The",
			"conformance it completes is what makes to_json reachable through",
			"JSON.stringify, which is why the member exists ahead of the decoder.",
		},
		Static:     true,
		Name:       fromJSONMethod,
		Parameters: []fsast.Parameter{{Name: jsonNodeParameter, Type: fstypes.Named(jsonNodeType)}},
		ReturnType: fstypes.Generic("JsonResult", fstypes.Named(plan.Name)),
		Body: []fsast.Node{
			fsast.Return{Value: fmt.Sprintf(
				"JsonResult[%s].fail(%s, %s)",
				plan.Name,
				strconv.Quote("JSON_PARSE_FAILED: "+plan.Name+" cannot be decoded from JSON yet"),
				strconv.Quote("$"),
			)},
		},
	}
}

// messageToJSONBody writes an ordinary message as a JSON object.
//
// Members come out in field-number order, oneofs included, for the same reason
// the wire encoder writes them that way: the document then reads against the
// .proto. Key sorting in JSON.stringify is off, so this order is what survives.
func messageToJSONBody(plan *messagePlan) []fsast.Node {
	body := []fsast.Node{
		line(0, "var "+jsonDocument+": Dictionary[String, "+jsonNodeType+"] = {}"),
	}
	written := map[string]bool{}
	for i := range plan.Fields {
		field := &plan.Fields[i]
		if field.OneofCase == "" {
			body = append(body, jsonField(field)...)
			continue
		}
		oneof := findOneof(plan.Oneofs, field.OneofField)
		if oneof == nil || written[oneof.Field] {
			continue
		}
		written[oneof.Field] = true
		body = append(body, jsonOneof(oneof)...)
	}
	return append(body, fsast.Return{Value: jsonNodeType + ".object_of(" + jsonDocument + ")"})
}

func jsonField(plan *fieldPlan) []fsast.Node {
	switch plan.Cardinality {
	case cardinalityRepeated:
		return jsonRepeatedField(plan)
	case cardinalityMap:
		return jsonMapField(plan)
	default:
		// The presence rule is the wire encoder's: a proto3 zero is
		// indistinguishable from unset and is left out, while an optional or
		// message member is written whenever it is present.
		return []fsast.Node{
			line(0, "if "+presenceCondition(plan)+":"),
			line(1, jsonMember(plan)+" = "+jsonValueExpression(plan.Value, plan.Name)),
		}
	}
}

// jsonMember is the document entry this field is written to, under the JSON
// name the field model resolved: an explicit [json_name] when the schema
// carries one, the camelCase derivation otherwise.
func jsonMember(plan *fieldPlan) string {
	return jsonDocument + "[" + strconv.Quote(plan.JSONName) + "]"
}

// jsonRepeatedField writes a repeated field as an array, omitting it when it is
// empty: proto3 gives a repeated field no presence of its own either.
func jsonRepeatedField(plan *fieldPlan) []fsast.Node {
	items := plan.Local("items")
	item := plan.Local("item")
	return []fsast.Node{
		line(0, fmt.Sprintf("if %s.size() > 0:", plan.Name)),
		line(1, fmt.Sprintf("var %s: Array[%s] = []", items, jsonNodeType)),
		line(1, fmt.Sprintf("for %s: %s in %s:", item, plan.Value.Type.Render(), plan.Name)),
		line(2, fmt.Sprintf("%s.append(%s)", items, jsonValueExpression(plan.Value, item))),
		line(1, jsonMember(plan)+" = "+jsonNodeType+".array_of("+items+")"),
	}
}

// jsonMapField writes a map as an object. JSON object keys are strings, so a
// key of any other type is stringified here rather than left to the encoder.
func jsonMapField(plan *fieldPlan) []fsast.Node {
	fields := plan.Local("fields")
	key := plan.Local("key")
	return []fsast.Node{
		line(0, fmt.Sprintf("if %s.size() > 0:", plan.Name)),
		line(1, fmt.Sprintf("var %s: Dictionary[String, %s] = {}", fields, jsonNodeType)),
		line(1, fmt.Sprintf("for %s: %s in %s:", key, plan.Key.Type.Render(), plan.Name)),
		line(2, fmt.Sprintf(
			"%s[%s] = %s",
			fields,
			jsonKeyExpression(plan.Key, key),
			jsonValueExpression(plan.Value, plan.Name+"["+key+"]"),
		)),
		line(1, jsonMember(plan)+" = "+jsonNodeType+".object_of("+fields+")"),
	}
}

// jsonOneof writes whichever member is set, and nothing when the union is
// unset: proto3 JSON has no spelling for an empty oneof.
func jsonOneof(oneof *oneofPlan) []fsast.Node {
	nodes := []fsast.Node{line(0, "match "+oneof.Field+":")}
	for i := range oneof.Members {
		member := &oneof.Members[i]
		bound := member.Local()
		nodes = append(nodes,
			line(1, fmt.Sprintf("%s(var %s):", member.OneofCase, bound)),
			line(2, jsonMember(member)+" = "+jsonValueExpression(member.Value, bound)),
		)
	}
	return append(nodes, line(1, "_:"), line(2, "pass"))
}

// jsonValueExpression is one value as a JsonNode, per the canonical mapping.
func jsonValueExpression(value valuePlan, expression string) string {
	switch value.Kind {
	case kindMessage:
		// Including a well-known type: its binding carries the special form, so
		// a reference to one recurses through the trait like any other message.
		return expression + "." + toJSONMethod + "()"
	case kindEnum:
		return jsonNodeType + ".Str(" + expression + "." + toJSONNameMethod + "())"
	}
	switch value.ProtoType {
	case "bool":
		return jsonNodeType + ".Bool(" + expression + ")"
	case "string":
		return jsonNodeType + ".Str(" + expression + ")"
	case "bytes":
		return jsonNodeType + ".Str(JsonBase64.encode(" + expression + "))"
	case "float", "double":
		return jsonFloatMethod + "(" + expression + ")"
	case "int64", "uint64", "sint64", "fixed64", "sfixed64":
		// A 64-bit integer is written as a string because that is the only form
		// that survives a round trip: sent as a bare JSON number, a value past
		// 2^53 comes back from the engine's parser as a Float, having lost
		// precision on the way.
		return jsonNodeType + ".Str(str(" + expression + "))"
	default:
		return jsonNodeType + ".Int(" + expression + ")"
	}
}

// jsonKeyExpression stringifies a map key. protobuf allows integral, boolean
// and string keys; the specification spells the first two as their JSON scalar
// forms rather than as whatever str() would produce for the host type.
func jsonKeyExpression(key valuePlan, expression string) string {
	switch key.ProtoType {
	case "string":
		return expression
	case "bool":
		return `"true" if ` + expression + ` else "false"`
	default:
		return "str(" + expression + ")"
	}
}

// jsonWritesAFloat reports whether this message's JSON output passes any value
// through the float helper, which is what decides whether the helper is
// emitted. A well-known form does not change the answer: the only well-known
// types with a float-valued field are the two floating-point wrappers and
// Value, and all three write that field.
func jsonWritesAFloat(plan *messagePlan) bool {
	for i := range plan.Fields {
		field := &plan.Fields[i]
		if isFloatProtoType(field.Value.ProtoType) {
			return true
		}
	}
	return false
}

func isFloatProtoType(protoType string) bool {
	return protoType == "float" || protoType == "double"
}

func jsonFloatFunction() fsast.Func {
	return fsast.Func{
		Doc:        jsonFloatDoc(),
		Static:     true,
		Name:       jsonFloatMethod,
		Parameters: []fsast.Parameter{{Name: jsonFloatParameter, Type: fstypes.Named("float")}},
		ReturnType: fstypes.Named(jsonNodeType),
		Body: []fsast.Node{
			line(0, "if is_nan("+jsonFloatParameter+"):"),
			line(1, `return `+jsonNodeType+`.Str("NaN")`),
			line(0, "if is_inf("+jsonFloatParameter+"):"),
			line(1, "if "+jsonFloatParameter+" > 0.0:"),
			line(2, `return `+jsonNodeType+`.Str("Infinity")`),
			line(1, `return `+jsonNodeType+`.Str("-Infinity")`),
			fsast.Return{Value: jsonNodeType + ".Float(" + jsonFloatParameter + ")"},
		},
	}
}

// enumJSONNameFunction hosts the JSON conversion on the enum, for the same
// reason to_wire is hosted there: the case names are declared once, and a
// message referencing the enum should not have to restate them.
func enumJSONNameFunction(enum *protoast.Enum) fsast.Func {
	wire := generatedPrefix + "wire"
	body := []fsast.Node{
		line(0, "var "+wire+": int = self as int"),
		line(0, "match "+wire+":"),
	}
	emitted := map[int]bool{}
	for _, value := range enum.Values {
		// An allow_alias enum declares the same number twice; the first
		// declared spelling wins, exactly as it does for from_wire.
		if emitted[value.Number] {
			continue
		}
		emitted[value.Number] = true
		body = append(body,
			line(1, strconv.Itoa(value.Number)+":"),
			line(2, "return "+strconv.Quote(value.Name)),
		)
	}
	// Every value of the enum is a declared case, so this arm is unreachable;
	// it is what makes the match total for the analyzer.
	body = append(body,
		line(1, "_:"),
		line(2, "return "+strconv.Quote(zeroValueName(enum))),
	)
	return fsast.Func{
		Doc:        toJSONNameDoc(),
		Name:       toJSONNameMethod,
		ReturnType: fstypes.Named("String"),
		Body:       body,
	}
}

// wellKnownToJSONBody is the body of to_json for a well-known type, or nil for
// an ordinary message. These do not serialize field by field: the canonical
// mapping gives each of them a JSON form of its own.
func wellKnownToJSONBody(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	switch form {
	case wellKnownJSONTimestamp, wellKnownJSONDuration:
		return wellKnownSecondsBody(form, plan)
	case wellKnownJSONFieldMask:
		return wellKnownFieldMaskBody(form, plan)
	case wellKnownJSONEmpty:
		return []fsast.Node{
			line(0, "var "+jsonDocument+": Dictionary[String, "+jsonNodeType+"] = {}"),
			fsast.Return{Value: jsonNodeType + ".object_of(" + jsonDocument + ")"},
		}
	case wellKnownJSONWrapper:
		return wellKnownWrapperBody(plan)
	case wellKnownJSONStruct:
		return wellKnownStructBody(plan)
	case wellKnownJSONListValue:
		return wellKnownListValueBody(plan)
	case wellKnownJSONValue:
		return wellKnownValueBody(plan)
	case wellKnownJSONAny:
		// Rendering an Any needs its type URL resolved to a generated binding,
		// which needs a runtime type registry that does not exist yet.
		return []fsast.Node{
			line(0, "push_error("+strconv.Quote(jsonAnyUnsupportedMessage)+")"),
			fsast.Return{Value: jsonNodeType + ".Null"},
		}
	default:
		return nil
	}
}

// wellKnownSecondsBody covers Timestamp and Duration, which differ only in
// which runtime helper formats their seconds and nanos.
func wellKnownSecondsBody(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	seconds := wellKnownField(plan.Fields, "seconds")
	nanos := wellKnownField(plan.Fields, "nanos")
	if seconds == nil || nanos == nil {
		return nil
	}
	return jsonTextBody(fmt.Sprintf("%s.format(%s, %s)", form.Helper(), seconds.Name, nanos.Name))
}

func wellKnownFieldMaskBody(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	paths := wellKnownField(plan.Fields, "paths")
	if paths == nil {
		return nil
	}
	return jsonTextBody(fmt.Sprintf("%s.%s(%s)", form.Helper(), toJSONMethod, paths.Name))
}

// jsonTextBody writes the string a runtime helper produced. to_json has no
// error channel of its own, so a value the helper refuses -- a timestamp
// outside the representable range, a mask path that is not lower_snake_case --
// is written as null, which is at least a value the decoder rejects rather than
// silently accepts as a valid one.
func jsonTextBody(call string) []fsast.Node {
	text := generatedPrefix + "text"
	failure := generatedPrefix + "error"
	return []fsast.Node{
		line(0, fmt.Sprintf("var (%s, %s) = %s", text, failure, call)),
		line(0, fmt.Sprintf("if %s != ProtobufError.OK:", failure)),
		line(1, "return "+jsonNodeType+".Null"),
		fsast.Return{Value: jsonNodeType + ".Str(" + text + ")"},
	}
}

// wellKnownWrapperBody writes the bare scalar the wrapper carries. Unlike a
// plain field it is written whatever its value: the wrapper exists to give a
// scalar explicit presence, and the message's own presence has already been
// decided by the member holding it.
func wellKnownWrapperBody(plan *messagePlan) []fsast.Node {
	value := wellKnownField(plan.Fields, "value")
	if value == nil {
		return nil
	}
	return []fsast.Node{fsast.Return{Value: jsonValueExpression(value.Value, value.Name)}}
}

func wellKnownStructBody(plan *messagePlan) []fsast.Node {
	fields := wellKnownField(plan.Fields, "fields")
	if fields == nil || fields.Cardinality != cardinalityMap {
		return nil
	}
	key := fields.Local("key")
	return []fsast.Node{
		line(0, "var "+jsonDocument+": Dictionary[String, "+jsonNodeType+"] = {}"),
		line(0, fmt.Sprintf("for %s: %s in %s:", key, fields.Key.Type.Render(), fields.Name)),
		line(1, fmt.Sprintf(
			"%s[%s] = %s",
			jsonDocument,
			key,
			jsonValueExpression(fields.Value, fields.Name+"["+key+"]"),
		)),
		fsast.Return{Value: jsonNodeType + ".object_of(" + jsonDocument + ")"},
	}
}

func wellKnownListValueBody(plan *messagePlan) []fsast.Node {
	values := wellKnownField(plan.Fields, "values")
	if values == nil || values.Cardinality != cardinalityRepeated {
		return nil
	}
	items := values.Local("items")
	item := values.Local("item")
	return []fsast.Node{
		line(0, fmt.Sprintf("var %s: Array[%s] = []", items, jsonNodeType)),
		line(0, fmt.Sprintf("for %s: %s in %s:", item, values.Value.Type.Render(), values.Name)),
		line(1, fmt.Sprintf("%s.append(%s)", items, jsonValueExpression(values.Value, item))),
		fsast.Return{Value: jsonNodeType + ".array_of(" + items + ")"},
	}
}

// wellKnownValueBody writes a Value as whichever JSON value its kind names. An
// unset Value, and the NullValue case, are both JSON null: the enum's case name
// is not what a null renders as, so this member is the one place the general
// enum rule does not apply.
func wellKnownValueBody(plan *messagePlan) []fsast.Node {
	if len(plan.Oneofs) != 1 {
		return nil
	}
	kind := &plan.Oneofs[0]
	nodes := []fsast.Node{line(0, "match "+kind.Field+":")}
	for i := range kind.Members {
		member := &kind.Members[i]
		bound := member.Local()
		value := jsonNodeType + ".Null"
		if member.RawName != "null_value" {
			value = jsonValueExpression(member.Value, bound)
		}
		nodes = append(nodes,
			line(1, fmt.Sprintf("%s(var %s):", member.OneofCase, bound)),
			line(2, "return "+value),
		)
	}
	return append(nodes,
		line(1, "_:"),
		line(2, "return "+jsonNodeType+".Null"),
	)
}
