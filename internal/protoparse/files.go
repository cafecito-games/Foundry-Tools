package protoparse

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

// ParsedFile is a parsed protobuf file and the filename used for diagnostics.
type ParsedFile struct {
	Filename string
	File     *protoast.ProtoFile
}

// ParseFiles parses root proto files using importRoots.
func ParseFiles(filenames, importRoots []string) ([]ParsedFile, error) {
	out := make([]ParsedFile, 0, len(filenames))
	for _, filename := range filenames {
		data, err := os.ReadFile(filename) //nolint:gosec // CLI input path is explicitly user-provided.
		if err != nil {
			return nil, err
		}
		tokens, err := Tokenize(string(data), filename)
		if err != nil {
			return nil, err
		}
		file, err := Parse(tokens, filename)
		if err != nil {
			return nil, err
		}
		importFS := &OSFS{
			BaseDir:      filepath.Dir(filename),
			IncludePaths: importRoots,
		}
		if err := validateImportedFiles(file, filename, importFS, map[string]bool{}); err != nil {
			return nil, err
		}
		if _, err := ResolveExternalWithFiles(file, filename, importFS); err != nil {
			return nil, err
		}
		out = append(out, ParsedFile{Filename: filename, File: file})
	}
	return out, nil
}

func validateImportedFiles(file *protoast.ProtoFile, owner string, fs FS, visited map[string]bool) error {
	for _, imp := range file.Imports {
		if visited[imp.Path] {
			continue
		}
		visited[imp.Path] = true
		if !fs.Exists(imp.Path) {
			return fmt.Errorf("import %q not found from %s", imp.Path, owner)
		}
		data, err := fs.Read(imp.Path)
		if err != nil {
			return fmt.Errorf("read import %q from %s: %w", imp.Path, owner, err)
		}
		tokens, err := Tokenize(string(data), imp.Path)
		if err != nil {
			return fmt.Errorf("tokenize import %q from %s: %w", imp.Path, owner, err)
		}
		imported, err := Parse(tokens, imp.Path)
		if err != nil {
			return fmt.Errorf("parse import %q from %s: %w", imp.Path, owner, err)
		}
		if err := validateImportedFiles(imported, imp.Path, fs, visited); err != nil {
			return err
		}
	}
	return nil
}
