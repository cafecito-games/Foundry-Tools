package packagemanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/installer"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/lockfile"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/manifest"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/project"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/source"
)

// InstallMode controls whether existing lockfile pins are honored.
type InstallMode int

const (
	// ModeInstall honors existing lockfile pins when they match packages.toml.
	ModeInstall InstallMode = iota
	// ModeUpdate re-resolves selected packages and rewrites lockfile pins.
	ModeUpdate
)

// Runner performs install orchestration. FetcherFor is injectable for tests.
type Runner struct {
	AddonsDir  string
	LockPath   string
	FetcherFor func(PackageSpec) (source.Fetcher, error)
}

// NewRunner builds a Runner wired to the real source layer.
func NewRunner(addonsDir, lockPath string, limits source.Limits) *Runner {
	return &Runner{
		AddonsDir:  addonsDir,
		LockPath:   lockPath,
		FetcherFor: source.FetcherForWithLimits(limits),
	}
}

// InstallPackages fetches and installs the named packages, or all packages when
// names is empty, and writes packages.lock.
func (r *Runner) InstallPackages(ctx context.Context, pkgManifest *Manifest, names []string, mode InstallMode) ([]PackageResult, error) {
	lock, err := lockfile.Load(r.LockPath)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if _, ok := pkgManifest.Packages[name]; !ok {
			return nil, &UsageError{Err: fmt.Errorf("unknown package %q", name)}
		}
	}
	targets := selectPackages(pkgManifest, names)
	results := make([]PackageResult, 0, len(targets))
	for i := range targets {
		spec := targets[i]
		useLock := mode == ModeInstall && !lockfile.NeedsResolve(spec, lock)

		effectiveSpec := spec
		if useLock && spec.Source == manifest.SourceGit {
			effectiveSpec.Version = lock.Packages[spec.Name].ResolvedVersion
		}

		fetcher, err := r.FetcherFor(effectiveSpec)
		if err != nil {
			return nil, err
		}
		fetched, err := fetcher.Fetch(ctx, effectiveSpec)
		if err != nil {
			return nil, err
		}

		if err := verifyChecksum(spec, lock, fetched, useLock); err != nil {
			_ = os.RemoveAll(fetched.Dir)
			return nil, err
		}

		err = installer.Install(fetched, spec, r.AddonsDir)
		_ = os.RemoveAll(fetched.Dir)
		if err != nil {
			return nil, err
		}
		lock.Packages[spec.Name] = lockfile.Entry{
			ResolvedVersion: fetched.ResolvedVersion,
			SourcePath:      spec.SourcePath,
			Checksum:        fetched.Checksum,
			SpecHash:        spec.Hash(),
		}
		if err := lock.Save(r.LockPath); err != nil {
			return nil, err
		}
		results = append(results, PackageResult{
			Name:            spec.Name,
			ResolvedVersion: fetched.ResolvedVersion,
			InstallPath:     spec.InstallName(),
		})
	}
	if len(names) == 0 {
		for name := range lock.Packages {
			if _, ok := pkgManifest.Packages[name]; !ok {
				delete(lock.Packages, name)
			}
		}
	}
	if err := lock.Save(r.LockPath); err != nil {
		return nil, err
	}
	return results, nil
}

func verifyChecksum(spec PackageSpec, lock *Lockfile, fetched source.FetchResult, useLock bool) error {
	if spec.Checksum != "" && fetched.Checksum != "" && spec.Checksum != fetched.Checksum {
		return &output.FetchError{Err: fmt.Errorf(
			"package %q: checksum mismatch (manifest: %s, fetched: %s)",
			spec.Name, spec.Checksum, fetched.Checksum)}
	}
	if useLock {
		entry := lock.Packages[spec.Name]
		if entry.Checksum != "" && fetched.Checksum != "" && entry.Checksum != fetched.Checksum {
			return &output.FetchError{Err: fmt.Errorf(
				"package %q: checksum mismatch (lock: %s, fetched: %s)",
				spec.Name, entry.Checksum, fetched.Checksum)}
		}
	}
	return nil
}

func selectPackages(pkgManifest *Manifest, names []string) []PackageSpec {
	if len(names) == 0 {
		sortedNames := make([]string, 0, len(pkgManifest.Packages))
		for name := range pkgManifest.Packages {
			sortedNames = append(sortedNames, name)
		}
		sort.Strings(sortedNames)
		out := make([]PackageSpec, 0, len(sortedNames))
		for _, name := range sortedNames {
			out = append(out, pkgManifest.Packages[name])
		}
		return out
	}
	out := make([]PackageSpec, 0, len(names))
	for _, name := range names {
		if spec, ok := pkgManifest.Packages[name]; ok {
			out = append(out, spec)
		}
	}
	return out
}

func listPackages(discovered *project.Project, pkgManifest *Manifest) []PackageListing {
	names := make([]string, 0, len(pkgManifest.Packages))
	for name := range pkgManifest.Packages {
		names = append(names, name)
	}
	sort.Strings(names)
	listings := make([]PackageListing, 0, len(names))
	for _, name := range names {
		spec := pkgManifest.Packages[name]
		installed := false
		if info, statErr := os.Stat(filepath.Join(discovered.AddonsDir, spec.InstallName())); statErr == nil {
			installed = info.IsDir()
		}
		listings = append(listings, PackageListing{
			Name:      name,
			Source:    string(spec.Source),
			Version:   spec.Version,
			Installed: installed,
		})
	}
	return listings
}
