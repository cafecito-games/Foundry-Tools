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
