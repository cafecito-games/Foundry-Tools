package wktcompat_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	goprotodesc "google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	foundrydesc "github.com/cafecito-games/foundry-tools/internal/proto/internal/desc"
	protoparse "github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
	"github.com/cafecito-games/foundry-tools/internal/proto/internal/wktcompat"
	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

func parseFile(t *testing.T, source string) *protoast.ProtoFile {
	t.Helper()
	tokens, err := protoparse.Tokenize(strings.TrimSpace(source), "candidate.proto")
	require.NoError(t, err)
	file, err := protoparse.Parse(tokens, "candidate.proto")
	require.NoError(t, err)
	return file
}

func TestCanonicalWellKnownSourcesAreCompatible(t *testing.T) {
	var candidates []wktcompat.SchemaFile
	for _, importPath := range wellknown.Files() {
		source, err := wellknown.Source(importPath)
		require.NoError(t, err)
		candidates = append(candidates, wktcompat.SchemaFile{
			ImportPath: importPath,
			File:       parseFile(t, string(source)),
		})
	}

	require.NoError(t, wktcompat.Check(candidates))
}

func TestCompatiblePresentationChangesAndAdditionsStayQuiet(t *testing.T) {
	candidate := parseFile(t, `
syntax = "proto3";
package google.protobuf;
option deprecated = true;
message Extra {}
message Timestamp {
  int64 renamed_seconds = 1;
  int32 renamed_nanos = 2;
  string future = 3;
}
enum ExtraEnum { EXTRA = 0; }
`)

	require.NoError(t, wktcompat.Check([]wktcompat.SchemaFile{{
		ImportPath: "google/protobuf/timestamp.proto",
		File:       candidate,
	}}))
}

func TestIncompatibleFieldsReportCanonicalIdentityInStableOrder(t *testing.T) {
	candidate := parseFile(t, `
syntax = "proto3";
package google.protobuf;
message Timestamp {
  repeated string seconds = 1;
}
`)

	err := wktcompat.Check([]wktcompat.SchemaFile{{
		ImportPath: "google/protobuf/timestamp.proto",
		File:       candidate,
	}})
	require.EqualError(t, err,
		"google/protobuf/timestamp.proto: google.protobuf.Timestamp.seconds (#1): "+
			"expected singular int64; found repeated string\n"+
			"google/protobuf/timestamp.proto: google.protobuf.Timestamp.nanos (#2): missing canonical field")
}

func canonicalSource(t *testing.T, importPath string) string {
	t.Helper()
	source, err := wellknown.Source(importPath)
	require.NoError(t, err)
	return string(source)
}

func replaceOnce(t *testing.T, source, old, replacement string) string {
	t.Helper()
	require.Contains(t, source, old)
	return strings.Replace(source, old, replacement, 1)
}

func checkSource(t *testing.T, importPath, source string) error {
	t.Helper()
	return wktcompat.Check([]wktcompat.SchemaFile{{
		ImportPath: importPath,
		File:       parseFile(t, source),
	}})
}

func TestStructuralCompatibilityMatrix(t *testing.T) {
	timestamp := canonicalSource(t, "google/protobuf/timestamp.proto")
	structProto := canonicalSource(t, "google/protobuf/struct.proto")
	splitOneof := replaceOnce(t, structProto,
		"    string string_value = 3;\n    // Represents a boolean value.",
		"    string string_value = 3;\n  }\n  oneof second {\n    // Represents a boolean value.")
	removedOneofMember := replaceOnce(t, structProto,
		"    // Represents a repeated `Value`.\n    ListValue list_value = 6;\n  }",
		"  }\n  // Represents a repeated `Value`.\n  ListValue list_value = 6;")
	timestampOneof := replaceOnce(t, timestamp,
		"message Timestamp {",
		"message Timestamp {\n  oneof time {")
	timestampOneof = replaceOnce(t, timestampOneof,
		"  int32 nanos = 2;\n}",
		"  int32 nanos = 2;\n  }\n}")

	tests := []struct {
		name       string
		importPath string
		source     string
		want       string
	}{
		{
			name:       "package",
			importPath: "google/protobuf/timestamp.proto",
			source:     replaceOnce(t, timestamp, "package google.protobuf;", "package custom;"),
			want:       "expected package google.protobuf; found custom",
		},
		{
			name:       "renamed type",
			importPath: "google/protobuf/timestamp.proto",
			source:     replaceOnce(t, timestamp, "message Timestamp {", "message RenamedTimestamp {"),
			want:       "missing canonical message google.protobuf.Timestamp",
		},
		{
			name:       "message becomes enum",
			importPath: "google/protobuf/timestamp.proto",
			source:     `syntax = "proto3"; package google.protobuf; enum Timestamp { ZERO = 0; }`,
			want:       "google.protobuf.Timestamp: expected message; found enum",
		},
		{
			name:       "moved field",
			importPath: "google/protobuf/timestamp.proto",
			source:     replaceOnce(t, timestamp, "int64 seconds = 1;", "int64 seconds = 9;"),
			want:       "Timestamp.seconds (#1): missing canonical field",
		},
		{
			name:       "scalar type",
			importPath: "google/protobuf/timestamp.proto",
			source:     replaceOnce(t, timestamp, "int64 seconds = 1;", "uint64 seconds = 1;"),
			want:       "expected singular int64; found singular uint64",
		},
		{
			name:       "repeated cardinality",
			importPath: "google/protobuf/timestamp.proto",
			source:     replaceOnce(t, timestamp, "int64 seconds = 1;", "repeated int64 seconds = 1;"),
			want:       "expected singular int64; found repeated int64",
		},
		{
			name:       "optional cardinality",
			importPath: "google/protobuf/timestamp.proto",
			source:     replaceOnce(t, timestamp, "int64 seconds = 1;", "optional int64 seconds = 1;"),
			want:       "expected singular int64; found optional int64",
		},
		{
			name:       "map key",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "map<string, Value> fields = 1;", "map<int64, Value> renamed = 1;"),
			want:       "expected map<string, google.protobuf.Value>; found map<int64, google.protobuf.Value>",
		},
		{
			name:       "map value",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "map<string, Value> fields = 1;", "map<string, NullValue> renamed = 1;"),
			want:       "expected map<string, google.protobuf.Value>; found map<string, google.protobuf.NullValue>",
		},
		{
			name:       "map becomes repeated",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "map<string, Value> fields = 1;", "repeated Value renamed = 1;"),
			want:       "expected map<string, google.protobuf.Value>; found repeated google.protobuf.Value",
		},
		{
			name:       "referenced type",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "Struct struct_value = 5;", "ListValue struct_value = 5;"),
			want:       "expected singular google.protobuf.Struct; found singular google.protobuf.ListValue",
		},
		{
			name:       "oneof rename",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "oneof kind {", "oneof renamed {"),
		},
		{
			name:       "oneof split",
			importPath: "google/protobuf/struct.proto",
			source:     splitOneof,
			want:       "canonical oneof fields [1 2 3 4 5 6]",
		},
		{
			name:       "oneof member removed",
			importPath: "google/protobuf/struct.proto",
			source:     removedOneofMember,
			want:       "canonical oneof fields [1 2 3 4 5 6]",
		},
		{
			name:       "ordinary fields inserted into oneof",
			importPath: "google/protobuf/timestamp.proto",
			source:     timestampOneof,
			want:       "Timestamp.seconds (#1): expected ordinary field; found oneof member",
		},
		{
			name:       "enum value renamed",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "NULL_VALUE = 0;", "RENAMED = 0;"),
		},
		{
			name:       "enum value added",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "NULL_VALUE = 0;", "NULL_VALUE = 0;\n  FUTURE = 1;"),
		},
		{
			name:       "enum number missing",
			importPath: "google/protobuf/struct.proto",
			source:     replaceOnce(t, structProto, "NULL_VALUE = 0;", "RENAMED = 1;"),
			want:       "google.protobuf.NullValue: missing canonical enum number 0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkSource(t, test.importPath, test.source)
			if test.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestDiagnosticsAreOrderedByCanonicalPath(t *testing.T) {
	timestamp := replaceOnce(t,
		canonicalSource(t, "google/protobuf/timestamp.proto"),
		"int32 nanos = 2;", "int32 nanos = 9;")
	structProto := replaceOnce(t,
		canonicalSource(t, "google/protobuf/struct.proto"),
		"NULL_VALUE = 0;", "RENAMED = 1;")

	err := wktcompat.Check([]wktcompat.SchemaFile{
		{ImportPath: "google/protobuf/timestamp.proto", File: parseFile(t, timestamp)},
		{ImportPath: "google/protobuf/struct.proto", File: parseFile(t, structProto)},
	})
	require.Error(t, err)
	lines := strings.Split(err.Error(), "\n")
	require.Len(t, lines, 2)
	require.Contains(t, lines[0], "google/protobuf/struct.proto")
	require.Contains(t, lines[1], "google/protobuf/timestamp.proto")
}

func TestDescriptorAndSourceCandidatesUseTheSameCompatibilityRules(t *testing.T) {
	descriptor := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("google/protobuf/timestamp.proto"),
		Package: proto.String("google.protobuf"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Timestamp"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:   proto.String("seconds"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				},
				{
					Name:   proto.String("nanos"),
					Number: proto.Int32(2),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				},
			},
		}},
	}

	file, err := foundrydesc.FromFileDescriptorProto(descriptor)
	require.NoError(t, err)
	require.NoError(t, wktcompat.Check([]wktcompat.SchemaFile{{
		ImportPath: descriptor.GetName(), File: file,
	}}))

	descriptor.MessageType[0].Field[0].Type = descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()
	file, err = foundrydesc.FromFileDescriptorProto(descriptor)
	require.NoError(t, err)
	err = wktcompat.Check([]wktcompat.SchemaFile{{
		ImportPath: descriptor.GetName(), File: file,
	}})
	require.ErrorContains(t, err, "Timestamp.seconds (#1): expected singular int64; found singular string")
}

func TestParsedSourceASTPreservesIncompatibleOneofCardinality(t *testing.T) {
	tests := []struct {
		name        string
		cardinality string
		mutate      func(*protoast.Field)
	}{
		{
			name:        "optional",
			cardinality: "optional",
			mutate: func(field *protoast.Field) {
				field.Optional = true
			},
		},
		{
			name:        "repeated",
			cardinality: "repeated",
			mutate: func(field *protoast.Field) {
				field.Repeated = true
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := parseFile(t, canonicalSource(t, "google/protobuf/struct.proto"))
			value := file.Messages[1]
			require.Equal(t, "Value", value.Name)
			require.Len(t, value.Oneofs, 1)
			require.Len(t, value.Oneofs[0].Fields, 6)
			test.mutate(value.Oneofs[0].Fields[0])

			err := wktcompat.Check([]wktcompat.SchemaFile{{
				ImportPath: "google/protobuf/struct.proto",
				File:       file,
			}})
			require.EqualError(t, err,
				"google/protobuf/struct.proto: google.protobuf.Value.null_value (#1): "+
					"expected singular google.protobuf.NullValue; found "+
					test.cardinality+" google.protobuf.NullValue")
		})
	}
}

func TestDescriptorConvertedASTPreservesIncompatibleOneofCardinality(t *testing.T) {
	canonical := goprotodesc.ToFileDescriptorProto(structpb.File_google_protobuf_struct_proto)
	file, err := foundrydesc.FromFileDescriptorProto(canonical)
	require.NoError(t, err)
	require.NoError(t, wktcompat.Check([]wktcompat.SchemaFile{{
		ImportPath: canonical.GetName(), File: file,
	}}), "a canonical descriptor oneof must remain compatible")

	tests := []struct {
		name        string
		cardinality string
		mutate      func(*descriptorpb.FieldDescriptorProto)
	}{
		{
			name:        "optional",
			cardinality: "optional",
			mutate: func(field *descriptorpb.FieldDescriptorProto) {
				field.Proto3Optional = proto.Bool(true)
			},
		},
		{
			name:        "repeated",
			cardinality: "repeated",
			mutate: func(field *descriptorpb.FieldDescriptorProto) {
				field.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			descriptor := proto.Clone(canonical).(*descriptorpb.FileDescriptorProto)
			value := descriptor.MessageType[1]
			require.Equal(t, "Value", value.GetName())
			require.Len(t, value.GetField(), 6)
			test.mutate(value.Field[0])

			file, err := foundrydesc.FromFileDescriptorProto(descriptor)
			require.NoError(t, err)
			err = wktcompat.Check([]wktcompat.SchemaFile{{
				ImportPath: descriptor.GetName(), File: file,
			}})
			require.EqualError(t, err,
				"google/protobuf/struct.proto: google.protobuf.Value.null_value (#1): "+
					"expected singular google.protobuf.NullValue; found "+
					test.cardinality+" google.protobuf.NullValue")
		})
	}
}
