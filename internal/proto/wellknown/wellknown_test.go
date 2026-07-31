package wellknown

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsWellKnownImportMatchesOnlyTheExactPath(t *testing.T) {
	require.True(t, IsWellKnownImport("google/protobuf/timestamp.proto"))
	require.True(t, IsWellKnownImport("google/protobuf/struct.proto"))
	require.False(t, IsWellKnownImport("cafecito/game/v1/player.proto"))
	require.False(t, IsWellKnownImport("google/protobuf/descriptor.proto"))

	// A path that merely ends in a well-known spelling is a different file.
	// Answering yes would skip bindings the caller asked for and, in import
	// resolution, stand the bundled schema in for a missing import.
	require.False(t, IsWellKnownImport("myorg/google/protobuf/timestamp.proto"))
	require.False(t, IsWellKnownImport("vendor/google/protobuf/timestamp.proto"))
	require.False(t, IsWellKnownImport("mygoogle/protobuf/timestamp.proto"))
	require.False(t, IsWellKnownImport("google/protobuf/nested/timestamp.proto"))
}

func TestIsWellKnownImportNormalizesSeparatorsAndTraversal(t *testing.T) {
	require.True(t, IsWellKnownImport(`google\protobuf\timestamp.proto`))
	require.True(t, IsWellKnownImport("google/protobuf/./timestamp.proto"))
}

// An include root is what turns a path on disk into an import path, so a
// vendored copy named under `-I vendor` is the well-known file itself.
func TestImportPathForResolvesAgainstIncludeRoots(t *testing.T) {
	name, err := ImportPathFor("vendor/google/protobuf/timestamp.proto", []string{"vendor"})
	require.NoError(t, err)
	require.Equal(t, "google/protobuf/timestamp.proto", name)
	require.True(t, IsWellKnownImport(name))

	name, err = ImportPathFor("third_party/protos/cafecito/game/v1/player.proto", []string{"third_party/protos"})
	require.NoError(t, err)
	require.Equal(t, "cafecito/game/v1/player.proto", name)
}

// The first root that contains the file wins, as it does for protoc.
func TestImportPathForUsesTheFirstContainingRoot(t *testing.T) {
	name, err := ImportPathFor("vendor/google/protobuf/timestamp.proto", []string{"other", "vendor", "."})
	require.NoError(t, err)
	require.Equal(t, "google/protobuf/timestamp.proto", name)
}

// The regression this model exists to fix: a schema of the caller's own whose
// import path only ends in a well-known spelling is an ordinary file, and the
// bindings it was asked for must be generated.
func TestImportPathForKeepsALookalikeUnderItsOwnRoot(t *testing.T) {
	name, err := ImportPathFor("myorg/google/protobuf/timestamp.proto", []string{"."})
	require.NoError(t, err)
	require.Equal(t, "myorg/google/protobuf/timestamp.proto", name)
	require.False(t, IsWellKnownImport(name))
}

// With no -I the path as given is already the import path, which is what protoc
// does with a relative path and no include path of its own.
func TestImportPathForTreatsAPathWithNoRootAsItsOwnImportPath(t *testing.T) {
	name, err := ImportPathFor("google/protobuf/timestamp.proto", nil)
	require.NoError(t, err)
	require.Equal(t, "google/protobuf/timestamp.proto", name)
	require.True(t, IsWellKnownImport(name))

	name, err = ImportPathFor("cafecito/game/v1/player.proto", nil)
	require.NoError(t, err)
	require.Equal(t, "cafecito/game/v1/player.proto", name)
}

// A path outside every include root that still spells out a google/protobuf
// file has no identity: it could be the well-known schema or an unrelated one.
// Skipping it would silently drop bindings, generating it would silently
// produce a second, incompatible Timestamp, so it is an error that says how to
// resolve it.
func TestImportPathForRejectsAnUnrootedGoogleProtobufPath(t *testing.T) {
	_, err := ImportPathFor("/abs/path/to/google/protobuf/timestamp.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be identified without an include path")
	require.Contains(t, err.Error(), "-I /abs/path/to")
	require.Contains(t, err.Error(), "google/protobuf/timestamp.proto")

	// An unshipped google/protobuf file is just as unidentifiable.
	_, err = ImportPathFor("vendor/google/protobuf/descriptor.proto", []string{"other"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be identified without an include path")

	// An ordinary path outside every root is not ambiguous at all.
	name, err := ImportPathFor("cafecito/game/v1/player.proto", []string{"other"})
	require.NoError(t, err)
	require.Equal(t, "cafecito/game/v1/player.proto", name)
}

// An absolute input is claimed by a relative root that contains it, since both
// sides resolve against the working directory.
func TestImportPathForResolvesAnAbsoluteInputAgainstARelativeRoot(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	absolute := filepath.Join(root, "vendor", "google", "protobuf", "timestamp.proto")
	name, err := ImportPathFor(absolute, []string{"vendor"})
	require.NoError(t, err)
	require.Equal(t, "google/protobuf/timestamp.proto", name)
}

func TestUnsupportedGoogleFileIsRejected(t *testing.T) {
	err := Check("google/protobuf/descriptor.proto")
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto")
	require.Contains(t, err.Error(), "not supported")
	require.NoError(t, Check("google/protobuf/timestamp.proto"))
	require.NoError(t, Check("cafecito/game/v1/player.proto"))
}

// Check reads an import path, so a file whose own import path merely ends in a
// google/protobuf spelling is the caller's file and not rejected.
func TestUnsupportedGoogleFileIsIdentifiedByImportPath(t *testing.T) {
	require.NoError(t, Check("myorg/google/protobuf/descriptor.proto"))
	require.NoError(t, Check("vendor/google/protobuf/descriptor.proto"))

	// Resolved against the root that vendors it, the same file is rejected.
	name, err := ImportPathFor("vendor/google/protobuf/descriptor.proto", []string{"vendor"})
	require.NoError(t, err)
	require.Error(t, Check(name))
}

func TestSourceMatchesTheImportPathExactly(t *testing.T) {
	source, err := Source("google/protobuf/timestamp.proto")
	require.NoError(t, err)
	require.Contains(t, string(source), "package google.protobuf;")

	_, err = Source("vendor/google/protobuf/timestamp.proto")
	require.Error(t, err)
}

func TestEmbeddedProtosArePresent(t *testing.T) {
	for _, name := range Files() {
		source, err := Source(name)
		require.NoError(t, err)
		require.Contains(t, string(source), "package google.protobuf;")
		require.Contains(t, string(source), "// Protocol Buffers - Google's data interchange format")
	}
	require.Len(t, Files(), 7)
}

func TestSourceRejectsUnvendoredFile(t *testing.T) {
	_, err := Source("google/protobuf/descriptor.proto")
	require.Error(t, err)
}
