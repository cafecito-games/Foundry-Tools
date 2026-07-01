package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManifestRoundTrip(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"dialogue": {Source: SourceGit, URL: "https://example.com/d.git", Version: "v1.0", SourcePath: "addons/dialogue"},
		"thing":    {Source: SourceArchive, URL: "https://example.com/t.zip"},
	}}
	path := filepath.Join(t.TempDir(), "packages.toml")
	require.NoError(t, m.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "dialogue", loaded.Packages["dialogue"].Name)
	require.Equal(t, SourceGit, loaded.Packages["dialogue"].Source)
	require.Equal(t, "v1.0", loaded.Packages["dialogue"].Version)
	require.Equal(t, m.Packages["dialogue"].Hash(), loaded.Packages["dialogue"].Hash())
}

func TestInstallNameDefaultsToKey(t *testing.T) {
	require.Equal(t, "foo", PackageSpec{Name: "foo"}.InstallName())
	require.Equal(t, "bar", PackageSpec{Name: "foo", InstallAs: "bar"}.InstallName())
}

func TestHashChangesWithFields(t *testing.T) {
	a := PackageSpec{Source: SourceGit, URL: "u", Version: "v1"}
	b := a
	b.Version = "v2"
	require.NotEqual(t, a.Hash(), b.Hash())
}

func TestHashChangesWithManifestExclusions(t *testing.T) {
	dir := t.TempDir()
	withoutExclude := filepath.Join(dir, "without.toml")
	require.NoError(t, os.WriteFile(withoutExclude, []byte(`
[packages]
[packages.dialogue]
source = "archive"
url = "https://example.com/dialogue.zip"
`), 0o644))
	withExclude := filepath.Join(dir, "with.toml")
	require.NoError(t, os.WriteFile(withExclude, []byte(`
[packages]
[packages.dialogue]
source = "archive"
url = "https://example.com/dialogue.zip"
exclude = ["dotnet"]
`), 0o644))

	a, err := Load(withoutExclude)
	require.NoError(t, err)
	b, err := Load(withExclude)
	require.NoError(t, err)

	require.NotEqual(t, a.Packages["dialogue"].Hash(), b.Packages["dialogue"].Hash())
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.toml"))
	require.Error(t, err)
}

func TestLoadBadTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	require.NoError(t, os.WriteFile(path, []byte("not = = valid"), 0o644))
	_, err := Load(path)
	require.Error(t, err)
}
