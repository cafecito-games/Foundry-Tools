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

	// Struct/Value/ListValue inherently bridge a dynamic native tree. Any's
	// registry also has a deliberately private, checked bridge between the
	// unrelated Message and JsonSerializable traits. Every other runtime
	// surface stays Variant-free.
	nonBridge := make(map[string]string, len(files)-4)
	for name, source := range files {
		switch name {
		case "foundry/proto/wkt/Struct.pb.fs", "foundry/proto/wkt/Value.pb.fs", "foundry/proto/wkt/ListValue.pb.fs",
			"foundry/proto/any_type_registry.fs":
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
	require.NotRegexp(t, `(?m)^static var .*Variant`, source)
	require.NotRegexp(t, `(?m)^static func [^_].*Variant`, source)
	require.NotRegexp(t, `(?m)^var .*Variant`, source)
	require.NotRegexp(t, `(?i)(prototype|network|http|load\()`, source)
}

func TestAnyTypeRegistrySerializesOrdinaryJSONThroughPrivateCheckedNarrowing(t *testing.T) {
	source := runtime.Files()["foundry/proto/any_type_registry.fs"]

	require.Contains(t, source,
		"static func _any_to_json(type_url: String, bytes: PackedByteArray) -> JsonNode:")
	require.Contains(t, source,
		"if type_url.is_empty() and bytes.is_empty():\n\t\treturn JsonNode.object_of({})")
	require.Contains(t, source, "var message_type_value: Variant = message_type")
	require.Contains(t, source, "if not (message_type_value is Type[JsonSerializable]):")
	require.Contains(t, source, `push_error("ANY_JSON_UNSUPPORTED:`)
	require.Contains(t, source, "var _json_message_type: Type[JsonSerializable] = message_type_value")
	require.Contains(t, source, "var message: Message = message_type.create_message()")
	require.Contains(t, source, "var merge_error: ProtobufError = message.merge_from_bytes(bytes)")
	require.Contains(t, source, "var message_value: Variant = message")
	require.Contains(t, source, "if not (message_value is JsonSerializable):")
	require.Contains(t, source, "var json_message: JsonSerializable = message_value")
	require.Contains(t, source, "var embedded: JsonNode = json_message.to_json()")
	require.Contains(t, source, `result["@type"] = JsonNode.Str(type_url)`)
	require.Contains(t, source, "for key: String in entries:")
	require.Contains(t, source, "result[key] = entries[key]")
	require.NotContains(t, source, `result["value"]`)

	capability := strings.Index(source, "if not (message_type_value is Type[JsonSerializable]):")
	decode := strings.Index(source, "message.merge_from_bytes(bytes)")
	require.NotEqual(t, -1, capability)
	require.NotEqual(t, -1, decode)
	require.Less(t, capability, decode, "JSON capability must be rejected before reading wire bytes")
}

func TestAnyTypeRegistryDecodesOrdinaryJSONTransactionallyAtRootPaths(t *testing.T) {
	source := runtime.Files()["foundry/proto/any_type_registry.fs"]

	require.Contains(t, source,
		"static func _any_from_json(node: JsonNode) -> (String, PackedByteArray, JsonDecodeError?):")
	require.Contains(t, source, "var no_error: JsonDecodeError? = null")
	require.Contains(t, source, "JsonNode.Null:\n\t\t\treturn (\"\", PackedByteArray(), no_error)")
	require.Contains(t, source, "if entries.is_empty():\n\t\t\t\treturn (\"\", PackedByteArray(), no_error)")
	require.Contains(t, source,
		`JsonDecodeError.create("JSON_TYPE_MISMATCH: google.protobuf.Any expects a JSON object", "$")`)
	require.Contains(t, source,
		`JsonDecodeError.create("JSON_PARSE_FAILED: google.protobuf.Any requires @type", "$[\"@type\"]")`)
	require.Contains(t, source,
		`JsonDecodeError.create("JSON_TYPE_MISMATCH: google.protobuf.Any @type expects a string", "$[\"@type\"]")`)
	require.Contains(t, source, "var message_type_value: Variant = message_type")
	require.Contains(t, source, "var json_message_type: Type[JsonSerializable] = message_type_value")
	require.Contains(t, source, "payload[key] = entries[key]")
	require.Contains(t, source,
		"var decoded: JsonResult[JsonSerializable] = json_message_type.from_json(JsonNode.object_of(payload))")
	require.Contains(t, source, "return (\"\", PackedByteArray(), decoded.error)")
	require.Contains(t, source, "var decoded_value: Variant = decoded.value")
	require.Contains(t, source, "if not (decoded_value is Message):")
	require.Contains(t, source, "var message: Message = decoded_value")
	require.Contains(t, source,
		"var packed: foundry.proto.wkt.Any = foundry.proto.wkt.Any.pack(message)")
	require.Contains(t, source, "packed.type_url = type_url")
	require.Contains(t, source, "return (packed.type_url, packed.value, no_error)")
	require.NotContains(t, source, `payload["value"]`)
	require.NotContains(t, source, `entries.has("value")`)
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
