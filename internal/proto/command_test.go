package proto

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

func TestPrintOptionsCommandIsWired(t *testing.T) {
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"print-options-proto"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), `package foundrytools;`)
}

func TestGenerateRequiresInputs(t *testing.T) {
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"generate", "-o", t.TempDir()})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one .proto file is required")
}

func TestWriteFilesUsesSourcePermissions(t *testing.T) {
	outDir := t.TempDir()
	oldUmask := syscall.Umask(0o077)
	t.Cleanup(func() {
		syscall.Umask(oldUmask)
	})

	require.NoError(t, writeFiles(outDir, map[string]string{
		"cafecito/game/v1/Player.pb.fs": "class_name Player\n",
	}))

	cafecitoInfo, err := os.Stat(filepath.Join(outDir, "cafecito"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), cafecitoInfo.Mode().Perm())

	gameInfo, err := os.Stat(filepath.Join(outDir, "cafecito", "game"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), gameInfo.Mode().Perm())

	dirInfo, err := os.Stat(filepath.Join(outDir, "cafecito", "game", "v1"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(filepath.Join(outDir, "cafecito", "game", "v1", "Player.pb.fs"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fileInfo.Mode().Perm())
}

// Naming a well-known file directly must produce nothing of its own; the
// runtime bindings, written alongside any generated output, are the only copy
// a project gets.
func TestGenerateSkipsWellKnownFiles(t *testing.T) {
	root := t.TempDir()
	name := "google/protobuf/timestamp.proto"
	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	source, err := wellknown.Source(name)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, source, 0o600))
	t.Chdir(root)

	outDir := t.TempDir()
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"generate", "-o", outDir, name})

	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "generated Foundry Script for 0 proto file(s)")
	_, err = os.Stat(filepath.Join(outDir, "google"))
	require.True(t, os.IsNotExist(err), "a well-known file must generate no output of its own")
	// The runtime copy still ships, so a project that asked for it gets one.
	_, err = os.Stat(filepath.Join(outDir, "foundry", "proto", "wkt", "Timestamp.pb.fs"))
	require.NoError(t, err)
	// And the rest of the runtime with it: it ships whenever generation ran,
	// not only when some schema produced a binding of its own.
	_, err = os.Stat(filepath.Join(outDir, "foundry", "proto", "wire.fs"))
	require.NoError(t, err)
}
