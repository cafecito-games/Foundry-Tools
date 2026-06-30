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
