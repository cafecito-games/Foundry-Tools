package packagemanager

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/lockfile"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/manifest"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/project"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/source"
)

type SourceType = manifest.SourceType

const (
	SourceGit           = manifest.SourceGit
	SourceGitHubRelease = manifest.SourceGitHubRelease
	SourceArchive       = manifest.SourceArchive
)

type PackageSpec = manifest.PackageSpec
type Manifest = manifest.Manifest
type Lockfile = lockfile.Lockfile
type LockEntry = lockfile.Entry
type Project = project.Project
type ExitCode = output.ExitCode

const (
	ExitOK       = output.ExitOK
	ExitGeneric  = output.ExitGeneric
	ExitUsage    = output.ExitUsage
	ExitManifest = output.ExitManifest
	ExitFetch    = output.ExitFetch
	ExitInstall  = output.ExitInstall
)

// UsageError marks bad package-manager flags or arguments.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// Options holds package-manager options shared by project operations.
type Options struct {
	Dir               string
	MaxDownloadBytes  int64
	MaxExtractedBytes int64
}

type InitOptions struct {
	Dir string
}

type InitResult struct {
	ManifestPath string `json:"manifest_path"`
}

type AddOptions struct {
	Options
	Spec PackageSpec
}

type InstallOptions struct {
	Options
	Names []string
}

type UpdateOptions struct {
	Options
	Names []string
}

type RemoveOptions struct {
	Options
	Name string
}

type ListOptions struct {
	Options
}

type PackageResult struct {
	Name            string `json:"name"`
	ResolvedVersion string `json:"resolved_version"`
	InstallPath     string `json:"install_path"`
}

type PackageListing struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
}

type runnerFactory func(addonsDir, lockPath string, limits source.Limits) *Runner

const starterManifest = `# Foundry package manifest managed by anvil.
# Add packages with ` + "`anvil pkg add`" + ` or by hand. Example:
#
# [packages.example]
# source      = "git"
# url         = "https://github.com/owner/example.git"
# version     = "v1.0.0"
# source_path = "addons/example"

[packages]
`

// Init creates a starter packages.toml in Dir or the current directory.
func Init(opts InitOptions) (*InitResult, error) {
	dir := opts.Dir
	if dir == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dir = workingDir
	}
	path := filepath.Join(dir, "packages.toml")
	if _, err := os.Stat(path); err == nil {
		return nil, &output.ManifestError{Err: fmt.Errorf("%s already exists", path)}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, &output.ManifestError{Err: err}
	}
	if err := os.WriteFile(path, []byte(starterManifest), 0o644); err != nil {
		return nil, &output.ManifestError{Err: err}
	}
	return &InitResult{ManifestPath: path}, nil
}

// Discover locates a Foundry project from dir or the current directory.
func Discover(dir string) (*Project, error) {
	if dir == "" {
		workingDir, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dir = workingDir
	}
	return project.Discover(dir)
}

// LoadManifest reads packages.toml at path.
func LoadManifest(path string) (*Manifest, error) {
	return manifest.Load(path)
}

// LoadLock reads packages.lock at path.
func LoadLock(path string) (*Lockfile, error) {
	return lockfile.Load(path)
}

// Add saves one package in packages.toml and installs it immediately.
func Add(ctx context.Context, opts AddOptions) ([]PackageResult, error) {
	return AddWithRunner(ctx, opts, NewRunner)
}

func AddWithRunner(ctx context.Context, opts AddOptions, newRunner runnerFactory) ([]PackageResult, error) {
	discovered, pkgManifest, err := loadProject(opts.Dir)
	if err != nil {
		return nil, err
	}
	if opts.Spec.Name == "" {
		return nil, &UsageError{Err: fmt.Errorf("package name is required")}
	}
	if _, exists := pkgManifest.Packages[opts.Spec.Name]; exists {
		return nil, &UsageError{Err: fmt.Errorf("package %q already exists", opts.Spec.Name)}
	}
	single := &Manifest{Packages: map[string]PackageSpec{opts.Spec.Name: opts.Spec}}
	if err := single.Validate(); err != nil {
		return nil, err
	}
	pkgManifest.Packages[opts.Spec.Name] = opts.Spec
	if err := pkgManifest.Save(discovered.ManifestPath); err != nil {
		return nil, &output.ManifestError{Err: err}
	}
	runner := newRunner(discovered.AddonsDir, discovered.LockPath, limitsFor(opts.Options))
	results, err := runner.InstallPackages(ctx, pkgManifest, []string{opts.Spec.Name}, ModeInstall)
	if err != nil {
		delete(pkgManifest.Packages, opts.Spec.Name)
		if rollbackErr := pkgManifest.Save(discovered.ManifestPath); rollbackErr != nil {
			return nil, errors.Join(err, rollbackErr)
		}
		return nil, err
	}
	return results, nil
}

// Install installs all packages or selected names while honoring lock pins.
func Install(ctx context.Context, opts InstallOptions) ([]PackageResult, error) {
	discovered, pkgManifest, err := loadProject(opts.Dir)
	if err != nil {
		return nil, err
	}
	return NewRunner(discovered.AddonsDir, discovered.LockPath, limitsFor(opts.Options)).
		InstallPackages(ctx, pkgManifest, opts.Names, ModeInstall)
}

// Update re-resolves all packages or selected names.
func Update(ctx context.Context, opts UpdateOptions) ([]PackageResult, error) {
	discovered, pkgManifest, err := loadProject(opts.Dir)
	if err != nil {
		return nil, err
	}
	return NewRunner(discovered.AddonsDir, discovered.LockPath, limitsFor(opts.Options)).
		InstallPackages(ctx, pkgManifest, opts.Names, ModeUpdate)
}

// Remove deletes an installed package and removes it from manifest and lockfile.
func Remove(opts RemoveOptions) error {
	discovered, pkgManifest, err := loadProject(opts.Dir)
	if err != nil {
		return err
	}
	spec, ok := pkgManifest.Packages[opts.Name]
	if !ok {
		return &UsageError{Err: fmt.Errorf("unknown package %q", opts.Name)}
	}
	if err := os.RemoveAll(filepath.Join(discovered.AddonsDir, spec.InstallName())); err != nil {
		return &output.InstallError{Err: err}
	}
	delete(pkgManifest.Packages, opts.Name)
	if err := pkgManifest.Save(discovered.ManifestPath); err != nil {
		return &output.ManifestError{Err: err}
	}
	lock, err := lockfile.Load(discovered.LockPath)
	if err != nil {
		return err
	}
	if _, exists := lock.Packages[opts.Name]; exists {
		delete(lock.Packages, opts.Name)
		if err := lock.Save(discovered.LockPath); err != nil {
			return err
		}
	}
	return nil
}

// List returns configured packages sorted by name.
func List(opts ListOptions) ([]PackageListing, error) {
	discovered, pkgManifest, err := loadProject(opts.Dir)
	if err != nil {
		return nil, err
	}
	return listPackages(discovered, pkgManifest), nil
}

// CodeFor maps an error to a package-manager exit code.
func CodeFor(err error) ExitCode {
	if err == nil {
		return ExitOK
	}
	var usageErr *UsageError
	if errors.As(err, &usageErr) {
		return ExitUsage
	}
	return output.CodeFor(err)
}

func limitsFor(opts Options) source.Limits {
	return source.Limits{
		MaxDownloadBytes:  opts.MaxDownloadBytes,
		MaxExtractedBytes: opts.MaxExtractedBytes,
	}
}

func loadProject(dir string) (*Project, *Manifest, error) {
	discovered, err := Discover(dir)
	if err != nil {
		return nil, nil, err
	}
	pkgManifest, err := manifest.Load(discovered.ManifestPath)
	if err != nil {
		return nil, nil, &output.ManifestError{Err: err}
	}
	if err := pkgManifest.Validate(); err != nil {
		return nil, nil, err
	}
	return discovered, pkgManifest, nil
}
