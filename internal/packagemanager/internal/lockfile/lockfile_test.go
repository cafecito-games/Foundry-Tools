package lockfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/manifest"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingLockfileReturnsEmpty(t *testing.T) {
	lock, err := Load(filepath.Join(t.TempDir(), "packages.lock"))
	require.NoError(t, err)
	require.NotNil(t, lock.Packages)
	require.Empty(t, lock.Packages)
}

func TestLockfileRoundTrip(t *testing.T) {
	lock := &Lockfile{Packages: map[string]Entry{
		"pkg": {ResolvedVersion: "abc123", SourcePath: "addons/pkg", Checksum: "deadbeef", SpecHash: "hash"},
	}}
	path := filepath.Join(t.TempDir(), "packages.lock")
	require.NoError(t, lock.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, lock.Packages["pkg"], loaded.Packages["pkg"])
}

func TestLoadBadLockfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "packages.lock")
	require.NoError(t, os.WriteFile(path, []byte("not = = toml"), 0o644))
	_, err := Load(path)
	require.Error(t, err)
}

func TestNeedsResolve(t *testing.T) {
	spec := manifest.PackageSpec{Name: "pkg", Source: manifest.SourceArchive, URL: "https://example.com/pkg.zip"}
	require.True(t, NeedsResolve(spec, &Lockfile{Packages: map[string]Entry{}}))

	lock := &Lockfile{Packages: map[string]Entry{
		"pkg": {SpecHash: spec.Hash()},
	}}
	require.False(t, NeedsResolve(spec, lock))

	changed := spec
	changed.URL = "https://example.com/other.zip"
	require.True(t, NeedsResolve(changed, lock))
}
