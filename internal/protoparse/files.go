package protoparse

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

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
		if err := parseFilesResolveExternalWithFiles(file, filename, importFS); err != nil {
			return nil, err
		}
		out = append(out, ParsedFile{Filename: filename, File: file})
	}
	return out, nil
}

func parseFilesResolveExternalWithFiles(file *protoast.ProtoFile, filename string, importFS FS) error {
	out := reflect.ValueOf(ResolveExternalWithFiles).Call([]reflect.Value{
		reflect.ValueOf(file),
		reflect.ValueOf(filename),
		reflect.ValueOf(importFS),
	})
	if errValue := out[1]; !errValue.IsNil() {
		return errValue.Interface().(error)
	}
	return nil
}

type parseFilesFS struct {
	baseDir      string
	includePaths []string
}

// Read locates and reads a proto import for ParseFiles.
func (f *parseFilesFS) Read(path string) ([]byte, error) {
	if found, ok := f.locate(path); ok {
		return os.ReadFile(found) //nolint:gosec // Import path is resolved from explicit CLI input and proto imports.
	}
	return nil, fmt.Errorf("import %q not found from %s", path, f.baseDir)
}

// Exists reports whether a proto import can be located for ParseFiles.
func (f *parseFilesFS) Exists(path string) bool {
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
