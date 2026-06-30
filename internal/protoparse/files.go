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
		importFS := &parseFilesFS{
			baseDir:      filepath.Dir(filename),
			includePaths: importRoots,
		}
		if err := resolveExternalForParseFiles(file, importFS); err != nil {
			return nil, err
		}
		out = append(out, ParsedFile{Filename: filename, File: file})
	}
	return out, nil
}

type parseFilesFS struct {
	baseDir      string
	includePaths []string
}

func resolveExternalForParseFiles(file *protoast.ProtoFile, fs *parseFilesFS) error {
	registry := map[string]importedType{}
	visited := map[string]bool{}
	for _, imp := range file.Imports {
		collectParseFilesImportedTypes(registry, visited, imp.Path, fs)
	}
	if len(registry) != 0 {
		lookup := buildLookup(registry, file.Package)
		for _, m := range file.Messages {
			annotateMessage(m, lookup)
		}
	}
	return nil
}

func collectParseFilesImportedTypes(out map[string]importedType, visited map[string]bool, path string, fs *parseFilesFS) {
	if visited[path] {
		return
	}
	visited[path] = true
	if !fs.exists(path) {
		return
	}
	data, err := fs.read(path)
	if err != nil {
		return
	}
	tokens, err := Tokenize(string(data), path)
	if err != nil {
		return
	}
	impFile, err := Parse(tokens, path)
	if err != nil {
		return
	}
	prefix := ""
	if impFile.Package != "" {
		prefix = impFile.Package + "."
	}
	for _, m := range impFile.Messages {
		collectFromMessage(out, m, prefix, "", path)
	}
	for _, e := range impFile.Enums {
		fullName := prefix + e.Name
		out[fullName] = importedType{
			SourceFile: path,
			IsEnum:     true,
			FullName:   fullName,
			ShortName:  e.Name,
			EnumValues: e.Values,
		}
	}
	for _, nested := range impFile.Imports {
		if nested.Public {
			collectParseFilesImportedTypes(out, visited, nested.Path, fs)
		}
	}
}

func (f *parseFilesFS) read(path string) ([]byte, error) {
	if found, ok := f.locate(path); ok {
		return os.ReadFile(found) //nolint:gosec // Import path is resolved from explicit CLI input and proto imports.
	}
	return nil, fmt.Errorf("import %q not found from %s", path, f.baseDir)
}

func (f *parseFilesFS) exists(path string) bool {
	_, ok := f.locate(path)
	return ok
}

func (f *parseFilesFS) locate(path string) (string, bool) {
	for _, inc := range f.includePaths {
		if c := filepath.Join(inc, path); parseFilesStatOK(c) {
			return c, true
		}
	}
	candidate := filepath.Join(f.baseDir, path)
	if parseFilesStatOK(candidate) {
		return candidate, true
	}
	dir := f.baseDir
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		if c := filepath.Join(dir, path); parseFilesStatOK(c) {
			return c, true
		}
	}
	return "", false
}

func parseFilesStatOK(path string) bool {
	_, err := os.Stat(path) //nolint:gosec // Import path is resolved from explicit CLI input and proto imports.
	return err == nil
}
