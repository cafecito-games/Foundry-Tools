package runtime_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	wellknowngen "github.com/cafecito-games/foundry-tools/internal/proto/wellknown/gen"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

func TestFilesReturnsRuntimeSources(t *testing.T) {
	files := runtime.Files()

	require.Contains(t, files, "foundry/proto/message.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func decode_bytes")
	require.Contains(t, files["foundry/proto/wire.fs"], "static func skip_field")

	// Struct/Value/ListValue are the one protobuf API that inherently bridges a
	// dynamic native tree. Keep that exception confined to their generated
	// bindings; every other runtime surface stays Variant-free.
	nonBridge := make(map[string]string, len(files)-3)
	for name, source := range files {
		switch name {
		case "foundry/proto/wkt/Struct.pb.fs", "foundry/proto/wkt/Value.pb.fs", "foundry/proto/wkt/ListValue.pb.fs":
			continue
		default:
			nonBridge[name] = source
		}
	}
	require.NotContains(t, runtime.PublicSource(nonBridge), "Variant")
	require.Contains(t, files["foundry/proto/wkt/Struct.pb.fs"],
		"func to_dictionary() -> Dictionary[String, Variant]:")
	require.Contains(t, files["foundry/proto/wkt/Value.pb.fs"],
		"static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError):")
	require.Contains(t, files["foundry/proto/wkt/ListValue.pb.fs"],
		"static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError):")
}

func TestAnyTypeRegistryHasTypedExplicitPublicSurface(t *testing.T) {
	files := runtime.Files()
	const path = "foundry/proto/any_type_registry.fs"
	source, ok := files[path]
	require.True(t, ok, "%s must be embedded with the runtime", path)
	require.Contains(t, source, "class_name AnyTypeRegistry extends RefCounted")
	require.Contains(t, source, "static var _types: Dictionary[String, Type[Message]] = {}")
	require.Contains(t, source, "static func register(message_type: Type[Message]) -> ProtobufError:")
	require.Contains(t, source, "static func clear() -> void:")
	require.Contains(t, source, "static func _resolve(type_url: String) -> (Type[Message]?, ProtobufError):")
	require.Contains(t, source, "static func _type_name_from_url(type_url: String) -> (String, ProtobufError):")
	require.NotRegexp(t, `(?m)^static func (resolve|type_name_from_url)\(`, source)
	require.NotContains(t, source, "Callable")
	require.NotContains(t, source, "Variant")
	require.NotRegexp(t, `(?i)(prototype|network|http|load\()`, source)
}

func TestAnyBindingCarriesGeneratedWireHelpers(t *testing.T) {
	const path = "foundry/proto/wkt/Any.pb.fs"
	source := runtime.Files()[path]
	require.Contains(t, source, "static func pack(message: Message) -> Any:")
	require.Contains(t, source, "func is_type(message_type: Type[Message]) -> bool:")
	require.Contains(t, source, "func unpack() -> (Message?, ProtobufError):")

	generated, err := wellknowngen.Generate()
	require.NoError(t, err)
	require.Equal(t, generated[path], source, "%s must only change through `task gen-wkt`", path)
}

// Trait requirements must be abstract; a bare func fails to resolve the trait
// body in every consumer that applies it.
func TestTraitRequirementsAreAbstract(t *testing.T) {
	files := runtime.Files()

	require.Contains(t, files["foundry/proto/message.fs"], "trait_name Message\n")
	require.Contains(t, files["foundry/proto/message.fs"], "abstract static func create_message() -> Self")
	require.Contains(t, files["foundry/proto/message.fs"], "abstract static func protobuf_type_name() -> String")
	require.Contains(t, files["foundry/proto/message.fs"], "abstract func type_name() -> String")
	require.Contains(t, files["foundry/proto/message.fs"], "abstract func to_bytes()")
	require.Contains(t, files["foundry/proto/codec.fs"], "abstract func encode(")
	require.NotRegexp(t, `(?m)^func `, files["foundry/proto/message.fs"])
	require.NotRegexp(t, `(?m)^func `, files["foundry/proto/codec.fs"])
}

// The read carriers are named tuples, one per file: a tuple_name file may
// contain nothing but its own declaration.
func TestReadCarriersAreSingleDeclarationTupleFiles(t *testing.T) {
	files := runtime.Files()

	for _, name := range []string{"varint_read", "string_read", "bytes_read", "skip_read"} {
		path := "foundry/proto/" + name + ".fs"
		require.Contains(t, files, path)
		require.Contains(t, files[path], "tuple_name ")
		require.NotContains(t, files[path], "class_name ")
		require.NotContains(t, files[path], "func ")
	}
	require.NotContains(t, files, "foundry/proto/field_read.fs")
	require.NotContains(t, files, "foundry/proto/decode_result.fs")
}

// The JSON error cases append after the existing wire-format cases without
// renumbering them, so a caller that already stored a ProtobufError value
// keeps the same meaning.
func TestProtobufErrorCarriesTheJSONCases(t *testing.T) {
	source := runtime.Files()["foundry/proto/protobuf_error.fs"]

	require.Contains(t, source, "UNKNOWN_REQUIRED_FEATURE = 6")
	require.Contains(t, source, "JSON_PARSE_FAILED = 7")
	require.Contains(t, source, "JSON_TYPE_MISMATCH = 8")
	require.Contains(t, source, "JSON_UNKNOWN_FIELD = 9")
	require.Contains(t, source, "JSON_VALUE_OUT_OF_RANGE = 10")
	require.Contains(t, source, "JSON_ANY_UNSUPPORTED = 11")
}

func TestProtobufErrorCarriesTheStructConversionCases(t *testing.T) {
	source := runtime.Files()["foundry/proto/protobuf_error.fs"]

	require.Contains(t, source, "JSON_ANY_UNSUPPORTED = 11")
	require.Contains(t, source, "STRUCT_KEY_NOT_STRING = 12")
	require.Contains(t, source, "STRUCT_VALUE_UNREPRESENTABLE = 13")
	require.Contains(t, source, "WELL_KNOWN_TIME_OUT_OF_RANGE = 14")
}

func TestProtobufErrorCarriesTheAnyRegistryCases(t *testing.T) {
	source := runtime.Files()["foundry/proto/protobuf_error.fs"]

	require.Contains(t, source, "JSON_ANY_UNSUPPORTED = 11")
	require.Contains(t, source, "STRUCT_KEY_NOT_STRING = 12")
	require.Contains(t, source, "STRUCT_VALUE_UNREPRESENTABLE = 13")
	require.Contains(t, source, "WELL_KNOWN_TIME_OUT_OF_RANGE = 14")
	require.Contains(t, source, "ANY_TYPE_NAME_INVALID = 15")
	require.Contains(t, source, "ANY_REGISTRY_CONFLICT = 16")
	require.Contains(t, source, "ANY_TYPE_URL_INVALID = 17")
	require.Contains(t, source, "ANY_TYPE_NOT_REGISTERED = 18")
	require.Contains(t, source, "ANY_JSON_UNSUPPORTED = 19")
}

// The well-known bindings are checked in so consumers get them without running
// the generator; regenerating here is what keeps the two in step.
func TestWellKnownBindingsAreUpToDate(t *testing.T) {
	generated, err := wellknowngen.Generate()
	require.NoError(t, err)
	require.NotEmpty(t, generated)

	embedded := runtime.Files()
	for name, want := range generated {
		got, ok := embedded[name]
		require.True(t, ok, "%s is generated but not checked in; run `task gen-wkt`", name)
		require.Equal(t, want, got, "%s is stale; run `task gen-wkt`", name)
	}

	for name := range embedded {
		if !strings.HasPrefix(name, "foundry/proto/wkt/") {
			continue
		}
		require.Contains(t, generated, name, "%s is checked in but no longer generated; run `task gen-wkt`", name)
		require.Contains(t, embedded[name], "namespace "+wellknown.Namespace+"\n", "%s must declare the shared namespace", name)
	}
}

// The narrowing a proto float needs is the encoder's own, so it lives beside
// the encoder rather than being restated by each emitter that needs it.
func TestWireNarrowsAProtoFloatToBinary32(t *testing.T) {
	source := runtime.Files()["foundry/proto/wire.fs"]

	require.Contains(t, source, "static func narrow_float32(value: float) -> float:")
	require.Contains(t, source, "narrowed.encode_float(0, value)")
}
