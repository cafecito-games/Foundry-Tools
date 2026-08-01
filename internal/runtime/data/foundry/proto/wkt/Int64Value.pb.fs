namespace foundry.proto.wkt
import foundry.proto

## Wrapper message for `int64`.
## The JSON representation for `Int64Value` is JSON string.
## Not recommended for use in new APIs, but still useful for legacy APIs and
## has no plan to be removed.
final class_name Int64Value extends RefCounted uses Message, JsonSerializable

## The int64 value.
var value: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

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
## Not generated yet: this reports a failure for every document. The
## conformance it completes is what makes to_json reachable through
## JSON.stringify, which is why the member exists ahead of the decoder.
static func from_json(_pb_node: JsonNode) -> JsonResult[Int64Value]:
	return JsonResult[Int64Value].fail("JSON_PARSE_FAILED: Int64Value cannot be decoded from JSON yet", "$")
