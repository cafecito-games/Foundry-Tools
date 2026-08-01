namespace foundry.proto.wkt
import foundry.proto

## Wrapper message for `bytes`.
## The JSON representation for `BytesValue` is JSON string.
## Not recommended for use in new APIs, but still useful for legacy APIs and
## has no plan to be removed.
final class_name BytesValue extends RefCounted uses Message, JsonSerializable

## The bytes value.
var value: PackedByteArray = PackedByteArray()

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new BytesValue message.
static func from_bytes(_pb_data: PackedByteArray) -> (BytesValue?, ProtobufError):
	var _pb_message: BytesValue = BytesValue.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: BytesValue? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if value.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(value.size()))
		_pb_result.append_array(value)
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
				var _pb_value_read: BytesRead = Wire.read_bytes(_pb_data, _pb_offset)
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
	return JsonNode.Str(JsonBase64.encode(value))

## Decodes a proto3 canonical JSON document into a new BytesValue message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[BytesValue]:
	var _pb_message: BytesValue = BytesValue.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[BytesValue].fail(_pb_error.message, _pb_error.path)
	return JsonResult[BytesValue].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	var (_pb_value_value, _pb_value_error) = _pb_json_read_bytes(_pb_node, "$")
	if _pb_value_error is JsonDecodeError:
		return _pb_value_error
	value = _pb_value_value
	return null

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
