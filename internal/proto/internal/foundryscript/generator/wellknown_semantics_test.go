package fsgenerator

import (
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func nativeWellKnownSource(t *testing.T, importPath, typeName string, messages ...*protoast.Message) string {
	t.Helper()
	files, err := Generate(namespacedFile(messages, []*protoast.Enum{{
		Name:   "NullValue",
		Values: []*protoast.EnumValue{{Name: "NULL_VALUE", Number: 0}},
	}}), importPath, nil, Options{})
	require.NoError(t, err)
	return files["cafecito/game/v1/"+typeName+".pb.fs"]
}

func TestWellKnownStructValueAndListValueEmitNativeConversions(t *testing.T) {
	value := &protoast.Message{Name: "Value", Oneofs: []*protoast.Oneof{{
		Name: "kind",
		Fields: []*protoast.Field{
			{FieldType: "NullValue", Name: "null_value", Number: 1, IsEnum: true},
			{FieldType: "double", Name: "number_value", Number: 2},
			{FieldType: "string", Name: "string_value", Number: 3},
			{FieldType: "bool", Name: "bool_value", Number: 4},
			{FieldType: "Struct", Name: "struct_value", Number: 5},
			{FieldType: "ListValue", Name: "list_value", Number: 6},
		},
	}}}
	structMessage := &protoast.Message{Name: "Struct", Maps: []*protoast.MapField{{
		KeyType: "string", ValueType: "Value", Name: "fields", Number: 1,
	}}}
	list := &protoast.Message{Name: "ListValue", Fields: []*protoast.Field{{
		FieldType: "Value", Name: "values", Number: 1, Repeated: true,
	}}}

	structSource := nativeWellKnownSource(t, "google/protobuf/struct.proto", "Struct", structMessage, value, list)
	require.Contains(t, structSource, "func to_dictionary() -> Dictionary[String, Variant]:")
	require.Contains(t, structSource, "static func from_dictionary(_pb_value: Dictionary) -> (Struct?, ProtobufError):")
	require.Contains(t, structSource, "if typeof(_pb_key) != TYPE_STRING:")
	require.Contains(t, structSource, "return (_pb_failed, ProtobufError.STRUCT_KEY_NOT_STRING)")
	require.Contains(t, structSource, "var (_pb_converted, _pb_error) = Value.from_variant(_pb_value[_pb_key])")

	valueSource := nativeWellKnownSource(t, "google/protobuf/struct.proto", "Value", structMessage, value, list)
	require.Contains(t, valueSource, "func to_variant() -> Variant:")
	require.Contains(t, valueSource, "static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError):")
	for _, variantType := range []string{
		"TYPE_NIL", "TYPE_BOOL", "TYPE_INT", "TYPE_FLOAT", "TYPE_STRING", "TYPE_DICTIONARY", "TYPE_ARRAY",
	} {
		require.Contains(t, valueSource, variantType+":")
	}
	require.Contains(t, valueSource, "return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)")
	require.Contains(t, valueSource, "ValueKindCase.StructValue(var _pb_kind_struct_value):")
	require.Contains(t, valueSource, "return _pb_kind_struct_value.to_dictionary()")
	require.Contains(t, valueSource, "ValueKindCase.ListValue(var _pb_kind_list_value):")
	require.Contains(t, valueSource, "return _pb_kind_list_value.to_array()")

	listSource := nativeWellKnownSource(t, "google/protobuf/struct.proto", "ListValue", structMessage, value, list)
	require.Contains(t, listSource, "func to_array() -> Array[Variant]:")
	require.Contains(t, listSource, "static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError):")
	require.Contains(t, listSource, "var (_pb_converted, _pb_error) = Value.from_variant(_pb_item)")
}

func TestWellKnownTimestampAndDurationEmitNativeTimeConversions(t *testing.T) {
	secondsAndNanos := []*protoast.Field{
		{FieldType: "int64", Name: "seconds", Number: 1},
		{FieldType: "int32", Name: "nanos", Number: 2},
	}

	timestamp := nativeWellKnownSource(t, "google/protobuf/timestamp.proto", "Timestamp",
		&protoast.Message{Name: "Timestamp", Fields: secondsAndNanos})
	require.Contains(t, timestamp, "static func from_unix_time(_pb_value: float) -> Timestamp:")
	require.Contains(t, timestamp, "static func now() -> Timestamp:")
	require.Contains(t, timestamp, "func to_unix_time() -> float:")
	require.Contains(t, timestamp, "var _pb_whole: int = floori(_pb_value)")
	require.Contains(t, timestamp, "_pb_result.nanos = roundi((_pb_value - float(_pb_whole)) * 1000000000.0)")
	require.Contains(t, timestamp, "if _pb_result.nanos >= 1000000000:")
	require.Contains(t, timestamp, "return Timestamp.from_unix_time(Time.get_unix_time_from_system())")

	duration := nativeWellKnownSource(t, "google/protobuf/duration.proto", "Duration",
		&protoast.Message{Name: "Duration", Fields: secondsAndNanos})
	require.Contains(t, duration, "static func from_seconds(_pb_value: float) -> Duration:")
	require.Contains(t, duration, "func to_seconds() -> float:")
	require.Contains(t, duration, "var _pb_whole: int = int(_pb_value)")
	require.Contains(t, duration, "elif _pb_result.nanos <= -1000000000:")
}

func TestNativeWellKnownConversionsAreSelectedByDeclarationFile(t *testing.T) {
	ordinary := generate(t, namespacedFile([]*protoast.Message{
		{Name: "Value"},
		{Name: "Timestamp", Fields: []*protoast.Field{
			{FieldType: "int64", Name: "seconds", Number: 1},
			{FieldType: "int32", Name: "nanos", Number: 2},
		}},
	}, nil))

	require.NotContains(t, ordinary["cafecito/game/v1/Value.pb.fs"], "to_variant")
	require.NotContains(t, ordinary["cafecito/game/v1/Timestamp.pb.fs"], "from_unix_time")

	anySource := nativeWellKnownSource(t, "google/protobuf/any.proto", "Any",
		&protoast.Message{Name: "Any"})
	require.NotContains(t, anySource, "pack(")
	require.NotContains(t, anySource, "unpack")

	wrapperSource := nativeWellKnownSource(t, "google/protobuf/wrappers.proto", "StringValue",
		&protoast.Message{Name: "StringValue", Fields: []*protoast.Field{{
			FieldType: "string", Name: "value", Number: 1,
		}}})
	require.NotContains(t, wrapperSource, "to_variant")
	require.NotContains(t, wrapperSource, "from_variant")
}
