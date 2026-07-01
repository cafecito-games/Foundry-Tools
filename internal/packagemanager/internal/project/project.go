// Package project discovers Foundry project roots and package-manager paths.
package project

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
)

// Project describes a located Foundry project and package-manager paths.
type Project struct {
	Root         string
	ManifestPath string
	LockPath     string
	AddonsDir    string
}

// ErrProjectNotFound marks discovery failure when no project.foundry exists.
var ErrProjectNotFound = errors.New("no project.foundry found")

// Discover walks up from startDir until it finds a directory containing
// project.foundry.
func Discover(startDir string) (*Project, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, &output.ManifestError{Err: err}
	}
	for {
		projectFile := filepath.Join(dir, "project.foundry")
		info, statErr := os.Stat(projectFile)
		if statErr == nil {
			if !info.Mode().IsRegular() {
				return nil, &output.ManifestError{Err: fmt.Errorf("%s is not a regular file", projectFile)}
			}
			return forRoot(dir), nil
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			return nil, &output.ManifestError{Err: fmt.Errorf("checking %s: %w", projectFile, statErr)}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, &output.ManifestError{
				Err: fmt.Errorf("%w in %s or any parent directory", ErrProjectNotFound, startDir),
			}
		}
		dir = parent
	}
}

func forRoot(root string) *Project {
	return &Project{
		Root:         root,
		ManifestPath: filepath.Join(root, "packages.toml"),
		LockPath:     filepath.Join(root, "packages.lock"),
		AddonsDir:    filepath.Join(root, "addons"),
	}
}
