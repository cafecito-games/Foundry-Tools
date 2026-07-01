package packagemanager

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandInitCreatesManifest(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"init", "--dir", dir})

	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(dir, "packages.toml"))
	require.Contains(t, stdout.String(), "Created ")
	require.Empty(t, stderr.String())
}

func TestCommandListPrintsTextAndJSON(t *testing.T) {
	project := newCommandTestProject(t)
	m := &Manifest{Packages: map[string]PackageSpec{
		"b": {Name: "b", Source: SourceArchive, URL: "https://example.com/b.zip", Version: "2"},
		"a": {Name: "a", Source: SourceGit, URL: "https://example.com/a.git", Version: "1", InstallAs: "installed_a"},
	}}
	require.NoError(t, m.Save(filepath.Join(project, "packages.toml")))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "addons", "installed_a"), 0o755))

	var textOut bytes.Buffer
	cmd := NewCommand(&textOut, &bytes.Buffer{})
	cmd.SetArgs([]string{"list", "--dir", project})
	require.NoError(t, cmd.Execute())
	require.Contains(t, textOut.String(), "[x] a")
	require.Contains(t, textOut.String(), "[ ] b")

	var jsonOut bytes.Buffer
	cmd = NewCommand(&jsonOut, &bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "list", "--dir", project})
	require.NoError(t, cmd.Execute())
	require.Contains(t, jsonOut.String(), `"name": "a"`)
	require.Contains(t, jsonOut.String(), `"installed": true`)
}

func TestCommandRemoveDeletesPackage(t *testing.T) {
	project := newCommandTestProject(t)
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Name: "pkg", Source: SourceArchive, URL: "https://example.com/pkg.zip"},
	}}
	require.NoError(t, m.Save(filepath.Join(project, "packages.toml")))
	lock := &Lockfile{Packages: map[string]LockEntry{
		"pkg": {ResolvedVersion: "1", SpecHash: m.Packages["pkg"].Hash()},
	}}
	require.NoError(t, lock.Save(filepath.Join(project, "packages.lock")))
	require.NoError(t, os.MkdirAll(filepath.Join(project, "addons", "pkg"), 0o755))

	var stdout bytes.Buffer
	cmd := NewCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"remove", "--dir", project, "pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "removed pkg")
	require.NoDirExists(t, filepath.Join(project, "addons", "pkg"))
}

func TestCommandInstallDelegatesAndRenders(t *testing.T) {
	restore := overrideCommandPackageOps(t)
	defer restore()
	packageInstall = func(_ context.Context, opts InstallOptions) ([]PackageResult, error) {
		require.Equal(t, "/tmp/project", opts.Dir)
		require.Equal(t, []string{"pkg"}, opts.Names)
		return []PackageResult{{Name: "pkg", ResolvedVersion: "1.0.0", InstallPath: "pkg"}}, nil
	}

	var stdout bytes.Buffer
	cmd := NewCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"install", "--dir", "/tmp/project", "pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "installed pkg @ 1.0.0")
}

func TestCommandUpdateDelegatesAndRendersJSON(t *testing.T) {
	restore := overrideCommandPackageOps(t)
	defer restore()
	packageUpdate = func(_ context.Context, opts UpdateOptions) ([]PackageResult, error) {
		require.Equal(t, []string{"pkg"}, opts.Names)
		return []PackageResult{{Name: "pkg", ResolvedVersion: "2.0.0", InstallPath: "pkg"}}, nil
	}

	var stdout bytes.Buffer
	cmd := NewCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "update", "pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), `"name": "pkg"`)
	require.Contains(t, stdout.String(), `"resolved_version": "2.0.0"`)
}

func TestCommandAddBuildsSpecAndDelegates(t *testing.T) {
	restore := overrideCommandPackageOps(t)
	defer restore()
	packageAdd = func(_ context.Context, opts AddOptions) ([]PackageResult, error) {
		require.Equal(t, "pkg", opts.Spec.Name)
		require.Equal(t, SourceArchive, opts.Spec.Source)
		require.Equal(t, "https://example.com/pkg.zip", opts.Spec.URL)
		require.Equal(t, "addons/pkg", opts.Spec.SourcePath)
		require.Equal(t, "installed_pkg", opts.Spec.InstallAs)
		return []PackageResult{{Name: "pkg", ResolvedVersion: "", InstallPath: "installed_pkg"}}, nil
	}

	var stdout bytes.Buffer
	cmd := NewCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"add", "--name", "pkg", "--source", "archive", "--url", "https://example.com/pkg.zip", "--source-path", "addons/pkg", "--install-as", "installed_pkg"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "added and installed pkg")
}

func TestCommandHelpIncludesSubcommands(t *testing.T) {
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"--help"})
	require.NoError(t, cmd.Execute())
	require.Contains(t, stdout.String(), "install")
	require.Contains(t, stdout.String(), "list")
}

func newCommandTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "project.foundry"), []byte("; config"), 0o644))
	return root
}

func overrideCommandPackageOps(t *testing.T) func() {
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
