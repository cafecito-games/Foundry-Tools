package protoparse

import (
	"os"
	"path/filepath"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
)

// ParsedFile is a parsed protobuf file and the filename used for diagnostics.
type ParsedFile struct {
	Filename string
	File     *protoast.ProtoFile
	// Imports are the files this one pulled in, transitively through public
	// re-exports. The generator needs them to resolve the namespace and the
	// enum defaults of any type declared outside this file.
	Imports []ImportedFile
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
		imported, err := ResolveExternalWithFiles(file, filename, importFS)
		if err != nil {
			return nil, err
		}
		out = append(out, ParsedFile{Filename: filename, File: file, Imports: imported})
	}
	return out, nil
}
