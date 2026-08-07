package fsgenerator

import (
	"fmt"
	"strconv"
	"strings"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// The engine's decode types, named unqualified for the same reason JsonNode is.
const (
	jsonResultType      = "JsonResult"
	jsonDecodeErrorType = "JsonDecodeError"
)

// The decode members a binding declares. from_json is the trait's half;
// _pb_merge_from_json carries the work, so a repeated, map, or oneof member is
// decoded once rather than once per entry point.
const (
	mergeFromJSONMethod  = generatedPrefix + "merge_from_json"
	fromJSONNameMethod   = "from_json_name"
	fromJSONNameArgument = "name"
)

// The locals the decoders introduce.
const (
	jsonPathParameter   = generatedPrefix + "path"
	jsonEntriesLocal    = generatedPrefix + "entries"
	jsonObjectLocal     = generatedPrefix + "object"
	jsonItemsLocal      = generatedPrefix + "items"
	jsonKeyLocal        = generatedPrefix + "key"
	jsonMemberLocal     = generatedPrefix + "member"
	jsonMemberPathLocal = generatedPrefix + "member_path"
	jsonMessageLocal    = generatedPrefix + "message"
	jsonErrorLocal      = generatedPrefix + "error"
	jsonTextLocal       = generatedPrefix + "text"
	jsonIntLocal        = generatedPrefix + "int"
	jsonFloatLocal      = generatedPrefix + "float"
	jsonBoolLocal       = generatedPrefix + "bool"
	jsonBytesLocal      = generatedPrefix + "bytes"
	jsonValueLocal      = generatedPrefix + "value"
)

// jsonRootPath is the JSONPath of the document a decode starts from. Every
// deeper path is built from it, so a failure always reads from the root.
const jsonRootPath = `"$"`

// The ProtobufError case names a JsonDecodeError message leads with. The cases
// themselves stay in the enum for numbering and are never returned; using their
// names here is what keeps the categories greppable across both halves.
const (
	jsonParseFailedPrefix     = "JSON_PARSE_FAILED: "
	jsonTypeMismatchPrefix    = "JSON_TYPE_MISMATCH: "
	jsonUnknownFieldPrefix    = "JSON_UNKNOWN_FIELD: "
	jsonValueOutOfRangePrefix = "JSON_VALUE_OUT_OF_RANGE: "
)

func fromJSONDoc(typeName string) []string {
	return []string{
		"Decodes a proto3 canonical JSON document into a new " + typeName + " message.",
		"",
		"JSON.parse_to_node(text).value produces the document; a malformed one is",
		"already reported through that JsonResult, so no text entry point is",
		"generated here.",
	}
}

func mergeFromJSONDoc() []string {
	return []string{
		"Merges a proto3 canonical JSON document into this message.",
		"",
		"A failure is returned rather than raised, matching the wire path, and",
		"carries the JSONPath of the value that could not be read.",
	}
}

func fromJSONNameDoc() []string {
	return []string{"Returns the case for a proto3 JSON name, or null if it names none."}
}

// jsonDecodeMembers are the decode members Options.JSON adds to a message.
func jsonDecodeMembers(plan *messagePlan, form wellKnownJSONForm) []fsast.Node {
	members := []fsast.Node{fromJSONFactory(plan.Name), mergeFromJSONFunction(plan, form)}
	for _, reader := range jsonReadersFor(plan, form) {
		members = append(members, jsonReaderFunction(reader))
	}
	return members
}

// fromJSONFactory is construct-then-merge: the decoding lives in the merge, so
// this only has to turn its failure into the trait's result type.
func fromJSONFactory(className string) fsast.Func {
	return fsast.Func{
		Doc:        fromJSONDoc(className),
		Static:     true,
		Name:       fromJSONMethod,
		Parameters: []fsast.Parameter{{Name: jsonNodeParameter, Type: fstypes.Named(jsonNodeType)}},
		ReturnType: fstypes.Generic(jsonResultType, fstypes.Named(className)),
		Body: []fsast.Node{
			line(0, fmt.Sprintf("var %s: %s = %s.new()", jsonMessageLocal, className, className)),
			line(0, fmt.Sprintf("var %s: %s? = %s.%s(%s)",
				jsonErrorLocal, jsonDecodeErrorType, jsonMessageLocal, mergeFromJSONMethod, jsonNodeParameter)),
			line(0, fmt.Sprintf("if %s is %s:", jsonErrorLocal, jsonDecodeErrorType)),
			line(1, fmt.Sprintf("return %s[%s].fail(%s.message, %s.path)",
				jsonResultType, className, jsonErrorLocal, jsonErrorLocal)),
			fsast.Return{Value: fmt.Sprintf("%s[%s].ok(%s)", jsonResultType, className, jsonMessageLocal)},
		},
	}
}

func mergeFromJSONFunction(plan *messagePlan, form wellKnownJSONForm) fsast.Func {
	body := wellKnownFromJSONBody(form, plan)
	if body == nil {
		body = messageFromJSONBody(plan)
	}
	return fsast.Func{
		Doc:        mergeFromJSONDoc(),
		Name:       mergeFromJSONMethod,
		Parameters: []fsast.Parameter{{Name: jsonNodeParameter, Type: fstypes.Named(jsonNodeType)}},
		ReturnType: fstypes.Nullable(fstypes.Named(jsonDecodeErrorType)),
		Body:       body,
	}
}

// messageFromJSONBody reads an ordinary message out of a JSON object.
//
// The document is walked member by member rather than field by field, because
// that is the order a failure has to be reported in and because a member the
// schema does not recognize has to be caught: JSON has no unknown-field
// preservation, so silently dropping one would lose it on the way back out.
func messageFromJSONBody(plan *messagePlan) []fsast.Node {
	body := jsonObjectGuard(0, plan.Name, jsonNodeParameter, jsonEntriesLocal, jsonRootPath)
	for _, flag := range jsonSeenFlags(plan) {
		body = append(body, line(0, fmt.Sprintf("var %s: bool = false", flag)))
	}
	body = append(body, line(0, fmt.Sprintf("for %s: String in %s:", jsonKeyLocal, jsonEntriesLocal)))
	arms := jsonFieldArms(plan)
	if len(arms) > 0 {
		body = append(body,
			line(1, fmt.Sprintf("var %s: %s = %s[%s]", jsonMemberLocal, jsonNodeType, jsonEntriesLocal, jsonKeyLocal)),
		)
	}
	body = append(body,
		line(1, fmt.Sprintf(`var %s: String = "$." + %s`, jsonMemberPathLocal, jsonKeyLocal)),
	)
	unknown := "return " + jsonFailExpression(
		strconv.Quote(jsonUnknownFieldPrefix+plan.Name+" has no field named ")+" + "+jsonKeyLocal,
		jsonMemberPathLocal,
	)
	if len(arms) == 0 {
		// A message with no fields has no key table to match against, so every
		// member of its document is unknown by construction.
		body = append(body, line(1, unknown))
		return append(body, fsast.Return{Value: "null"})
	}
	body = append(body, line(1, "match "+jsonKeyLocal+":"))
	body = append(body, arms...)
	body = append(body, line(2, "_:"), line(3, unknown))
	return append(body, fsast.Return{Value: "null"})
}

// jsonObjectGuard binds the entries of a JSON object, refusing a document of
// any other shape. A JSON null is accepted and leaves the message at its
// defaults, which is what the canonical mapping says a null member means.
func jsonObjectGuard(depth int, typeName, source, entries, path string) []fsast.Node {
	return []fsast.Node{
		line(depth, fmt.Sprintf("var %s: Dictionary[String, %s] = {}", entries, jsonNodeType)),
		line(depth, "match "+source+":"),
		line(depth+1, fmt.Sprintf("%s.Object(var %s):", jsonNodeType, jsonObjectLocal)),
		line(depth+2, entries+" = "+jsonObjectLocal),
		line(depth+1, jsonNodeType+".Null:"),
		line(depth+2, "pass"),
		line(depth+1, "_:"),
		line(depth+2, "return "+jsonFail(jsonTypeMismatchPrefix+typeName+" expects a JSON object", path)),
	}
}

// jsonFieldArms is the key table, in field-number order so it reads against the
// .proto exactly as the wire decoder's match does.
func jsonFieldArms(plan *messagePlan) []fsast.Node {
	var arms []fsast.Node
	for i := range plan.Fields {
		field := &plan.Fields[i]
		arms = append(arms, line(2, jsonMemberPatterns(field)+":"))
		arms = append(arms, jsonSeenGuard(plan, field)...)
		arms = append(arms, jsonMergeField(plan.Name, field)...)
	}
	return arms
}

// jsonSeenFlags are the flags this message needs to catch a document that names
// one logical field twice. Only two shapes can do that, so only those carry a
// flag: a field whose two accepted spellings differ, since a JSON object cannot
// repeat one key but can carry both spellings of one field, and a oneof, whose
// members are distinct keys naming one member slot.
func jsonSeenFlags(plan *messagePlan) []string {
	var flags []string
	seen := map[string]bool{}
	for i := range plan.Fields {
		flag := jsonSeenFlag(plan, &plan.Fields[i])
		if flag == "" || seen[flag] {
			continue
		}
		seen[flag] = true
		flags = append(flags, flag)
	}
	return flags
}

func jsonSeenFlag(plan *messagePlan, field *fieldPlan) string {
	if field.OneofCase != "" {
		oneof := findOneof(plan.Oneofs, field.OneofField)
		if oneof == nil {
			return ""
		}
		return localName(oneof.RawField, "seen")
	}
	if field.JSONName == field.RawName {
		return ""
	}
	return field.Local("seen")
}

// jsonSeenGuard refuses a second member for a field already read. proto3 JSON
// has no merge semantics for a repeated key: taking the last one would make the
// decode depend on member order, and a oneof would silently lose whichever
// alternative the document listed first.
func jsonSeenGuard(plan *messagePlan, field *fieldPlan) []fsast.Node {
	flag := jsonSeenFlag(plan, field)
	if flag == "" {
		return nil
	}
	report := jsonParseFailedPrefix + field.RawName + " was given more than once"
	if field.OneofCase != "" {
		oneof := findOneof(plan.Oneofs, field.OneofField)
		report = jsonParseFailedPrefix + oneof.RawField + " has more than one member set"
	}
	return []fsast.Node{
		line(3, "if "+flag+":"),
		line(4, "return "+jsonFail(report, jsonMemberPathLocal)),
		line(3, flag+" = true"),
	}
}

// jsonMemberPatterns are the spellings this field answers to: the JSON name it
// is written under, and the original proto name, which the canonical mapping
// requires a parser to accept as well. A field whose two spellings agree is
// listed once; two identical patterns in one arm would not compile.
func jsonMemberPatterns(plan *fieldPlan) string {
	if plan.JSONName == plan.RawName {
		return strconv.Quote(plan.JSONName)
	}
	return strconv.Quote(plan.JSONName) + ", " + strconv.Quote(plan.RawName)
}

// jsonMergeField reads one member into the field it names.
func jsonMergeField(className string, plan *fieldPlan) []fsast.Node {
	context := jsonDecodeContext{
		class:  className,
		local:  plan.Local(),
		source: jsonMemberLocal,
		path:   jsonMemberPathLocal,
		key:    jsonKeyLocal,
		assign: jsonFieldAssignment(plan),
	}
	switch plan.Cardinality {
	case cardinalityRepeated:
		return jsonRepeatedBody(3, plan, className, jsonMemberLocal, jsonMemberPathLocal, jsonKeyLocal)
	case cardinalityMap:
		return jsonMapBody(3, plan, className, jsonMemberLocal, jsonMemberPathLocal, jsonKeyLocal)
	}
	if !jsonFieldIsNullable(plan) {
		return jsonDecodeValue(3, plan.Value, context)
	}
	// A nullable member distinguishes unset from its default, so a JSON null
	// clears it rather than writing the default over it.
	nodes := []fsast.Node{
		line(3, "match "+jsonMemberLocal+":"),
		line(4, jsonNodeType+".Null:"),
		line(5, jsonNullAssignment(plan)),
		line(4, "_:"),
	}
	return append(nodes, jsonDecodeValue(5, plan.Value, context)...)
}

// jsonFieldIsNullable reports whether a JSON null has a meaning for this field
// beyond the value's own default: an explicit-presence member, a message-typed
// member, and a oneof member all read as unset.
func jsonFieldIsNullable(plan *fieldPlan) bool {
	return plan.Cardinality == cardinalityOptional ||
		plan.Value.Kind == kindMessage ||
		plan.OneofCase != ""
}

func jsonNullAssignment(plan *fieldPlan) string {
	if plan.OneofCase != "" {
		return plan.OneofField + " = null"
	}
	return plan.Name + " = null"
}

func jsonFieldAssignment(plan *fieldPlan) func(string) string {
	if plan.OneofCase != "" {
		return func(expression string) string {
			return fmt.Sprintf("%s = %s(%s)", plan.OneofField, plan.OneofCase, expression)
		}
	}
	return func(expression string) string {
		return plan.Name + " = " + expression
	}
}

// jsonDecodeContext is everything reading one value needs beyond the value
// itself: where the node is, where it sits in the document, and what to do with
// what comes back.
type jsonDecodeContext struct {
	// class is the message being decoded into, which is the type parameter the
	// re-rooting of a nested failure is spelled through.
	class string
	// local is the stem for the temporaries this read introduces.
	local string
	// source is the JsonNode expression to read.
	source string
	// path is the String expression naming this value's place in the document.
	path string
	// key is the String expression a nested failure is re-rooted under, or
	// empty when the nested value is the whole document and its failure is
	// already rooted where it belongs.
	key    string
	assign func(string) string
}

func jsonDecodeValue(depth int, value valuePlan, context jsonDecodeContext) []fsast.Node {
	switch value.Kind {
	case kindMessage:
		return jsonDecodeMessageValue(depth, value, context)
	case kindEnum:
		return jsonDecodeEnumValue(depth, value, context)
	default:
		return jsonDecodeScalarValue(depth, value, context)
	}
}

func jsonDecodeScalarValue(depth int, value valuePlan, context jsonDecodeContext) []fsast.Node {
	reader := jsonReaderFor(value)
	valueLocal := context.local + "_value"
	errorLocal := context.local + "_error"
	decoded := valueLocal
	if value.ProtoType == "float" {
		// A proto float is binary32 and the member holding it is binary64, so a
		// document carrying more precision than the field can is narrowed here
		// exactly as the encoder narrows it on the way out.
		decoded = "Wire.narrow_float32(" + valueLocal + ")"
	}
	return []fsast.Node{
		line(depth, fmt.Sprintf("var (%s, %s) = %s(%s, %s)",
			valueLocal, errorLocal, reader.Method(), context.source, context.path)),
		line(depth, fmt.Sprintf("if %s is %s:", errorLocal, jsonDecodeErrorType)),
		line(depth+1, "return "+errorLocal),
		line(depth, context.assign(decoded)),
	}
}

// jsonDecodeMessageValue recurses through the trait, which is what makes a
// well-known type's special form reach a field that references one without the
// field needing a case of its own. A failure comes back rooted at the nested
// document, so it is re-rooted here under the member it was read from.
func jsonDecodeMessageValue(depth int, value valuePlan, context jsonDecodeContext) []fsast.Node {
	typeName := value.Type.Render()
	resultLocal := context.local + "_result"
	errorLocal := context.local + "_error"
	valueLocal := context.local + "_value"
	return []fsast.Node{
		line(depth, fmt.Sprintf("var %s: %s[%s] = %s.%s(%s)",
			resultLocal, jsonResultType, typeName, typeName, fromJSONMethod, context.source)),
		line(depth, fmt.Sprintf("var %s: %s? = %s.error", errorLocal, jsonDecodeErrorType, resultLocal)),
		line(depth, fmt.Sprintf("if %s is %s:", errorLocal, jsonDecodeErrorType)),
		line(depth+1, jsonNestedFailure(context, errorLocal)),
		line(depth, fmt.Sprintf("var %s: %s? = %s.value", valueLocal, typeName, resultLocal)),
		line(depth, fmt.Sprintf("if not (%s is %s):", valueLocal, typeName)),
		line(depth+1, "return "+jsonFail(jsonTypeMismatchPrefix+typeName+" decoded to no value", context.path)),
		line(depth, context.assign(valueLocal)),
	}
}

// jsonNestedFailure re-roots a failure that came back from a nested decode, so
// the path it reports reads from this document's root. A nested value that is
// the whole document -- a Value's list or struct case -- is already rooted
// there, and re-rooting it would name a member that does not exist.
func jsonNestedFailure(context jsonDecodeContext, errorLocal string) string {
	if context.key == "" {
		return "return " + errorLocal
	}
	return fmt.Sprintf("return %s[%s].nested(%s, %s).error",
		jsonResultType, context.class, errorLocal, context.key)
}

// jsonDecodeEnumValue reads an enum from either spelling the canonical mapping
// allows. An unrecognized number takes the default, matching what the wire path
// does with a repeated or map-valued enum: JSON has nowhere to retain it. An
// unrecognized name is refused instead, because a name no case answers to is a
// misspelling rather than a value from a newer schema.
func jsonDecodeEnumValue(depth int, value valuePlan, context jsonDecodeContext) []fsast.Node {
	typeName := value.Type.Render()
	valueLocal := context.local + "_value"
	nameLocal := context.local + "_name"
	caseLocal := context.local + "_case"
	numberLocal := context.local + "_number"
	wireLocal := context.local + "_wire"
	return []fsast.Node{
		line(depth, fmt.Sprintf("var %s: %s = %s", valueLocal, typeName, value.ZeroValue)),
		line(depth, "match "+context.source+":"),
		line(depth+1, jsonNodeType+".Null:"),
		line(depth+2, "pass"),
		line(depth+1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, nameLocal)),
		line(depth+2, fmt.Sprintf("var %s: %s? = %s.%s(%s)", caseLocal, typeName, typeName, fromJSONNameMethod, nameLocal)),
		line(depth+2, fmt.Sprintf("if not (%s is %s):", caseLocal, typeName)),
		line(depth+3, "return "+jsonFail(jsonValueOutOfRangePrefix+typeName+" has no case with this JSON name", context.path)),
		line(depth+2, valueLocal+" = "+caseLocal),
		line(depth+1, fmt.Sprintf("%s.Int(var %s):", jsonNodeType, numberLocal)),
		line(depth+2, fmt.Sprintf("var %s: %s? = %s.from_wire(%s)", wireLocal, typeName, typeName, numberLocal)),
		line(depth+2, fmt.Sprintf("if %s is %s:", wireLocal, typeName)),
		line(depth+3, valueLocal+" = "+wireLocal),
		line(depth+1, "_:"),
		line(depth+2, "return "+jsonFail(jsonTypeMismatchPrefix+typeName+" takes a case name or a number", context.path)),
		line(depth, context.assign(valueLocal)),
	}
}

// jsonRepeatedBody reads a JSON array into a repeated field. Each element
// reports its own index, so a failure names the element rather than the field.
func jsonRepeatedBody(depth int, plan *fieldPlan, className, source, path, key string) []fsast.Node {
	items := plan.Local("items")
	index := plan.Local("index")
	element := plan.Local("element")
	context := jsonDecodeContext{
		class:  className,
		local:  element,
		source: fmt.Sprintf("%s[%s]", items, index),
		path:   jsonJoin(path, "str("+index+")"),
		key:    jsonJoin(key, "str("+index+")"),
		assign: func(expression string) string { return plan.Name + ".append(" + expression + ")" },
	}
	nodes := []fsast.Node{
		line(depth, "match "+source+":"),
		line(depth+1, jsonNodeType+".Null:"),
		line(depth+2, plan.Name+" = []"),
		line(depth+1, fmt.Sprintf("%s.Array(var %s):", jsonNodeType, items)),
		line(depth+2, plan.Name+" = []"),
		line(depth+2, fmt.Sprintf("var %s: int = 0", index)),
		line(depth+2, fmt.Sprintf("while %s < %s.size():", index, items)),
	}
	nodes = append(nodes, jsonDecodeValue(depth+3, plan.Value, context)...)
	return append(nodes,
		line(depth+3, index+" += 1"),
		line(depth+1, "_:"),
		line(depth+2, "return "+jsonFail(jsonTypeMismatchPrefix+plan.RawName+" expects a JSON array", path)),
	)
}

// jsonMapBody reads a JSON object into a map field. JSON object keys are
// strings, so a key of any other type is parsed back out of the key text rather
// than read as a JSON value of its own.
func jsonMapBody(depth int, plan *fieldPlan, className, source, path, key string) []fsast.Node {
	entries := plan.Local("entries")
	entryKey := plan.Local("key")
	context := jsonDecodeContext{
		class:  className,
		local:  plan.Local(),
		source: fmt.Sprintf("%s[%s]", entries, entryKey),
		path:   jsonJoin(path, entryKey),
		key:    jsonJoin(key, entryKey),
	}
	nodes := []fsast.Node{
		line(depth, "match "+source+":"),
		line(depth+1, jsonNodeType+".Null:"),
		line(depth+2, plan.Name+" = {}"),
		line(depth+1, fmt.Sprintf("%s.Object(var %s):", jsonNodeType, entries)),
		line(depth+2, plan.Name+" = {}"),
		line(depth+2, fmt.Sprintf("for %s: String in %s:", entryKey, entries)),
	}
	keyExpression, keyNodes := jsonMapKey(depth+3, plan, entryKey, path)
	nodes = append(nodes, keyNodes...)
	context.assign = func(expression string) string {
		return fmt.Sprintf("%s[%s] = %s", plan.Name, keyExpression, expression)
	}
	nodes = append(nodes, jsonDecodeValue(depth+3, plan.Value, context)...)
	return append(nodes,
		line(depth+1, "_:"),
		line(depth+2, "return "+jsonFail(jsonTypeMismatchPrefix+plan.RawName+" expects a JSON object", path)),
	)
}

// jsonMapKey converts one JSON object key back to the map's key type. protobuf
// allows integral, boolean and string keys, which the specification spells as
// their JSON scalar forms rather than as whatever str() produced for them.
func jsonMapKey(depth int, plan *fieldPlan, entryKey, path string) (string, []fsast.Node) {
	keyPath := plan.Local("key", "path")
	pathNode := line(depth, fmt.Sprintf("var %s: String = %s", keyPath, jsonJoin(path, entryKey)))
	switch plan.Key.ProtoType {
	case "string":
		return entryKey, nil
	case "bool":
		valueLocal := plan.Local("key", "value")
		return valueLocal, []fsast.Node{
			pathNode,
			line(depth, fmt.Sprintf("var %s: bool = false", valueLocal)),
			line(depth, fmt.Sprintf(`if %s == "true":`, entryKey)),
			line(depth+1, valueLocal+" = true"),
			line(depth, fmt.Sprintf(`elif %s != "false":`, entryKey)),
			line(depth+1, "return "+jsonFail(jsonTypeMismatchPrefix+`a bool map key takes "true" or "false"`, keyPath)),
		}
	default:
		valueLocal := plan.Local("key", "value")
		errorLocal := plan.Local("key", "error")
		reader := jsonReaderFor(plan.Key)
		return valueLocal, []fsast.Node{
			pathNode,
			line(depth, fmt.Sprintf("var (%s, %s) = %s(%s.Str(%s), %s)",
				valueLocal, errorLocal, reader.Method(), jsonNodeType, entryKey, keyPath)),
			line(depth, fmt.Sprintf("if %s is %s:", errorLocal, jsonDecodeErrorType)),
			line(depth+1, "return "+errorLocal),
		}
	}
}

// jsonJoin appends one segment to a JSONPath expression. An empty base is the
// document root itself, which a segment names on its own.
func jsonJoin(path, segment string) string {
	if path == "" {
		return segment
	}
	// A literal base takes the separator inside the quotes rather than
	// concatenating a one-character string at run time.
	if strings.HasPrefix(path, `"`) && strings.HasSuffix(path, `"`) {
		return path[:len(path)-1] + `." + ` + segment
	}
	return path + ` + "." + ` + segment
}

func jsonFail(message, pathExpression string) string {
	return jsonFailExpression(strconv.Quote(message), pathExpression)
}

func jsonFailExpression(messageExpression, pathExpression string) string {
	return fmt.Sprintf("%s.create(%s, %s)", jsonDecodeErrorType, messageExpression, pathExpression)
}

// enumFromJSONNameFunction hosts the reverse of to_json_name on the enum, for
// the same reason from_wire is hosted there: the case names are declared once.
func enumFromJSONNameFunction(typeName string, enum *protoast.Enum) fsast.Func {
	body := []fsast.Node{line(0, "match "+fromJSONNameArgument+":")}
	for _, value := range enum.Values {
		// An allow_alias enum declares one number twice but never one name, so
		// every declared spelling names a case of its own.
		body = append(body,
			line(1, strconv.Quote(value.Name)+":"),
			line(2, "return "+typeName+"."+EnumValueName(value.Name)),
		)
	}
	body = append(body, line(1, "_:"), line(2, "return null"))
	return fsast.Func{
		Doc:        fromJSONNameDoc(),
		Static:     true,
		Name:       fromJSONNameMethod,
		Parameters: []fsast.Parameter{{Name: fromJSONNameArgument, Type: fstypes.Named("String")}},
		// Self is the spelling that resolves to the namespaced enum from inside
		// its own body, exactly as it is for from_wire.
		ReturnType: fstypes.Nullable(fstypes.Named("Self")),
		Body:       body,
	}
}

// wellKnownFromJSONBody is the body of _pb_merge_from_json for a well-known
// type, or nil for an ordinary message. Empty is deliberately absent: an empty
// JSON object is what the ordinary body already reads, and going through it is
// what makes an unexpected member an error rather than something ignored.
func wellKnownFromJSONBody(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	switch form {
	case wellKnownJSONTimestamp, wellKnownJSONDuration:
		return wellKnownSecondsFromJSONBody(form, plan)
	case wellKnownJSONFieldMask:
		return wellKnownFieldMaskFromJSONBody(form, plan)
	case wellKnownJSONWrapper:
		return wellKnownWrapperFromJSONBody(plan)
	case wellKnownJSONStruct:
		return wellKnownStructFromJSONBody(plan)
	case wellKnownJSONListValue:
		return wellKnownListValueFromJSONBody(plan)
	case wellKnownJSONValue:
		return wellKnownValueFromJSONBody(plan)
	case wellKnownJSONAny:
		return []fsast.Node{
			line(0, "var (_pb_type_url, _pb_value, _pb_error) = AnyTypeRegistry._any_from_json(_pb_node)"),
			line(0, "if _pb_error is JsonDecodeError:"),
			line(1, "return _pb_error"),
			line(0, "type_url = _pb_type_url"),
			line(0, "value = _pb_value"),
			fsast.Return{Value: "null"},
		}
	default:
		return nil
	}
}

// wellKnownSecondsFromJSONBody covers Timestamp and Duration, which differ only
// in which runtime helper parses their text.
func wellKnownSecondsFromJSONBody(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	seconds := wellKnownField(plan.Fields, "seconds")
	nanos := wellKnownField(plan.Fields, "nanos")
	if seconds == nil || nanos == nil {
		return nil
	}
	secondsLocal := generatedPrefix + "seconds"
	nanosLocal := generatedPrefix + "nanos"
	return jsonTextFromJSONBody(plan.Name, []fsast.Node{
		line(2, fmt.Sprintf("var (%s, %s, %s) = %s.parse(%s)",
			secondsLocal, nanosLocal, jsonErrorLocal, form.Helper(), jsonTextLocal)),
		line(2, fmt.Sprintf("if %s != ProtobufError.OK:", jsonErrorLocal)),
		line(3, "return "+jsonFail(
			jsonValueOutOfRangePrefix+plan.Name+" cannot be decoded from this JSON string", jsonRootPath)),
		line(2, seconds.Name+" = "+secondsLocal),
		line(2, nanos.Name+" = "+nanosLocal),
	})
}

func wellKnownFieldMaskFromJSONBody(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	paths := wellKnownField(plan.Fields, "paths")
	if paths == nil {
		return nil
	}
	pathsLocal := generatedPrefix + "paths"
	return jsonTextFromJSONBody(plan.Name, []fsast.Node{
		line(2, fmt.Sprintf("var (%s, %s) = %s.%s(%s)",
			pathsLocal, jsonErrorLocal, form.Helper(), fromJSONMethod, jsonTextLocal)),
		line(2, fmt.Sprintf("if %s != ProtobufError.OK:", jsonErrorLocal)),
		line(3, "return "+jsonFail(
			jsonValueOutOfRangePrefix+plan.Name+" cannot be decoded from this JSON string", jsonRootPath)),
		line(2, paths.Name+" = "+pathsLocal),
	})
}

// jsonTextFromJSONBody frames a form whose whole JSON representation is one
// string, leaving the parse itself to the caller's statements.
func jsonTextFromJSONBody(typeName string, parse []fsast.Node) []fsast.Node {
	nodes := []fsast.Node{
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
	}
	nodes = append(nodes, parse...)
	return append(nodes,
		line(1, "_:"),
		line(2, "return "+jsonFail(jsonTypeMismatchPrefix+typeName+" expects a JSON string", jsonRootPath)),
		fsast.Return{Value: "null"},
	)
}

// wellKnownWrapperFromJSONBody reads the bare scalar the wrapper carries: the
// whole document is the value, since a wrapper's presence was already decided
// by the member that held it.
func wellKnownWrapperFromJSONBody(plan *messagePlan) []fsast.Node {
	value := wellKnownField(plan.Fields, "value")
	if value == nil {
		return nil
	}
	nodes := jsonDecodeValue(0, value.Value, jsonDecodeContext{
		class:  plan.Name,
		local:  value.Local(),
		source: jsonNodeParameter,
		path:   jsonRootPath,
		assign: func(expression string) string { return value.Name + " = " + expression },
	})
	return append(nodes, fsast.Return{Value: "null"})
}

func wellKnownStructFromJSONBody(plan *messagePlan) []fsast.Node {
	fields := wellKnownField(plan.Fields, "fields")
	if fields == nil || fields.Cardinality != cardinalityMap {
		return nil
	}
	entryKey := fields.Local("key")
	nodes := jsonObjectGuard(0, plan.Name, jsonNodeParameter, jsonEntriesLocal, jsonRootPath)
	nodes = append(nodes,
		line(0, fields.Name+" = {}"),
		line(0, fmt.Sprintf("for %s: String in %s:", entryKey, jsonEntriesLocal)),
	)
	nodes = append(nodes, jsonDecodeValue(1, fields.Value, jsonDecodeContext{
		class:  plan.Name,
		local:  fields.Local(),
		source: fmt.Sprintf("%s[%s]", jsonEntriesLocal, entryKey),
		path:   jsonJoin(jsonRootPath, entryKey),
		key:    entryKey,
		assign: func(expression string) string {
			return fmt.Sprintf("%s[%s] = %s", fields.Name, entryKey, expression)
		},
	})...)
	return append(nodes, fsast.Return{Value: "null"})
}

func wellKnownListValueFromJSONBody(plan *messagePlan) []fsast.Node {
	values := wellKnownField(plan.Fields, "values")
	if values == nil || values.Cardinality != cardinalityRepeated {
		return nil
	}
	nodes := jsonRepeatedBody(0, values, plan.Name, jsonNodeParameter, jsonRootPath, "")
	return append(nodes, fsast.Return{Value: "null"})
}

// wellKnownValueFromJSONBody maps the engine's JsonNode onto Value case for
// case, which is what makes the two agree about what a JSON value is. Int and
// Float both land on number_value, because Value carries a double and has no
// integral case of its own.
func wellKnownValueFromJSONBody(plan *messagePlan) []fsast.Node {
	if len(plan.Oneofs) != 1 {
		return nil
	}
	kind := &plan.Oneofs[0]
	nodes := []fsast.Node{line(0, "match "+jsonNodeParameter+":")}
	if member := wellKnownOneofMember(kind, "null_value"); member != nil {
		nodes = append(nodes,
			line(1, jsonNodeType+".Null:"),
			line(2, fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, member.Value.ZeroValue)),
		)
	}
	if member := wellKnownOneofMember(kind, "bool_value"); member != nil {
		nodes = append(nodes,
			line(1, fmt.Sprintf("%s.Bool(var %s):", jsonNodeType, jsonBoolLocal)),
			line(2, fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, jsonBoolLocal)),
		)
	}
	if member := wellKnownOneofMember(kind, "number_value"); member != nil {
		nodes = append(nodes,
			line(1, fmt.Sprintf("%s.Int(var %s):", jsonNodeType, jsonIntLocal)),
			line(2, fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, jsonIntLocal)),
			line(1, fmt.Sprintf("%s.Float(var %s):", jsonNodeType, jsonFloatLocal)),
			line(2, fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, jsonFloatLocal)),
		)
	}
	if member := wellKnownOneofMember(kind, "string_value"); member != nil {
		nodes = append(nodes,
			line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
			line(2, fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, jsonTextLocal)),
		)
	}
	if member := wellKnownOneofMember(kind, "list_value"); member != nil {
		nodes = append(nodes, line(1, fmt.Sprintf("%s.Array(var %s):", jsonNodeType, jsonItemsLocal)))
		nodes = append(nodes, jsonDecodeValue(2, member.Value, jsonDecodeContext{
			class:  plan.Name,
			local:  member.Local(),
			source: fmt.Sprintf("%s.array_of(%s)", jsonNodeType, jsonItemsLocal),
			path:   jsonRootPath,
			assign: func(expression string) string {
				return fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, expression)
			},
		})...)
	}
	if member := wellKnownOneofMember(kind, "struct_value"); member != nil {
		nodes = append(nodes, line(1, fmt.Sprintf("%s.Object(var %s):", jsonNodeType, jsonObjectLocal)))
		nodes = append(nodes, jsonDecodeValue(2, member.Value, jsonDecodeContext{
			class:  plan.Name,
			local:  member.Local(),
			source: fmt.Sprintf("%s.object_of(%s)", jsonNodeType, jsonObjectLocal),
			path:   jsonRootPath,
			assign: func(expression string) string {
				return fmt.Sprintf("%s = %s(%s)", kind.Field, member.OneofCase, expression)
			},
		})...)
	}
	return append(nodes,
		line(1, "_:"),
		line(2, "return "+jsonFail(jsonTypeMismatchPrefix+plan.Name+" cannot represent this JSON value", jsonRootPath)),
		fsast.Return{Value: "null"},
	)
}

// wellKnownOneofMember is a union member found by its protobuf name, for the
// same reason wellKnownField is: a well-known binding is generated like any
// other, so its members carry whatever escaping the naming rules applied.
func wellKnownOneofMember(oneof *oneofPlan, protoName string) *fieldPlan {
	for i := range oneof.Members {
		if oneof.Members[i].RawName == protoName {
			return &oneof.Members[i]
		}
	}
	return nil
}

// jsonReader is one of the scalar readers a binding hosts. They are emitted per
// message rather than shared in the runtime for the same reason the float
// writer is: they are generated members of the binding, so a project that never
// asks for JSON carries none of them.
type jsonReader int

const (
	jsonReaderNone jsonReader = iota
	jsonReaderInt32
	jsonReaderUint32
	jsonReaderInt64
	jsonReaderUint64
	jsonReaderFloat
	jsonReaderBool
	jsonReaderString
	jsonReaderBytes
)

// jsonReaderOrder is the order readers are emitted in, so the output of two
// runs over the same schema is identical.
var jsonReaderOrder = []jsonReader{
	jsonReaderInt32, jsonReaderUint32, jsonReaderInt64, jsonReaderUint64,
	jsonReaderFloat, jsonReaderBool, jsonReaderString, jsonReaderBytes,
}

// jsonReaderFor is the reader one value is read through. The two unsigned
// 64-bit types get a reader of their own, matching what the serializer writes
// for them: the top half of their range has no signed spelling, so neither the
// bounds nor the text conversion of the signed reader applies to them.
func jsonReaderFor(value valuePlan) jsonReader {
	if value.Kind != kindScalar {
		return jsonReaderNone
	}
	switch value.ProtoType {
	case "int32", "sint32", "sfixed32":
		return jsonReaderInt32
	case "uint32", "fixed32":
		return jsonReaderUint32
	case "int64", "sint64", "sfixed64":
		return jsonReaderInt64
	case "uint64", "fixed64":
		return jsonReaderUint64
	case "float", "double":
		return jsonReaderFloat
	case "bool":
		return jsonReaderBool
	case "string":
		return jsonReaderString
	case "bytes":
		return jsonReaderBytes
	default:
		return jsonReaderNone
	}
}

// Method is the emitted name of this reader.
func (r jsonReader) Method() string {
	switch r {
	case jsonReaderInt32:
		return generatedPrefix + "json_read_int32"
	case jsonReaderUint32:
		return generatedPrefix + "json_read_uint32"
	case jsonReaderInt64:
		return generatedPrefix + "json_read_int64"
	case jsonReaderUint64:
		return generatedPrefix + "json_read_uint64"
	case jsonReaderFloat:
		return generatedPrefix + "json_read_float"
	case jsonReaderBool:
		return generatedPrefix + "json_read_bool"
	case jsonReaderString:
		return generatedPrefix + "json_read_string"
	case jsonReaderBytes:
		return generatedPrefix + "json_read_bytes"
	default:
		return ""
	}
}

// jsonReadersFor are the readers this message calls, so none is emitted that
// would be dead code in the file.
func jsonReadersFor(plan *messagePlan, form wellKnownJSONForm) []jsonReader {
	needed := map[jsonReader]bool{}
	switch form {
	case wellKnownJSONNone, wellKnownJSONEmpty:
		for i := range plan.Fields {
			field := &plan.Fields[i]
			needed[jsonReaderFor(field.Value)] = true
			if field.Cardinality == cardinalityMap && field.Key.ProtoType != "bool" {
				needed[jsonReaderFor(field.Key)] = true
			}
		}
	case wellKnownJSONWrapper:
		if value := wellKnownField(plan.Fields, "value"); value != nil {
			needed[jsonReaderFor(value.Value)] = true
		}
	}
	readers := make([]jsonReader, 0, len(needed))
	for _, reader := range jsonReaderOrder {
		if needed[reader] {
			readers = append(readers, reader)
		}
	}
	return readers
}

func jsonReaderFunction(reader jsonReader) fsast.Func {
	doc, resultType, body := jsonReaderShape(reader)
	return fsast.Func{
		Doc:    doc,
		Static: true,
		Name:   reader.Method(),
		Parameters: []fsast.Parameter{
			{Name: jsonNodeParameter, Type: fstypes.Named(jsonNodeType)},
			{Name: jsonPathParameter, Type: fstypes.Named("String")},
		},
		ReturnType: fstypes.Tuple(fstypes.Named(resultType), fstypes.Nullable(fstypes.Named(jsonDecodeErrorType))),
		Body:       body,
	}
}

func jsonReaderShape(reader jsonReader) (doc []string, resultType string, body []fsast.Node) {
	switch reader {
	case jsonReaderInt32:
		return jsonIntegerReaderDoc("a signed 32-bit integer"),
			"int",
			jsonIntegerReaderBody("a signed 32-bit integer", "-2147483648", "2147483647", "-2147483648.0", "2147483648.0", "int")
	case jsonReaderUint32:
		return jsonIntegerReaderDoc("an unsigned 32-bit integer"),
			"uint",
			jsonIntegerReaderBody("an unsigned 32-bit integer", "0", "4294967295", "0.0", "4294967296.0", "uint")
	case jsonReaderInt64:
		return jsonWideIntegerReaderDoc(),
			"long",
			jsonIntegerReaderBody("a 64-bit integer", "", "", "-9223372036854775808.0", "9223372036854775808.0", "long")
	case jsonReaderUint64:
		return jsonUnsignedWideIntegerReaderDoc(), "ulong", jsonUnsignedWideIntegerReaderBody()
	case jsonReaderFloat:
		return jsonFloatReaderDoc(), "float", jsonFloatReaderBody()
	case jsonReaderBool:
		return jsonReaderDoc("a bool"), "bool", jsonBoolReaderBody()
	case jsonReaderString:
		return jsonReaderDoc("a string"), "String", jsonStringReaderBody()
	default:
		return jsonBytesReaderDoc(), "PackedByteArray", jsonBytesReaderBody()
	}
}

func jsonReaderDoc(description string) []string {
	return []string{"Reads " + description + " field out of a JSON value."}
}

func jsonIntegerReaderDoc(description string) []string {
	return []string{
		"Reads " + description + " field out of a JSON value.",
		"",
		"The canonical mapping accepts a JSON string and a whole JSON number as",
		"well as the number this emitter writes, so all three are read here. A",
		"value outside the field's domain is refused rather than truncated.",
	}
}

func jsonWideIntegerReaderDoc() []string {
	return []string{
		"Reads a 64-bit integer field out of a JSON value.",
		"",
		"A string is exact and is what this emitter writes. A bare number is",
		"accepted because the canonical mapping requires it, and is lossy past",
		"2^53: the engine's parser produces a double, so a value that large does",
		"not even arrive as a JsonNode.Int.",
	}
}

func jsonUnsignedWideIntegerReaderDoc() []string {
	return []string{
		"Reads an unsigned 64-bit integer field out of a JSON value.",
		"",
		"The top half of the range has no signed spelling, so the text goes",
		"through the runtime helper rather than String.to_int(), which wraps to",
		"the smallest signed value there. A bare number is accepted because the",
		"canonical mapping requires it, and is lossy past 2^53: the engine's",
		"parser produces a double, so a value that large does not even arrive as",
		"a JsonNode.Int. The widest value rounds to 2^64 on the way in and is",
		"read as the value it rounded from rather than refused.",
	}
}

// jsonUnsignedWideIntegerReaderBody reads a uint64 or fixed64. The value is
// held as a ulong, matching the field type and what the serializer writes back.
//
// The float arm is bounded before it is converted, for the same reason the
// signed reader's is. Exactly 2^64 is the widest value having been rounded
// rather than a value out of range — no double holds those twenty digits — so
// it is the documented lossy edge the signed reader has at 2^63, not a document
// to refuse.
func jsonUnsignedWideIntegerReaderBody() []fsast.Node {
	const description = "an unsigned 64-bit integer"
	outOfRange := "return (0UL, " + jsonFail(
		jsonValueOutOfRangePrefix+description+" field cannot hold this value", jsonPathParameter) + ")"
	unsigned := generatedPrefix + "unsigned"
	unsignedError := generatedPrefix + "unsigned_error"
	return append([]fsast.Node{
		line(0, fmt.Sprintf("var %s: ulong = 0UL", jsonValueLocal)),
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Int(var %s):", jsonNodeType, jsonIntLocal)),
		line(2, fmt.Sprintf("if %s < 0:", jsonIntLocal)),
		line(3, outOfRange),
		line(2, jsonValueLocal+" = "+jsonIntLocal+" as ulong"),
		line(1, fmt.Sprintf("%s.Float(var %s):", jsonNodeType, jsonFloatLocal)),
		line(2, fmt.Sprintf("if %s != floor(%s):", jsonFloatLocal, jsonFloatLocal)),
		line(3, "return (0UL, "+jsonFail(
			jsonTypeMismatchPrefix+description+" field cannot take a fractional number", jsonPathParameter)+")"),
		line(2, fmt.Sprintf("if %s > 18446744073709551616.0 or %s < 0.0:", jsonFloatLocal, jsonFloatLocal)),
		line(3, outOfRange),
		line(2, fmt.Sprintf("if %s == 18446744073709551616.0:", jsonFloatLocal)),
		line(3, jsonValueLocal+" = 18446744073709551615UL"),
		line(2, "else:"),
		line(3, fmt.Sprintf("%s = (%s as long) as ulong", jsonValueLocal, jsonFloatLocal)),
		line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
		line(2, fmt.Sprintf("var (%s, %s) = %s.parse(%s)", unsigned, unsignedError, jsonUint64Type, jsonTextLocal)),
		line(2, fmt.Sprintf("if %s == ProtobufError.JSON_VALUE_OUT_OF_RANGE:", unsignedError)),
		line(3, outOfRange),
		line(2, fmt.Sprintf("if %s != ProtobufError.OK:", unsignedError)),
		line(3, "return (0UL, "+jsonFail(
			jsonTypeMismatchPrefix+description+" field cannot take this string", jsonPathParameter)+")"),
		line(2, jsonValueLocal+" = "+unsigned),
		line(1, "_:"),
		line(2, "return (0UL, "+jsonFail(
			jsonTypeMismatchPrefix+description+" field takes a number or a string", jsonPathParameter)+")"),
	}, jsonReaderSuccess(jsonValueLocal)...)
}

func jsonFloatReaderDoc() []string {
	return []string{
		"Reads a floating-point field out of a JSON value.",
		"",
		"The three non-finite values have no JSON number form, so the canonical",
		"mapping spells them as strings and they are read back from those.",
	}
}

func jsonBytesReaderDoc() []string {
	return []string{
		"Reads a bytes field out of a JSON value.",
		"",
		"The runtime helper accepts the URL-safe alphabet and optional padding as",
		"well as the standard form, which is what the mapping asks a parser for.",
	}
}

// jsonIntegerReaderBody reads one integer. A JSON null leaves the field at its
// proto default, which is what the canonical mapping says a null member means.
//
// The float arm is bounded before it is converted rather than after: a value
// too large for the host int does not survive the conversion intact, so
// checking the result would be checking a number the document never carried.
func jsonIntegerReaderBody(description, minimum, maximum, floatMinimum, floatBound, resultType string) []fsast.Node {
	// Error returns use the literal form of the declared result type, because
	// the function signature fixes the tuple's first slot to that type and an
	// unsuffixed 0 infers int — the wrong carrier for uint/ulong returns.
	zero := "0"
	switch resultType {
	case "uint":
		zero = "0U"
	case "long":
		zero = "0L"
	case "ulong":
		zero = "0UL"
	}
	body := []fsast.Node{
		line(0, fmt.Sprintf("var %s: long = 0", jsonValueLocal)),
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Int(var %s):", jsonNodeType, jsonIntLocal)),
		line(2, jsonValueLocal+" = "+jsonIntLocal),
		line(1, fmt.Sprintf("%s.Float(var %s):", jsonNodeType, jsonFloatLocal)),
		line(2, fmt.Sprintf("if %s != floor(%s):", jsonFloatLocal, jsonFloatLocal)),
		line(3, "return ("+zero+", "+jsonFail(
			jsonTypeMismatchPrefix+description+" field cannot take a fractional number", jsonPathParameter)+")"),
		line(2, fmt.Sprintf("if %s %s %s or %s < %s:",
			jsonFloatLocal, jsonFloatComparison(minimum), floatBound, jsonFloatLocal, floatMinimum)),
		line(3, "return ("+zero+", "+jsonFail(
			jsonValueOutOfRangePrefix+description+" field cannot hold this value", jsonPathParameter)+")"),
		line(2, fmt.Sprintf("%s = %s as long", jsonValueLocal, jsonFloatLocal)),
		line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
		line(2, fmt.Sprintf("if not %s.is_valid_int():", jsonTextLocal)),
		line(3, "return ("+zero+", "+jsonFail(
			jsonTypeMismatchPrefix+description+" field cannot take this string", jsonPathParameter)+")"),
		line(2, fmt.Sprintf("%s = %s.to_int() as long", jsonValueLocal, jsonTextLocal)),
	}
	if minimum == "" {
		// A 64-bit field has no range check after the match to catch a string
		// the host int cannot hold, and to_int() wraps rather than reporting.
		// Requiring the text to be exactly what the value prints as is what
		// separates a value that survived from one that came back different.
		body = append(body,
			line(2, fmt.Sprintf("if str(%s) != %s:", jsonValueLocal, jsonTextLocal)),
			line(3, "return ("+zero+", "+jsonFail(
				jsonValueOutOfRangePrefix+description+" field takes a decimal string it can hold exactly",
				jsonPathParameter)+")"),
		)
	}
	body = append(body, []fsast.Node{
		line(1, "_:"),
		line(2, "return ("+zero+", "+jsonFail(
			jsonTypeMismatchPrefix+description+" field takes a number or a string", jsonPathParameter)+")"),
	}...)
	if minimum != "" {
		body = append(body,
			line(0, fmt.Sprintf("if %s < %s or %s > %s:", jsonValueLocal, minimum, jsonValueLocal, maximum)),
			line(1, "return ("+zero+", "+jsonFail(
				jsonValueOutOfRangePrefix+description+" field cannot hold this value", jsonPathParameter)+")"),
		)
	}
	return append(body, jsonReaderSuccessTyped(jsonValueLocal, resultType)...)
}

func jsonFloatReaderBody() []fsast.Node {
	return []fsast.Node{
		line(0, fmt.Sprintf("var %s: float = 0.0", jsonValueLocal)),
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Float(var %s):", jsonNodeType, jsonFloatLocal)),
		line(2, jsonValueLocal+" = "+jsonFloatLocal),
		line(1, fmt.Sprintf("%s.Int(var %s):", jsonNodeType, jsonIntLocal)),
		line(2, jsonValueLocal+" = "+jsonIntLocal),
		line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
		line(2, fmt.Sprintf(`if %s == "NaN":`, jsonTextLocal)),
		line(3, jsonValueLocal+" = NAN"),
		line(2, fmt.Sprintf(`elif %s == "Infinity":`, jsonTextLocal)),
		line(3, jsonValueLocal+" = INF"),
		line(2, fmt.Sprintf(`elif %s == "-Infinity":`, jsonTextLocal)),
		line(3, jsonValueLocal+" = -INF"),
		line(2, "else:"),
		line(3, "return (0.0, "+jsonFail(
			jsonTypeMismatchPrefix+"a floating-point field takes a number or one of the three non-finite strings",
			jsonPathParameter)+")"),
		line(1, "_:"),
		line(2, "return (0.0, "+jsonFail(
			jsonTypeMismatchPrefix+"a floating-point field takes a number or one of the three non-finite strings",
			jsonPathParameter)+")"),
		line(0, fmt.Sprintf("var %s: %s? = null", jsonErrorLocal, jsonDecodeErrorType)),
		fsast.Return{Value: fmt.Sprintf("(%s, %s)", jsonValueLocal, jsonErrorLocal)},
	}
}

func jsonBoolReaderBody() []fsast.Node {
	return []fsast.Node{
		line(0, fmt.Sprintf("var %s: bool = false", jsonValueLocal)),
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Bool(var %s):", jsonNodeType, jsonBoolLocal)),
		line(2, jsonValueLocal+" = "+jsonBoolLocal),
		line(1, "_:"),
		line(2, "return (false, "+jsonFail(
			jsonTypeMismatchPrefix+"a bool field takes true or false", jsonPathParameter)+")"),
		line(0, fmt.Sprintf("var %s: %s? = null", jsonErrorLocal, jsonDecodeErrorType)),
		fsast.Return{Value: fmt.Sprintf("(%s, %s)", jsonValueLocal, jsonErrorLocal)},
	}
}

func jsonStringReaderBody() []fsast.Node {
	return []fsast.Node{
		line(0, fmt.Sprintf(`var %s: String = ""`, jsonValueLocal)),
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
		line(2, jsonValueLocal+" = "+jsonTextLocal),
		line(1, "_:"),
		line(2, `return ("", `+jsonFail(
			jsonTypeMismatchPrefix+"a string field takes a JSON string", jsonPathParameter)+")"),
		line(0, fmt.Sprintf("var %s: %s? = null", jsonErrorLocal, jsonDecodeErrorType)),
		fsast.Return{Value: fmt.Sprintf("(%s, %s)", jsonValueLocal, jsonErrorLocal)},
	}
}

func jsonBytesReaderBody() []fsast.Node {
	decodeError := generatedPrefix + "bytes_error"
	return []fsast.Node{
		line(0, fmt.Sprintf("var %s: PackedByteArray = PackedByteArray()", jsonValueLocal)),
		line(0, "match "+jsonNodeParameter+":"),
		line(1, jsonNodeType+".Null:"),
		line(2, "pass"),
		line(1, fmt.Sprintf("%s.Str(var %s):", jsonNodeType, jsonTextLocal)),
		line(2, fmt.Sprintf("var (%s, %s) = JsonBase64.decode(%s)", jsonBytesLocal, decodeError, jsonTextLocal)),
		line(2, fmt.Sprintf("if %s != ProtobufError.OK:", decodeError)),
		line(3, "return (PackedByteArray(), "+jsonFail(
			jsonTypeMismatchPrefix+"a bytes field takes base64 text", jsonPathParameter)+")"),
		line(2, jsonValueLocal+" = "+jsonBytesLocal),
		line(1, "_:"),
		line(2, "return (PackedByteArray(), "+jsonFail(
			jsonTypeMismatchPrefix+"a bytes field takes base64 text", jsonPathParameter)+")"),
		line(0, fmt.Sprintf("var %s: %s? = null", jsonErrorLocal, jsonDecodeErrorType)),
		fsast.Return{Value: fmt.Sprintf("(%s, %s)", jsonValueLocal, jsonErrorLocal)},
	}
}

// jsonReaderSuccess returns a value with no failure. The null is declared with
// its type first: a bare null in a tuple does not carry the element type the
// return needs.
func jsonReaderSuccess(valueLocal string) []fsast.Node {
	return jsonReaderSuccessTyped(valueLocal, "")
}

// jsonReaderSuccessTyped emits the shared success epilogue, narrowing value to
// castType when one is given. The integer readers keep their intermediate in a
// long so the JSON parsing and range checks work uniformly, then narrow once at
// the boundary to the field's declared type.
func jsonReaderSuccessTyped(valueLocal, castType string) []fsast.Node {
	valueExpr := valueLocal
	if castType != "" {
		valueExpr = valueLocal + " as " + castType
	}
	return []fsast.Node{
		line(0, fmt.Sprintf("var %s: %s? = null", jsonErrorLocal, jsonDecodeErrorType)),
		fsast.Return{Value: fmt.Sprintf("(%s, %s)", valueExpr, jsonErrorLocal)},
	}
}

// jsonFloatComparison bounds the float arm of an integer reader. A 32-bit field
// is bounded by the first value it cannot hold, while a 64-bit one is bounded
// by the first value past the host int: the engine's parser produces a double,
// so the largest int64 arrives as exactly 2^63 and is the documented lossy case
// rather than a value to refuse.
func jsonFloatComparison(minimum string) string {
	if minimum == "" {
		return ">"
	}
	return ">="
}
