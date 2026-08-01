namespace cafecito.json.v1
import foundry.proto

## A message reached through a field rather than declared inline.
## The two fields are declared in the reverse of their numbering, because JSON
## member order follows the field number rather than the order of declaration.
## Ordering them the same way in both would leave the rule unpinned.
final class_name Reference extends RefCounted uses Message, JsonSerializable

## The label protobuf field.
var label: String = ""

## The weight protobuf field.
var weight: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Reference message.
static func from_bytes(_pb_data: PackedByteArray) -> (Reference?, ProtobufError):
	var _pb_message: Reference = Reference.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Reference? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if label != "":
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_label_data: PackedByteArray = Wire.encode_string(label)
		_pb_result.append_array(Wire.encode_varint(_pb_label_data.size()))
		_pb_result.append_array(_pb_label_data)
	if weight != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(weight))
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
				var _pb_label_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_label_read.error != ProtobufError.OK:
					return _pb_label_read.error
				label = _pb_label_read.value
				_pb_offset = _pb_label_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_weight_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_weight_read.error != ProtobufError.OK:
					return _pb_weight_read.error
				weight = _pb_weight_read.value
				_pb_offset = _pb_weight_read.offset
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
	if label != "":
		_pb_json["label"] = JsonNode.Str(label)
	if weight != 0:
		_pb_json["weight"] = JsonNode.Str(str(weight))
	return JsonNode.object_of(_pb_json)

## Decodes a proto3 canonical JSON document into a new Reference message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Reference]:
	var _pb_message: Reference = Reference.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Reference].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Reference].ok(_pb_message)

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
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Reference expects a JSON object", "$")
	for _pb_key: String in _pb_entries:
		var _pb_member: JsonNode = _pb_entries[_pb_key]
		var _pb_member_path: String = "$." + _pb_key
		match _pb_key:
			"label":
				var (_pb_label_value, _pb_label_error) = _pb_json_read_string(_pb_member, _pb_member_path)
				if _pb_label_error is JsonDecodeError:
					return _pb_label_error
				label = _pb_label_value
			"weight":
				var (_pb_weight_value, _pb_weight_error) = _pb_json_read_int64(_pb_member, _pb_member_path)
				if _pb_weight_error is JsonDecodeError:
					return _pb_weight_error
				weight = _pb_weight_value
			_:
				return JsonDecodeError.create("JSON_UNKNOWN_FIELD: Reference has no field named " + _pb_key, _pb_member_path)
	return null

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
