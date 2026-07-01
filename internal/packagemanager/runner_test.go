package packagemanager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/source"
	"github.com/stretchr/testify/require"
)

type fakeFetcher struct {
	version  string
	checksum string
	files    map[string]string
	err      error
}

func (f fakeFetcher) Fetch(_ context.Context, _ PackageSpec) (source.FetchResult, error) {
	if f.err != nil {
		return source.FetchResult{}, f.err
	}
	dir, err := os.MkdirTemp("", "fake-package-*")
	if err != nil {
		return source.FetchResult{}, err
	}
	files := f.files
	if files == nil {
		files = map[string]string{"plugin.cfg": "[plugin]"}
	}
	for relative, body := range files {
		path := filepath.Join(dir, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return source.FetchResult{}, err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return source.FetchResult{}, err
		}
	}
	return source.FetchResult{Dir: dir, ResolvedVersion: f.version, Checksum: f.checksum}, nil
}

func TestRunnerInstallPackagesWritesLock(t *testing.T) {
	root := t.TempDir()
	runner := &Runner{
		AddonsDir: filepath.Join(root, "addons"),
		LockPath:  filepath.Join(root, "packages.lock"),
		FetcherFor: func(PackageSpec) (source.Fetcher, error) {
			return fakeFetcher{version: "1.0.0", checksum: "abc123"}, nil
		},
	}
	m := &Manifest{Packages: map[string]PackageSpec{
		"dlg": {Name: "dlg", Source: SourceArchive, URL: "https://example.com/dlg.zip"},
	}}

	results, err := runner.InstallPackages(context.Background(), m, nil, ModeInstall)
	require.NoError(t, err)
	require.Equal(t, []PackageResult{{Name: "dlg", ResolvedVersion: "1.0.0", InstallPath: "dlg"}}, results)
	require.FileExists(t, filepath.Join(root, "addons", "dlg", "plugin.cfg"))

	lock, err := LoadLock(filepath.Join(root, "packages.lock"))
	require.NoError(t, err)
	require.Equal(t, "1.0.0", lock.Packages["dlg"].ResolvedVersion)
	require.Equal(t, "abc123", lock.Packages["dlg"].Checksum)
	require.Equal(t, m.Packages["dlg"].Hash(), lock.Packages["dlg"].SpecHash)
}

func TestRunnerInstallPackagesHonorsLockedGitVersion(t *testing.T) {
	root := t.TempDir()
	spec := PackageSpec{Name: "pkg", Source: SourceGit, URL: "https://example.com/repo.git", Version: "main"}
	lock := &Lockfile{Packages: map[string]LockEntry{
		"pkg": {ResolvedVersion: "locked-sha", SpecHash: spec.Hash()},
	}}
	require.NoError(t, lock.Save(filepath.Join(root, "packages.lock")))

	var fetchedVersion string
	runner := &Runner{
		AddonsDir: filepath.Join(root, "addons"),
		LockPath:  filepath.Join(root, "packages.lock"),
		FetcherFor: func(spec PackageSpec) (source.Fetcher, error) {
			fetchedVersion = spec.Version
			return fakeFetcher{version: spec.Version}, nil
		},
	}

	_, err := runner.InstallPackages(context.Background(), &Manifest{Packages: map[string]PackageSpec{"pkg": spec}}, nil, ModeInstall)
	require.NoError(t, err)
	require.Equal(t, "locked-sha", fetchedVersion)
}

func TestRunnerUpdatePackagesReResolvesSelectedNames(t *testing.T) {
	root := t.TempDir()
	spec := PackageSpec{Name: "pkg", Source: SourceGit, URL: "https://example.com/repo.git", Version: "main"}
	lock := &Lockfile{Packages: map[string]LockEntry{
		"pkg": {ResolvedVersion: "locked-sha", SpecHash: spec.Hash()},
	}}
	require.NoError(t, lock.Save(filepath.Join(root, "packages.lock")))

	var fetchedVersion string
	runner := &Runner{
		AddonsDir: filepath.Join(root, "addons"),
		LockPath:  filepath.Join(root, "packages.lock"),
		FetcherFor: func(spec PackageSpec) (source.Fetcher, error) {
			fetchedVersion = spec.Version
			return fakeFetcher{version: "new-sha"}, nil
		},
	}

	results, err := runner.InstallPackages(context.Background(), &Manifest{Packages: map[string]PackageSpec{"pkg": spec}}, []string{"pkg"}, ModeUpdate)
	require.NoError(t, err)
	require.Equal(t, "main", fetchedVersion)
	require.Equal(t, "new-sha", results[0].ResolvedVersion)
}

func TestRunnerInstallPackagesRejectsUnknownName(t *testing.T) {
	runner := &Runner{AddonsDir: t.TempDir(), LockPath: filepath.Join(t.TempDir(), "packages.lock")}
	_, err := runner.InstallPackages(context.Background(), &Manifest{Packages: map[string]PackageSpec{}}, []string{"missing"}, ModeInstall)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown package "missing"`)
}

func TestAddPackageRollsBackManifestOnInstallFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.foundry"), []byte("; config"), 0o644))
	require.NoError(t, (&Manifest{Packages: map[string]PackageSpec{}}).Save(filepath.Join(root, "packages.toml")))

	fetchErr := &output.FetchError{Err: errors.New("boom")}
	_, err := addWithRunner(context.Background(), AddOptions{
		Options: Options{Dir: root},
		Spec:    PackageSpec{Name: "pkg", Source: SourceArchive, URL: "https://example.com/pkg.zip"},
	}, func(addonsDir, lockPath string, limits source.Limits) *Runner {
		return &Runner{
			AddonsDir: addonsDir,
			LockPath:  lockPath,
			FetcherFor: func(PackageSpec) (source.Fetcher, error) {
				return fakeFetcher{err: fetchErr}, nil
			},
		}
	})

	require.ErrorIs(t, err, fetchErr)
	loaded, loadErr := LoadManifest(filepath.Join(root, "packages.toml"))
	require.NoError(t, loadErr)
	require.Empty(t, loaded.Packages)
}

func TestRemovePackageDeletesDiskManifestAndLock(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.foundry"), []byte("; config"), 0o644))
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Name: "pkg", Source: SourceArchive, URL: "https://example.com/pkg.zip", InstallAs: "installed_pkg"},
	}}
	require.NoError(t, m.Save(filepath.Join(root, "packages.toml")))
	lock := &Lockfile{Packages: map[string]LockEntry{"pkg": {ResolvedVersion: "1", SpecHash: m.Packages["pkg"].Hash()}}}
	require.NoError(t, lock.Save(filepath.Join(root, "packages.lock")))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "addons", "installed_pkg"), 0o755))

	require.NoError(t, Remove(RemoveOptions{Options: Options{Dir: root}, Name: "pkg"}))
	require.NoDirExists(t, filepath.Join(root, "addons", "installed_pkg"))

	loaded, err := LoadManifest(filepath.Join(root, "packages.toml"))
	require.NoError(t, err)
	require.Empty(t, loaded.Packages)
	loadedLock, err := LoadLock(filepath.Join(root, "packages.lock"))
	require.NoError(t, err)
	require.Empty(t, loadedLock.Packages)
}

func TestListPackagesReturnsSortedInstalledStatus(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.foundry"), []byte("; config"), 0o644))
	m := &Manifest{Packages: map[string]PackageSpec{
		"b": {Name: "b", Source: SourceArchive, URL: "https://example.com/b.zip", Version: "2"},
		"a": {Name: "a", Source: SourceGit, URL: "https://example.com/a.git", Version: "1", InstallAs: "installed_a"},
	}}
	require.NoError(t, m.Save(filepath.Join(root, "packages.toml")))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "addons", "installed_a"), 0o755))

	listings, err := List(ListOptions{Options: Options{Dir: root}})
	require.NoError(t, err)
	require.Equal(t, []PackageListing{
		{Name: "a", Source: "git", Version: "1", Installed: true},
		{Name: "b", Source: "archive", Version: "2", Installed: false},
	}, listings)
}
