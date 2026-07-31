// Package wellknowngen renders the Foundry Script bindings for the vendored
// well-known types. It is a package rather than a bare command so the runtime
// can regenerate them in a test and fail when the checked-in output drifts.
package wellknowngen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cafecito-games/foundry-tools/internal/proto"
	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

// DataDir is the embedded runtime data root the generated bindings are checked
// into, relative to the repository root. Generated paths are namespace-derived,
// so a binding lands at DataDir/foundry/proto/wkt/<type>.pb.fs.
const DataDir = "internal/runtime/data"

// Generate renders every vendored well-known file, keyed by the output path the
// generator assigns it. The vendored files declare `package google.protobuf`,
// so the namespace option is overridden to keep the bindings in the single
// runtime namespace instead of giving each project its own copy.
func Generate() (map[string]string, error) {
	root, err := os.MkdirTemp("", "foundry-tools-wkt")
	if err != nil {
		return nil, fmt.Errorf("create proto staging directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(root)
	}()

	names := wellknown.Files()
	paths := make([]string, 0, len(names))
	for _, name := range names {
		source, err := wellknown.Source(name)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create proto staging directory: %w", err)
		}
		if err := os.WriteFile(path, source, 0o600); err != nil {
			return nil, fmt.Errorf("stage %s: %w", name, err)
		}
		paths = append(paths, path)
	}

	// Staging exists only so the parser has bytes to read; it is not a way to
	// disguise these files. The staging directory is passed as the include
	// root, so each staged copy resolves to exactly the import path it was
	// staged under -- google/protobuf/timestamp.proto and so on -- which is
	// also the name diagnostics should carry rather than the staging copy's.
	//
	// What lets this package render files every other caller skips is that it
	// does not route through a caller that skips them: the well-known check
	// belongs to the `anvil proto generate` command and the protoc plugin, and
	// this calls proto.Validate and proto.Generate directly instead. Adding a
	// classification check to those two would silently empty this output, which
	// is why TestWellKnownBindingsAreUpToDate regenerates and compares.
	parsedFiles, err := proto.ParseFiles(paths, []string{root})
	if err != nil {
		return nil, err
	}

	files := make(map[string]string)
	for index := range parsedFiles {
		parsed := parsedFiles[index]
		sourceName := parsed.ImportPath

		if parsed.File.Options == nil {
			parsed.File.Options = make(map[string]any)
		}
		parsed.File.Options[proto.NamespaceOptionKey] = wellknown.Namespace

		if validationErrors := proto.Validate(parsed.File, sourceName); len(validationErrors) != 0 {
			return nil, fmt.Errorf("%s", proto.FormatValidationErrors(validationErrors))
		}
		// wellknown.Namespace is reserved against every other caller precisely
		// because this package fills it, so this is the one path that renders
		// into it. proto.Generate would reject these files.
		generated, err := proto.GenerateIntoRuntimeNamespace(parsed.File, sourceName, proto.ImportsOf(parsed))
		if err != nil {
			return nil, err
		}
		for name, source := range generated {
			if existing, duplicate := files[name]; duplicate && existing != source {
				return nil, fmt.Errorf("%s is generated twice with different contents", name)
			}
			files[name] = source
		}
	}
	return files, nil
}
