package cli

import (
	"bytes"
	"os"
	"path/filepath"
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
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `package foundrytools;`)
	require.Empty(t, stderr.String())
}

func TestProtoGenerateRequiresInputs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "generate", "-o", t.TempDir()})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one .proto file is required")
}

func TestWriteFilesUsesSourcePermissions(t *testing.T) {
	outDir := t.TempDir()

	require.NoError(t, writeFiles(outDir, map[string]string{
		"cafecito/game/v1/Player.pb.fs": "class_name Player\n",
	}))

	dirInfo, err := os.Stat(filepath.Join(outDir, "cafecito", "game", "v1"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(filepath.Join(outDir, "cafecito", "game", "v1", "Player.pb.fs"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fileInfo.Mode().Perm())
}
