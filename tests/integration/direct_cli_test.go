//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectCLIGeneratesFoundryScript(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "run", "./cmd/foundry-tools", "proto", "generate", "-I", "tests/integration/fixtures/basic", "-o", outDir, "tests/integration/fixtures/basic/player.proto")

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito/game/v1/Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "namespace cafecito.game.v1")
	require.Contains(t, string(data), "class_name Player")
	require.Contains(t, string(data), "## Player state shared with Foundry Script.\nfinal class_name Player extends RefCounted")
	require.Contains(t, string(data), "## Player display name.\nfunc get_name() -> String:")
}
