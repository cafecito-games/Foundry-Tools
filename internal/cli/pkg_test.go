package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager"
	"github.com/stretchr/testify/require"
)

func TestPkgInitCreatesManifest(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"pkg", "init", "--dir", dir})

	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(dir, "packages.toml"))
	require.Contains(t, stdout.String(), "Created ")
	require.Empty(t, stderr.String())
}

func TestPkgListPrintsTextAndJSON(t *testing.T) {
	project := newPkgTestProject(t)
	m := &packagemanager.Manifest{Packages: map[string]packagemanager.PackageSpec{
		"b": {Name: "b", Source: packagemanager.SourceArchive, URL: "https://example.com/b.zip", Version: "2"},
		"a": {Name: "a", Source: packagemanager.SourceGit, URL: "https://example.com/a.git", Version: "1", InstallAs: "installed_a"},
	}}
	require.NoError(t, m.Save(filepath.Join(project, "packages.toml")))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "addons", "installed_a"), 0o755))

	var textOut bytes.Buffer
	cmd := NewRootCommand(&textOut, &bytes.Buffer{})
	cmd.SetArgs([]string{"pkg", "list", "--dir", project})
	require.NoError(t, cmd.Execute())
	require.Contains(t, textOut.String(), "[x] a")
	require.Contains(t, textOut.String(), "[ ] b")

	var jsonOut bytes.Buffer
	cmd = NewRootCommand(&jsonOut, &bytes.Buffer{})
	cmd.SetArgs([]string{"pkg", "--json", "list", "--dir", project})
	require.NoError(t, cmd.Execute())
	require.Contains(t, jsonOut.String(), `"name": "a"`)
	require.Contains(t, jsonOut.String(), `"installed": true`)
}

func TestPkgRemoveDeletesPackage(t *testing.T) {
	project := newPkgTestProject(t)
	m := &packagemanager.Manifest{Packages: map[string]packagemanager.PackageSpec{
		"pkg": {Name: "pkg", Source: packagemanager.SourceArchive, URL: "https://example.com/pkg.zip"},
	}}
	require.NoError(t, m.Save(filepath.Join(project, "packages.toml")))
	lock := &packagemanager.Lockfile{Packages: map[string]packagemanager.LockEntry{
		"pkg": {ResolvedVersion: "1", SpecHash: m.Packages["pkg"].Hash()},
	}}
	require.NoError(t, lock.Save(filepath.Join(project, "packages.lock")))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "addons", "pkg"), 0o755))

	var stdout bytes.Buffer
	cmd := NewRootCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"pkg", "remove", "--dir", project, "pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "removed pkg")
	require.NoDirExists(t, filepath.Join(project, "addons", "pkg"))
}

func TestPkgInstallDelegatesAndRenders(t *testing.T) {
	restore := overridePackageOps(t)
	defer restore()
	packageInstall = func(_ context.Context, opts packagemanager.InstallOptions) ([]packagemanager.PackageResult, error) {
		require.Equal(t, "/tmp/project", opts.Dir)
		require.Equal(t, []string{"pkg"}, opts.Names)
		return []packagemanager.PackageResult{{Name: "pkg", ResolvedVersion: "1.0.0", InstallPath: "pkg"}}, nil
	}

	var stdout bytes.Buffer
	cmd := NewRootCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"pkg", "install", "--dir", "/tmp/project", "pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "installed pkg @ 1.0.0")
}

func TestPkgUpdateDelegatesAndRendersJSON(t *testing.T) {
	restore := overridePackageOps(t)
	defer restore()
	packageUpdate = func(_ context.Context, opts packagemanager.UpdateOptions) ([]packagemanager.PackageResult, error) {
		require.Equal(t, []string{"pkg"}, opts.Names)
		return []packagemanager.PackageResult{{Name: "pkg", ResolvedVersion: "2.0.0", InstallPath: "pkg"}}, nil
	}

	var stdout bytes.Buffer
	cmd := NewRootCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"pkg", "--json", "update", "pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), `"name": "pkg"`)
	require.Contains(t, stdout.String(), `"resolved_version": "2.0.0"`)
}

func TestPkgAddBuildsSpecAndDelegates(t *testing.T) {
	restore := overridePackageOps(t)
	defer restore()
	packageAdd = func(_ context.Context, opts packagemanager.AddOptions) ([]packagemanager.PackageResult, error) {
		require.Equal(t, "pkg", opts.Spec.Name)
		require.Equal(t, packagemanager.SourceArchive, opts.Spec.Source)
		require.Equal(t, "https://example.com/pkg.zip", opts.Spec.URL)
		require.Equal(t, "addons/pkg", opts.Spec.SourcePath)
		require.Equal(t, "installed_pkg", opts.Spec.InstallAs)
		return []packagemanager.PackageResult{{Name: "pkg", ResolvedVersion: "", InstallPath: "installed_pkg"}}, nil
	}

	var stdout bytes.Buffer
	cmd := NewRootCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"pkg", "add", "--name", "pkg", "--source", "archive", "--url", "https://example.com/pkg.zip", "--source-path", "addons/pkg", "--install-as", "installed_pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "added and installed pkg")
}

func TestRootHelpIncludesPkg(t *testing.T) {
	var stdout bytes.Buffer
	cmd := NewRootCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "pkg")
}

func newPkgTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.foundry"), []byte("; config"), 0o644))
	return root
}

func overridePackageOps(t *testing.T) func() {
	t.Helper()
	oldAdd := packageAdd
	oldInstall := packageInstall
	oldUpdate := packageUpdate
	return func() {
		packageAdd = oldAdd
		packageInstall = oldInstall
		packageUpdate = oldUpdate
	}
}
