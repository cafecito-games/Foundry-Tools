namespace foundry.proto.wkt
import foundry.proto

## A generic empty message that you can re-use to avoid defining duplicated
## empty messages in your APIs. A typical example is to use it as the request
## or the response type of an API method. For instance:
## service Foo {
## rpc Bar(google.protobuf.Empty) returns (google.protobuf.Empty);
## }
final class_name Empty extends RefCounted uses Message, JsonSerializable

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Empty message.
static func from_bytes(_pb_data: PackedByteArray) -> (Empty?, ProtobufError):
	var _pb_message: Empty = Empty.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Empty? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
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
	return JsonNode.object_of(_pb_json)

## Decodes a proto3 canonical JSON document into a new Empty message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Empty]:
	var _pb_message: Empty = Empty.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Empty].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Empty].ok(_pb_message)

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
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Empty expects a JSON object", "$")
	for _pb_key: String in _pb_entries:
		var _pb_member_path: String = "$." + _pb_key
		return JsonDecodeError.create("JSON_UNKNOWN_FIELD: Empty has no field named " + _pb_key, _pb_member_path)
	return null
