package proto_test

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto"
)

var updateGolden = flag.Bool("update", false, "rewrite examples/golden from the current generator output")

const (
	goldenProto = "../../examples/example.proto"
	goldenDir   = "../../examples/golden"
)

// TestGoldenExampleProto keeps examples/golden in lockstep with the emitter.
// Run `go test ./internal/proto -run TestGolden -update` to regenerate.
func TestGoldenExampleProto(t *testing.T) {
	parsedFiles, err := proto.ParseFiles([]string{goldenProto}, []string{filepath.Dir(goldenProto)})
	require.NoError(t, err)
	require.Len(t, parsedFiles, 1)

	parsed := parsedFiles[0]
	require.Empty(t, proto.Validate(parsed.File, parsed.Filename))

	generated, err := proto.Generate(parsed.File, parsed.Filename, nil)
	require.NoError(t, err)

	if *updateGolden {
		writeGolden(t, generated)
		return
	}

	require.Equal(t, sortedKeys(generated), goldenFileNames(t),
		"generated file set differs from examples/golden; rerun with -update")
	for name, source := range generated {
		want, err := os.ReadFile(filepath.Join(goldenDir, name))
		require.NoError(t, err)
		require.Equal(t, string(want), source,
			"%s is stale; rerun with -update", name)
	}
}

func writeGolden(t *testing.T, generated proto.GeneratedFiles) {
	t.Helper()

	require.NoError(t, os.RemoveAll(goldenDir))
	for name, source := range generated {
		path := filepath.Join(goldenDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o644))
	}
	t.Logf("wrote %d golden file(s) to %s", len(generated), goldenDir)
}

func goldenFileNames(t *testing.T) []string {
	t.Helper()

	var names []string
	err := filepath.WalkDir(goldenDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name, err := filepath.Rel(goldenDir, path)
		if err != nil {
			return err
		}
		names = append(names, filepath.ToSlash(name))
		return nil
	})
	require.NoError(t, err)
	sort.Strings(names)
	return names
}

func sortedKeys(files proto.GeneratedFiles) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
