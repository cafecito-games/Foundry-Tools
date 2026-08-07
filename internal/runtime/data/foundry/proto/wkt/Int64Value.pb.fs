namespace foundry.proto.wkt
import foundry.proto

## Wrapper message for `int64`.
## The JSON representation for `Int64Value` is JSON string.
## Not recommended for use in new APIs, but still useful for legacy APIs and
## has no plan to be removed.
final class_name Int64Value extends RefCounted uses Message, JsonSerializable

## The int64 value.
var value: long = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> Int64Value:
	return Int64Value.new()

static func protobuf_type_name() -> String:
	return "google.protobuf.Int64Value"

func type_name() -> String:
	return Int64Value.protobuf_type_name()

static func _pb_any_uses_value() -> bool:
	return true

## Decodes protobuf wire data into a new Int64Value message.
static func from_bytes(_pb_data: PackedByteArray) -> (Int64Value?, ProtobufError):
	var _pb_message: Int64Value = Int64Value.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Int64Value? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if value != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(value))
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
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_value_read.error != ProtobufError.OK:
					return _pb_value_read.error
				value = _pb_value_read.value
				_pb_offset = _pb_value_read.offset
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
	return JsonNode.Str(str(value))

## Decodes a proto3 canonical JSON document into a new Int64Value message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Int64Value]:
	var _pb_message: Int64Value = Int64Value.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Int64Value].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Int64Value].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	var (_pb_value_value, _pb_value_error) = _pb_json_read_int64(_pb_node, "$")
	if _pb_value_error is JsonDecodeError:
		return _pb_value_error
	value = _pb_value_value
	return null

## Reads a 64-bit integer field out of a JSON value.
##
## A string is exact and is what this emitter writes. A bare number is
## accepted because the canonical mapping requires it, and is lossy past
## 2^53: the engine's parser produces a double, so a value that large does
## not even arrive as a JsonNode.Int.
static func _pb_json_read_int64(_pb_node: JsonNode, _pb_path: String) -> (long, JsonDecodeError?):
	var _pb_value: long = 0
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Int(var _pb_int):
			_pb_value = _pb_int
		JsonNode.Float(var _pb_float):
			if _pb_float != floor(_pb_float):
				return (0L, JsonDecodeError.create("JSON_TYPE_MISMATCH: a 64-bit integer field cannot take a fractional number", _pb_path))
			if _pb_float > 9223372036854775808.0 or _pb_float < -9223372036854775808.0:
				return (0L, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a 64-bit integer field cannot hold this value", _pb_path))
			_pb_value = _pb_float as long
		JsonNode.Str(var _pb_text):
			if not _pb_text.is_valid_int():
				return (0L, JsonDecodeError.create("JSON_TYPE_MISMATCH: a 64-bit integer field cannot take this string", _pb_path))
			_pb_value = _pb_text.to_int() as long
			if str(_pb_value) != _pb_text:
				return (0L, JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a 64-bit integer field takes a decimal string it can hold exactly", _pb_path))
		_:
			return (0L, JsonDecodeError.create("JSON_TYPE_MISMATCH: a 64-bit integer field takes a number or a string", _pb_path))
	var _pb_error: JsonDecodeError? = null
	return (_pb_value as long, _pb_error)
