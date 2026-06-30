package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommandPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "foundry-tools dev")
	require.Empty(t, stderr.String())
}

func TestProtoPrintOptionsCommandIsWired(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "print-options-proto"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "options proto support is not wired")
}
