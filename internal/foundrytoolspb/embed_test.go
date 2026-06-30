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
