namespace foundry.proto.wkt
import foundry.proto

## `ListValue` is a wrapper around a repeated field of values.
## The JSON representation for `ListValue` is JSON array.
final class_name ListValue extends RefCounted uses Message, JsonSerializable

## Repeated field of dynamically typed values.
var values: Array[Value] = []

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new ListValue message.
static func from_bytes(_pb_data: PackedByteArray) -> (ListValue?, ProtobufError):
	var _pb_message: ListValue = ListValue.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: ListValue? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	for _pb_values_item: Value in values:
		var _pb_values_data: PackedByteArray = _pb_values_item.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_values_data.size()))
		_pb_result.append_array(_pb_values_data)
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
				var _pb_values_message: Value = Value.new()
				var _pb_values_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_values_message)
				if _pb_values_read.error != ProtobufError.OK:
					return _pb_values_read.error
				values.append(_pb_values_message)
				_pb_offset = _pb_values_read.offset
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
	var _pb_values_items: Array[JsonNode] = []
	for _pb_values_item: Value in values:
		_pb_values_items.append(_pb_values_item.to_json())
	return JsonNode.array_of(_pb_values_items)

## Decodes a proto3 canonical JSON document into a new ListValue message.
##
## Not generated yet: this reports a failure for every document. The
## conformance it completes is what makes to_json reachable through
## JSON.stringify, which is why the member exists ahead of the decoder.
static func from_json(_pb_node: JsonNode) -> JsonResult[ListValue]:
	return JsonResult[ListValue].fail("JSON_PARSE_FAILED: ListValue cannot be decoded from JSON yet", "$")
