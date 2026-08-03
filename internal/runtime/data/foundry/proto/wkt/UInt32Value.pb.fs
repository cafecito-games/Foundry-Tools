namespace foundry.proto.wkt
import foundry.proto

## Wrapper message for `uint32`.
## The JSON representation for `UInt32Value` is JSON number.
## Not recommended for use in new APIs, but still useful for legacy APIs and
## has no plan to be removed.
final class_name UInt32Value extends RefCounted uses Message, JsonSerializable

## The uint32 value.
var value: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> UInt32Value:
	return UInt32Value.new()

static func protobuf_type_name() -> String:
	return "google.protobuf.UInt32Value"

func type_name() -> String:
	return UInt32Value.protobuf_type_name()

static func _pb_any_uses_value() -> bool:
	return true

## Decodes protobuf wire data into a new UInt32Value message.
static func from_bytes(_pb_data: PackedByteArray) -> (UInt32Value?, ProtobufError):
	var _pb_message: UInt32Value = UInt32Value.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: UInt32Value? = null
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
	return JsonNode.Int(value)

## Decodes a proto3 canonical JSON document into a new UInt32Value message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[UInt32Value]:
	var _pb_message: UInt32Value = UInt32Value.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[UInt32Value].fail(_pb_error.message, _pb_error.path)
	return JsonResult[UInt32Value].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	var (_pb_value_value, _pb_value_error) = _pb_json_read_uint32(_pb_node, "$")
	if _pb_value_error is JsonDecodeError:
		return _pb_value_error
	value = _pb_value_value
	return null

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
