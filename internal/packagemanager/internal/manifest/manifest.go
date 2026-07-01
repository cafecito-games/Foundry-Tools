package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Load reads and parses packages.toml at path. Each PackageSpec.Name is set
// from its TOML table key.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	m := &Manifest{}
	if err := toml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	if m.Packages == nil {
		m.Packages = map[string]PackageSpec{}
	}
	for name, pkg := range m.Packages {
		pkg.Name = name
		m.Packages[name] = pkg
	}
	return m, nil
}

// Save writes the manifest to path as TOML using an atomic rename.
func (m *Manifest) Save(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".packages-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp manifest: %w", err)
	}
	tmpName := tmp.Name()

	if err := toml.NewEncoder(tmp).Encode(m); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("encoding manifest %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("syncing manifest %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing manifest %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("installing manifest %s: %w", path, err)
	}
	return nil
}
