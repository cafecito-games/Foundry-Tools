package packagemanager

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
	"github.com/stretchr/testify/require"
)

func TestInitCreatesPackagesManifest(t *testing.T) {
	dir := t.TempDir()
	result, err := Init(InitOptions{Dir: dir})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "packages.toml"), result.ManifestPath)
	require.FileExists(t, result.ManifestPath)
}

func TestInitFailsWhenManifestExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "packages.toml"), []byte("[packages]\n"), 0o644))
	_, err := Init(InitOptions{Dir: dir})
	require.Error(t, err)
}

func TestCodeForMapsErrors(t *testing.T) {
	require.Equal(t, ExitOK, CodeFor(nil))
	require.Equal(t, ExitUsage, CodeFor(&UsageError{Err: errors.New("bad args")}))
	require.Equal(t, ExitManifest, CodeFor(&output.ManifestError{Err: errors.New("bad manifest")}))
	require.Equal(t, ExitFetch, CodeFor(&output.FetchError{Err: errors.New("bad fetch")}))
	require.Equal(t, ExitInstall, CodeFor(&output.InstallError{Err: errors.New("bad install")}))
	require.Equal(t, ExitGeneric, CodeFor(errors.New("bad")))
}
