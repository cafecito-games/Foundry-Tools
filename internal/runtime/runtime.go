package runtime

import (
	"embed"
	"io/fs"
	pathpkg "path"
	"sort"
	"strings"
)

//go:embed data/*/**/*.fs
var runtimeFS embed.FS

// Files returns embedded runtime source files keyed by generated output path.
func Files() map[string]string {
	files := make(map[string]string)

	err := fs.WalkDir(runtimeFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || pathpkg.Ext(path) != ".fs" {
			return nil
		}

		source, err := runtimeFS.ReadFile(path)
		if err != nil {
			return err
		}

		key := strings.TrimPrefix(path, "data/")
		files[key] = string(source)
		return nil
	})
	if err != nil {
		panic(err)
	}

	return files
}

// PublicSource concatenates source contents in stable sorted key order.
func PublicSource(files map[string]string) string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(files[key])
	}
	return builder.String()
}
