namespace foundry.proto.wkt
import foundry.proto

## `Value` represents a dynamically typed value which can be either
## null, a number, a string, a boolean, a recursive struct value, or a
## list of values. A producer of value is expected to set one of these
## variants. Absence of any variant indicates an error.
## The JSON representation for `Value` is JSON value.
final class_name Value extends RefCounted uses Message

## The kind of value.
var kind: ValueKindCase? = null:
	set(_pb_value):
		_pb_kind_null_value_unknown = PackedByteArray()
		kind = _pb_value

## Raw bytes of an unrecognized null_value value, kept so a re-encode is lossless.
var _pb_kind_null_value_unknown: PackedByteArray = PackedByteArray()

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

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
