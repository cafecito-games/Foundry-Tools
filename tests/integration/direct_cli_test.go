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

	run(t, root, "go", "run", "./cmd/anvil", "proto", "generate", "-I", "tests/integration/fixtures/basic", "-o", outDir, "tests/integration/fixtures/basic/player.proto")

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito/game/v1/Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "namespace cafecito.game.v1")
	require.Contains(t, string(data), "class_name Player")
	require.Contains(t, string(data), "## Player state shared with Foundry Script.\nfinal class_name Player extends RefCounted uses Message")
	require.Contains(t, string(data), "## Player display name.\nvar name: String = \"\"")
	require.Contains(t, string(data), "import cafecito.inventory.v1")
	require.Contains(t, string(data), "var rarity: Rarity = Rarity.RARITY_UNSPECIFIED")
}

func TestDirectCLIReportsEngineTypeCollision(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	output := runFailure(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/collisions",
		"-o", outDir,
		"tests/integration/fixtures/collisions/types.proto")

	require.Contains(t, output, `native class "Node"`)
	require.Contains(t, output, "(foundrytools.type_prefix)")

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestDirectCLIGenerationFailureIsAtomic(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	output := runFailure(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/collisions",
		"-o", outDir,
		"tests/integration/fixtures/collisions/prefixed.proto",
		"tests/integration/fixtures/collisions/types.proto")

	require.Contains(t, output, `native class "Node"`)
	require.Contains(t, output, "(foundrytools.type_prefix)")

	require.NoFileExists(t, filepath.Join(outDir, "probe/collisions/v1/GameNode.pb.fs"))

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestDirectCLIAppliesTypePrefix(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/collisions",
		"-o", outDir,
		"tests/integration/fixtures/collisions/prefixed.proto")

	data, err := os.ReadFile(filepath.Join(outDir, "probe/collisions/v1/GameNode.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "class_name GameNode")

	_, err = os.Stat(filepath.Join(outDir, "probe/collisions/v1/Node.pb.fs"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDirectCLIResolvesAbsoluteLocalTypeReferences(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/absolute_local",
		"-o", outDir,
		"tests/integration/fixtures/absolute_local/types.proto")

	holder, err := os.ReadFile(filepath.Join(outDir, "probe/local/v1/GameHolder.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(holder), "var node: GameNode? = null")
	require.Contains(t, string(holder), "var nodes: Dictionary[String, GameNode] = {}")
	require.Contains(t, string(holder),
		"var _pb_selection_selected_message: GameNode = GameNode.new()")

	union, err := os.ReadFile(filepath.Join(outDir, "probe/local/v1/GameHolderSelectionCase.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(union), "\tSelected(selected: GameNode)\n")

	generated := string(holder) + string(union)
	require.NotContains(t, generated, "GameProbe")
	require.NotContains(t, generated, "GameLocal")
	require.NotContains(t, generated, "GameV1")
}
