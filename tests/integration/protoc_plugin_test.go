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
	pluginPath := buildProtocPlugin(t, root)

	run(t, root, "protoc",
		"--plugin=protoc-gen-foundryscript="+pluginPath,
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
	pluginPath := buildProtocPlugin(t, root)

	run(t, root, "protoc",
		"--plugin=protoc-gen-foundryscript="+pluginPath,
		"--foundryscript_out="+outDir,
		"-I", "tests/integration/fixtures/member_collisions",
		"tests/integration/fixtures/member_collisions/fields.proto",
	)

	requireMemberProbeEscapes(t, outDir)
}

func TestProtocPluginAcceptsBundledWellKnown(t *testing.T) {
	root := repoRoot(t)
	includeRoot := t.TempDir()
	outDir := t.TempDir()
	pluginPath := buildProtocPlugin(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(includeRoot, "event.proto"), []byte(`syntax = "proto3";
package cafecito.game.v1;
import "google/protobuf/timestamp.proto";
message Event { google.protobuf.Timestamp occurred_at = 1; }
`), 0o600))

	run(t, root, "protoc",
		"--plugin=protoc-gen-foundryscript="+pluginPath,
		"--foundryscript_out="+outDir,
		"-I", includeRoot,
		filepath.Join(includeRoot, "event.proto"),
	)

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito", "game", "v1", "Event.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "import foundry.proto.wkt")
	require.Contains(t, string(data), "var occurred_at: Timestamp? = null")
}

func buildProtocPlugin(t *testing.T, root string) string {
	t.Helper()

	pluginPath := filepath.Join(t.TempDir(), "protoc-gen-foundryscript")
	run(t, root, "go", "build", "-o", pluginPath, "./cmd/protoc-gen-foundryscript")
	return pluginPath
}
