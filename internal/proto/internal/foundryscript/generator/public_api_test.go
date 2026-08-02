package fsgenerator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRejectsPublicVariantSignatures(t *testing.T) {
	source := "func get_value() -> Variant:\n\treturn value\n"
	err := CheckPublicAPI(source)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public Variant")
}

func TestRejectsPublicNullableVariantSignatures(t *testing.T) {
	source := "func get_value() -> Variant?:\n\treturn value\n"
	err := CheckPublicAPI(source)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public Variant")
}

func TestAllowsPrivateVariantSignatures(t *testing.T) {
	source := "func _decode_dynamic(value: Variant) -> int:\n\treturn 0\n"
	require.NoError(t, CheckPublicAPI(source))
}

func TestAllowsOnlyTheWellKnownNativeVariantBridge(t *testing.T) {
	for _, source := range []string{
		"func to_variant() -> Variant:\n\treturn null\n",
		"static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError):\n\treturn (null, ProtobufError.OK)\n",
		"func to_dictionary() -> Dictionary[String, Variant]:\n\treturn {}\n",
		"func to_array() -> Array[Variant]:\n\treturn []\n",
		"static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError):\n\treturn (null, ProtobufError.OK)\n",
	} {
		require.NoError(t, CheckPublicAPI(source))
	}

	for _, source := range []string{
		"func other() -> Variant:\n\treturn null\n",
		"static func from_variant(value: Variant) -> (Value?, ProtobufError):\n\treturn (null, ProtobufError.OK)\n",
		"func to_dictionary() -> Dictionary[int, Variant]:\n\treturn {}\n",
		"static func from_array(value: Array[Variant]) -> (ListValue?, ProtobufError):\n\treturn (null, ProtobufError.OK)\n",
	} {
		require.Error(t, CheckPublicAPI(source))
	}
}
