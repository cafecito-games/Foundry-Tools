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
	require.False(t, IsWellKnown("vendor/google/protobuf/timestamp.proto"))
}

func TestUnsupportedGoogleFileIsRejected(t *testing.T) {
	err := Check("google/protobuf/descriptor.proto")
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto")
	require.Contains(t, err.Error(), "not supported")
	require.NoError(t, Check("google/protobuf/timestamp.proto"))
	require.NoError(t, Check("cafecito/game/v1/player.proto"))
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
