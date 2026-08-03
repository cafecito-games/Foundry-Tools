//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// The upstream conformance schema is the exhaustive statement of proto3, so
// generating it whole is what "we support proto3" means. Before the
// fixed-width and zig-zag scalars landed this failed on the first sint32.
//
// It asserts structure rather than exact source: the point is that every
// construct in the file survives generation, and pinning the emitted text
// here would duplicate examples/golden while making an upstream refresh look
// like a generator regression.
func TestConformanceSchemaGenerates(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	fixture := "tests/integration/fixtures/conformance"

	args := []string{"run", "./cmd/anvil", "proto", "generate", "-I", fixture, "-o", outDir,
		filepath.Join(fixture, "test_messages_proto3.proto"),
	}
	for _, wellKnown := range []string{"any", "duration", "empty", "field_mask", "struct", "timestamp", "wrappers"} {
		args = append(args, filepath.Join(fixture, "google/protobuf", wellKnown+".proto"))
	}
	run(t, root, "go", args...)

	// The schema's own declarations, plus the oneof union that carries a type
	// nested in the message declaring it.
	for _, name := range []string{
		"protobuf_test_messages/proto3/TestAllTypesProto3.pb.fs",
		"protobuf_test_messages/proto3/TestAllTypesProto3OneofFieldCase.pb.fs",
		"protobuf_test_messages/proto3/ForeignMessage.pb.fs",
		"protobuf_test_messages/proto3/ForeignEnum.pb.fs",
	} {
		_, err := os.Stat(filepath.Join(outDir, name))
		require.NoError(t, err, "conformance schema did not produce %s", name)
	}

	// The fixture vendors its own google/protobuf copies and the invocation
	// above names them outright, which is how a repo with a vendored tree calls
	// the generator. They are still the well-known types, so they arrive as the
	// runtime's single set of bindings -- including the recursive Value -- and
	// not as a project-local second copy the runtime would not recognize.
	for _, name := range []string{
		"foundry/proto/wkt/Struct.pb.fs",
		"foundry/proto/wkt/Value.pb.fs",
		"foundry/proto/wkt/Any.pb.fs",
	} {
		_, err := os.Stat(filepath.Join(outDir, name))
		require.NoError(t, err, "the runtime did not ship %s", name)
	}
	_, statErr := os.Stat(filepath.Join(outDir, "google"))
	require.True(t, os.IsNotExist(statErr), "a vendored well-known file must generate no binding of its own")

	suite, err := os.ReadFile(filepath.Join(outDir, "protobuf_test_messages/proto3/TestAllTypesProto3.pb.fs"))
	require.NoError(t, err)
	source := string(suite)

	// One assertion per framing the eight scalars introduced, so a regression
	// that drops back to varints fails here rather than only in the engine.
	require.Contains(t, source, "Wire.encode_float(optional_float)")
	require.Contains(t, source, "Wire.encode_double(optional_double)")
	require.Contains(t, source, "Wire.encode_fixed32(optional_fixed32)")
	require.Contains(t, source, "Wire.encode_fixed64(optional_fixed64)")
	require.Contains(t, source, "Wire.encode_sint32(optional_sint32)")
	require.Contains(t, source, "Wire.encode_sint64(optional_sint64)")
	require.Contains(t, source, "var _pb_optional_sfixed32_read: FixedRead = Wire.read_sfixed32(_pb_data, _pb_offset)")
	require.Contains(t, source, "var _pb_optional_sfixed64_read: FixedRead = Wire.read_fixed64(_pb_data, _pb_offset)")

	// Recursion, direct and mutual, resolves rather than looping the emitter.
	require.Contains(t, source, "var recursive_message: TestAllTypesProto3? = null")
}

func TestAnyProtoJSONFixturesMatchProtobuf(t *testing.T) {
	root := repoRoot(t)
	resolver, ordinary := anyFixtureResolver(t, root)

	ordinary.Set(ordinary.Descriptor().Fields().ByName("optional_int32"), protoreflect.ValueOfInt32(7))
	ordinary.Set(ordinary.Descriptor().Fields().ByName("optional_string"), protoreflect.ValueOfString("Ada"))
	object, err := structpb.NewStruct(map[string]any{"enabled": true, "name": "Ada"})
	require.NoError(t, err)
	list, err := structpb.NewList([]any{"first", nil, float64(7)})
	require.NoError(t, err)
	inner, err := anypb.New(wrapperspb.String("nested"))
	require.NoError(t, err)

	cases := []struct {
		name     string
		file     string
		typeURL  string
		embedded proto.Message
	}{
		{
			name:     "ordinary canonical URL",
			file:     "ordinary.json",
			typeURL:  "type.googleapis.com/protobuf_test_messages.proto3.TestAllTypesProto3",
			embedded: ordinary,
		},
		{
			name:     "ordinary foreign URL",
			file:     "ordinary-foreign-prefix.json",
			typeURL:  "https://peer.example/types/protobuf_test_messages.proto3.TestAllTypesProto3",
			embedded: ordinary,
		},
		{
			name:     "timestamp",
			file:     "timestamp.json",
			typeURL:  "type.googleapis.com/google.protobuf.Timestamp",
			embedded: timestamppb.New(time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)),
		},
		{
			name:     "wrapper",
			file:     "int32-wrapper.json",
			typeURL:  "type.googleapis.com/google.protobuf.Int32Value",
			embedded: wrapperspb.Int32(7),
		},
		{
			name:     "struct",
			file:     "struct.json",
			typeURL:  "type.googleapis.com/google.protobuf.Struct",
			embedded: object,
		},
		{
			name:     "value",
			file:     "value.json",
			typeURL:  "type.googleapis.com/google.protobuf.Value",
			embedded: structpb.NewNullValue(),
		},
		{
			name:     "list value",
			file:     "list-value.json",
			typeURL:  "type.googleapis.com/google.protobuf.ListValue",
			embedded: list,
		},
		{
			name:     "nested Any",
			file:     "nested-any.json",
			typeURL:  "type.googleapis.com/google.protobuf.Any",
			embedded: inner,
		},
	}

	fixtureDir := filepath.Join(root, "tests/integration/fixtures/conformance/any_protojson")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document, err := os.ReadFile(filepath.Join(fixtureDir, tc.file))
			require.NoError(t, err)

			var decoded anypb.Any
			require.NoError(t, (protojson.UnmarshalOptions{Resolver: resolver}).Unmarshal(document, &decoded))
			require.Equal(t, tc.typeURL, decoded.TypeUrl, "a valid supplied URL must be retained verbatim")

			expectedWire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(tc.embedded)
			require.NoError(t, err)
			require.Equal(t, expectedWire, decoded.Value, "Any.value must contain the canonical embedded wire bytes")

			emitted, err := (protojson.MarshalOptions{Resolver: resolver}).Marshal(&decoded)
			require.NoError(t, err)
			require.JSONEq(t, string(document), string(emitted), "fixture must use protobuf's canonical Any JSON structure")
		})
	}

	// The approved Foundry mapping deliberately treats Empty as an ordinary
	// message, so its Any has no value envelope. The engine acceptance test
	// exercises its decode and exact empty payload; this fixture pins the one
	// intentional shape difference from Go protojson's custom-WKT treatment.
	emptyDocument, err := os.ReadFile(filepath.Join(fixtureDir, "empty.json"))
	require.NoError(t, err)
	var emptyObject map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(emptyDocument, &emptyObject))
	require.Equal(t, map[string]json.RawMessage{
		"@type": json.RawMessage(`"type.googleapis.com/google.protobuf.Empty"`),
	}, emptyObject)
	emptyWire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(&emptypb.Empty{})
	require.NoError(t, err)
	require.Empty(t, emptyWire)
}

func TestAnyProtoJSONMalformedFixturesAreRejected(t *testing.T) {
	root := repoRoot(t)
	resolver, _ := anyFixtureResolver(t, root)
	fixtureDir := filepath.Join(root, "tests/integration/fixtures/conformance/any_protojson")

	for _, name := range []string{
		"malformed-missing-type.json",
		"malformed-nonstring-type.json",
		"malformed-trailing-slash.json",
		"malformed-type-name.json",
		"malformed-unregistered-type.json",
	} {
		t.Run(name, func(t *testing.T) {
			document, err := os.ReadFile(filepath.Join(fixtureDir, name))
			require.NoError(t, err)
			var decoded anypb.Any
			require.Error(t, (protojson.UnmarshalOptions{Resolver: resolver}).Unmarshal(document, &decoded))
		})
	}
}

func anyFixtureResolver(t *testing.T, root string) (*protoregistry.Types, *dynamicpb.Message) {
	t.Helper()
	descriptorPath := filepath.Join(t.TempDir(), "conformance.desc")
	fixture := "tests/integration/fixtures/conformance"
	run(t, root, "protoc",
		"--descriptor_set_out="+descriptorPath,
		"--include_imports",
		"-I", fixture,
		filepath.Join(fixture, "test_messages_proto3.proto"),
	)

	raw, err := os.ReadFile(descriptorPath)
	require.NoError(t, err)
	var set descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(raw, &set))
	files, err := protodesc.NewFiles(&set)
	require.NoError(t, err)
	descriptor, err := files.FindDescriptorByName("protobuf_test_messages.proto3.TestAllTypesProto3")
	require.NoError(t, err)
	ordinaryDescriptor := descriptor.(protoreflect.MessageDescriptor)

	resolver := new(protoregistry.Types)
	for _, message := range []proto.Message{
		&anypb.Any{},
		&durationpb.Duration{},
		&emptypb.Empty{},
		&fieldmaskpb.FieldMask{},
		&structpb.Struct{},
		&structpb.Value{},
		&structpb.ListValue{},
		&timestamppb.Timestamp{},
		&wrapperspb.BoolValue{},
		&wrapperspb.Int32Value{},
		&wrapperspb.Int64Value{},
		&wrapperspb.UInt32Value{},
		&wrapperspb.UInt64Value{},
		&wrapperspb.FloatValue{},
		&wrapperspb.DoubleValue{},
		&wrapperspb.StringValue{},
		&wrapperspb.BytesValue{},
	} {
		require.NoError(t, resolver.RegisterMessage(message.ProtoReflect().Type()))
	}
	registerDynamicMessages(t, resolver, ordinaryDescriptor.ParentFile().Messages())
	return resolver, dynamicpb.NewMessage(ordinaryDescriptor)
}

func registerDynamicMessages(t *testing.T, resolver *protoregistry.Types, messages protoreflect.MessageDescriptors) {
	t.Helper()
	for i := 0; i < messages.Len(); i++ {
		descriptor := messages.Get(i)
		require.NoError(t, resolver.RegisterMessage(dynamicpb.NewMessageType(descriptor)))
		registerDynamicMessages(t, resolver, descriptor.Messages())
	}
}
