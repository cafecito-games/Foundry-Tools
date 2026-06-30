package protoparse

import (
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
		if _, err := ResolveExternalWithFiles(file, filename, importFS); err != nil {
			return nil, err
		}
		out = append(out, ParsedFile{Filename: filename, File: file})
	}
	return out, nil
}
