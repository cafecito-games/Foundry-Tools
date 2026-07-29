namespace cafecito.game.v1
import foundry.proto

## An equippable inventory slot.
final class_name Slot extends RefCounted uses Message

## The label protobuf field.
var label: String = ""

## The quantity protobuf field.
var quantity: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Slot message.
static func from_bytes(data: PackedByteArray) -> (Slot?, ProtobufError):
	var _message: Slot = Slot.new()
	var _error: ProtobufError = _message.merge_from_bytes(data)
	if _error != ProtobufError.OK:
		var _failed: Slot? = null
		return (_failed, _error)
	return (_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _result: PackedByteArray = PackedByteArray()
	if label != "":
		_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _label_data: PackedByteArray = Wire.encode_string(label)
		_result.append_array(Wire.encode_varint(_label_data.size()))
		_result.append_array(_label_data)
	if quantity != 0:
		_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_result.append_array(Wire.encode_varint(quantity))
	_result.append_array(_unknown_fields)
	return _result

## Merges protobuf wire data into this message.
func merge_from_bytes(_data: PackedByteArray) -> ProtobufError:
	var _offset: int = 0
	while _offset < _data.size():
		var _tag: VarintRead = Wire.decode_varint(_data, _offset)
		if _tag.error != ProtobufError.OK:
			return _tag.error
		_offset = _tag.offset
		var _wire_type: int = Wire.get_wire_type(_tag.value)
		match Wire.get_field_number(_tag.value):
			1:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _label_read: StringRead = Wire.read_string(_data, _offset)
				if _label_read.error != ProtobufError.OK:
					return _label_read.error
				label = _label_read.value
				_offset = _label_read.offset
			2:
				if _wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _quantity_read: VarintRead = Wire.decode_varint(_data, _offset)
				if _quantity_read.error != ProtobufError.OK:
					return _quantity_read.error
				quantity = _quantity_read.value
				_offset = _quantity_read.offset
			_:
				var _skipped: SkipRead = Wire.capture_field(_data, _offset, _tag.value, _wire_type, _unknown_fields)
				if _skipped.error != ProtobufError.OK:
					return _skipped.error
				_offset = _skipped.offset
	return ProtobufError.OK
