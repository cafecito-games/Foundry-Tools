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

// A repo that vendors the well-known protos names them by an absolute or
// vendor-prefixed path under `-I vendor`, which resolves to the bare import
// spelling and so is the same schema. Generating a project-local binding for it
// would hand the project a second, incompatible Timestamp alongside the
// runtime's.
func TestGenerateSkipsWellKnownFilesNamedByVendoredPath(t *testing.T) {
	root := t.TempDir()
	name := "google/protobuf/timestamp.proto"
	vendoredPath := filepath.Join(root, "vendor", filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(vendoredPath), 0o755))
	source, err := wellknown.Source(name)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(vendoredPath, source, 0o600))
	require.True(t, filepath.IsAbs(vendoredPath))

	outDir := t.TempDir()
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"generate", "-o", outDir, "-I", filepath.Join(root, "vendor"), vendoredPath})

	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "generated Foundry Script for 0 proto file(s)")

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	require.Equal(t, []string{"foundry"}, names, "the runtime is the only thing a vendored well-known file may produce")

	_, err = os.Stat(filepath.Join(outDir, "foundry", "proto", "wkt", "Timestamp.pb.fs"))
	require.NoError(t, err)
}

// A schema of the caller's own whose import path only ends in a well-known
// spelling is an ordinary file. Its bindings were asked for and must be
// emitted; skipping it would drop the caller's requested output with no error.
func TestGenerateEmitsASchemaWhoseImportPathOnlyEndsInAWellKnownSpelling(t *testing.T) {
	root := t.TempDir()
	name := "myorg/google/protobuf/timestamp.proto"
	path := filepath.Join(root, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`syntax = "proto3";
package myorg.google.protobuf;

message Timestamp {
  int64 seconds = 1;
}
`), 0o600))
	t.Chdir(root)

	outDir := t.TempDir()
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"generate", "-o", outDir, "-I", ".", name})

	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "generated Foundry Script for 1 proto file(s)")

	source, err := os.ReadFile(filepath.Join(outDir, "myorg", "google", "protobuf", "Timestamp.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(source), "class_name Timestamp")
}

// Named with no include root that contains it, a google/protobuf path has no
// identity: it is either the well-known schema or an unrelated one, and both
// guesses are silent damage. The error names the include path that would settle
// it and points out that the bindings already ship.
func TestGenerateRejectsAnUnrootedGoogleProtobufPath(t *testing.T) {
	root := t.TempDir()
	name := "google/protobuf/timestamp.proto"
	vendoredPath := filepath.Join(root, "vendor", filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(vendoredPath), 0o755))
	source, err := wellknown.Source(name)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(vendoredPath, source, 0o600))

	outDir := t.TempDir()
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"generate", "-o", outDir, vendoredPath})

	err = cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be identified without an include path")
	require.Contains(t, err.Error(), "-I "+filepath.Join(root, "vendor"))
	require.Contains(t, err.Error(), "google/protobuf/timestamp.proto")

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	require.Empty(t, entries, "a rejected input must produce no output at all")
}
