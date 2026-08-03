namespace foundry.proto.wkt
import foundry.proto

## `Struct` represents a structured data value, consisting of fields
## which map to dynamically typed values. In some languages, `Struct`
## might be supported by a native representation. For example, in
## scripting languages like JS a struct is represented as an
## object. The details of that representation are described together
## with the proto support for the language.
## The JSON representation for `Struct` is JSON object.
final class_name Struct extends RefCounted uses Message, JsonSerializable

## Unordered map of dynamically typed values.
var fields: Dictionary[String, Value] = {}

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> Struct:
	return Struct.new()

static func protobuf_type_name() -> String:
	return "google.protobuf.Struct"

func type_name() -> String:
	return Struct.protobuf_type_name()

static func _pb_any_uses_value() -> bool:
	return true

## Decodes protobuf wire data into a new Struct message.
static func from_bytes(_pb_data: PackedByteArray) -> (Struct?, ProtobufError):
	var _pb_message: Struct = Struct.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Struct? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	for _pb_fields_key: String in fields:
		var _pb_fields_entry: PackedByteArray = PackedByteArray()
		_pb_fields_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_fields_key_data: PackedByteArray = Wire.encode_string(_pb_fields_key)
		_pb_fields_entry.append_array(Wire.encode_varint(_pb_fields_key_data.size()))
		_pb_fields_entry.append_array(_pb_fields_key_data)
		var _pb_fields_value_data: PackedByteArray = fields[_pb_fields_key].to_bytes()
		_pb_fields_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_fields_entry.append_array(Wire.encode_varint(_pb_fields_value_data.size()))
		_pb_fields_entry.append_array(_pb_fields_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_fields_entry.size()))
		_pb_result.append_array(_pb_fields_entry)
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
				var _pb_fields_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_fields_length.error != ProtobufError.OK:
					return _pb_fields_length.error
				var _pb_fields_end: int = _pb_fields_length.offset + _pb_fields_length.value
				_pb_offset = _pb_fields_length.offset
				var _pb_fields_key: String = ""
				var _pb_fields_value: Value = Value.new()
				while _pb_offset < _pb_fields_end:
					var _pb_fields_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_fields_entry_tag.error != ProtobufError.OK:
						return _pb_fields_entry_tag.error
					if _pb_fields_entry_tag.offset > _pb_fields_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_fields_entry_tag.offset
					var _pb_fields_entry_wire_type: int = Wire.get_wire_type(_pb_fields_entry_tag.value)
					match Wire.get_field_number(_pb_fields_entry_tag.value):
						1:
							if _pb_fields_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_fields_key_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_fields_key_read.error != ProtobufError.OK:
								return _pb_fields_key_read.error
							if _pb_fields_key_read.offset > _pb_fields_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_fields_key = _pb_fields_key_read.value
							_pb_offset = _pb_fields_key_read.offset
						2:
							if _pb_fields_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_fields_value_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_fields_value)
							if _pb_fields_value_read.error != ProtobufError.OK:
								return _pb_fields_value_read.error
							if _pb_fields_value_read.offset > _pb_fields_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_fields_value_read.offset
						_:
							var _pb_fields_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_fields_entry_wire_type)
							if _pb_fields_skip.error != ProtobufError.OK:
								return _pb_fields_skip.error
							if _pb_fields_skip.offset > _pb_fields_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_fields_skip.offset
				fields[_pb_fields_key] = _pb_fields_value
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK

## Returns this Struct as a native string-keyed Dictionary.
##
## Every Value case has a native representation, so this direction is total.
func to_dictionary() -> Dictionary[String, Variant]:
	var _pb_result: Dictionary[String, Variant] = {}
	for _pb_key: String in fields:
		_pb_result[_pb_key] = fields[_pb_key].to_variant()
	return _pb_result

## Converts a native Dictionary into a Struct.
##
## Keys must be Strings and every value must have a protobuf Value mapping.
## A nested failure returns no partial Struct.
static func from_dictionary(_pb_value: Dictionary) -> (Struct?, ProtobufError):
	var _pb_ancestors: Array[Variant] = []
	return Struct._from_dictionary(_pb_value, _pb_ancestors)

static func _from_dictionary(_pb_value: Dictionary, _pb_ancestors: Array[Variant]) -> (Struct?, ProtobufError):
	var _pb_failed: Struct? = null
	for _pb_ancestor: Variant in _pb_ancestors:
		if is_same(_pb_ancestor, _pb_value):
			return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
	_pb_ancestors.append(_pb_value)
	var _pb_result: Struct = Struct.new()
	for _pb_key: Variant in _pb_value:
		if typeof(_pb_key) != TYPE_STRING:
			_pb_ancestors.pop_back()
			return (_pb_failed, ProtobufError.STRUCT_KEY_NOT_STRING)
		var (_pb_converted, _pb_error) = Value._from_variant(_pb_value[_pb_key], _pb_ancestors)
		if _pb_error != ProtobufError.OK:
			_pb_ancestors.pop_back()
			return (_pb_failed, _pb_error)
		if not (_pb_converted is Value):
			_pb_ancestors.pop_back()
			return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
		_pb_result.fields[str(_pb_key)] = _pb_converted
	_pb_ancestors.pop_back()
	return (_pb_result, ProtobufError.OK)

## Returns this message as a proto3 canonical JSON document.
##
## JSON.stringify(message, "", false) renders it as text; the third argument
## turns off key sorting, which keeps members in field declaration order.
func to_json() -> JsonNode:
	var _pb_json: Dictionary[String, JsonNode] = {}
	for _pb_fields_key: String in fields:
		_pb_json[_pb_fields_key] = fields[_pb_fields_key].to_json()
	return JsonNode.object_of(_pb_json)

## Decodes a proto3 canonical JSON document into a new Struct message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Struct]:
	var _pb_message: Struct = Struct.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Struct].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Struct].ok(_pb_message)

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
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Struct expects a JSON object", "$")
	fields = {}
	for _pb_fields_key: String in _pb_entries:
		var _pb_fields_result: JsonResult[Value] = Value.from_json(_pb_entries[_pb_fields_key])
		var _pb_fields_error: JsonDecodeError? = _pb_fields_result.error
		if _pb_fields_error is JsonDecodeError:
			return JsonResult[Struct].nested(_pb_fields_error, _pb_fields_key).error
		var _pb_fields_value: Value? = _pb_fields_result.value
		if not (_pb_fields_value is Value):
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Value decoded to no value", "$." + _pb_fields_key)
		fields[_pb_fields_key] = _pb_fields_value
	return null
