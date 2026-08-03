namespace foundry.proto.wkt
import foundry.proto

## `ListValue` is a wrapper around a repeated field of values.
## The JSON representation for `ListValue` is JSON array.
final class_name ListValue extends RefCounted uses Message, JsonSerializable

## Repeated field of dynamically typed values.
var values: Array[Value] = []

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> ListValue:
	return ListValue.new()

static func protobuf_type_name() -> String:
	return "google.protobuf.ListValue"

func type_name() -> String:
	return ListValue.protobuf_type_name()

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

## Returns this ListValue as a native Array.
func to_array() -> Array[Variant]:
	var _pb_result: Array[Variant] = []
	for _pb_item: Value in values:
		_pb_result.append(_pb_item.to_variant())
	return _pb_result

## Converts a native Array into a ListValue.
## A nested failure returns no partial ListValue.
static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError):
	var _pb_ancestors: Array[Variant] = []
	return ListValue._from_array(_pb_value, _pb_ancestors)

static func _from_array(_pb_value: Array[Variant], _pb_ancestors: Array[Variant]) -> (ListValue?, ProtobufError):
	var _pb_failed: ListValue? = null
	for _pb_ancestor: Variant in _pb_ancestors:
		if is_same(_pb_ancestor, _pb_value):
			return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
	_pb_ancestors.append(_pb_value)
	var _pb_result: ListValue = ListValue.new()
	for _pb_item: Variant in _pb_value:
		var (_pb_converted, _pb_error) = Value._from_variant(_pb_item, _pb_ancestors)
		if _pb_error != ProtobufError.OK:
			_pb_ancestors.pop_back()
			return (_pb_failed, _pb_error)
		if not (_pb_converted is Value):
			_pb_ancestors.pop_back()
			return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
		_pb_result.values.append(_pb_converted)
	_pb_ancestors.pop_back()
	return (_pb_result, ProtobufError.OK)

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
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[ListValue]:
	var _pb_message: ListValue = ListValue.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[ListValue].fail(_pb_error.message, _pb_error.path)
	return JsonResult[ListValue].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	match _pb_node:
		JsonNode.Null:
			values = []
		JsonNode.Array(var _pb_values_items):
			values = []
			var _pb_values_index: int = 0
			while _pb_values_index < _pb_values_items.size():
				var _pb_values_element_result: JsonResult[Value] = Value.from_json(_pb_values_items[_pb_values_index])
				var _pb_values_element_error: JsonDecodeError? = _pb_values_element_result.error
				if _pb_values_element_error is JsonDecodeError:
					return JsonResult[ListValue].nested(_pb_values_element_error, str(_pb_values_index)).error
				var _pb_values_element_value: Value? = _pb_values_element_result.value
				if not (_pb_values_element_value is Value):
					return JsonDecodeError.create("JSON_TYPE_MISMATCH: Value decoded to no value", "$." + str(_pb_values_index))
				values.append(_pb_values_element_value)
				_pb_values_index += 1
		_:
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: values expects a JSON array", "$")
	return null
