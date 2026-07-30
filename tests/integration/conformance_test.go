//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The upstream conformance schema is the exhaustive statement of proto3, so
// generating it whole is what "we support proto3" means. Before the
// fixed-width and zig-zag scalars landed this failed on the first sint32.
//
// It asserts structure rather than exact source: the point is that every
// construct in the file survives generation, and pinning the emitted text
// here would duplicate examples/golden while making an upstream refresh look
// like a generator regression.
func TestConformanceSchemaGenerates(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	fixture := "tests/integration/fixtures/conformance"

	args := []string{"run", "./cmd/anvil", "proto", "generate", "-I", fixture, "-o", outDir,
		filepath.Join(fixture, "test_messages_proto3.proto"),
	}
	for _, wellKnown := range []string{"any", "duration", "empty", "field_mask", "struct", "timestamp", "wrappers"} {
		args = append(args, filepath.Join(fixture, "google/protobuf", wellKnown+".proto"))
	}
	run(t, root, "go", args...)

	// The schema's own declarations, plus the oneof union that carries a type
	// nested in the message declaring it.
	for _, name := range []string{
		"protobuf_test_messages/proto3/TestAllTypesProto3.pb.fs",
		"protobuf_test_messages/proto3/TestAllTypesProto3OneofFieldCase.pb.fs",
		"protobuf_test_messages/proto3/ForeignMessage.pb.fs",
		"protobuf_test_messages/proto3/ForeignEnum.pb.fs",
	} {
		_, err := os.Stat(filepath.Join(outDir, name))
		require.NoError(t, err, "conformance schema did not produce %s", name)
	}

	// The fixture vendors its own google/protobuf copies and the invocation
	// above names them outright, which is how a repo with a vendored tree calls
	// the generator. They are still the well-known types, so they arrive as the
	// runtime's single set of bindings -- including the recursive Value -- and
	// not as a project-local second copy the runtime would not recognize.
	for _, name := range []string{
		"foundry/proto/wkt/Struct.pb.fs",
		"foundry/proto/wkt/Value.pb.fs",
		"foundry/proto/wkt/Any.pb.fs",
	} {
		_, err := os.Stat(filepath.Join(outDir, name))
		require.NoError(t, err, "the runtime did not ship %s", name)
	}
	_, statErr := os.Stat(filepath.Join(outDir, "google"))
	require.True(t, os.IsNotExist(statErr), "a vendored well-known file must generate no binding of its own")

	suite, err := os.ReadFile(filepath.Join(outDir, "protobuf_test_messages/proto3/TestAllTypesProto3.pb.fs"))
	require.NoError(t, err)
	source := string(suite)

	// One assertion per framing the eight scalars introduced, so a regression
	// that drops back to varints fails here rather than only in the engine.
	require.Contains(t, source, "Wire.encode_float(optional_float)")
	require.Contains(t, source, "Wire.encode_double(optional_double)")
	require.Contains(t, source, "Wire.encode_fixed32(optional_fixed32)")
	require.Contains(t, source, "Wire.encode_fixed64(optional_fixed64)")
	require.Contains(t, source, "Wire.encode_sint32(optional_sint32)")
	require.Contains(t, source, "Wire.encode_sint64(optional_sint64)")
	require.Contains(t, source, "var _pb_optional_sfixed32_read: FixedRead = Wire.read_sfixed32(_pb_data, _pb_offset)")
	require.Contains(t, source, "var _pb_optional_sfixed64_read: FixedRead = Wire.read_fixed64(_pb_data, _pb_offset)")

	// Recursion, direct and mutual, resolves rather than looping the emitter.
	require.Contains(t, source, "var recursive_message: TestAllTypesProto3? = null")
}
