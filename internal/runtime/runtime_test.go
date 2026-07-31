package runtime_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	wellknowngen "github.com/cafecito-games/foundry-tools/internal/proto/wellknown/gen"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

func TestFilesReturnsRuntimeSources(t *testing.T) {
	files := runtime.Files()

	require.Contains(t, files, "foundry/proto/message.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func decode_bytes")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func skip_field")
	require.NotContains(t, runtime.PublicSource(files), "Variant")
}

// Trait requirements must be abstract; a bare func fails to resolve the trait
// body in every consumer that applies it.
func TestTraitRequirementsAreAbstract(t *testing.T) {
	files := runtime.Files()

	require.Contains(t, files["foundry/proto/message.fs"], "trait_name Message\n")
	require.Contains(t, files["foundry/proto/message.fs"], "abstract func to_bytes()")
	require.Contains(t, files["foundry/proto/codec.fs"], "abstract func encode(")
	require.NotRegexp(t, `(?m)^func `, files["foundry/proto/message.fs"])
	require.NotRegexp(t, `(?m)^func `, files["foundry/proto/codec.fs"])
}

// The read carriers are named tuples, one per file: a tuple_name file may
// contain nothing but its own declaration.
func TestReadCarriersAreSingleDeclarationTupleFiles(t *testing.T) {
	files := runtime.Files()

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

// The well-known bindings are checked in so consumers get them without running
// the generator; regenerating here is what keeps the two in step.
func TestWellKnownBindingsAreUpToDate(t *testing.T) {
	generated, err := wellknowngen.Generate()
	require.NoError(t, err)
	require.NotEmpty(t, generated)

	embedded := runtime.Files()
	for name, want := range generated {
		got, ok := embedded[name]
		require.True(t, ok, "%s is generated but not checked in; run `task gen-wkt`", name)
		require.Equal(t, want, got, "%s is stale; run `task gen-wkt`", name)
	}

	for name := range embedded {
		if !strings.HasPrefix(name, "foundry/proto/wkt/") {
			continue
		}
		require.Contains(t, generated, name, "%s is checked in but no longer generated; run `task gen-wkt`", name)
		require.Contains(t, embedded[name], "namespace "+wellknown.Namespace+"\n", "%s must declare the shared namespace", name)
	}
}
