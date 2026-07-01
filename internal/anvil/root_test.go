package anvil_test

import (
	"bytes"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/anvil"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager"
	protocmd "github.com/cafecito-games/foundry-tools/internal/proto"
	"github.com/cafecito-games/foundry-tools/internal/version"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootCommandComposesVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newTestRoot(&stdout, &stderr)
	cmd.SetArgs([]string{"version"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "anvil dev")
	require.Empty(t, stderr.String())
}

func TestRootCommandComposesProtoCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newTestRoot(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "print-options-proto"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), `package foundrytools;`)
	require.Empty(t, stderr.String())
}

func TestRootCommandComposesProtoGenerateValidation(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newTestRoot(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "generate", "-o", t.TempDir()})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one .proto file is required")
}

func TestExecuteRendersPackageErrors(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newTestRoot(&stdout, &stderr)
	cmd.SetArgs([]string{"pkg", "list", "--dir", t.TempDir()})

	code := anvil.Execute(cmd, &stdout, &stderr)

	require.Equal(t, 3, code)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "anvil:")
	require.Contains(t, stderr.String(), "no project.foundry found")
}

func TestExecuteRendersPackageErrorsAsJSON(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newTestRoot(&stdout, &stderr)
	cmd.SetArgs([]string{"pkg", "--json", "list", "--dir", t.TempDir()})

	code := anvil.Execute(cmd, &stdout, &stderr)

	require.Equal(t, 3, code)
	require.Empty(t, stderr.String())
	require.Contains(t, stdout.String(), `"code": 3`)
	require.Contains(t, stdout.String(), `"error": "no project.foundry found`)
}

func newTestRoot(stdout, stderr *bytes.Buffer) *cobra.Command {
	root := anvil.NewRootCommand(stdout, stderr)
	root.AddCommand(version.NewCommand(stdout))
	root.AddCommand(protocmd.NewCommand(stdout))
	root.AddCommand(packagemanager.NewCommand(stdout, stderr))
	return root
}
