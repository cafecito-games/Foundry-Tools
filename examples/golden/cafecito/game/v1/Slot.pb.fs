namespace cafecito.game.v1
import foundry.proto

## An equippable inventory slot.
final class_name Slot extends RefCounted uses Message

## The label protobuf field.
var label: String = ""

## The quantity protobuf field.
var quantity: int = 0

## Decodes protobuf wire data into a new Slot message.
static func from_bytes(data: PackedByteArray) -> (Slot?, ProtobufError):
	var message: Slot = Slot.new()
	var error: ProtobufError = message.merge_from_bytes(data)
	if error != ProtobufError.OK:
		var failed: Slot? = null
		return (failed, error)
	return (message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	if label != "":
		result.append_array(Wire.encode_varint(10))
		var label_data: PackedByteArray = Wire.encode_string(label)
		result.append_array(Wire.encode_varint(label_data.size()))
		result.append_array(label_data)
	if quantity != 0:
		result.append_array(Wire.encode_varint(16))
		result.append_array(Wire.encode_varint(quantity))
	return result

## Merges protobuf wire data into this message.
func merge_from_bytes(data: PackedByteArray) -> ProtobufError:
	var offset: int = 0
	while offset < data.size():
		var tag_read: VarintRead = Wire.decode_varint(data, offset)
		if tag_read.error != ProtobufError.OK:
			return tag_read.error
		offset = tag_read.offset
		var wire_type: int = Wire.get_wire_type(tag_read.value)
		match Wire.get_field_number(tag_read.value):
			1:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var label_length: VarintRead = Wire.decode_varint(data, offset)
				if label_length.error != ProtobufError.OK:
					return label_length.error
				offset = label_length.offset
				if label_length.value < 0 or offset + label_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var label_read: StringRead = Wire.decode_string(data, offset, label_length.value)
				if label_read.error != ProtobufError.OK:
					return label_read.error
				label = label_read.value
				offset = label_read.offset
			2:
				if wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var quantity_read: VarintRead = Wire.decode_varint(data, offset)
				if quantity_read.error != ProtobufError.OK:
					return quantity_read.error
				quantity = quantity_read.value
				offset = quantity_read.offset
			_:
				var skipped: SkipRead = Wire.skip_field(data, offset, wire_type)
				if skipped.error != ProtobufError.OK:
					return skipped.error
				offset = skipped.offset
	return ProtobufError.OK
