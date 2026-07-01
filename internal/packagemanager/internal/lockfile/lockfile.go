// Package lockfile reads and writes reproducible package pins.
package lockfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/manifest"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
)

// Entry pins one resolved package for reproducible installs.
type Entry struct {
	ResolvedVersion string `toml:"resolved_version"`
	SourcePath      string `toml:"source_path"`
	Checksum        string `toml:"checksum,omitempty"`
	SpecHash        string `toml:"spec_hash"`
}

// Lockfile is the parsed contents of packages.lock.
type Lockfile struct {
	Packages map[string]Entry `toml:"packages"`
}

// Load reads packages.lock at path. A missing file yields an empty Lockfile.
func Load(path string) (*Lockfile, error) {
	data, err := os.ReadFile(path) //nolint:gosec // Lockfile path is discovered from the Foundry project root.
	if errors.Is(err, fs.ErrNotExist) {
		return &Lockfile{Packages: map[string]Entry{}}, nil
	}
	if err != nil {
		return nil, &output.ManifestError{Err: fmt.Errorf("reading lockfile %s: %w", path, err)}
	}
	lockfile := &Lockfile{}
	if err := toml.Unmarshal(data, lockfile); err != nil {
		return nil, &output.ManifestError{Err: fmt.Errorf("parsing lockfile %s: %w", path, err)}
	}
	if lockfile.Packages == nil {
		lockfile.Packages = map[string]Entry{}
	}
	return lockfile, nil
}

// Save writes the lockfile to path as TOML using an atomic rename.
func (lockfile *Lockfile) Save(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".packages-lock-*.tmp")
	if err != nil {
		return &output.ManifestError{Err: fmt.Errorf("creating temp lockfile: %w", err)}
	}
	tmpName := tmp.Name()

	if err := toml.NewEncoder(tmp).Encode(lockfile); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return &output.ManifestError{Err: fmt.Errorf("encoding lockfile %s: %w", path, err)}
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return &output.ManifestError{Err: fmt.Errorf("syncing lockfile %s: %w", path, err)}
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return &output.ManifestError{Err: fmt.Errorf("closing lockfile %s: %w", path, err)}
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return &output.ManifestError{Err: fmt.Errorf("installing lockfile %s: %w", path, err)}
	}
	return nil
}

// NeedsResolve reports whether spec must be re-fetched rather than installed
// from its existing lock pin.
func NeedsResolve(spec manifest.PackageSpec, lock *Lockfile) bool {
	entry, ok := lock.Packages[spec.Name]
	if !ok {
		return true
	}
	return entry.SpecHash != spec.Hash()
}
