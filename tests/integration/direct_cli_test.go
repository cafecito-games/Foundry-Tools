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

func TestDirectCLIEscapesFoundryMemberCollisions(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/member_collisions",
		"-o", outDir,
		"tests/integration/fixtures/member_collisions/fields.proto")

	requireMemberProbeEscapes(t, outDir)
}

func requireMemberProbeEscapes(t *testing.T, outDir string) {
	t.Helper()

	message, err := os.ReadFile(filepath.Join(outDir, "probe/members/v1/MemberProbe.pb.fs"))
	require.NoError(t, err)
	for _, declaration := range []string{
		`var Node_: String = ""`,
		"var String_: String? = null",
		"var Timer_: Array[String] = []",
		"var Resource_: Dictionary[String, int] = {}",
		"var Object_: MemberProbeObjectCase? = null",
	} {
		require.Contains(t, string(message), declaration)
	}
	require.Contains(t, string(message), "Object_ = MemberProbeObjectCase.Image(")

	oneof, err := os.ReadFile(filepath.Join(outDir, "probe/members/v1/MemberProbeObjectCase.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(oneof), "\tImage(Image: String)")
	require.NotContains(t, string(oneof), "Image_")
}

func TestDirectCLIMemberCollisionFailureIsAtomic(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	output := runFailure(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/member_collisions",
		"-o", outDir,
		"tests/integration/fixtures/member_collisions/fields.proto",
		"tests/integration/fixtures/member_collisions/secondary.proto")

	require.Contains(t, output, "secondary.proto:6:3:")
	require.Contains(t, output, "secondary.proto:7:3:")
	require.Contains(t, output, "field probe.members.v1.SecondaryCollision.Node ")
	require.Contains(t, output, "field probe.members.v1.SecondaryCollision.Node_ ")
	require.Contains(t, output, `Foundry member "Node_"`)
	require.Contains(t, output, `native class "Node"`)

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}
