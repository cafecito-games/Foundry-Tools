package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilesReturnsRuntimeSources(t *testing.T) {
	files := Files()

	require.Contains(t, files, "foundry/proto/message.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func decode_bytes")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func skip_field")
	require.NotContains(t, PublicSource(files), "Variant")
}

// Trait requirements must be abstract; a bare func fails to resolve the trait
// body in every consumer that applies it.
func TestTraitRequirementsAreAbstract(t *testing.T) {
	files := Files()

	require.Contains(t, files["foundry/proto/message.fs"], "trait_name Message\n")
	require.Contains(t, files["foundry/proto/message.fs"], "abstract func to_bytes()")
	require.Contains(t, files["foundry/proto/codec.fs"], "abstract func encode(")
	require.NotRegexp(t, `(?m)^func `, files["foundry/proto/message.fs"])
	require.NotRegexp(t, `(?m)^func `, files["foundry/proto/codec.fs"])
}

// The read carriers are named tuples, one per file: a tuple_name file may
// contain nothing but its own declaration.
func TestReadCarriersAreSingleDeclarationTupleFiles(t *testing.T) {
	files := Files()

	for _, name := range []string{"varint_read", "string_read", "bytes_read", "skip_read"} {
		path := "foundry/proto/" + name + ".fs"
		require.Contains(t, files, path)
		require.Contains(t, files[path], "tuple_name ")
		require.NotContains(t, files[path], "class_name ")
		require.NotContains(t, files[path], "func ")
	}
	require.NotContains(t, files, "foundry/proto/field_read.fs")
	require.NotContains(t, files, "foundry/proto/decode_result.fs")
}
