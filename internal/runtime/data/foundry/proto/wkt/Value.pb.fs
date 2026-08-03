namespace foundry.proto.wkt
import foundry.proto

## `Value` represents a dynamically typed value which can be either
## null, a number, a string, a boolean, a recursive struct value, or a
## list of values. A producer of value is expected to set one of these
## variants. Absence of any variant indicates an error.
## The JSON representation for `Value` is JSON value.
final class_name Value extends RefCounted uses Message, JsonSerializable

## The kind of value.
var kind: ValueKindCase? = null:
	set(_pb_value):
		_pb_kind_null_value_unknown = PackedByteArray()
		kind = _pb_value

## Raw bytes of an unrecognized null_value value, kept so a re-encode is lossless.
var _pb_kind_null_value_unknown: PackedByteArray = PackedByteArray()

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> Value:
	return Value.new()

static func protobuf_type_name() -> String:
	return "google.protobuf.Value"

func type_name() -> String:
	return Value.protobuf_type_name()

static func _pb_any_uses_value() -> bool:
	return true

## Decodes protobuf wire data into a new Value message.
static func from_bytes(_pb_data: PackedByteArray) -> (Value?, ProtobufError):
	var _pb_message: Value = Value.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Value? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if _pb_kind_null_value_unknown.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_result.append_array(_pb_kind_null_value_unknown)
	else:
		match kind:
			ValueKindCase.NullValue(var _pb_kind_null_value):
				_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
				_pb_result.append_array(Wire.encode_varint(_pb_kind_null_value.to_wire()))
			ValueKindCase.NumberValue(var _pb_kind_number_value):
				_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_64BIT)))
				_pb_result.append_array(Wire.encode_double(_pb_kind_number_value))
			ValueKindCase.StringValue(var _pb_kind_string_value):
				_pb_result.append_array(Wire.encode_varint(Wire.make_tag(3, Wire.WIRE_LENGTH_DELIMITED)))
				var _pb_kind_string_value_data: PackedByteArray = Wire.encode_string(_pb_kind_string_value)
				_pb_result.append_array(Wire.encode_varint(_pb_kind_string_value_data.size()))
				_pb_result.append_array(_pb_kind_string_value_data)
			ValueKindCase.BoolValue(var _pb_kind_bool_value):
				_pb_result.append_array(Wire.encode_varint(Wire.make_tag(4, Wire.WIRE_VARINT)))
				_pb_result.append_array(Wire.encode_varint(1 if _pb_kind_bool_value else 0))
			ValueKindCase.StructValue(var _pb_kind_struct_value):
				var _pb_kind_struct_value_data: PackedByteArray = _pb_kind_struct_value.to_bytes()
				_pb_result.append_array(Wire.encode_varint(Wire.make_tag(5, Wire.WIRE_LENGTH_DELIMITED)))
				_pb_result.append_array(Wire.encode_varint(_pb_kind_struct_value_data.size()))
				_pb_result.append_array(_pb_kind_struct_value_data)
			ValueKindCase.ListValue(var _pb_kind_list_value):
				var _pb_kind_list_value_data: PackedByteArray = _pb_kind_list_value.to_bytes()
				_pb_result.append_array(Wire.encode_varint(Wire.make_tag(6, Wire.WIRE_LENGTH_DELIMITED)))
				_pb_result.append_array(Wire.encode_varint(_pb_kind_list_value_data.size()))
				_pb_result.append_array(_pb_kind_list_value_data)
			_:
				pass
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
				var _pb_kind_null_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_kind_null_value_read.error != ProtobufError.OK:
					return _pb_kind_null_value_read.error
				var _pb_kind_null_value_case: NullValue? = NullValue.from_wire(_pb_kind_null_value_read.value)
				if _pb_kind_null_value_case is NullValue:
					kind = ValueKindCase.NullValue(_pb_kind_null_value_case)
				else:
					kind = null
					_pb_kind_null_value_unknown = _pb_data.slice(_pb_offset, _pb_kind_null_value_read.offset)
				_pb_offset = _pb_kind_null_value_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_64BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_kind_number_value_read: FloatRead = Wire.read_double(_pb_data, _pb_offset)
				if _pb_kind_number_value_read.error != ProtobufError.OK:
					return _pb_kind_number_value_read.error
				kind = ValueKindCase.NumberValue(_pb_kind_number_value_read.value)
				_pb_offset = _pb_kind_number_value_read.offset
			3:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_kind_string_value_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_kind_string_value_read.error != ProtobufError.OK:
					return _pb_kind_string_value_read.error
				kind = ValueKindCase.StringValue(_pb_kind_string_value_read.value)
				_pb_offset = _pb_kind_string_value_read.offset
			4:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_kind_bool_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_kind_bool_value_read.error != ProtobufError.OK:
					return _pb_kind_bool_value_read.error
				kind = ValueKindCase.BoolValue(_pb_kind_bool_value_read.value != 0)
				_pb_offset = _pb_kind_bool_value_read.offset
			5:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_kind_struct_value_message: Struct = Struct.new()
				var _pb_kind_struct_value_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_kind_struct_value_message)
				if _pb_kind_struct_value_read.error != ProtobufError.OK:
					return _pb_kind_struct_value_read.error
				kind = ValueKindCase.StructValue(_pb_kind_struct_value_message)
				_pb_offset = _pb_kind_struct_value_read.offset
			6:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_kind_list_value_message: ListValue = ListValue.new()
				var _pb_kind_list_value_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_kind_list_value_message)
				if _pb_kind_list_value_read.error != ProtobufError.OK:
					return _pb_kind_list_value_read.error
				kind = ValueKindCase.ListValue(_pb_kind_list_value_message)
				_pb_offset = _pb_kind_list_value_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK

## Returns this protobuf Value as its native Foundry representation.
##
## An unset Value is represented as null, matching its proto3 JSON form.
func to_variant() -> Variant:
	match kind:
		ValueKindCase.NullValue(_):
			return null
		ValueKindCase.NumberValue(var _pb_kind_number_value):
			return _pb_kind_number_value
		ValueKindCase.StringValue(var _pb_kind_string_value):
			return _pb_kind_string_value
		ValueKindCase.BoolValue(var _pb_kind_bool_value):
			return _pb_kind_bool_value
		ValueKindCase.StructValue(var _pb_kind_struct_value):
			return _pb_kind_struct_value.to_dictionary()
		ValueKindCase.ListValue(var _pb_kind_list_value):
			return _pb_kind_list_value.to_array()
		_:
			return null

## Converts a native Foundry value into a protobuf Value.
##
## An int narrows to Value's float and is lossy beyond 2^53. Unsupported
## Variant kinds and Dictionaries with non-String keys are refused.
static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError):
	var _pb_ancestors: Array[Variant] = []
	return Value._from_variant(_pb_value, _pb_ancestors)

static func _from_variant(_pb_value: Variant, _pb_ancestors: Array[Variant]) -> (Value?, ProtobufError):
	var _pb_failed: Value? = null
	var _pb_result: Value = Value.new()
	match typeof(_pb_value):
		TYPE_NIL:
			_pb_result.kind = ValueKindCase.NullValue(NullValue.NULL_VALUE)
		TYPE_BOOL:
			_pb_result.kind = ValueKindCase.BoolValue(_pb_value)
		TYPE_INT:
			_pb_result.kind = ValueKindCase.NumberValue(float(_pb_value))
		TYPE_FLOAT:
			_pb_result.kind = ValueKindCase.NumberValue(_pb_value)
		TYPE_STRING:
			_pb_result.kind = ValueKindCase.StringValue(_pb_value)
		TYPE_DICTIONARY:
			var (_pb_converted, _pb_error) = Struct._from_dictionary(_pb_value, _pb_ancestors)
			if _pb_error != ProtobufError.OK:
				return (_pb_failed, _pb_error)
			if not (_pb_converted is Struct):
				return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
			_pb_result.kind = ValueKindCase.StructValue(_pb_converted)
		TYPE_ARRAY:
			var (_pb_converted, _pb_error) = ListValue._from_array(_pb_value, _pb_ancestors)
			if _pb_error != ProtobufError.OK:
				return (_pb_failed, _pb_error)
			if not (_pb_converted is ListValue):
				return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
			_pb_result.kind = ValueKindCase.ListValue(_pb_converted)
		_:
			return (_pb_failed, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)
	return (_pb_result, ProtobufError.OK)

## Returns this message as a proto3 canonical JSON document.
##
## JSON.stringify(message, "", false) renders it as text; the third argument
## turns off key sorting, which keeps members in field declaration order.
func to_json() -> JsonNode:
	match kind:
		ValueKindCase.NullValue(var _pb_kind_null_value):
			return JsonNode.Null
		ValueKindCase.NumberValue(var _pb_kind_number_value):
			return _pb_json_float(_pb_kind_number_value)
		ValueKindCase.StringValue(var _pb_kind_string_value):
			return JsonNode.Str(_pb_kind_string_value)
		ValueKindCase.BoolValue(var _pb_kind_bool_value):
			return JsonNode.Bool(_pb_kind_bool_value)
		ValueKindCase.StructValue(var _pb_kind_struct_value):
			return _pb_kind_struct_value.to_json()
		ValueKindCase.ListValue(var _pb_kind_list_value):
			return _pb_kind_list_value.to_json()
		_:
			return JsonNode.Null

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

## Decodes a proto3 canonical JSON document into a new Value message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Value]:
	var _pb_message: Value = Value.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Value].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Value].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	match _pb_node:
		JsonNode.Null:
			kind = ValueKindCase.NullValue(NullValue.NULL_VALUE)
		JsonNode.Bool(var _pb_bool):
			kind = ValueKindCase.BoolValue(_pb_bool)
		JsonNode.Int(var _pb_int):
			kind = ValueKindCase.NumberValue(_pb_int)
		JsonNode.Float(var _pb_float):
			kind = ValueKindCase.NumberValue(_pb_float)
		JsonNode.Str(var _pb_text):
			kind = ValueKindCase.StringValue(_pb_text)
		JsonNode.Array(var _pb_items):
			var _pb_kind_list_value_result: JsonResult[ListValue] = ListValue.from_json(JsonNode.array_of(_pb_items))
			var _pb_kind_list_value_error: JsonDecodeError? = _pb_kind_list_value_result.error
			if _pb_kind_list_value_error is JsonDecodeError:
				return _pb_kind_list_value_error
			var _pb_kind_list_value_value: ListValue? = _pb_kind_list_value_result.value
			if not (_pb_kind_list_value_value is ListValue):
				return JsonDecodeError.create("JSON_TYPE_MISMATCH: ListValue decoded to no value", "$")
			kind = ValueKindCase.ListValue(_pb_kind_list_value_value)
		JsonNode.Object(var _pb_object):
			var _pb_kind_struct_value_result: JsonResult[Struct] = Struct.from_json(JsonNode.object_of(_pb_object))
			var _pb_kind_struct_value_error: JsonDecodeError? = _pb_kind_struct_value_result.error
			if _pb_kind_struct_value_error is JsonDecodeError:
				return _pb_kind_struct_value_error
			var _pb_kind_struct_value_value: Struct? = _pb_kind_struct_value_result.value
			if not (_pb_kind_struct_value_value is Struct):
				return JsonDecodeError.create("JSON_TYPE_MISMATCH: Struct decoded to no value", "$")
			kind = ValueKindCase.StructValue(_pb_kind_struct_value_value)
		_:
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Value cannot represent this JSON value", "$")
	return null
