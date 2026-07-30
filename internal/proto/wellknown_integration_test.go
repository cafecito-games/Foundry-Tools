package proto_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto"
)

// A reference to a well-known type must name the runtime binding, in every
// position a reference can take, and must leave no per-project copy behind.
func TestWellKnownReferenceResolvesToRuntimeNamespace(t *testing.T) {
	files := generateSchema(t, `
syntax = "proto3";

package cafecito.game.v1;

import "google/protobuf/any.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";

message Event {
  google.protobuf.Timestamp occurred_at = 1;
  google.protobuf.Struct attributes = 2;
  repeated google.protobuf.Any attachments = 3;
  map<string, google.protobuf.Timestamp> checkpoints = 4;
  oneof detail {
    google.protobuf.Any payload = 5;
    string note = 6;
  }
}
`)

	source := files["cafecito/game/v1/Event.pb.fs"]
	require.Contains(t, source, "import foundry.proto.wkt")
	require.Contains(t, source, "var occurred_at: Timestamp? = null")
	require.Contains(t, source, "var attributes: Struct? = null")
	require.Contains(t, source, "var attachments: Array[Any] = []")
	require.Contains(t, source, "var checkpoints: Dictionary[String, Timestamp] = {}")

	oneof := files["cafecito/game/v1/EventDetailCase.pb.fs"]
	require.Contains(t, oneof, "import foundry.proto.wkt")
	require.Contains(t, oneof, "Payload(payload: Any)")

	for name := range files {
		require.NotContains(t, name, "google/protobuf/",
			"well-known types must come from the runtime, not per-project generation")
	}
}

// A schema that mentions no well-known type must not import their namespace:
// the import is what makes a schema type of the same name need qualifying.
func TestWellKnownNamespaceIsNotImportedWithoutAReference(t *testing.T) {
	files := generateSchema(t, `
syntax = "proto3";

package cafecito.game.v1;

message Event {
  string note = 1;
}
`)

	require.NotContains(t, files["cafecito/game/v1/Event.pb.fs"], "foundry.proto.wkt")
}

// A schema may declare a type named like a well-known one. The local
// declaration keeps its name and shadows the import; only the well-known
// reference is namespace-qualified.
func TestSchemaTypeNamedLikeAWellKnownTypeKeepsItsName(t *testing.T) {
	files := generateSchema(t, `
syntax = "proto3";

package cafecito.game.v1;

import "google/protobuf/timestamp.proto";

message Timestamp {
  string label = 1;
}

message Reading {
  Timestamp local = 1;
  google.protobuf.Timestamp remote = 2;
  repeated Timestamp locals = 3;
  map<string, google.protobuf.Timestamp> remotes = 4;
}
`)

	local := files["cafecito/game/v1/Timestamp.pb.fs"]
	require.Contains(t, local, "final class_name Timestamp extends RefCounted uses Message")
	require.NotContains(t, local, "foundry.proto.wkt")

	reading := files["cafecito/game/v1/Reading.pb.fs"]
	require.Contains(t, reading, "var local: Timestamp? = null")
	require.Contains(t, reading, "var remote: foundry.proto.wkt.Timestamp? = null")
	require.Contains(t, reading, "var locals: Array[Timestamp] = []")
	require.Contains(t, reading, "var remotes: Dictionary[String, foundry.proto.wkt.Timestamp] = {}")
}

// Naming an unshipped google/protobuf file is an error rather than a second,
// incompatible copy of a type the runtime already defines.
func TestUnsupportedWellKnownFileIsRejected(t *testing.T) {
	_, err := proto.ParseFiles([]string{"google/protobuf/descriptor.proto"}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto")
	require.Contains(t, err.Error(), "not supported")
}

// descriptor.proto arrives as a dependency of any schema that uses custom
// options, so importing it must stay harmless. Only a reference into it fails.
func TestUnsupportedWellKnownDependencyFailsOnlyWhenReferenced(t *testing.T) {
	descriptor := `
syntax = "proto2";
package google.protobuf;
message FileOptions {
  optional string go_package = 11;
}
`
	unused := generateSchema(t, `
syntax = "proto3";

package cafecito.game.v1;

import "google/protobuf/descriptor.proto";

message Event {
  string note = 1;
}
`, schemaFile{"google/protobuf/descriptor.proto", descriptor})
	require.Contains(t, unused["cafecito/game/v1/Event.pb.fs"], "var note: String")

	_, err := generateSchemaError(t, `
syntax = "proto3";

package cafecito.game.v1;

import "google/protobuf/descriptor.proto";

message Event {
  google.protobuf.FileOptions options = 1;
}
`, schemaFile{"google/protobuf/descriptor.proto", descriptor})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}

type schemaFile struct {
	name   string
	source string
}

func generateSchema(t *testing.T, source string, extra ...schemaFile) proto.GeneratedFiles {
	t.Helper()

	files, err := generateSchemaError(t, source, extra...)
	require.NoError(t, err)
	return files
}

// generateSchemaError parses and generates source as a schema named
// schema.proto, with extra written alongside it so imports resolve.
func generateSchemaError(t *testing.T, source string, extra ...schemaFile) (proto.GeneratedFiles, error) {
	t.Helper()

	root := t.TempDir()
	for _, file := range extra {
		path := filepath.Join(root, filepath.FromSlash(file.name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(strings.TrimLeft(file.source, "\n")), 0o600))
	}
	path := filepath.Join(root, "schema.proto")
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimLeft(source, "\n")), 0o600))

	parsedFiles, err := proto.ParseFiles([]string{path}, []string{root})
	if err != nil {
		return nil, err
	}
	require.Len(t, parsedFiles, 1)

	parsed := parsedFiles[0]
	require.Empty(t, proto.Validate(parsed.File, parsed.Filename))
	return proto.Generate(parsed.File, parsed.Filename, proto.ImportsOf(parsed))
}
