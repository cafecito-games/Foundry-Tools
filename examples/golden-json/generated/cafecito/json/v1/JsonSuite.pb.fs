namespace cafecito.json.v1
import foundry.proto
import foundry.proto.wkt

## Every field kind the canonical JSON mapping has a rule for, in one message.
## The mapping is not uniform across the scalars: a 64-bit integer is a JSON
## string, bytes are base64, an enum is its declared name, a float is a bare
## number except when it is not finite. Holding all of them here is what makes
## the generated JSON surface diffable in one place.
final class_name JsonSuite extends RefCounted uses Message, JsonSerializable

## Nested declarations keep proto's scoping: JsonSuite.Inner.
final class Inner extends RefCounted uses Message, JsonSerializable:
	## The code protobuf field.
	var code: String = ""

	## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
	var _pb_unknown_fields: PackedByteArray = PackedByteArray()

	static func create_message() -> Inner:
		return Inner.new()

	static func protobuf_type_name() -> String:
		return "cafecito.json.v1.JsonSuite.Inner"

	func type_name() -> String:
		return Inner.protobuf_type_name()

	## Decodes protobuf wire data into a new Inner message.
	static func from_bytes(_pb_data: PackedByteArray) -> (Inner?, ProtobufError):
		var _pb_message: Inner = Inner.new()
		var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
		if _pb_error != ProtobufError.OK:
			var _pb_failed: Inner? = null
			return (_pb_failed, _pb_error)
		return (_pb_message, ProtobufError.OK)

	## Serializes this message to protobuf wire data.
	func to_bytes() -> PackedByteArray:
		var _pb_result: PackedByteArray = PackedByteArray()
		if code != "":
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
			var _pb_code_data: PackedByteArray = Wire.encode_string(code)
			_pb_result.append_array(Wire.encode_varint(_pb_code_data.size()))
			_pb_result.append_array(_pb_code_data)
		_pb_result.append_array(_pb_unknown_fields)
		return _pb_result

	## Merges protobuf wire data into this message.
	func merge_from_bytes(_pb_data: PackedByteArray) -> ProtobufError:
		var _pb_offset: int = 0
		while _pb_offset < _pb_data.size():
			var _pb_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
			if _pb_tag.error != ProtobufError.OK:
				return _pb_tag.error
			_pb_offset = _pb_tag.offset
			var _pb_wire_type: int = Wire.get_wire_type(_pb_tag.value)
			match Wire.get_field_number(_pb_tag.value):
				1:
					if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
						return ProtobufError.WIRE_TYPE_MISMATCH
					var _pb_code_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
					if _pb_code_read.error != ProtobufError.OK:
						return _pb_code_read.error
					code = _pb_code_read.value
					_pb_offset = _pb_code_read.offset
				_:
					var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
					if _pb_skipped.error != ProtobufError.OK:
						return _pb_skipped.error
					_pb_offset = _pb_skipped.offset
		return ProtobufError.OK

	## Returns this message as a proto3 canonical JSON document.
	##
	## JSON.stringify(message, "", false) renders it as text; the third argument
	## turns off key sorting, which keeps members in field declaration order.
	func to_json() -> JsonNode:
		var _pb_json: Dictionary[String, JsonNode] = {}
		if code != "":
			_pb_json["code"] = JsonNode.Str(code)
		return JsonNode.object_of(_pb_json)

	## Decodes a proto3 canonical JSON document into a new Inner message.
	##
	## JSON.parse_to_node(text).value produces the document; a malformed one is
	## already reported through that JsonResult, so no text entry point is
	## generated here.
	static func from_json(_pb_node: JsonNode) -> JsonResult[Inner]:
		var _pb_message: Inner = Inner.new()
		var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
		if _pb_error is JsonDecodeError:
			return JsonResult[Inner].fail(_pb_error.message, _pb_error.path)
		return JsonResult[Inner].ok(_pb_message)

	## Merges a proto3 canonical JSON document into this message.
	##
	## A failure is returned rather than raised, matching the wire path, and
	## carries the JSONPath of the value that could not be read.
	func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
		var _pb_entries: Dictionary[String, JsonNode] = {}
		match _pb_node:
			JsonNode.Object(var _pb_object):
				_pb_entries = _pb_object
			JsonNode.Null:
				pass
			_:
				return JsonDecodeError.create("JSON_TYPE_MISMATCH: Inner expects a JSON object", "$")
		for _pb_key: String in _pb_entries:
			var _pb_member: JsonNode = _pb_entries[_pb_key]
			var _pb_member_path: String = "$." + _pb_key
			match _pb_key:
				"code":
					var (_pb_code_value, _pb_code_error) = _pb_json_read_string(_pb_member, _pb_member_path)
					if _pb_code_error is JsonDecodeError:
						return _pb_code_error
					code = _pb_code_value
				_:
					return JsonDecodeError.create("JSON_UNKNOWN_FIELD: Inner has no field named " + _pb_key, _pb_member_path)
		return null

	## Reads a string field out of a JSON value.
	static func _pb_json_read_string(_pb_node: JsonNode, _pb_path: String) -> (String, JsonDecodeError?):
		var _pb_value: String = ""
		match _pb_node:
			JsonNode.Null:
				pass
			JsonNode.Str(var _pb_text):
				_pb_value = _pb_text
			_:
				return ("", JsonDecodeError.create("JSON_TYPE_MISMATCH: a string field takes a JSON string", _pb_path))
		var _pb_error: JsonDecodeError? = null
		return (_pb_value, _pb_error)

## The double_value protobuf field.
var double_value: float = 0.0

## The float_value protobuf field.
var float_value: float = 0.0

## The int32_value protobuf field.
var int32_value: int = 0

## The int64_value protobuf field.
var int64_value: int = 0

## The uint32_value protobuf field.
var uint32_value: int = 0

## The uint64_value protobuf field.
var uint64_value: int = 0

## The sint32_value protobuf field.
var sint32_value: int = 0

## The sint64_value protobuf field.
var sint64_value: int = 0

## The fixed32_value protobuf field.
var fixed32_value: int = 0

## The fixed64_value protobuf field.
var fixed64_value: int = 0

## The sfixed32_value protobuf field.
var sfixed32_value: int = 0

## The sfixed64_value protobuf field.
var sfixed64_value: int = 0

## The bool_value protobuf field.
var bool_value: bool = false

## The string_value protobuf field.
var string_value: String = ""

## The bytes_value protobuf field.
var bytes_value: PackedByteArray = PackedByteArray()

## A camelCase JSON name is derived from a snake_case proto one, and both
## spellings are accepted on the way in.
var two_word_name: String = ""

## The flavor protobuf field.
var flavor: Flavor = Flavor.FLAVOR_UNSPECIFIED:
	set(_pb_value):
		_pb_flavor_unknown = PackedByteArray()
		flavor = _pb_value

## The primary protobuf field.
var primary: Reference? = null

## Explicit presence: an absent nickname is omitted, an empty one is written.
var nickname: String? = null

## The tags protobuf field.
var tags: Array[String] = []

## The tallies protobuf field.
var tallies: Array[int] = []

## The flavors protobuf field.
var flavors: Array[Flavor] = []

## The references protobuf field.
var references: Array[Reference] = []

## A JSON member name is always a string, so every map key kind is written as
## one, and the 64-bit key kinds go through the same text as the fields do.
var counts: Dictionary[String, int] = {}

## The int32_keyed protobuf field.
var int32_keyed: Dictionary[int, String] = {}

## The int64_keyed protobuf field.
var int64_keyed: Dictionary[int, String] = {}

## The uint64_keyed protobuf field.
var uint64_keyed: Dictionary[int, String] = {}

## The bool_keyed protobuf field.
var bool_keyed: Dictionary[bool, Reference] = {}

## The well-known types carry the special JSON form on their own bindings, so
## a reference to one recurses through the trait like any other message.
var occurred_at: Timestamp? = null

## The elapsed protobuf field.
var elapsed: Duration? = null

## The attributes protobuf field.
var attributes: Struct? = null

## The setting protobuf field.
var setting: Value? = null

## The label protobuf field.
var label: StringValue? = null

## The inner protobuf field.
var inner: Inner? = null

## The choice protobuf oneof; null when no case is set.
var choice: JsonSuiteChoiceCase? = null

## Raw bytes of an unrecognized flavor value, kept so a re-encode is lossless.
var _pb_flavor_unknown: PackedByteArray = PackedByteArray()

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> JsonSuite:
	return JsonSuite.new()

static func protobuf_type_name() -> String:
	return "cafecito.json.v1.JsonSuite"

func type_name() -> String:
	return JsonSuite.protobuf_type_name()

## Decodes protobuf wire data into a new JsonSuite message.
static func from_bytes(_pb_data: PackedByteArray) -> (JsonSuite?, ProtobufError):
	var _pb_message: JsonSuite = JsonSuite.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: JsonSuite? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if not Wire.is_default_float(double_value):
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_64BIT)))
		_pb_result.append_array(Wire.encode_double(double_value))
	if not Wire.is_default_float32(float_value):
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_32BIT)))
		_pb_result.append_array(Wire.encode_float(float_value))
	if int32_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(3, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(int32_value))
	if int64_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(4, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(int64_value))
	if uint32_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(5, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(uint32_value))
	if uint64_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(6, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(uint64_value))
	if sint32_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(7, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_sint32(sint32_value))
	if sint64_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_sint64(sint64_value))
	if fixed32_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(9, Wire.WIRE_32BIT)))
		_pb_result.append_array(Wire.encode_fixed32(fixed32_value))
	if fixed64_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(10, Wire.WIRE_64BIT)))
		_pb_result.append_array(Wire.encode_fixed64(fixed64_value))
	if sfixed32_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(11, Wire.WIRE_32BIT)))
		_pb_result.append_array(Wire.encode_fixed32(sfixed32_value))
	if sfixed64_value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(12, Wire.WIRE_64BIT)))
		_pb_result.append_array(Wire.encode_fixed64(sfixed64_value))
	if bool_value:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(13, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(1 if bool_value else 0))
	if string_value != "":
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(14, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_string_value_data: PackedByteArray = Wire.encode_string(string_value)
		_pb_result.append_array(Wire.encode_varint(_pb_string_value_data.size()))
		_pb_result.append_array(_pb_string_value_data)
	if bytes_value.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(15, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(bytes_value.size()))
		_pb_result.append_array(bytes_value)
	if two_word_name != "":
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(16, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_two_word_name_data: PackedByteArray = Wire.encode_string(two_word_name)
		_pb_result.append_array(Wire.encode_varint(_pb_two_word_name_data.size()))
		_pb_result.append_array(_pb_two_word_name_data)
	if _pb_flavor_unknown.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(17, Wire.WIRE_VARINT)))
		_pb_result.append_array(_pb_flavor_unknown)
	elif flavor != Flavor.FLAVOR_UNSPECIFIED:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(17, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(flavor.to_wire()))
	if primary is Reference:
		var _pb_primary_data: PackedByteArray = primary.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(18, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_primary_data.size()))
		_pb_result.append_array(_pb_primary_data)
	if nickname is String:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(19, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_nickname_data: PackedByteArray = Wire.encode_string(nickname)
		_pb_result.append_array(Wire.encode_varint(_pb_nickname_data.size()))
		_pb_result.append_array(_pb_nickname_data)
	for _pb_tags_item: String in tags:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(20, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_tags_data: PackedByteArray = Wire.encode_string(_pb_tags_item)
		_pb_result.append_array(Wire.encode_varint(_pb_tags_data.size()))
		_pb_result.append_array(_pb_tags_data)
	if tallies.size() > 0:
		var _pb_tallies_data: PackedByteArray = PackedByteArray()
		for _pb_tallies_item: int in tallies:
			_pb_tallies_data.append_array(Wire.encode_varint(_pb_tallies_item))
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(21, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_tallies_data.size()))
		_pb_result.append_array(_pb_tallies_data)
	if flavors.size() > 0:
		var _pb_flavors_data: PackedByteArray = PackedByteArray()
		for _pb_flavors_item: Flavor in flavors:
			_pb_flavors_data.append_array(Wire.encode_varint(_pb_flavors_item.to_wire()))
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(22, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_flavors_data.size()))
		_pb_result.append_array(_pb_flavors_data)
	for _pb_references_item: Reference in references:
		var _pb_references_data: PackedByteArray = _pb_references_item.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(23, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_references_data.size()))
		_pb_result.append_array(_pb_references_data)
	for _pb_counts_key: String in counts:
		var _pb_counts_entry: PackedByteArray = PackedByteArray()
		_pb_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_counts_key_data: PackedByteArray = Wire.encode_string(_pb_counts_key)
		_pb_counts_entry.append_array(Wire.encode_varint(_pb_counts_key_data.size()))
		_pb_counts_entry.append_array(_pb_counts_key_data)
		_pb_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_pb_counts_entry.append_array(Wire.encode_varint(counts[_pb_counts_key]))
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(24, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_counts_entry.size()))
		_pb_result.append_array(_pb_counts_entry)
	for _pb_int32_keyed_key: int in int32_keyed:
		var _pb_int32_keyed_entry: PackedByteArray = PackedByteArray()
		_pb_int32_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_int32_keyed_entry.append_array(Wire.encode_varint(_pb_int32_keyed_key))
		_pb_int32_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_int32_keyed_value_data: PackedByteArray = Wire.encode_string(int32_keyed[_pb_int32_keyed_key])
		_pb_int32_keyed_entry.append_array(Wire.encode_varint(_pb_int32_keyed_value_data.size()))
		_pb_int32_keyed_entry.append_array(_pb_int32_keyed_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(25, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_int32_keyed_entry.size()))
		_pb_result.append_array(_pb_int32_keyed_entry)
	for _pb_int64_keyed_key: int in int64_keyed:
		var _pb_int64_keyed_entry: PackedByteArray = PackedByteArray()
		_pb_int64_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_int64_keyed_entry.append_array(Wire.encode_varint(_pb_int64_keyed_key))
		_pb_int64_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_int64_keyed_value_data: PackedByteArray = Wire.encode_string(int64_keyed[_pb_int64_keyed_key])
		_pb_int64_keyed_entry.append_array(Wire.encode_varint(_pb_int64_keyed_value_data.size()))
		_pb_int64_keyed_entry.append_array(_pb_int64_keyed_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(26, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_int64_keyed_entry.size()))
		_pb_result.append_array(_pb_int64_keyed_entry)
	for _pb_uint64_keyed_key: int in uint64_keyed:
		var _pb_uint64_keyed_entry: PackedByteArray = PackedByteArray()
		_pb_uint64_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_uint64_keyed_entry.append_array(Wire.encode_varint(_pb_uint64_keyed_key))
		_pb_uint64_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_uint64_keyed_value_data: PackedByteArray = Wire.encode_string(uint64_keyed[_pb_uint64_keyed_key])
		_pb_uint64_keyed_entry.append_array(Wire.encode_varint(_pb_uint64_keyed_value_data.size()))
		_pb_uint64_keyed_entry.append_array(_pb_uint64_keyed_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(27, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_uint64_keyed_entry.size()))
		_pb_result.append_array(_pb_uint64_keyed_entry)
	for _pb_bool_keyed_key: bool in bool_keyed:
		var _pb_bool_keyed_entry: PackedByteArray = PackedByteArray()
		_pb_bool_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_bool_keyed_entry.append_array(Wire.encode_varint(1 if _pb_bool_keyed_key else 0))
		var _pb_bool_keyed_value_data: PackedByteArray = bool_keyed[_pb_bool_keyed_key].to_bytes()
		_pb_bool_keyed_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_bool_keyed_entry.append_array(Wire.encode_varint(_pb_bool_keyed_value_data.size()))
		_pb_bool_keyed_entry.append_array(_pb_bool_keyed_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(28, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_bool_keyed_entry.size()))
		_pb_result.append_array(_pb_bool_keyed_entry)
	match choice:
		JsonSuiteChoiceCase.Note(var _pb_choice_note):
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(29, Wire.WIRE_LENGTH_DELIMITED)))
			var _pb_choice_note_data: PackedByteArray = Wire.encode_string(_pb_choice_note)
			_pb_result.append_array(Wire.encode_varint(_pb_choice_note_data.size()))
			_pb_result.append_array(_pb_choice_note_data)
		JsonSuiteChoiceCase.Tally(var _pb_choice_tally):
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(30, Wire.WIRE_VARINT)))
			_pb_result.append_array(Wire.encode_varint(_pb_choice_tally))
		JsonSuiteChoiceCase.Detail(var _pb_choice_detail):
			var _pb_choice_detail_data: PackedByteArray = _pb_choice_detail.to_bytes()
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(31, Wire.WIRE_LENGTH_DELIMITED)))
			_pb_result.append_array(Wire.encode_varint(_pb_choice_detail_data.size()))
			_pb_result.append_array(_pb_choice_detail_data)
		_:
			pass
	if occurred_at is Timestamp:
		var _pb_occurred_at_data: PackedByteArray = occurred_at.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(32, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_occurred_at_data.size()))
		_pb_result.append_array(_pb_occurred_at_data)
	if elapsed is Duration:
		var _pb_elapsed_data: PackedByteArray = elapsed.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(33, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_elapsed_data.size()))
		_pb_result.append_array(_pb_elapsed_data)
	if attributes is Struct:
		var _pb_attributes_data: PackedByteArray = attributes.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(34, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_attributes_data.size()))
		_pb_result.append_array(_pb_attributes_data)
	if setting is Value:
		var _pb_setting_data: PackedByteArray = setting.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(35, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_setting_data.size()))
		_pb_result.append_array(_pb_setting_data)
	if label is StringValue:
		var _pb_label_data: PackedByteArray = label.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(36, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_label_data.size()))
		_pb_result.append_array(_pb_label_data)
	if inner is Inner:
		var _pb_inner_data: PackedByteArray = inner.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(37, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_inner_data.size()))
		_pb_result.append_array(_pb_inner_data)
	_pb_result.append_array(_pb_unknown_fields)
	return _pb_result

## Merges protobuf wire data into this message.
func merge_from_bytes(_pb_data: PackedByteArray) -> ProtobufError:
	var _pb_offset: int = 0
	while _pb_offset < _pb_data.size():
		var _pb_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
		if _pb_tag.error != ProtobufError.OK:
			return _pb_tag.error
		_pb_offset = _pb_tag.offset
		var _pb_wire_type: int = Wire.get_wire_type(_pb_tag.value)
		match Wire.get_field_number(_pb_tag.value):
			1:
				if _pb_wire_type != Wire.WIRE_64BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_double_value_read: FloatRead = Wire.read_double(_pb_data, _pb_offset)
				if _pb_double_value_read.error != ProtobufError.OK:
					return _pb_double_value_read.error
				double_value = _pb_double_value_read.value
				_pb_offset = _pb_double_value_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_32BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_float_value_read: FloatRead = Wire.read_float(_pb_data, _pb_offset)
				if _pb_float_value_read.error != ProtobufError.OK:
					return _pb_float_value_read.error
				float_value = _pb_float_value_read.value
				_pb_offset = _pb_float_value_read.offset
			3:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_int32_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_int32_value_read.error != ProtobufError.OK:
					return _pb_int32_value_read.error
				int32_value = _pb_int32_value_read.value
				_pb_offset = _pb_int32_value_read.offset
			4:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_int64_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_int64_value_read.error != ProtobufError.OK:
					return _pb_int64_value_read.error
				int64_value = _pb_int64_value_read.value
				_pb_offset = _pb_int64_value_read.offset
			5:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_uint32_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_uint32_value_read.error != ProtobufError.OK:
					return _pb_uint32_value_read.error
				uint32_value = _pb_uint32_value_read.value
				_pb_offset = _pb_uint32_value_read.offset
			6:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_uint64_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_uint64_value_read.error != ProtobufError.OK:
					return _pb_uint64_value_read.error
				uint64_value = _pb_uint64_value_read.value
				_pb_offset = _pb_uint64_value_read.offset
			7:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_sint32_value_read: VarintRead = Wire.read_sint32(_pb_data, _pb_offset)
				if _pb_sint32_value_read.error != ProtobufError.OK:
					return _pb_sint32_value_read.error
				sint32_value = _pb_sint32_value_read.value
				_pb_offset = _pb_sint32_value_read.offset
			8:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_sint64_value_read: VarintRead = Wire.read_sint64(_pb_data, _pb_offset)
				if _pb_sint64_value_read.error != ProtobufError.OK:
					return _pb_sint64_value_read.error
				sint64_value = _pb_sint64_value_read.value
				_pb_offset = _pb_sint64_value_read.offset
			9:
				if _pb_wire_type != Wire.WIRE_32BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_fixed32_value_read: FixedRead = Wire.read_fixed32(_pb_data, _pb_offset)
				if _pb_fixed32_value_read.error != ProtobufError.OK:
					return _pb_fixed32_value_read.error
				fixed32_value = _pb_fixed32_value_read.value
				_pb_offset = _pb_fixed32_value_read.offset
			10:
				if _pb_wire_type != Wire.WIRE_64BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_fixed64_value_read: FixedRead = Wire.read_fixed64(_pb_data, _pb_offset)
				if _pb_fixed64_value_read.error != ProtobufError.OK:
					return _pb_fixed64_value_read.error
				fixed64_value = _pb_fixed64_value_read.value
				_pb_offset = _pb_fixed64_value_read.offset
			11:
				if _pb_wire_type != Wire.WIRE_32BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_sfixed32_value_read: FixedRead = Wire.read_sfixed32(_pb_data, _pb_offset)
				if _pb_sfixed32_value_read.error != ProtobufError.OK:
					return _pb_sfixed32_value_read.error
				sfixed32_value = _pb_sfixed32_value_read.value
				_pb_offset = _pb_sfixed32_value_read.offset
			12:
				if _pb_wire_type != Wire.WIRE_64BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_sfixed64_value_read: FixedRead = Wire.read_fixed64(_pb_data, _pb_offset)
				if _pb_sfixed64_value_read.error != ProtobufError.OK:
					return _pb_sfixed64_value_read.error
				sfixed64_value = _pb_sfixed64_value_read.value
				_pb_offset = _pb_sfixed64_value_read.offset
			13:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_bool_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_bool_value_read.error != ProtobufError.OK:
					return _pb_bool_value_read.error
				bool_value = _pb_bool_value_read.value != 0
				_pb_offset = _pb_bool_value_read.offset
			14:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_string_value_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_string_value_read.error != ProtobufError.OK:
					return _pb_string_value_read.error
				string_value = _pb_string_value_read.value
				_pb_offset = _pb_string_value_read.offset
			15:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_bytes_value_read: BytesRead = Wire.read_bytes(_pb_data, _pb_offset)
				if _pb_bytes_value_read.error != ProtobufError.OK:
					return _pb_bytes_value_read.error
				bytes_value = _pb_bytes_value_read.value
				_pb_offset = _pb_bytes_value_read.offset
			16:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_two_word_name_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_two_word_name_read.error != ProtobufError.OK:
					return _pb_two_word_name_read.error
				two_word_name = _pb_two_word_name_read.value
				_pb_offset = _pb_two_word_name_read.offset
			17:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_flavor_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_flavor_read.error != ProtobufError.OK:
					return _pb_flavor_read.error
				var _pb_flavor_case: Flavor? = Flavor.from_wire(_pb_flavor_read.value)
				if _pb_flavor_case is Flavor:
					flavor = _pb_flavor_case
				else:
					flavor = Flavor.FLAVOR_UNSPECIFIED
					_pb_flavor_unknown = _pb_data.slice(_pb_offset, _pb_flavor_read.offset)
				_pb_offset = _pb_flavor_read.offset
			18:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (primary is Reference):
					primary = Reference.new()
				var _pb_primary_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, primary)
				if _pb_primary_read.error != ProtobufError.OK:
					return _pb_primary_read.error
				_pb_offset = _pb_primary_read.offset
			19:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_nickname_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_nickname_read.error != ProtobufError.OK:
					return _pb_nickname_read.error
				nickname = _pb_nickname_read.value
				_pb_offset = _pb_nickname_read.offset
			20:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_tags_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_tags_read.error != ProtobufError.OK:
					return _pb_tags_read.error
				tags.append(_pb_tags_read.value)
				_pb_offset = _pb_tags_read.offset
			21:
				if _pb_wire_type == Wire.WIRE_LENGTH_DELIMITED:
					var _pb_tallies_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
					if _pb_tallies_length.error != ProtobufError.OK:
						return _pb_tallies_length.error
					var _pb_tallies_end: int = _pb_tallies_length.offset + _pb_tallies_length.value
					_pb_offset = _pb_tallies_length.offset
					while _pb_offset < _pb_tallies_end:
						var _pb_tallies_packed: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
						if _pb_tallies_packed.error != ProtobufError.OK:
							return _pb_tallies_packed.error
						if _pb_tallies_packed.offset > _pb_tallies_end:
							return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
						tallies.append(_pb_tallies_packed.value)
						_pb_offset = _pb_tallies_packed.offset
				elif _pb_wire_type == Wire.WIRE_VARINT:
					var _pb_tallies_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_tallies_read.error != ProtobufError.OK:
						return _pb_tallies_read.error
					tallies.append(_pb_tallies_read.value)
					_pb_offset = _pb_tallies_read.offset
				else:
					return ProtobufError.WIRE_TYPE_MISMATCH
			22:
				if _pb_wire_type == Wire.WIRE_LENGTH_DELIMITED:
					var _pb_flavors_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
					if _pb_flavors_length.error != ProtobufError.OK:
						return _pb_flavors_length.error
					var _pb_flavors_end: int = _pb_flavors_length.offset + _pb_flavors_length.value
					_pb_offset = _pb_flavors_length.offset
					while _pb_offset < _pb_flavors_end:
						var _pb_flavors_packed: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
						if _pb_flavors_packed.error != ProtobufError.OK:
							return _pb_flavors_packed.error
						if _pb_flavors_packed.offset > _pb_flavors_end:
							return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
						var _pb_flavors_packed_case: Flavor? = Flavor.from_wire(_pb_flavors_packed.value)
						if _pb_flavors_packed_case is Flavor:
							flavors.append(_pb_flavors_packed_case)
						else:
							flavors.append(Flavor.FLAVOR_UNSPECIFIED)
						_pb_offset = _pb_flavors_packed.offset
				elif _pb_wire_type == Wire.WIRE_VARINT:
					var _pb_flavors_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_flavors_read.error != ProtobufError.OK:
						return _pb_flavors_read.error
					var _pb_flavors_case: Flavor? = Flavor.from_wire(_pb_flavors_read.value)
					if _pb_flavors_case is Flavor:
						flavors.append(_pb_flavors_case)
					else:
						flavors.append(Flavor.FLAVOR_UNSPECIFIED)
					_pb_offset = _pb_flavors_read.offset
				else:
					return ProtobufError.WIRE_TYPE_MISMATCH
			23:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_references_message: Reference = Reference.new()
				var _pb_references_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_references_message)
				if _pb_references_read.error != ProtobufError.OK:
					return _pb_references_read.error
				references.append(_pb_references_message)
				_pb_offset = _pb_references_read.offset
			24:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_counts_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_counts_length.error != ProtobufError.OK:
					return _pb_counts_length.error
				var _pb_counts_end: int = _pb_counts_length.offset + _pb_counts_length.value
				_pb_offset = _pb_counts_length.offset
				var _pb_counts_key: String = ""
				var _pb_counts_value: int = 0
				while _pb_offset < _pb_counts_end:
					var _pb_counts_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_counts_entry_tag.error != ProtobufError.OK:
						return _pb_counts_entry_tag.error
					if _pb_counts_entry_tag.offset > _pb_counts_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_counts_entry_tag.offset
					var _pb_counts_entry_wire_type: int = Wire.get_wire_type(_pb_counts_entry_tag.value)
					match Wire.get_field_number(_pb_counts_entry_tag.value):
						1:
							if _pb_counts_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_counts_key_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_counts_key_read.error != ProtobufError.OK:
								return _pb_counts_key_read.error
							if _pb_counts_key_read.offset > _pb_counts_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_counts_key = _pb_counts_key_read.value
							_pb_offset = _pb_counts_key_read.offset
						2:
							if _pb_counts_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_counts_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
							if _pb_counts_value_read.error != ProtobufError.OK:
								return _pb_counts_value_read.error
							if _pb_counts_value_read.offset > _pb_counts_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_counts_value = _pb_counts_value_read.value
							_pb_offset = _pb_counts_value_read.offset
						_:
							var _pb_counts_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_counts_entry_wire_type)
							if _pb_counts_skip.error != ProtobufError.OK:
								return _pb_counts_skip.error
							if _pb_counts_skip.offset > _pb_counts_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_counts_skip.offset
				counts[_pb_counts_key] = _pb_counts_value
			25:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_int32_keyed_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_int32_keyed_length.error != ProtobufError.OK:
					return _pb_int32_keyed_length.error
				var _pb_int32_keyed_end: int = _pb_int32_keyed_length.offset + _pb_int32_keyed_length.value
				_pb_offset = _pb_int32_keyed_length.offset
				var _pb_int32_keyed_key: int = 0
				var _pb_int32_keyed_value: String = ""
				while _pb_offset < _pb_int32_keyed_end:
					var _pb_int32_keyed_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_int32_keyed_entry_tag.error != ProtobufError.OK:
						return _pb_int32_keyed_entry_tag.error
					if _pb_int32_keyed_entry_tag.offset > _pb_int32_keyed_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_int32_keyed_entry_tag.offset
					var _pb_int32_keyed_entry_wire_type: int = Wire.get_wire_type(_pb_int32_keyed_entry_tag.value)
					match Wire.get_field_number(_pb_int32_keyed_entry_tag.value):
						1:
							if _pb_int32_keyed_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_int32_keyed_key_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
							if _pb_int32_keyed_key_read.error != ProtobufError.OK:
								return _pb_int32_keyed_key_read.error
							if _pb_int32_keyed_key_read.offset > _pb_int32_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_int32_keyed_key = _pb_int32_keyed_key_read.value
							_pb_offset = _pb_int32_keyed_key_read.offset
						2:
							if _pb_int32_keyed_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_int32_keyed_value_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_int32_keyed_value_read.error != ProtobufError.OK:
								return _pb_int32_keyed_value_read.error
							if _pb_int32_keyed_value_read.offset > _pb_int32_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_int32_keyed_value = _pb_int32_keyed_value_read.value
							_pb_offset = _pb_int32_keyed_value_read.offset
						_:
							var _pb_int32_keyed_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_int32_keyed_entry_wire_type)
							if _pb_int32_keyed_skip.error != ProtobufError.OK:
								return _pb_int32_keyed_skip.error
							if _pb_int32_keyed_skip.offset > _pb_int32_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_int32_keyed_skip.offset
				int32_keyed[_pb_int32_keyed_key] = _pb_int32_keyed_value
			26:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_int64_keyed_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_int64_keyed_length.error != ProtobufError.OK:
					return _pb_int64_keyed_length.error
				var _pb_int64_keyed_end: int = _pb_int64_keyed_length.offset + _pb_int64_keyed_length.value
				_pb_offset = _pb_int64_keyed_length.offset
				var _pb_int64_keyed_key: int = 0
				var _pb_int64_keyed_value: String = ""
				while _pb_offset < _pb_int64_keyed_end:
					var _pb_int64_keyed_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_int64_keyed_entry_tag.error != ProtobufError.OK:
						return _pb_int64_keyed_entry_tag.error
					if _pb_int64_keyed_entry_tag.offset > _pb_int64_keyed_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_int64_keyed_entry_tag.offset
					var _pb_int64_keyed_entry_wire_type: int = Wire.get_wire_type(_pb_int64_keyed_entry_tag.value)
					match Wire.get_field_number(_pb_int64_keyed_entry_tag.value):
						1:
							if _pb_int64_keyed_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_int64_keyed_key_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
							if _pb_int64_keyed_key_read.error != ProtobufError.OK:
								return _pb_int64_keyed_key_read.error
							if _pb_int64_keyed_key_read.offset > _pb_int64_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_int64_keyed_key = _pb_int64_keyed_key_read.value
							_pb_offset = _pb_int64_keyed_key_read.offset
						2:
							if _pb_int64_keyed_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_int64_keyed_value_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_int64_keyed_value_read.error != ProtobufError.OK:
								return _pb_int64_keyed_value_read.error
							if _pb_int64_keyed_value_read.offset > _pb_int64_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_int64_keyed_value = _pb_int64_keyed_value_read.value
							_pb_offset = _pb_int64_keyed_value_read.offset
						_:
							var _pb_int64_keyed_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_int64_keyed_entry_wire_type)
							if _pb_int64_keyed_skip.error != ProtobufError.OK:
								return _pb_int64_keyed_skip.error
							if _pb_int64_keyed_skip.offset > _pb_int64_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_int64_keyed_skip.offset
				int64_keyed[_pb_int64_keyed_key] = _pb_int64_keyed_value
			27:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_uint64_keyed_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_uint64_keyed_length.error != ProtobufError.OK:
					return _pb_uint64_keyed_length.error
				var _pb_uint64_keyed_end: int = _pb_uint64_keyed_length.offset + _pb_uint64_keyed_length.value
				_pb_offset = _pb_uint64_keyed_length.offset
				var _pb_uint64_keyed_key: int = 0
				var _pb_uint64_keyed_value: String = ""
				while _pb_offset < _pb_uint64_keyed_end:
					var _pb_uint64_keyed_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_uint64_keyed_entry_tag.error != ProtobufError.OK:
						return _pb_uint64_keyed_entry_tag.error
					if _pb_uint64_keyed_entry_tag.offset > _pb_uint64_keyed_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_uint64_keyed_entry_tag.offset
					var _pb_uint64_keyed_entry_wire_type: int = Wire.get_wire_type(_pb_uint64_keyed_entry_tag.value)
					match Wire.get_field_number(_pb_uint64_keyed_entry_tag.value):
						1:
							if _pb_uint64_keyed_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_uint64_keyed_key_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
							if _pb_uint64_keyed_key_read.error != ProtobufError.OK:
								return _pb_uint64_keyed_key_read.error
							if _pb_uint64_keyed_key_read.offset > _pb_uint64_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_uint64_keyed_key = _pb_uint64_keyed_key_read.value
							_pb_offset = _pb_uint64_keyed_key_read.offset
						2:
							if _pb_uint64_keyed_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_uint64_keyed_value_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_uint64_keyed_value_read.error != ProtobufError.OK:
								return _pb_uint64_keyed_value_read.error
							if _pb_uint64_keyed_value_read.offset > _pb_uint64_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_uint64_keyed_value = _pb_uint64_keyed_value_read.value
							_pb_offset = _pb_uint64_keyed_value_read.offset
						_:
							var _pb_uint64_keyed_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_uint64_keyed_entry_wire_type)
							if _pb_uint64_keyed_skip.error != ProtobufError.OK:
								return _pb_uint64_keyed_skip.error
							if _pb_uint64_keyed_skip.offset > _pb_uint64_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_uint64_keyed_skip.offset
				uint64_keyed[_pb_uint64_keyed_key] = _pb_uint64_keyed_value
			28:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_bool_keyed_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_bool_keyed_length.error != ProtobufError.OK:
					return _pb_bool_keyed_length.error
				var _pb_bool_keyed_end: int = _pb_bool_keyed_length.offset + _pb_bool_keyed_length.value
				_pb_offset = _pb_bool_keyed_length.offset
				var _pb_bool_keyed_key: bool = false
				var _pb_bool_keyed_value: Reference = Reference.new()
				while _pb_offset < _pb_bool_keyed_end:
					var _pb_bool_keyed_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_bool_keyed_entry_tag.error != ProtobufError.OK:
						return _pb_bool_keyed_entry_tag.error
					if _pb_bool_keyed_entry_tag.offset > _pb_bool_keyed_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_bool_keyed_entry_tag.offset
					var _pb_bool_keyed_entry_wire_type: int = Wire.get_wire_type(_pb_bool_keyed_entry_tag.value)
					match Wire.get_field_number(_pb_bool_keyed_entry_tag.value):
						1:
							if _pb_bool_keyed_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_bool_keyed_key_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
							if _pb_bool_keyed_key_read.error != ProtobufError.OK:
								return _pb_bool_keyed_key_read.error
							if _pb_bool_keyed_key_read.offset > _pb_bool_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_bool_keyed_key = _pb_bool_keyed_key_read.value != 0
							_pb_offset = _pb_bool_keyed_key_read.offset
						2:
							if _pb_bool_keyed_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_bool_keyed_value_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_bool_keyed_value)
							if _pb_bool_keyed_value_read.error != ProtobufError.OK:
								return _pb_bool_keyed_value_read.error
							if _pb_bool_keyed_value_read.offset > _pb_bool_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_bool_keyed_value_read.offset
						_:
							var _pb_bool_keyed_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_bool_keyed_entry_wire_type)
							if _pb_bool_keyed_skip.error != ProtobufError.OK:
								return _pb_bool_keyed_skip.error
							if _pb_bool_keyed_skip.offset > _pb_bool_keyed_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_bool_keyed_skip.offset
				bool_keyed[_pb_bool_keyed_key] = _pb_bool_keyed_value
			29:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_choice_note_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_choice_note_read.error != ProtobufError.OK:
					return _pb_choice_note_read.error
				choice = JsonSuiteChoiceCase.Note(_pb_choice_note_read.value)
				_pb_offset = _pb_choice_note_read.offset
			30:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_choice_tally_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_choice_tally_read.error != ProtobufError.OK:
					return _pb_choice_tally_read.error
				choice = JsonSuiteChoiceCase.Tally(_pb_choice_tally_read.value)
				_pb_offset = _pb_choice_tally_read.offset
			31:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_choice_detail_message: Reference = Reference.new()
				var _pb_choice_detail_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_choice_detail_message)
				if _pb_choice_detail_read.error != ProtobufError.OK:
					return _pb_choice_detail_read.error
				choice = JsonSuiteChoiceCase.Detail(_pb_choice_detail_message)
				_pb_offset = _pb_choice_detail_read.offset
			32:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (occurred_at is Timestamp):
					occurred_at = Timestamp.new()
				var _pb_occurred_at_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, occurred_at)
				if _pb_occurred_at_read.error != ProtobufError.OK:
					return _pb_occurred_at_read.error
				_pb_offset = _pb_occurred_at_read.offset
			33:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (elapsed is Duration):
					elapsed = Duration.new()
				var _pb_elapsed_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, elapsed)
				if _pb_elapsed_read.error != ProtobufError.OK:
					return _pb_elapsed_read.error
				_pb_offset = _pb_elapsed_read.offset
			34:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (attributes is Struct):
					attributes = Struct.new()
				var _pb_attributes_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, attributes)
				if _pb_attributes_read.error != ProtobufError.OK:
					return _pb_attributes_read.error
				_pb_offset = _pb_attributes_read.offset
			35:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (setting is Value):
					setting = Value.new()
				var _pb_setting_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, setting)
				if _pb_setting_read.error != ProtobufError.OK:
					return _pb_setting_read.error
				_pb_offset = _pb_setting_read.offset
			36:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (label is StringValue):
					label = StringValue.new()
				var _pb_label_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, label)
				if _pb_label_read.error != ProtobufError.OK:
					return _pb_label_read.error
				_pb_offset = _pb_label_read.offset
			37:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (inner is Inner):
					inner = Inner.new()
				var _pb_inner_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, inner)
				if _pb_inner_read.error != ProtobufError.OK:
					return _pb_inner_read.error
				_pb_offset = _pb_inner_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK

## Returns this message as a proto3 canonical JSON document.
##
## JSON.stringify(message, "", false) renders it as text; the third argument
## turns off key sorting, which keeps members in field declaration order.
func to_json() -> JsonNode:
	var _pb_json: Dictionary[String, JsonNode] = {}
	if not Wire.is_default_float(double_value):
		_pb_json["doubleValue"] = _pb_json_float(double_value)
	if not Wire.is_default_float32(float_value):
		_pb_json["floatValue"] = _pb_json_float(Wire.narrow_float32(float_value))
	if int32_value != 0:
		_pb_json["int32Value"] = JsonNode.Int(int32_value)
	if int64_value != 0:
		_pb_json["int64Value"] = JsonNode.Str(str(int64_value))
	if uint32_value != 0:
		_pb_json["uint32Value"] = JsonNode.Int(uint32_value)
	if uint64_value != 0:
		_pb_json["uint64Value"] = JsonNode.Str(JsonUint64.format(uint64_value))
	if sint32_value != 0:
		_pb_json["sint32Value"] = JsonNode.Int(sint32_value)
	if sint64_value != 0:
		_pb_json["sint64Value"] = JsonNode.Str(str(sint64_value))
	if fixed32_value != 0:
		_pb_json["fixed32Value"] = JsonNode.Int(fixed32_value)
	if fixed64_value != 0:
		_pb_json["fixed64Value"] = JsonNode.Str(JsonUint64.format(fixed64_value))
	if sfixed32_value != 0:
		_pb_json["sfixed32Value"] = JsonNode.Int(sfixed32_value)
	if sfixed64_value != 0:
		_pb_json["sfixed64Value"] = JsonNode.Str(str(sfixed64_value))
	if bool_value:
		_pb_json["boolValue"] = JsonNode.Bool(bool_value)
	if string_value != "":
		_pb_json["stringValue"] = JsonNode.Str(string_value)
	if bytes_value.size() > 0:
		_pb_json["bytesValue"] = JsonNode.Str(JsonBase64.encode(bytes_value))
	if two_word_name != "":
		_pb_json["twoWordName"] = JsonNode.Str(two_word_name)
	if flavor != Flavor.FLAVOR_UNSPECIFIED:
		_pb_json["flavor"] = JsonNode.Str(flavor.to_json_name())
	if primary is Reference:
		_pb_json["primary"] = primary.to_json()
	if nickname is String:
		_pb_json["nickname"] = JsonNode.Str(nickname)
	if tags.size() > 0:
		var _pb_tags_items: Array[JsonNode] = []
		for _pb_tags_item: String in tags:
			_pb_tags_items.append(JsonNode.Str(_pb_tags_item))
		_pb_json["tags"] = JsonNode.array_of(_pb_tags_items)
	if tallies.size() > 0:
		var _pb_tallies_items: Array[JsonNode] = []
		for _pb_tallies_item: int in tallies:
			_pb_tallies_items.append(JsonNode.Str(str(_pb_tallies_item)))
		_pb_json["tallies"] = JsonNode.array_of(_pb_tallies_items)
	if flavors.size() > 0:
		var _pb_flavors_items: Array[JsonNode] = []
		for _pb_flavors_item: Flavor in flavors:
			_pb_flavors_items.append(JsonNode.Str(_pb_flavors_item.to_json_name()))
		_pb_json["flavors"] = JsonNode.array_of(_pb_flavors_items)
	if references.size() > 0:
		var _pb_references_items: Array[JsonNode] = []
		for _pb_references_item: Reference in references:
			_pb_references_items.append(_pb_references_item.to_json())
		_pb_json["references"] = JsonNode.array_of(_pb_references_items)
	if counts.size() > 0:
		var _pb_counts_fields: Dictionary[String, JsonNode] = {}
		for _pb_counts_key: String in counts:
			_pb_counts_fields[_pb_counts_key] = JsonNode.Int(counts[_pb_counts_key])
		_pb_json["counts"] = JsonNode.object_of(_pb_counts_fields)
	if int32_keyed.size() > 0:
		var _pb_int32_keyed_fields: Dictionary[String, JsonNode] = {}
		for _pb_int32_keyed_key: int in int32_keyed:
			_pb_int32_keyed_fields[str(_pb_int32_keyed_key)] = JsonNode.Str(int32_keyed[_pb_int32_keyed_key])
		_pb_json["int32Keyed"] = JsonNode.object_of(_pb_int32_keyed_fields)
	if int64_keyed.size() > 0:
		var _pb_int64_keyed_fields: Dictionary[String, JsonNode] = {}
		for _pb_int64_keyed_key: int in int64_keyed:
			_pb_int64_keyed_fields[str(_pb_int64_keyed_key)] = JsonNode.Str(int64_keyed[_pb_int64_keyed_key])
		_pb_json["int64Keyed"] = JsonNode.object_of(_pb_int64_keyed_fields)
	if uint64_keyed.size() > 0:
		var _pb_uint64_keyed_fields: Dictionary[String, JsonNode] = {}
		for _pb_uint64_keyed_key: int in uint64_keyed:
			_pb_uint64_keyed_fields[JsonUint64.format(_pb_uint64_keyed_key)] = JsonNode.Str(uint64_keyed[_pb_uint64_keyed_key])
		_pb_json["uint64Keyed"] = JsonNode.object_of(_pb_uint64_keyed_fields)
	if bool_keyed.size() > 0:
		var _pb_bool_keyed_fields: Dictionary[String, JsonNode] = {}
		for _pb_bool_keyed_key: bool in bool_keyed:
			_pb_bool_keyed_fields["true" if _pb_bool_keyed_key else "false"] = bool_keyed[_pb_bool_keyed_key].to_json()
		_pb_json["boolKeyed"] = JsonNode.object_of(_pb_bool_keyed_fields)
	match choice:
		JsonSuiteChoiceCase.Note(var _pb_choice_note):
			_pb_json["note"] = JsonNode.Str(_pb_choice_note)
		JsonSuiteChoiceCase.Tally(var _pb_choice_tally):
			_pb_json["tally"] = JsonNode.Str(str(_pb_choice_tally))
		JsonSuiteChoiceCase.Detail(var _pb_choice_detail):
			_pb_json["detail"] = _pb_choice_detail.to_json()
		_:
			pass
	if occurred_at is Timestamp:
		_pb_json["occurredAt"] = occurred_at.to_json()
	if elapsed is Duration:
		_pb_json["elapsed"] = elapsed.to_json()
	if attributes is Struct:
		_pb_json["attributes"] = attributes.to_json()
	if setting is Value:
		_pb_json["setting"] = setting.to_json()
	if label is StringValue:
		_pb_json["label"] = label.to_json()
	if inner is Inner:
		_pb_json["inner"] = inner.to_json()
	return JsonNode.object_of(_pb_json)

## Returns one float as canonical proto3 JSON.
##
## A non-finite value never reaches the Float case: the encoder writes NaN as
## null and the infinities as ±1e99999, none of which is canonical, so the
## three specified string forms are produced here instead.
static func _pb_json_float(_pb_value: float) -> JsonNode:
	if is_nan(_pb_value):
		return JsonNode.Str("NaN")
	if is_inf(_pb_value):
		if _pb_value > 0.0:
			return JsonNode.Str("Infinity")
		return JsonNode.Str("-Infinity")
	return JsonNode.Float(_pb_value)

## Decodes a proto3 canonical JSON document into a new JsonSuite message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[JsonSuite]:
	var _pb_message: JsonSuite = JsonSuite.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[JsonSuite].fail(_pb_error.message, _pb_error.path)
	return JsonResult[JsonSuite].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	var _pb_entries: Dictionary[String, JsonNode] = {}
	match _pb_node:
		JsonNode.Object(var _pb_object):
			_pb_entries = _pb_object
		JsonNode.Null:
			pass
		_:
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: JsonSuite expects a JSON object", "$")
	var _pb_double_value_seen: bool = false
	var _pb_float_value_seen: bool = false
	var _pb_int32_value_seen: bool = false
	var _pb_int64_value_seen: bool = false
	var _pb_uint32_value_seen: bool = false
	var _pb_uint64_value_seen: bool = false
	var _pb_sint32_value_seen: bool = false
	var _pb_sint64_value_seen: bool = false
	var _pb_fixed32_value_seen: bool = false
	var _pb_fixed64_value_seen: bool = false
	var _pb_sfixed32_value_seen: bool = false
	var _pb_sfixed64_value_seen: bool = false
	var _pb_bool_value_seen: bool = false
	var _pb_string_value_seen: bool = false
	var _pb_bytes_value_seen: bool = false
	var _pb_two_word_name_seen: bool = false
	var _pb_int32_keyed_seen: bool = false
	var _pb_int64_keyed_seen: bool = false
	var _pb_uint64_keyed_seen: bool = false
	var _pb_bool_keyed_seen: bool = false
	var _pb_choice_seen: bool = false
	var _pb_occurred_at_seen: bool = false
	for _pb_key: String in _pb_entries:
		var _pb_member: JsonNode = _pb_entries[_pb_key]
		var _pb_member_path: String = "$." + _pb_key
		match _pb_key:
			"doubleValue", "double_value":
				if _pb_double_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: double_value was given more than once", _pb_member_path)
				_pb_double_value_seen = true
				var (_pb_double_value_value, _pb_double_value_error) = _pb_json_read_float(_pb_member, _pb_member_path)
				if _pb_double_value_error is JsonDecodeError:
					return _pb_double_value_error
				double_value = _pb_double_value_value
			"floatValue", "float_value":
				if _pb_float_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: float_value was given more than once", _pb_member_path)
				_pb_float_value_seen = true
				var (_pb_float_value_value, _pb_float_value_error) = _pb_json_read_float(_pb_member, _pb_member_path)
				if _pb_float_value_error is JsonDecodeError:
					return _pb_float_value_error
				float_value = Wire.narrow_float32(_pb_float_value_value)
			"int32Value", "int32_value":
				if _pb_int32_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: int32_value was given more than once", _pb_member_path)
				_pb_int32_value_seen = true
				var (_pb_int32_value_value, _pb_int32_value_error) = _pb_json_read_int32(_pb_member, _pb_member_path)
				if _pb_int32_value_error is JsonDecodeError:
					return _pb_int32_value_error
				int32_value = _pb_int32_value_value
			"int64Value", "int64_value":
				if _pb_int64_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: int64_value was given more than once", _pb_member_path)
				_pb_int64_value_seen = true
				var (_pb_int64_value_value, _pb_int64_value_error) = _pb_json_read_int64(_pb_member, _pb_member_path)
				if _pb_int64_value_error is JsonDecodeError:
					return _pb_int64_value_error
				int64_value = _pb_int64_value_value
			"uint32Value", "uint32_value":
				if _pb_uint32_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: uint32_value was given more than once", _pb_member_path)
				_pb_uint32_value_seen = true
				var (_pb_uint32_value_value, _pb_uint32_value_error) = _pb_json_read_uint32(_pb_member, _pb_member_path)
				if _pb_uint32_value_error is JsonDecodeError:
					return _pb_uint32_value_error
				uint32_value = _pb_uint32_value_value
			"uint64Value", "uint64_value":
				if _pb_uint64_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: uint64_value was given more than once", _pb_member_path)
				_pb_uint64_value_seen = true
				var (_pb_uint64_value_value, _pb_uint64_value_error) = _pb_json_read_uint64(_pb_member, _pb_member_path)
				if _pb_uint64_value_error is JsonDecodeError:
					return _pb_uint64_value_error
				uint64_value = _pb_uint64_value_value
			"sint32Value", "sint32_value":
				if _pb_sint32_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: sint32_value was given more than once", _pb_member_path)
				_pb_sint32_value_seen = true
				var (_pb_sint32_value_value, _pb_sint32_value_error) = _pb_json_read_int32(_pb_member, _pb_member_path)
				if _pb_sint32_value_error is JsonDecodeError:
					return _pb_sint32_value_error
				sint32_value = _pb_sint32_value_value
			"sint64Value", "sint64_value":
				if _pb_sint64_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: sint64_value was given more than once", _pb_member_path)
				_pb_sint64_value_seen = true
				var (_pb_sint64_value_value, _pb_sint64_value_error) = _pb_json_read_int64(_pb_member, _pb_member_path)
				if _pb_sint64_value_error is JsonDecodeError:
					return _pb_sint64_value_error
				sint64_value = _pb_sint64_value_value
			"fixed32Value", "fixed32_value":
				if _pb_fixed32_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: fixed32_value was given more than once", _pb_member_path)
				_pb_fixed32_value_seen = true
				var (_pb_fixed32_value_value, _pb_fixed32_value_error) = _pb_json_read_uint32(_pb_member, _pb_member_path)
				if _pb_fixed32_value_error is JsonDecodeError:
					return _pb_fixed32_value_error
				fixed32_value = _pb_fixed32_value_value
			"fixed64Value", "fixed64_value":
				if _pb_fixed64_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: fixed64_value was given more than once", _pb_member_path)
				_pb_fixed64_value_seen = true
				var (_pb_fixed64_value_value, _pb_fixed64_value_error) = _pb_json_read_uint64(_pb_member, _pb_member_path)
				if _pb_fixed64_value_error is JsonDecodeError:
					return _pb_fixed64_value_error
				fixed64_value = _pb_fixed64_value_value
			"sfixed32Value", "sfixed32_value":
				if _pb_sfixed32_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: sfixed32_value was given more than once", _pb_member_path)
				_pb_sfixed32_value_seen = true
				var (_pb_sfixed32_value_value, _pb_sfixed32_value_error) = _pb_json_read_int32(_pb_member, _pb_member_path)
				if _pb_sfixed32_value_error is JsonDecodeError:
					return _pb_sfixed32_value_error
				sfixed32_value = _pb_sfixed32_value_value
			"sfixed64Value", "sfixed64_value":
				if _pb_sfixed64_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: sfixed64_value was given more than once", _pb_member_path)
				_pb_sfixed64_value_seen = true
				var (_pb_sfixed64_value_value, _pb_sfixed64_value_error) = _pb_json_read_int64(_pb_member, _pb_member_path)
				if _pb_sfixed64_value_error is JsonDecodeError:
					return _pb_sfixed64_value_error
				sfixed64_value = _pb_sfixed64_value_value
			"boolValue", "bool_value":
				if _pb_bool_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: bool_value was given more than once", _pb_member_path)
				_pb_bool_value_seen = true
				var (_pb_bool_value_value, _pb_bool_value_error) = _pb_json_read_bool(_pb_member, _pb_member_path)
				if _pb_bool_value_error is JsonDecodeError:
					return _pb_bool_value_error
				bool_value = _pb_bool_value_value
			"stringValue", "string_value":
				if _pb_string_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: string_value was given more than once", _pb_member_path)
				_pb_string_value_seen = true
				var (_pb_string_value_value, _pb_string_value_error) = _pb_json_read_string(_pb_member, _pb_member_path)
				if _pb_string_value_error is JsonDecodeError:
					return _pb_string_value_error
				string_value = _pb_string_value_value
			"bytesValue", "bytes_value":
				if _pb_bytes_value_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: bytes_value was given more than once", _pb_member_path)
				_pb_bytes_value_seen = true
				var (_pb_bytes_value_value, _pb_bytes_value_error) = _pb_json_read_bytes(_pb_member, _pb_member_path)
				if _pb_bytes_value_error is JsonDecodeError:
					return _pb_bytes_value_error
				bytes_value = _pb_bytes_value_value
			"twoWordName", "two_word_name":
				if _pb_two_word_name_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: two_word_name was given more than once", _pb_member_path)
				_pb_two_word_name_seen = true
				var (_pb_two_word_name_value, _pb_two_word_name_error) = _pb_json_read_string(_pb_member, _pb_member_path)
				if _pb_two_word_name_error is JsonDecodeError:
					return _pb_two_word_name_error
				two_word_name = _pb_two_word_name_value
			"flavor":
				var _pb_flavor_value: Flavor = Flavor.FLAVOR_UNSPECIFIED
				match _pb_member:
					JsonNode.Null:
						pass
					JsonNode.Str(var _pb_flavor_name):
						var _pb_flavor_case: Flavor? = Flavor.from_json_name(_pb_flavor_name)
						if not (_pb_flavor_case is Flavor):
							return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: Flavor has no case with this JSON name", _pb_member_path)
						_pb_flavor_value = _pb_flavor_case
					JsonNode.Int(var _pb_flavor_number):
						var _pb_flavor_wire: Flavor? = Flavor.from_wire(_pb_flavor_number)
						if _pb_flavor_wire is Flavor:
							_pb_flavor_value = _pb_flavor_wire
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: Flavor takes a case name or a number", _pb_member_path)
				flavor = _pb_flavor_value
			"primary":
				match _pb_member:
					JsonNode.Null:
						primary = null
					_:
						var _pb_primary_result: JsonResult[Reference] = Reference.from_json(_pb_member)
						var _pb_primary_error: JsonDecodeError? = _pb_primary_result.error
						if _pb_primary_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_primary_error, _pb_key).error
						var _pb_primary_value: Reference? = _pb_primary_result.value
						if not (_pb_primary_value is Reference):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Reference decoded to no value", _pb_member_path)
						primary = _pb_primary_value
			"nickname":
				match _pb_member:
					JsonNode.Null:
						nickname = null
					_:
						var (_pb_nickname_value, _pb_nickname_error) = _pb_json_read_string(_pb_member, _pb_member_path)
						if _pb_nickname_error is JsonDecodeError:
							return _pb_nickname_error
						nickname = _pb_nickname_value
			"tags":
				match _pb_member:
					JsonNode.Null:
						tags = []
					JsonNode.Array(var _pb_tags_items):
						tags = []
						var _pb_tags_index: int = 0
						while _pb_tags_index < _pb_tags_items.size():
							var (_pb_tags_element_value, _pb_tags_element_error) = _pb_json_read_string(_pb_tags_items[_pb_tags_index], _pb_member_path + "." + str(_pb_tags_index))
							if _pb_tags_element_error is JsonDecodeError:
								return _pb_tags_element_error
							tags.append(_pb_tags_element_value)
							_pb_tags_index += 1
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: tags expects a JSON array", _pb_member_path)
			"tallies":
				match _pb_member:
					JsonNode.Null:
						tallies = []
					JsonNode.Array(var _pb_tallies_items):
						tallies = []
						var _pb_tallies_index: int = 0
						while _pb_tallies_index < _pb_tallies_items.size():
							var (_pb_tallies_element_value, _pb_tallies_element_error) = _pb_json_read_int64(_pb_tallies_items[_pb_tallies_index], _pb_member_path + "." + str(_pb_tallies_index))
							if _pb_tallies_element_error is JsonDecodeError:
								return _pb_tallies_element_error
							tallies.append(_pb_tallies_element_value)
							_pb_tallies_index += 1
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: tallies expects a JSON array", _pb_member_path)
			"flavors":
				match _pb_member:
					JsonNode.Null:
						flavors = []
					JsonNode.Array(var _pb_flavors_items):
						flavors = []
						var _pb_flavors_index: int = 0
						while _pb_flavors_index < _pb_flavors_items.size():
							var _pb_flavors_element_value: Flavor = Flavor.FLAVOR_UNSPECIFIED
							match _pb_flavors_items[_pb_flavors_index]:
								JsonNode.Null:
									pass
								JsonNode.Str(var _pb_flavors_element_name):
									var _pb_flavors_element_case: Flavor? = Flavor.from_json_name(_pb_flavors_element_name)
									if not (_pb_flavors_element_case is Flavor):
										return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: Flavor has no case with this JSON name", _pb_member_path + "." + str(_pb_flavors_index))
									_pb_flavors_element_value = _pb_flavors_element_case
								JsonNode.Int(var _pb_flavors_element_number):
									var _pb_flavors_element_wire: Flavor? = Flavor.from_wire(_pb_flavors_element_number)
									if _pb_flavors_element_wire is Flavor:
										_pb_flavors_element_value = _pb_flavors_element_wire
								_:
									return JsonDecodeError.create("JSON_TYPE_MISMATCH: Flavor takes a case name or a number", _pb_member_path + "." + str(_pb_flavors_index))
							flavors.append(_pb_flavors_element_value)
							_pb_flavors_index += 1
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: flavors expects a JSON array", _pb_member_path)
			"references":
				match _pb_member:
					JsonNode.Null:
						references = []
					JsonNode.Array(var _pb_references_items):
						references = []
						var _pb_references_index: int = 0
						while _pb_references_index < _pb_references_items.size():
							var _pb_references_element_result: JsonResult[Reference] = Reference.from_json(_pb_references_items[_pb_references_index])
							var _pb_references_element_error: JsonDecodeError? = _pb_references_element_result.error
							if _pb_references_element_error is JsonDecodeError:
								return JsonResult[JsonSuite].nested(_pb_references_element_error, _pb_key + "." + str(_pb_references_index)).error
							var _pb_references_element_value: Reference? = _pb_references_element_result.value
							if not (_pb_references_element_value is Reference):
								return JsonDecodeError.create("JSON_TYPE_MISMATCH: Reference decoded to no value", _pb_member_path + "." + str(_pb_references_index))
							references.append(_pb_references_element_value)
							_pb_references_index += 1
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: references expects a JSON array", _pb_member_path)
			"counts":
				match _pb_member:
					JsonNode.Null:
						counts = {}
					JsonNode.Object(var _pb_counts_entries):
						counts = {}
						for _pb_counts_key: String in _pb_counts_entries:
							var (_pb_counts_value, _pb_counts_error) = _pb_json_read_int32(_pb_counts_entries[_pb_counts_key], _pb_member_path + "." + _pb_counts_key)
							if _pb_counts_error is JsonDecodeError:
								return _pb_counts_error
							counts[_pb_counts_key] = _pb_counts_value
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: counts expects a JSON object", _pb_member_path)
			"int32Keyed", "int32_keyed":
				if _pb_int32_keyed_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: int32_keyed was given more than once", _pb_member_path)
				_pb_int32_keyed_seen = true
				match _pb_member:
					JsonNode.Null:
						int32_keyed = {}
					JsonNode.Object(var _pb_int32_keyed_entries):
						int32_keyed = {}
						for _pb_int32_keyed_key: String in _pb_int32_keyed_entries:
							var _pb_int32_keyed_key_path: String = _pb_member_path + "." + _pb_int32_keyed_key
							var (_pb_int32_keyed_key_value, _pb_int32_keyed_key_error) = _pb_json_read_int32(JsonNode.Str(_pb_int32_keyed_key), _pb_int32_keyed_key_path)
							if _pb_int32_keyed_key_error is JsonDecodeError:
								return _pb_int32_keyed_key_error
							var (_pb_int32_keyed_value, _pb_int32_keyed_error) = _pb_json_read_string(_pb_int32_keyed_entries[_pb_int32_keyed_key], _pb_member_path + "." + _pb_int32_keyed_key)
							if _pb_int32_keyed_error is JsonDecodeError:
								return _pb_int32_keyed_error
							int32_keyed[_pb_int32_keyed_key_value] = _pb_int32_keyed_value
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: int32_keyed expects a JSON object", _pb_member_path)
			"int64Keyed", "int64_keyed":
				if _pb_int64_keyed_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: int64_keyed was given more than once", _pb_member_path)
				_pb_int64_keyed_seen = true
				match _pb_member:
					JsonNode.Null:
						int64_keyed = {}
					JsonNode.Object(var _pb_int64_keyed_entries):
						int64_keyed = {}
						for _pb_int64_keyed_key: String in _pb_int64_keyed_entries:
							var _pb_int64_keyed_key_path: String = _pb_member_path + "." + _pb_int64_keyed_key
							var (_pb_int64_keyed_key_value, _pb_int64_keyed_key_error) = _pb_json_read_int64(JsonNode.Str(_pb_int64_keyed_key), _pb_int64_keyed_key_path)
							if _pb_int64_keyed_key_error is JsonDecodeError:
								return _pb_int64_keyed_key_error
							var (_pb_int64_keyed_value, _pb_int64_keyed_error) = _pb_json_read_string(_pb_int64_keyed_entries[_pb_int64_keyed_key], _pb_member_path + "." + _pb_int64_keyed_key)
							if _pb_int64_keyed_error is JsonDecodeError:
								return _pb_int64_keyed_error
							int64_keyed[_pb_int64_keyed_key_value] = _pb_int64_keyed_value
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: int64_keyed expects a JSON object", _pb_member_path)
			"uint64Keyed", "uint64_keyed":
				if _pb_uint64_keyed_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: uint64_keyed was given more than once", _pb_member_path)
				_pb_uint64_keyed_seen = true
				match _pb_member:
					JsonNode.Null:
						uint64_keyed = {}
					JsonNode.Object(var _pb_uint64_keyed_entries):
						uint64_keyed = {}
						for _pb_uint64_keyed_key: String in _pb_uint64_keyed_entries:
							var _pb_uint64_keyed_key_path: String = _pb_member_path + "." + _pb_uint64_keyed_key
							var (_pb_uint64_keyed_key_value, _pb_uint64_keyed_key_error) = _pb_json_read_uint64(JsonNode.Str(_pb_uint64_keyed_key), _pb_uint64_keyed_key_path)
							if _pb_uint64_keyed_key_error is JsonDecodeError:
								return _pb_uint64_keyed_key_error
							var (_pb_uint64_keyed_value, _pb_uint64_keyed_error) = _pb_json_read_string(_pb_uint64_keyed_entries[_pb_uint64_keyed_key], _pb_member_path + "." + _pb_uint64_keyed_key)
							if _pb_uint64_keyed_error is JsonDecodeError:
								return _pb_uint64_keyed_error
							uint64_keyed[_pb_uint64_keyed_key_value] = _pb_uint64_keyed_value
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: uint64_keyed expects a JSON object", _pb_member_path)
			"boolKeyed", "bool_keyed":
				if _pb_bool_keyed_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: bool_keyed was given more than once", _pb_member_path)
				_pb_bool_keyed_seen = true
				match _pb_member:
					JsonNode.Null:
						bool_keyed = {}
					JsonNode.Object(var _pb_bool_keyed_entries):
						bool_keyed = {}
						for _pb_bool_keyed_key: String in _pb_bool_keyed_entries:
							var _pb_bool_keyed_key_path: String = _pb_member_path + "." + _pb_bool_keyed_key
							var _pb_bool_keyed_key_value: bool = false
							if _pb_bool_keyed_key == "true":
								_pb_bool_keyed_key_value = true
							elif _pb_bool_keyed_key != "false":
								return JsonDecodeError.create("JSON_TYPE_MISMATCH: a bool map key takes \"true\" or \"false\"", _pb_bool_keyed_key_path)
							var _pb_bool_keyed_result: JsonResult[Reference] = Reference.from_json(_pb_bool_keyed_entries[_pb_bool_keyed_key])
							var _pb_bool_keyed_error: JsonDecodeError? = _pb_bool_keyed_result.error
							if _pb_bool_keyed_error is JsonDecodeError:
								return JsonResult[JsonSuite].nested(_pb_bool_keyed_error, _pb_key + "." + _pb_bool_keyed_key).error
							var _pb_bool_keyed_value: Reference? = _pb_bool_keyed_result.value
							if not (_pb_bool_keyed_value is Reference):
								return JsonDecodeError.create("JSON_TYPE_MISMATCH: Reference decoded to no value", _pb_member_path + "." + _pb_bool_keyed_key)
							bool_keyed[_pb_bool_keyed_key_value] = _pb_bool_keyed_value
					_:
						return JsonDecodeError.create("JSON_TYPE_MISMATCH: bool_keyed expects a JSON object", _pb_member_path)
			"note":
				if _pb_choice_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: choice has more than one member set", _pb_member_path)
				_pb_choice_seen = true
				match _pb_member:
					JsonNode.Null:
						choice = null
					_:
						var (_pb_choice_note_value, _pb_choice_note_error) = _pb_json_read_string(_pb_member, _pb_member_path)
						if _pb_choice_note_error is JsonDecodeError:
							return _pb_choice_note_error
						choice = JsonSuiteChoiceCase.Note(_pb_choice_note_value)
			"tally":
				if _pb_choice_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: choice has more than one member set", _pb_member_path)
				_pb_choice_seen = true
				match _pb_member:
					JsonNode.Null:
						choice = null
					_:
						var (_pb_choice_tally_value, _pb_choice_tally_error) = _pb_json_read_int64(_pb_member, _pb_member_path)
						if _pb_choice_tally_error is JsonDecodeError:
							return _pb_choice_tally_error
						choice = JsonSuiteChoiceCase.Tally(_pb_choice_tally_value)
			"detail":
				if _pb_choice_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: choice has more than one member set", _pb_member_path)
				_pb_choice_seen = true
				match _pb_member:
					JsonNode.Null:
						choice = null
					_:
						var _pb_choice_detail_result: JsonResult[Reference] = Reference.from_json(_pb_member)
						var _pb_choice_detail_error: JsonDecodeError? = _pb_choice_detail_result.error
						if _pb_choice_detail_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_choice_detail_error, _pb_key).error
						var _pb_choice_detail_value: Reference? = _pb_choice_detail_result.value
						if not (_pb_choice_detail_value is Reference):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Reference decoded to no value", _pb_member_path)
						choice = JsonSuiteChoiceCase.Detail(_pb_choice_detail_value)
			"occurredAt", "occurred_at":
				if _pb_occurred_at_seen:
					return JsonDecodeError.create("JSON_PARSE_FAILED: occurred_at was given more than once", _pb_member_path)
				_pb_occurred_at_seen = true
				match _pb_member:
					JsonNode.Null:
						occurred_at = null
					_:
						var _pb_occurred_at_result: JsonResult[Timestamp] = Timestamp.from_json(_pb_member)
						var _pb_occurred_at_error: JsonDecodeError? = _pb_occurred_at_result.error
						if _pb_occurred_at_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_occurred_at_error, _pb_key).error
						var _pb_occurred_at_value: Timestamp? = _pb_occurred_at_result.value
						if not (_pb_occurred_at_value is Timestamp):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Timestamp decoded to no value", _pb_member_path)
						occurred_at = _pb_occurred_at_value
			"elapsed":
				match _pb_member:
					JsonNode.Null:
						elapsed = null
					_:
						var _pb_elapsed_result: JsonResult[Duration] = Duration.from_json(_pb_member)
						var _pb_elapsed_error: JsonDecodeError? = _pb_elapsed_result.error
						if _pb_elapsed_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_elapsed_error, _pb_key).error
						var _pb_elapsed_value: Duration? = _pb_elapsed_result.value
						if not (_pb_elapsed_value is Duration):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Duration decoded to no value", _pb_member_path)
						elapsed = _pb_elapsed_value
			"attributes":
				match _pb_member:
					JsonNode.Null:
						attributes = null
					_:
						var _pb_attributes_result: JsonResult[Struct] = Struct.from_json(_pb_member)
						var _pb_attributes_error: JsonDecodeError? = _pb_attributes_result.error
						if _pb_attributes_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_attributes_error, _pb_key).error
						var _pb_attributes_value: Struct? = _pb_attributes_result.value
						if not (_pb_attributes_value is Struct):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Struct decoded to no value", _pb_member_path)
						attributes = _pb_attributes_value
			"setting":
				match _pb_member:
					JsonNode.Null:
						setting = null
					_:
						var _pb_setting_result: JsonResult[Value] = Value.from_json(_pb_member)
						var _pb_setting_error: JsonDecodeError? = _pb_setting_result.error
						if _pb_setting_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_setting_error, _pb_key).error
						var _pb_setting_value: Value? = _pb_setting_result.value
						if not (_pb_setting_value is Value):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Value decoded to no value", _pb_member_path)
						setting = _pb_setting_value
			"label":
				match _pb_member:
					JsonNode.Null:
						label = null
					_:
						var _pb_label_result: JsonResult[StringValue] = StringValue.from_json(_pb_member)
						var _pb_label_error: JsonDecodeError? = _pb_label_result.error
						if _pb_label_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_label_error, _pb_key).error
						var _pb_label_value: StringValue? = _pb_label_result.value
						if not (_pb_label_value is StringValue):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: StringValue decoded to no value", _pb_member_path)
						label = _pb_label_value
			"inner":
				match _pb_member:
					JsonNode.Null:
						inner = null
					_:
						var _pb_inner_result: JsonResult[Inner] = Inner.from_json(_pb_member)
						var _pb_inner_error: JsonDecodeError? = _pb_inner_result.error
						if _pb_inner_error is JsonDecodeError:
							return JsonResult[JsonSuite].nested(_pb_inner_error, _pb_key).error
						var _pb_inner_value: Inner? = _pb_inner_result.value
						if not (_pb_inner_value is Inner):
							return JsonDecodeError.create("JSON_TYPE_MISMATCH: Inner decoded to no value", _pb_member_path)
						inner = _pb_inner_value
			_:
				return JsonDecodeError.create("JSON_UNKNOWN_FIELD: JsonSuite has no field named " + _pb_key, _pb_member_path)
	return null

## Reads a signed 32-bit integer field out of a JSON value.
##
## The canonical mapping accepts a JSON string and a whole JSON number as
## well as the number this emitter writes, so all three are read here. A
## value outside the field's domain is refused rather than truncated.
static func _pb_json_read_int32(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):
	var _pb_value: int = 0
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Int(var _pb_int):
			_pb_value = _pb_int
		JsonNode.Float(var _pb_float):
			if _pb_float != floor(_pb_float):
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a signed 32-bit integer field cannot take a fractional number", _pb_path))
			if _pb_float >= 2147483648.0 or _pb_float < -2147483648.0:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a signed 32-bit integer field cannot hold this value", _pb_path))
			_pb_value = int(_pb_float)
		JsonNode.Str(var _pb_text):
			if not _pb_text.is_valid_int():
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a signed 32-bit integer field cannot take this string", _pb_path))
			_pb_value = _pb_text.to_int()
		_:
			return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a signed 32-bit integer field takes a number or a string", _pb_path))
	if _pb_value < -2147483648 or _pb_value > 2147483647:
		return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a signed 32-bit integer field cannot hold this value", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads an unsigned 32-bit integer field out of a JSON value.
##
## The canonical mapping accepts a JSON string and a whole JSON number as
## well as the number this emitter writes, so all three are read here. A
## value outside the field's domain is refused rather than truncated.
static func _pb_json_read_uint32(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):
	var _pb_value: int = 0
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Int(var _pb_int):
			_pb_value = _pb_int
		JsonNode.Float(var _pb_float):
			if _pb_float != floor(_pb_float):
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: an unsigned 32-bit integer field cannot take a fractional number", _pb_path))
			if _pb_float >= 4294967296.0 or _pb_float < 0.0:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: an unsigned 32-bit integer field cannot hold this value", _pb_path))
			_pb_value = int(_pb_float)
		JsonNode.Str(var _pb_text):
			if not _pb_text.is_valid_int():
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: an unsigned 32-bit integer field cannot take this string", _pb_path))
			_pb_value = _pb_text.to_int()
		_:
			return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: an unsigned 32-bit integer field takes a number or a string", _pb_path))
	if _pb_value < 0 or _pb_value > 4294967295:
		return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: an unsigned 32-bit integer field cannot hold this value", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads a 64-bit integer field out of a JSON value.
##
## A string is exact and is what this emitter writes. A bare number is
## accepted because the canonical mapping requires it, and is lossy past
## 2^53: the engine's parser produces a double, so a value that large does
## not even arrive as a JsonNode.Int.
static func _pb_json_read_int64(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):
	var _pb_value: int = 0
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Int(var _pb_int):
			_pb_value = _pb_int
		JsonNode.Float(var _pb_float):
			if _pb_float != floor(_pb_float):
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a 64-bit integer field cannot take a fractional number", _pb_path))
			if _pb_float > 9223372036854775808.0 or _pb_float < -9223372036854775808.0:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a 64-bit integer field cannot hold this value", _pb_path))
			_pb_value = int(_pb_float)
		JsonNode.Str(var _pb_text):
			if not _pb_text.is_valid_int():
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a 64-bit integer field cannot take this string", _pb_path))
			_pb_value = _pb_text.to_int()
			if str(_pb_value) != _pb_text:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a 64-bit integer field takes a decimal string it can hold exactly", _pb_path))
		_:
			return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a 64-bit integer field takes a number or a string", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads an unsigned 64-bit integer field out of a JSON value.
##
## The top half of the range has no signed spelling, so the text goes
## through the runtime helper rather than String.to_int(), which wraps to
## the smallest signed value there. A bare number is accepted because the
## canonical mapping requires it, and is lossy past 2^53: the engine's
## parser produces a double, so a value that large does not even arrive as
## a JsonNode.Int. The widest value rounds to 2^64 on the way in and is
## read as the value it rounded from rather than refused.
static func _pb_json_read_uint64(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):
	var _pb_value: int = 0
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Int(var _pb_int):
			if _pb_int < 0:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: an unsigned 64-bit integer field cannot hold this value", _pb_path))
			_pb_value = _pb_int
		JsonNode.Float(var _pb_float):
			if _pb_float != floor(_pb_float):
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: an unsigned 64-bit integer field cannot take a fractional number", _pb_path))
			if _pb_float > 18446744073709551616.0 or _pb_float < 0.0:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: an unsigned 64-bit integer field cannot hold this value", _pb_path))
			if _pb_float == 18446744073709551616.0:
				_pb_value = JsonUint64.WIDEST_BITS
			elif _pb_float >= 9223372036854775808.0:
				_pb_value = int(_pb_float - 18446744073709551616.0)
			else:
				_pb_value = int(_pb_float)
		JsonNode.Str(var _pb_text):
			var (_pb_unsigned, _pb_unsigned_error) = JsonUint64.parse(_pb_text)
			if _pb_unsigned_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE:
				return (0, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: an unsigned 64-bit integer field cannot hold this value", _pb_path))
			if _pb_unsigned_error != ProtobufError.OK:
				return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: an unsigned 64-bit integer field cannot take this string", _pb_path))
			_pb_value = _pb_unsigned
		_:
			return (0, JsonDecodeError.create("JSON_TYPE_MISMATCH: an unsigned 64-bit integer field takes a number or a string", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads a floating-point field out of a JSON value.
##
## The three non-finite values have no JSON number form, so the canonical
## mapping spells them as strings and they are read back from those.
static func _pb_json_read_float(_pb_node: JsonNode, _pb_path: String) -> (float, JsonDecodeError?):
	var _pb_value: float = 0.0
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Float(var _pb_float):
			_pb_value = _pb_float
		JsonNode.Int(var _pb_int):
			_pb_value = _pb_int
		JsonNode.Str(var _pb_text):
			if _pb_text == "NaN":
				_pb_value = NAN
			elif _pb_text == "Infinity":
				_pb_value = INF
			elif _pb_text == "-Infinity":
				_pb_value = -INF
			else:
				return (0.0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a floating-point field takes a number or one of the three non-finite strings", _pb_path))
		_:
			return (0.0, JsonDecodeError.create("JSON_TYPE_MISMATCH: a floating-point field takes a number or one of the three non-finite strings", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads a bool field out of a JSON value.
static func _pb_json_read_bool(_pb_node: JsonNode, _pb_path: String) -> (bool, JsonDecodeError?):
	var _pb_value: bool = false
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Bool(var _pb_bool):
			_pb_value = _pb_bool
		_:
			return (false, JsonDecodeError.create("JSON_TYPE_MISMATCH: a bool field takes true or false", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads a string field out of a JSON value.
static func _pb_json_read_string(_pb_node: JsonNode, _pb_path: String) -> (String, JsonDecodeError?):
	var _pb_value: String = ""
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Str(var _pb_text):
			_pb_value = _pb_text
		_:
			return ("", JsonDecodeError.create("JSON_TYPE_MISMATCH: a string field takes a JSON string", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)

## Reads a bytes field out of a JSON value.
##
## The runtime helper accepts the URL-safe alphabet and optional padding as
## well as the standard form, which is what the mapping asks a parser for.
static func _pb_json_read_bytes(_pb_node: JsonNode, _pb_path: String) -> (PackedByteArray, JsonDecodeError?):
	var _pb_value: PackedByteArray = PackedByteArray()
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Str(var _pb_text):
			var (_pb_bytes, _pb_bytes_error) = JsonBase64.decode(_pb_text)
			if _pb_bytes_error != ProtobufError.OK:
				return (PackedByteArray(), JsonDecodeError.create("JSON_TYPE_MISMATCH: a bytes field takes base64 text", _pb_path))
			_pb_value = _pb_bytes
		_:
			return (PackedByteArray(), JsonDecodeError.create("JSON_TYPE_MISMATCH: a bytes field takes base64 text", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value, _pb_error)
