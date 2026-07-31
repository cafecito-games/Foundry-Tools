package protoparse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFilesParsesRootProto(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "player.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";
package cafecito.game.v1;
message Player {
  string name = 1;
}
`), 0o644))

	files, err := ParseFiles([]string{protoPath}, []string{dir})
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, protoPath, files[0].Filename)
	require.Equal(t, "cafecito.game.v1", files[0].File.Package)
	require.Len(t, files[0].File.Messages, 1)
}

func TestParseFilesFailsMissingImport(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "player.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";
package cafecito.game.v1;
import "missing.proto";
message Player {
  string name = 1;
}
`), 0o644))

	_, err := ParseFiles([]string{protoPath}, []string{dir})
	require.Error(t, err)
	require.Contains(t, err.Error(), `import "missing.proto" not found`)
}

// A schema importing a well-known type builds with no include path, because
// the vendored copy stands in for one.
func TestParseFilesResolvesWellKnownImportWithoutAnIncludePath(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "event.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";
package cafecito.game.v1;
import "google/protobuf/timestamp.proto";
message Event {
  google.protobuf.Timestamp occurred_at = 1;
}
`), 0o644))

	files, err := ParseFiles([]string{protoPath}, nil)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Len(t, files[0].Imports, 1)
	require.Equal(t, "google/protobuf/timestamp.proto", files[0].Imports[0].Filename)
	require.Equal(t, "google.protobuf", files[0].Imports[0].File.Package)
}

// An include path is an explicit statement about which sources to use, so a
// copy the caller supplies wins over the vendored one.
func TestWellKnownFallbackYieldsToACallerSuppliedCopy(t *testing.T) {
	dir := t.TempDir()
	vendored := filepath.Join(dir, "google", "protobuf", "timestamp.proto")
	require.NoError(t, os.MkdirAll(filepath.Dir(vendored), 0o755))
	require.NoError(t, os.WriteFile(vendored, []byte(`syntax = "proto3";
package google.protobuf;
message Timestamp {
  int64 seconds = 1;
}
`), 0o644))

	fs := WellKnownFS{Next: &OSFS{BaseDir: dir}}
	source, err := fs.Read("google/protobuf/timestamp.proto")
	require.NoError(t, err)
	require.NotContains(t, string(source), "nanos")
}

// An import path is a literal reference, not a spelling of a file the caller
// already holds. A path that merely ends in a well-known name is a different
// file, and standing in for it would turn a typo into a silent substitution.
func TestWellKnownFSDoesNotSatisfyAnImportThatOnlyEndsInAWellKnownPath(t *testing.T) {
	fs := WellKnownFS{Next: &OSFS{BaseDir: t.TempDir()}}

	require.False(t, fs.Exists("myorg/google/protobuf/timestamp.proto"))
	_, err := fs.Read("myorg/google/protobuf/timestamp.proto")
	require.Error(t, err)

	require.True(t, fs.Exists("google/protobuf/timestamp.proto"))
}

func TestParseFilesRejectsAnUnsupportedGoogleFile(t *testing.T) {
	_, err := ParseFiles([]string{"google/protobuf/descriptor.proto"}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}
