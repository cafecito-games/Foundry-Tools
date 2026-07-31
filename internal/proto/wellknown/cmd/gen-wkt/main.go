// Package main regenerates the checked-in well-known type bindings.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	wellknowngen "github.com/cafecito-games/foundry-tools/internal/proto/wellknown/gen"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	files, err := wellknowngen.Generate()
	if err != nil {
		return err
	}

	bindingDir := filepath.Join(wellknowngen.DataDir, filepath.FromSlash(namespacePath()))
	// A removed or renamed type must not leave a stale binding behind, and the
	// directory holds nothing but generated output, so it is rebuilt each run.
	if err := os.RemoveAll(bindingDir); err != nil {
		return fmt.Errorf("clear %s: %w", bindingDir, err)
	}

	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if !strings.HasPrefix(name, namespacePath()+"/") {
			return fmt.Errorf("%s was generated outside %s", name, namespacePath())
		}
		path := filepath.Join(wellknowngen.DataDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // Runtime source directories are checked in and project-readable.
			return err
		}
		if err := os.WriteFile(path, []byte(files[name]), 0o644); err != nil { //nolint:gosec // Runtime source files are checked in and project-readable.
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	_, err = fmt.Printf("generated %d well-known type binding(s)\n", len(names))
	return err
}

func namespacePath() string {
	return strings.ReplaceAll(wellknown.Namespace, ".", "/")
}
