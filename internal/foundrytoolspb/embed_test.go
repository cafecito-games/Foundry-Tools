package foundrytoolspb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytesReturnsOptionsProto(t *testing.T) {
	text := string(Bytes())

	require.Contains(t, text, `package foundrytools;`)
	require.Contains(t, text, `optional string namespace = 52000;`)
	require.Contains(t, text, `optional string type_prefix = 52001;`)
	require.Contains(t, text, `optional bool emit_runtime = 52002;`)
	require.True(t, strings.HasSuffix(text, "\n"))
}

func TestEmbeddedOptionsProtoDocumentsTypePrefix(t *testing.T) {
	text := string(Bytes())

	require.Contains(t, text, `// Literal prefix applied to every generated type in this file.`)
	require.Contains(t, text, `// Use it to avoid Foundry built-in, native-class, or project-name collisions.`)
}
