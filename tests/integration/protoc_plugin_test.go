//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocPluginGeneratesFoundryScript(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "build", "-o", "bin/protoc-gen-foundryscript", "./cmd/protoc-gen-foundryscript")
	run(t, root, "protoc",
		"--plugin=protoc-gen-foundryscript="+filepath.Join(root, "bin/protoc-gen-foundryscript"),
		"--foundryscript_out="+outDir,
		"-I", "tests/integration/fixtures/basic",
		"tests/integration/fixtures/basic/player.proto",
	)

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito/game/v1/Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "func to_bytes() -> PackedByteArray:")
	require.Contains(t, string(data), "## Player state shared with Foundry Script.\nfinal class_name Player extends RefCounted")
	require.Contains(t, string(data), "## Player display name.\nfunc get_name() -> String:")
}
