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
	require.Contains(t, string(data), "## Player state shared with Foundry Script.\nfinal class_name Player extends RefCounted uses Message")
	require.Contains(t, string(data), "## Player display name.\nvar name: String = \"\"")
	// A type from a dependency resolves through the descriptors protoc supplies,
	// which is the only way the plugin path can learn its namespace and default.
	require.Contains(t, string(data), "import cafecito.inventory.v1")
	require.Contains(t, string(data), "var rarity: Rarity = Rarity.RARITY_UNSPECIFIED")
}

func TestProtocPluginEscapesFoundryMemberCollisions(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "build", "-o", "bin/protoc-gen-foundryscript", "./cmd/protoc-gen-foundryscript")
	run(t, root, "protoc",
		"--plugin=protoc-gen-foundryscript="+filepath.Join(root, "bin/protoc-gen-foundryscript"),
		"--foundryscript_out="+outDir,
		"-I", "tests/integration/fixtures/member_collisions",
		"tests/integration/fixtures/member_collisions/fields.proto",
	)

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
