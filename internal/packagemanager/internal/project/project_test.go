package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscoverFindsProjectFoundryInParent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.foundry"), []byte("; config"), 0o644))
	nested := filepath.Join(root, "src", "systems")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	discovered, err := Discover(nested)
	require.NoError(t, err)
	require.Equal(t, root, discovered.Root)
	require.Equal(t, filepath.Join(root, "packages.toml"), discovered.ManifestPath)
	require.Equal(t, filepath.Join(root, "packages.lock"), discovered.LockPath)
	require.Equal(t, filepath.Join(root, "addons"), discovered.AddonsDir)
}

func TestDiscoverRejectsMissingProject(t *testing.T) {
	_, err := Discover(t.TempDir())
	require.Error(t, err)
}

func TestDiscoverRejectsNonRegularProjectMarker(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "project.foundry"), 0o755))
	_, err := Discover(root)
	require.Error(t, err)
}
