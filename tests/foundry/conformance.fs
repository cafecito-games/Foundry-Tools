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
	suite.map_uint64_uint64 = {-1: -1}
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

	if failures > 0:
		printerr("conformance round trip failed with ", failures, " error(s)")
		quit(1)
		return
	print("conformance round trip ok")
	quit(0)
