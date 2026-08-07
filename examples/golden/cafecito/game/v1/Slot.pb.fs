namespace cafecito.game.v1
import foundry.proto

## An equippable inventory slot.
final class_name Slot extends RefCounted uses Message

## The label protobuf field.
var label: String = ""

## The quantity protobuf field.
var quantity: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

static func create_message() -> Slot:
	return Slot.new()

static func protobuf_type_name() -> String:
	return "cafecito.game.v1.Slot"

func type_name() -> String:
	return Slot.protobuf_type_name()

static func _pb_any_uses_value() -> bool:
	return false

## Decodes protobuf wire data into a new Slot message.
static func from_bytes(_pb_data: PackedByteArray) -> (Slot?, ProtobufError):
	var _pb_message: Slot = Slot.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Slot? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if label != "":
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_label_data: PackedByteArray = Wire.encode_string(label)
		_pb_result.append_array(Wire.encode_varint(_pb_label_data.size()))
		_pb_result.append_array(_pb_label_data)
	if quantity != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(quantity))
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
				var _pb_label_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_label_read.error != ProtobufError.OK:
					return _pb_label_read.error
				label = _pb_label_read.value
				_pb_offset = _pb_label_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_quantity_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_quantity_read.error != ProtobufError.OK:
					return _pb_quantity_read.error
				quantity = _pb_quantity_read.value as int
				_pb_offset = _pb_quantity_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK
