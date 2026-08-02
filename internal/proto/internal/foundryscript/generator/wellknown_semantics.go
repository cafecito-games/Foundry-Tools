package fsgenerator

import (
	"fmt"

	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

const (
	nativeValueParameter = generatedPrefix + "value"
	nativeResultLocal    = generatedPrefix + "result"
	nativeFailedLocal    = generatedPrefix + "failed"
	nativeConvertedLocal = generatedPrefix + "converted"
	nativeErrorLocal     = generatedPrefix + "error"
	nativeKeyLocal       = generatedPrefix + "key"
	nativeItemLocal      = generatedPrefix + "item"
)

var variantType = fstypes.Named("Variant")

// wellKnownNativeMembers are the non-wire conveniences hosted by the runtime's
// generated well-known bindings. The form table already distinguishes these
// declarations from an identically named type in a caller's schema, so native
// semantics use the same discriminator as canonical JSON.
func wellKnownNativeMembers(form wellKnownJSONForm, plan *messagePlan) []fsast.Node {
	switch form {
	case wellKnownJSONStruct:
		return structNativeMembers(plan)
	case wellKnownJSONValue:
		return valueNativeMembers(plan)
	case wellKnownJSONListValue:
		return listValueNativeMembers(plan)
	case wellKnownJSONTimestamp:
		return timestampNativeMembers(plan)
	case wellKnownJSONDuration:
		return durationNativeMembers(plan)
	default:
		return nil
	}
}

func timestampNativeMembers(plan *messagePlan) []fsast.Node {
	seconds := wellKnownField(plan.Fields, "seconds")
	nanos := wellKnownField(plan.Fields, "nanos")
	if seconds == nil || nanos == nil {
		return nil
	}

	whole := generatedPrefix + "whole"
	return []fsast.Node{
		fsast.Func{
			Doc: []string{
				"Converts unix seconds into a normalized Timestamp.",
				"",
				"The seconds field is floored so a negative timestamp keeps nonnegative",
				"nanos. Nanoseconds rounded to a full second carry into seconds.",
			},
			Static:     true,
			Name:       "from_unix_time",
			Parameters: []fsast.Parameter{{Name: nativeValueParameter, Type: fstypes.Named("float")}},
			ReturnType: fstypes.Named(plan.Name),
			Body: []fsast.Node{
				line(0, fmt.Sprintf("var %s: %s = %s.new()", nativeResultLocal, plan.Name, plan.Name)),
				line(0, fmt.Sprintf("var %s: int = floori(%s)", whole, nativeValueParameter)),
				line(0, fmt.Sprintf("%s.%s = %s", nativeResultLocal, seconds.Name, whole)),
				line(0, fmt.Sprintf("%s.%s = roundi((%s - float(%s)) * 1000000000.0)",
					nativeResultLocal, nanos.Name, nativeValueParameter, whole)),
				line(0, fmt.Sprintf("if %s.%s >= 1000000000:", nativeResultLocal, nanos.Name)),
				line(1, fmt.Sprintf("%s.%s -= 1000000000", nativeResultLocal, nanos.Name)),
				line(1, fmt.Sprintf("%s.%s += 1", nativeResultLocal, seconds.Name)),
				fsast.Return{Value: nativeResultLocal},
			},
		},
		fsast.Func{
			Doc:        []string{"Returns the current system time as a Timestamp."},
			Static:     true,
			Name:       "now",
			ReturnType: fstypes.Named(plan.Name),
			Body: []fsast.Node{
				fsast.Return{Value: plan.Name + ".from_unix_time(Time.get_unix_time_from_system())"},
			},
		},
		fsast.Func{
			Doc: []string{
				"Returns this Timestamp as unix seconds.",
				"",
				"This direction is lossy: a float near the present epoch resolves to about",
				"238 nanoseconds. Read seconds and nanos directly when full precision matters.",
			},
			Name:       "to_unix_time",
			ReturnType: fstypes.Named("float"),
			Body: []fsast.Node{
				fsast.Return{Value: fmt.Sprintf("float(%s) + float(%s) / 1000000000.0", seconds.Name, nanos.Name)},
			},
		},
	}
}

func durationNativeMembers(plan *messagePlan) []fsast.Node {
	seconds := wellKnownField(plan.Fields, "seconds")
	nanos := wellKnownField(plan.Fields, "nanos")
	if seconds == nil || nanos == nil {
		return nil
	}

	whole := generatedPrefix + "whole"
	return []fsast.Node{
		fsast.Func{
			Doc: []string{
				"Converts float seconds into a normalized Duration.",
				"",
				"Whole seconds truncate toward zero so seconds and nanos keep compatible",
				"signs. Nanoseconds rounded to a full second carry in either direction.",
			},
			Static:     true,
			Name:       "from_seconds",
			Parameters: []fsast.Parameter{{Name: nativeValueParameter, Type: fstypes.Named("float")}},
			ReturnType: fstypes.Named(plan.Name),
			Body: []fsast.Node{
				line(0, fmt.Sprintf("var %s: %s = %s.new()", nativeResultLocal, plan.Name, plan.Name)),
				line(0, fmt.Sprintf("var %s: int = int(%s)", whole, nativeValueParameter)),
				line(0, fmt.Sprintf("%s.%s = %s", nativeResultLocal, seconds.Name, whole)),
				line(0, fmt.Sprintf("%s.%s = roundi((%s - float(%s)) * 1000000000.0)",
					nativeResultLocal, nanos.Name, nativeValueParameter, whole)),
				line(0, fmt.Sprintf("if %s.%s >= 1000000000:", nativeResultLocal, nanos.Name)),
				line(1, fmt.Sprintf("%s.%s -= 1000000000", nativeResultLocal, nanos.Name)),
				line(1, fmt.Sprintf("%s.%s += 1", nativeResultLocal, seconds.Name)),
				line(0, fmt.Sprintf("elif %s.%s <= -1000000000:", nativeResultLocal, nanos.Name)),
				line(1, fmt.Sprintf("%s.%s += 1000000000", nativeResultLocal, nanos.Name)),
				line(1, fmt.Sprintf("%s.%s -= 1", nativeResultLocal, seconds.Name)),
				fsast.Return{Value: nativeResultLocal},
			},
		},
		fsast.Func{
			Doc: []string{
				"Returns this Duration as float seconds.",
				"Read seconds and nanos directly when nanosecond precision matters.",
			},
			Name:       "to_seconds",
			ReturnType: fstypes.Named("float"),
			Body: []fsast.Node{
				fsast.Return{Value: fmt.Sprintf("float(%s) + float(%s) / 1000000000.0", seconds.Name, nanos.Name)},
			},
		},
	}
}

func structNativeMembers(plan *messagePlan) []fsast.Node {
	fields := wellKnownField(plan.Fields, "fields")
	if fields == nil || fields.Cardinality != cardinalityMap {
		return nil
	}

	return []fsast.Node{
		fsast.Func{
			Doc: []string{
				"Returns this Struct as a native string-keyed Dictionary.",
				"",
				"Every Value case has a native representation, so this direction is total.",
			},
			Name:       "to_dictionary",
			ReturnType: fstypes.Dictionary(fstypes.Named("String"), variantType),
			Body: []fsast.Node{
				line(0, fmt.Sprintf("var %s: Dictionary[String, Variant] = {}", nativeResultLocal)),
				line(0, fmt.Sprintf("for %s: String in %s:", nativeKeyLocal, fields.Name)),
				line(1, fmt.Sprintf("%s[%s] = %s[%s].to_variant()",
					nativeResultLocal, nativeKeyLocal, fields.Name, nativeKeyLocal)),
				fsast.Return{Value: nativeResultLocal},
			},
		},
		fsast.Func{
			Doc: []string{
				"Converts a native Dictionary into a Struct.",
				"",
				"Keys must be Strings and every value must have a protobuf Value mapping.",
				"A nested failure returns no partial Struct.",
			},
			Static: true,
			Name:   "from_dictionary",
			Parameters: []fsast.Parameter{{
				Name: nativeValueParameter,
				Type: fstypes.Named("Dictionary"),
			}},
			ReturnType: fstypes.Tuple(fstypes.Nullable(fstypes.Named(plan.Name)), fstypes.Named("ProtobufError")),
			Body: []fsast.Node{
				line(0, fmt.Sprintf("var %s: %s? = null", nativeFailedLocal, plan.Name)),
				line(0, fmt.Sprintf("var %s: %s = %s.new()", nativeResultLocal, plan.Name, plan.Name)),
				line(0, fmt.Sprintf("for %s: Variant in %s:", nativeKeyLocal, nativeValueParameter)),
				line(1, fmt.Sprintf("if typeof(%s) != TYPE_STRING:", nativeKeyLocal)),
				line(2, fmt.Sprintf("return (%s, ProtobufError.STRUCT_KEY_NOT_STRING)", nativeFailedLocal)),
				line(1, fmt.Sprintf("var (%s, %s) = Value.from_variant(%s[%s])",
					nativeConvertedLocal, nativeErrorLocal, nativeValueParameter, nativeKeyLocal)),
				line(1, fmt.Sprintf("if %s != ProtobufError.OK:", nativeErrorLocal)),
				line(2, fmt.Sprintf("return (%s, %s)", nativeFailedLocal, nativeErrorLocal)),
				line(1, fmt.Sprintf("if not (%s is Value):", nativeConvertedLocal)),
				line(2, fmt.Sprintf("return (%s, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)", nativeFailedLocal)),
				line(1, fmt.Sprintf("%s.%s[str(%s)] = %s",
					nativeResultLocal, fields.Name, nativeKeyLocal, nativeConvertedLocal)),
				fsast.Return{Value: fmt.Sprintf("(%s, ProtobufError.OK)", nativeResultLocal)},
			},
		},
	}
}

func valueNativeMembers(plan *messagePlan) []fsast.Node {
	if len(plan.Oneofs) != 1 {
		return nil
	}
	kind := &plan.Oneofs[0]
	nullValue := wellKnownOneofMember(kind, "null_value")
	numberValue := wellKnownOneofMember(kind, "number_value")
	stringValue := wellKnownOneofMember(kind, "string_value")
	boolValue := wellKnownOneofMember(kind, "bool_value")
	structValue := wellKnownOneofMember(kind, "struct_value")
	listValue := wellKnownOneofMember(kind, "list_value")
	if nullValue == nil || numberValue == nil || stringValue == nil || boolValue == nil || structValue == nil || listValue == nil {
		return nil
	}

	toVariant := []fsast.Node{line(0, "match "+kind.Field+":")}
	toVariant = append(toVariant,
		line(1, nullValue.OneofCase+"(_):"),
		line(2, "return null"),
		line(1, numberValue.OneofCase+"(var "+numberValue.Local()+"):"),
		line(2, "return "+numberValue.Local()),
		line(1, stringValue.OneofCase+"(var "+stringValue.Local()+"):"),
		line(2, "return "+stringValue.Local()),
		line(1, boolValue.OneofCase+"(var "+boolValue.Local()+"):"),
		line(2, "return "+boolValue.Local()),
		line(1, structValue.OneofCase+"(var "+structValue.Local()+"):"),
		line(2, "return "+structValue.Local()+".to_dictionary()"),
		line(1, listValue.OneofCase+"(var "+listValue.Local()+"):"),
		line(2, "return "+listValue.Local()+".to_array()"),
		line(1, "_:"),
		line(2, "return null"),
	)

	fromVariant := []fsast.Node{
		line(0, fmt.Sprintf("var %s: %s? = null", nativeFailedLocal, plan.Name)),
		line(0, fmt.Sprintf("var %s: %s = %s.new()", nativeResultLocal, plan.Name, plan.Name)),
		line(0, "match typeof("+nativeValueParameter+"):"),
		line(1, "TYPE_NIL:"),
		line(2, fmt.Sprintf("%s.%s = %s(%s)", nativeResultLocal, kind.Field, nullValue.OneofCase, nullValue.Value.ZeroValue)),
		line(1, "TYPE_BOOL:"),
		line(2, fmt.Sprintf("%s.%s = %s(%s)", nativeResultLocal, kind.Field, boolValue.OneofCase, nativeValueParameter)),
		line(1, "TYPE_INT:"),
		line(2, fmt.Sprintf("%s.%s = %s(float(%s))", nativeResultLocal, kind.Field, numberValue.OneofCase, nativeValueParameter)),
		line(1, "TYPE_FLOAT:"),
		line(2, fmt.Sprintf("%s.%s = %s(%s)", nativeResultLocal, kind.Field, numberValue.OneofCase, nativeValueParameter)),
		line(1, "TYPE_STRING:"),
		line(2, fmt.Sprintf("%s.%s = %s(%s)", nativeResultLocal, kind.Field, stringValue.OneofCase, nativeValueParameter)),
		line(1, "TYPE_DICTIONARY:"),
		line(2, fmt.Sprintf("var (%s, %s) = Struct.from_dictionary(%s)", nativeConvertedLocal, nativeErrorLocal, nativeValueParameter)),
		line(2, fmt.Sprintf("if %s != ProtobufError.OK:", nativeErrorLocal)),
		line(3, fmt.Sprintf("return (%s, %s)", nativeFailedLocal, nativeErrorLocal)),
		line(2, fmt.Sprintf("if not (%s is Struct):", nativeConvertedLocal)),
		line(3, fmt.Sprintf("return (%s, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)", nativeFailedLocal)),
		line(2, fmt.Sprintf("%s.%s = %s(%s)", nativeResultLocal, kind.Field, structValue.OneofCase, nativeConvertedLocal)),
		line(1, "TYPE_ARRAY:"),
		line(2, fmt.Sprintf("var (%s, %s) = ListValue.from_array(%s)", nativeConvertedLocal, nativeErrorLocal, nativeValueParameter)),
		line(2, fmt.Sprintf("if %s != ProtobufError.OK:", nativeErrorLocal)),
		line(3, fmt.Sprintf("return (%s, %s)", nativeFailedLocal, nativeErrorLocal)),
		line(2, fmt.Sprintf("if not (%s is ListValue):", nativeConvertedLocal)),
		line(3, fmt.Sprintf("return (%s, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)", nativeFailedLocal)),
		line(2, fmt.Sprintf("%s.%s = %s(%s)", nativeResultLocal, kind.Field, listValue.OneofCase, nativeConvertedLocal)),
		line(1, "_:"),
		line(2, fmt.Sprintf("return (%s, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)", nativeFailedLocal)),
		fsast.Return{Value: fmt.Sprintf("(%s, ProtobufError.OK)", nativeResultLocal)},
	}

	return []fsast.Node{
		fsast.Func{
			Doc: []string{
				"Returns this protobuf Value as its native Foundry representation.",
				"",
				"An unset Value is represented as null, matching its proto3 JSON form.",
			},
			Name:       "to_variant",
			ReturnType: variantType,
			Body:       toVariant,
		},
		fsast.Func{
			Doc: []string{
				"Converts a native Foundry value into a protobuf Value.",
				"",
				"An int narrows to Value's float and is lossy beyond 2^53. Unsupported",
				"Variant kinds and Dictionaries with non-String keys are refused.",
			},
			Static:     true,
			Name:       "from_variant",
			Parameters: []fsast.Parameter{{Name: nativeValueParameter, Type: variantType}},
			ReturnType: fstypes.Tuple(fstypes.Nullable(fstypes.Named(plan.Name)), fstypes.Named("ProtobufError")),
			Body:       fromVariant,
		},
	}
}

func listValueNativeMembers(plan *messagePlan) []fsast.Node {
	values := wellKnownField(plan.Fields, "values")
	if values == nil || values.Cardinality != cardinalityRepeated {
		return nil
	}

	return []fsast.Node{
		fsast.Func{
			Doc:        []string{"Returns this ListValue as a native Array."},
			Name:       "to_array",
			ReturnType: fstypes.Array(variantType),
			Body: []fsast.Node{
				line(0, fmt.Sprintf("var %s: Array[Variant] = []", nativeResultLocal)),
				line(0, fmt.Sprintf("for %s: Value in %s:", nativeItemLocal, values.Name)),
				line(1, fmt.Sprintf("%s.append(%s.to_variant())", nativeResultLocal, nativeItemLocal)),
				fsast.Return{Value: nativeResultLocal},
			},
		},
		fsast.Func{
			Doc: []string{
				"Converts a native Array into a ListValue.",
				"A nested failure returns no partial ListValue.",
			},
			Static: true,
			Name:   "from_array",
			Parameters: []fsast.Parameter{{
				Name: nativeValueParameter,
				Type: fstypes.Array(variantType),
			}},
			ReturnType: fstypes.Tuple(fstypes.Nullable(fstypes.Named(plan.Name)), fstypes.Named("ProtobufError")),
			Body: []fsast.Node{
				line(0, fmt.Sprintf("var %s: %s? = null", nativeFailedLocal, plan.Name)),
				line(0, fmt.Sprintf("var %s: %s = %s.new()", nativeResultLocal, plan.Name, plan.Name)),
				line(0, fmt.Sprintf("for %s: Variant in %s:", nativeItemLocal, nativeValueParameter)),
				line(1, fmt.Sprintf("var (%s, %s) = Value.from_variant(%s)", nativeConvertedLocal, nativeErrorLocal, nativeItemLocal)),
				line(1, fmt.Sprintf("if %s != ProtobufError.OK:", nativeErrorLocal)),
				line(2, fmt.Sprintf("return (%s, %s)", nativeFailedLocal, nativeErrorLocal)),
				line(1, fmt.Sprintf("if not (%s is Value):", nativeConvertedLocal)),
				line(2, fmt.Sprintf("return (%s, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)", nativeFailedLocal)),
				line(1, fmt.Sprintf("%s.%s.append(%s)", nativeResultLocal, values.Name, nativeConvertedLocal)),
				fsast.Return{Value: fmt.Sprintf("(%s, ProtobufError.OK)", nativeResultLocal)},
			},
		},
	}
}
