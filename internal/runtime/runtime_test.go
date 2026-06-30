package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilesReturnsRuntimeSources(t *testing.T) {
	files := Files()

	require.Contains(t, files, "foundry/proto/message.fs")
	require.Contains(t, files, "foundry/proto/decode_result.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	require.Contains(t, files["foundry/proto/message.fs"], "trait_name Message[T]")
	require.Contains(t, files["foundry/proto/decode_result.fs"], "class_name DecodeResult[T]")
	require.NotContains(t, PublicSource(files), "Variant")
}
