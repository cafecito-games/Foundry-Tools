import foundry.proto
import foundry.proto.wkt
import protobuf_test_messages.proto3

extends SceneTree

var failures: int = 0

func check(condition: bool, label: String) -> void:
	if not condition:
		printerr("FAIL: ", label)
		failures += 1

func _init() -> void:
	var direct: TestAllTypesProto3 = TestAllTypesProto3.new()
	direct.optional_string = "direct child"

	var mutual: TestAllTypesProto3 = TestAllTypesProto3.new()
	mutual.optional_string = "mutual child"
	var nested: TestAllTypesProto3.NestedMessage = TestAllTypesProto3.NestedMessage.new()
	nested.a = 7
	nested.corecursive = mutual

	var suite: TestAllTypesProto3 = TestAllTypesProto3.new()
	suite.recursive_message = direct
	suite.optional_nested_message = nested
	suite.optional_nested_enum = TestAllTypesProto3.NestedEnum.NEG
	suite.optional_aliased_enum = TestAllTypesProto3.AliasedEnum.MOO
	suite.map_uint64_uint64 = {0xFFFFFFFFFFFFFFFFUL: 0xFFFFFFFFFFFFFFFFUL}
	suite.oneof_field = TestAllTypesProto3OneofFieldCase.OneofNullValue(NullValue.NULL_VALUE)

	var (decoded, decode_error) = TestAllTypesProto3.from_bytes(suite.to_bytes())
	check(decode_error == ProtobufError.OK, "upstream conformance message decodes")
	if not (decoded is TestAllTypesProto3):
		printerr("FAIL: upstream conformance message was null")
		quit(1)
		return

	check(decoded.recursive_message is TestAllTypesProto3,
		"a directly recursive message edge round trips")
	if decoded.recursive_message is TestAllTypesProto3:
		check(decoded.recursive_message.optional_string == "direct child",
			"the directly recursive child keeps its fields")

	check(decoded.optional_nested_message is TestAllTypesProto3.NestedMessage,
		"the nested message round trips")
	if decoded.optional_nested_message is TestAllTypesProto3.NestedMessage:
		check(decoded.optional_nested_message.a == 7, "the nested message keeps its fields")
		check(decoded.optional_nested_message.corecursive is TestAllTypesProto3,
			"a nested message can recurse to its parent type")
		if decoded.optional_nested_message.corecursive is TestAllTypesProto3:
			check(decoded.optional_nested_message.corecursive.optional_string == "mutual child",
				"the mutually recursive child keeps its fields")

	check(decoded.optional_nested_enum == TestAllTypesProto3.NestedEnum.NEG,
		"a negative enum value round trips")
	check(decoded.optional_aliased_enum == TestAllTypesProto3.AliasedEnum.MOO,
		"an aliased enum keeps its numeric value")
	check(decoded.map_uint64_uint64 == {-1: -1},
		"the widest uint64 map key and value round trip")

	match decoded.oneof_field:
		TestAllTypesProto3OneofFieldCase.OneofNullValue(var null_value):
			check(null_value == NullValue.NULL_VALUE,
				"a default-valued NullValue oneof remains present")
		_:
			printerr("FAIL: the NullValue oneof case did not round trip")
			failures += 1

	check(TestAllTypesProto3.protobuf_type_name() == "protobuf_test_messages.proto3.TestAllTypesProto3",
		"the upstream top-level message keeps its package-qualified identity")
	check(TestAllTypesProto3.NestedMessage.protobuf_type_name() ==
		"protobuf_test_messages.proto3.TestAllTypesProto3.NestedMessage",
		"the upstream nested message keeps its protobuf-qualified identity")
	var dynamic_top_level: Message = TestAllTypesProto3.create_message()
	var dynamic_nested: Message = TestAllTypesProto3.NestedMessage.create_message()
	check(dynamic_top_level is TestAllTypesProto3,
		"the upstream top-level factory keeps its concrete dynamic type")
	check(dynamic_nested is TestAllTypesProto3.NestedMessage,
		"the upstream nested factory keeps its concrete dynamic type")

	AnyTypeRegistry.clear()
	var suite_bytes: PackedByteArray = suite.to_bytes()
	var packed: Any = Any.pack(suite)
	check(packed.type_url ==
		"type.googleapis.com/protobuf_test_messages.proto3.TestAllTypesProto3",
		"the upstream message packs with its canonical type URL")
	check(packed.value == suite_bytes,
		"the upstream message packs its exact wire bytes")
	var (unregistered_message, unregistered_error) = packed.unpack()
	check(unregistered_message == null and
		unregistered_error == ProtobufError.ANY_TYPE_NOT_REGISTERED,
		"packing the upstream message does not register it implicitly")
	check(AnyTypeRegistry.register(TestAllTypesProto3) == ProtobufError.OK,
		"the upstream top-level message registers explicitly")
	check(AnyTypeRegistry.register(TestAllTypesProto3.NestedMessage) == ProtobufError.OK,
		"the upstream nested message registers explicitly")
	var (unpacked, unpack_error) = packed.unpack()
	check(unpack_error == ProtobufError.OK and unpacked is TestAllTypesProto3,
		"the registry dynamically unpacks the upstream message")
	if unpacked is TestAllTypesProto3:
		check(unpacked.to_bytes() == suite_bytes,
			"dynamic upstream unpack preserves exact wire bytes")
	var packed_nested: Any = Any.pack(nested)
	var (unpacked_nested, unpacked_nested_error) = packed_nested.unpack()
	check(unpacked_nested_error == ProtobufError.OK and
		unpacked_nested is TestAllTypesProto3.NestedMessage,
		"the registry dynamically unpacks the upstream nested message")
	check(packed.to_json() == JsonNode.Null,
		"a generated wire-only upstream message rejects Any JSON")
	var wire_only_json: JsonResult[Any] = Any.from_json(JsonNode.object_of({
		"@type": JsonNode.Str(packed.type_url),
	}))
	check(not wire_only_json.is_ok() and
		wire_only_json.error.path == '$["@type"]',
		"wire-only upstream Any JSON fails at @type")
	check(wire_only_json.error.message.find("ANY_JSON_UNSUPPORTED") >= 0,
		"wire-only upstream Any JSON reports ANY_JSON_UNSUPPORTED")
	AnyTypeRegistry.clear()
	var (_cleared_type, cleared_error) = AnyTypeRegistry._resolve(
		TestAllTypesProto3.protobuf_type_name())
	check(cleared_error == ProtobufError.ANY_TYPE_NOT_REGISTERED,
		"the upstream conformance case leaves the global registry isolated")

	if failures > 0:
		printerr("conformance round trip failed with ", failures, " error(s)")
		quit(1)
		return
	print("conformance round trip ok")
	quit(0)
