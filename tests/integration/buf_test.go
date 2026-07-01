//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBufGeneratesFoundryScript(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "tests/integration/fixtures/basic")
	outDir := filepath.Join(fixture, "out")
	run(t, root, "go", "build", "-o", "bin/protoc-gen-foundryscript", "./cmd/protoc-gen-foundryscript")
	require.NoError(t, os.RemoveAll(outDir))
	t.Cleanup(func() { _ = os.RemoveAll(outDir) })
	run(t, fixture, "buf", "generate")

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito", "game", "v1", "Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "class_name Player")
}
