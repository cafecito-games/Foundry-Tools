package wellknown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWellKnown(t *testing.T) {
	require.True(t, IsWellKnown("google/protobuf/timestamp.proto"))
	require.True(t, IsWellKnown("google/protobuf/struct.proto"))
	require.False(t, IsWellKnown("cafecito/game/v1/player.proto"))
}

func TestIsWellKnownNormalizesSeparatorsAndTraversal(t *testing.T) {
	require.True(t, IsWellKnown(`google\protobuf\timestamp.proto`))
	require.True(t, IsWellKnown("google/protobuf/./timestamp.proto"))
}

// A vendored or absolute copy is the same schema as the bare import spelling,
// so it is still the well-known type. Treating it as a project file instead
// would generate a second Timestamp the runtime does not recognize, which is
// precisely the duplicate this package exists to prevent.
func TestIsWellKnownMatchesVendoredAndAbsoluteCopies(t *testing.T) {
	require.True(t, IsWellKnown("google/protobuf/timestamp.proto"))
	require.True(t, IsWellKnown("vendor/google/protobuf/timestamp.proto"))
	require.True(t, IsWellKnown("third_party/google/protobuf/struct.proto"))
	require.True(t, IsWellKnown("/abs/path/to/google/protobuf/timestamp.proto"))
	require.True(t, IsWellKnown(`C:\vendor\google\protobuf\timestamp.proto`))

	// A directory that merely ends in the same words is not the import path.
	require.False(t, IsWellKnown("mygoogle/protobuf/timestamp.proto"))
	require.False(t, IsWellKnown("google/protobuf/nested/timestamp.proto"))
	require.False(t, IsWellKnown("vendor/google/protobuf/descriptor.proto"))
}

func TestUnsupportedGoogleFileIsRejected(t *testing.T) {
	err := Check("google/protobuf/descriptor.proto")
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto")
	require.Contains(t, err.Error(), "not supported")
	require.NoError(t, Check("google/protobuf/timestamp.proto"))
	require.NoError(t, Check("cafecito/game/v1/player.proto"))
}

// An unshipped google/protobuf file is unsupported wherever the copy lives; the
// path relative to the include root is what identifies it.
func TestUnsupportedGoogleFileIsRejectedWhenVendored(t *testing.T) {
	err := Check("vendor/google/protobuf/descriptor.proto")
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto")
	require.Contains(t, err.Error(), "not supported")

	require.Error(t, Check("/abs/path/to/google/protobuf/descriptor.proto"))
	require.NoError(t, Check("vendor/google/protobuf/timestamp.proto"))
}

// Source keys off the same import path, so a vendored or absolute spelling
// still resolves to the vendored text rather than failing the lookup.
func TestSourceAcceptsVendoredSpelling(t *testing.T) {
	source, err := Source("vendor/google/protobuf/timestamp.proto")
	require.NoError(t, err)
	require.Contains(t, string(source), "package google.protobuf;")
}

func TestEmbeddedProtosArePresent(t *testing.T) {
	for _, name := range Files() {
		source, err := Source(name)
		require.NoError(t, err)
		require.Contains(t, string(source), "package google.protobuf;")
		require.Contains(t, string(source), "// Protocol Buffers - Google's data interchange format")
	}
	require.Len(t, Files(), 7)
}

func TestSourceRejectsUnvendoredFile(t *testing.T) {
	_, err := Source("google/protobuf/descriptor.proto")
	require.Error(t, err)
}
