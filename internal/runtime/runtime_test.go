package runtime_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	wellknowngen "github.com/cafecito-games/foundry-tools/internal/proto/wellknown/gen"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

// The one runtime file allowed to name Variant. JSON.parse_string and
// JSON.stringify are Variant-typed engine APIs, so a JSON document has to meet
// a dynamic value somewhere; json_node.fs is that somewhere, and the exemption
// is scoped to it by name so every other runtime file, present and future,
// stays Variant-free.
const jsonNodePath = "foundry/proto/json_node.fs"

func TestFilesReturnsRuntimeSources(t *testing.T) {
	files := runtime.Files()

	require.Contains(t, files, "foundry/proto/message.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func decode_bytes")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func skip_field")
	require.NotContains(t, runtime.PublicSource(sourcesOutsideTheVariantBoundary(files)), "Variant")
}

// The exemption above is only meaningful while the boundary file is the one
// that needs it: if json_node.fs ever stops naming Variant, the carve-out has
// outlived its reason and should go rather than sit there widening the check.
func TestTheVariantBoundaryIsTheJSONNode(t *testing.T) {
	files := runtime.Files()

	require.Contains(t, files, jsonNodePath)
	// Inside an enum body the enum's own name does not resolve to the enum
	// being declared, so the conversions spell their own type out in full.
	require.Contains(t, files[jsonNodePath], "static func to_variant(_pb_node: foundry.proto.JsonNode) -> Variant")
	require.Contains(t, files[jsonNodePath],
		"static func from_variant(_pb_value: Variant) -> (foundry.proto.JsonNode?, ProtobufError)")
}

func sourcesOutsideTheVariantBoundary(files map[string]string) map[string]string {
	rest := make(map[string]string, len(files))
	for name, source := range files {
		if name == jsonNodePath {
			continue
		}
		rest[name] = source
	}
	return rest
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

// The JSON error cases append after the existing wire-format cases without
// renumbering them, so a caller that already stored a ProtobufError value
// keeps the same meaning.
func TestProtobufErrorCarriesTheJSONCases(t *testing.T) {
	source := runtime.Files()["foundry/proto/protobuf_error.fs"]

	require.Contains(t, source, "UNKNOWN_REQUIRED_FEATURE = 6")
	require.Contains(t, source, "JSON_PARSE_FAILED = 7")
	require.Contains(t, source, "JSON_TYPE_MISMATCH = 8")
	require.Contains(t, source, "JSON_UNKNOWN_FIELD = 9")
	require.Contains(t, source, "JSON_VALUE_OUT_OF_RANGE = 10")
	require.Contains(t, source, "JSON_ANY_UNSUPPORTED = 11")
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
