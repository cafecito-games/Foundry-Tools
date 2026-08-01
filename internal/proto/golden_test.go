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

	wellKnownGoldenProto = "../../examples/golden-wkt/event.proto"
	wellKnownGoldenDir   = "../../examples/golden-wkt/generated"

	jsonGoldenProto = "../../examples/golden-json/json_suite.proto"
	jsonGoldenDir   = "../../examples/golden-json/generated"
)

// TestGoldenExampleProto keeps examples/golden in lockstep with the emitter.
// Run `go test ./internal/proto -run TestGolden -update` to regenerate.
func TestGoldenExampleProto(t *testing.T) {
	requireGolden(t, goldenProto, goldenDir, proto.Options{})
}

// TestGoldenWellKnownProto covers a schema whose every reference is to a
// well-known type, in each position a reference can take.
func TestGoldenWellKnownProto(t *testing.T) {
	requireGolden(t, wellKnownGoldenProto, wellKnownGoldenDir, proto.Options{})
}

// TestGoldenJSONProto pins the emitter with the JSON option on. It is a corpus
// of its own rather than a regeneration of examples/golden so that every other
// golden assertion keeps covering the option's off-path.
func TestGoldenJSONProto(t *testing.T) {
	requireGolden(t, jsonGoldenProto, jsonGoldenDir, proto.Options{JSON: true})
}

func requireGolden(t *testing.T, protoPath, goldenPath string, options proto.Options) {
	t.Helper()

	parsedFiles, err := proto.ParseFiles([]string{protoPath}, []string{filepath.Dir(protoPath)})
	require.NoError(t, err)
	require.Len(t, parsedFiles, 1)

	parsed := parsedFiles[0]
	require.Empty(t, proto.Validate(parsed.File, parsed.Filename))

	generated, err := proto.Generate(parsed.File, parsed.Filename, proto.ImportsOf(parsed), options)
	require.NoError(t, err)

	if *updateGolden {
		writeGolden(t, goldenPath, generated)
		return
	}

	require.Equal(t, sortedKeys(generated), goldenFileNames(t, goldenPath),
		"generated file set differs from %s; rerun with -update", goldenPath)
	for name, source := range generated {
		want, err := os.ReadFile(filepath.Join(goldenPath, name))
		require.NoError(t, err)
		require.Equal(t, string(want), source,
			"%s is stale; rerun with -update", name)
	}
}

func writeGolden(t *testing.T, goldenDir string, generated proto.GeneratedFiles) {
	t.Helper()

	require.NoError(t, os.RemoveAll(goldenDir))
	for name, source := range generated {
		path := filepath.Join(goldenDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(source), 0o644))
	}
	t.Logf("wrote %d golden file(s) to %s", len(generated), goldenDir)
}

func goldenFileNames(t *testing.T, goldenDir string) []string {
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
