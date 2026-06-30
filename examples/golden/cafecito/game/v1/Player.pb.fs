namespace cafecito.game.v1
import foundry.proto

## Generated protobuf message binding for Player.
final class_name Player extends RefCounted

var _name: String = ""

## Sets the name protobuf field.
func set_name(value: String) -> void:
	_name = value

## Returns the name protobuf field.
func get_name() -> String:
	return _name

## Decodes protobuf wire data into a new Player message.
static func from_bytes(data: PackedByteArray) -> DecodeResult[Player]:
	var message: Player = Player.new()
	var err: foundry.proto.ProtobufError = message.merge_from_bytes(data)
	return DecodeResult[Player].from(message, err)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	if _name != "":
		result.append_array(foundry.proto.Wire.encode_varint(10))
		var name_data: PackedByteArray = foundry.proto.Wire.encode_string(_name)
		result.append_array(foundry.proto.Wire.encode_varint(name_data.size()))
		result.append_array(name_data)
	return result

## Merges protobuf wire data into this message.
func merge_from_bytes(data: PackedByteArray) -> foundry.proto.ProtobufError:
	var offset: int = 0
	while offset < data.size():
		var tag_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)
		if tag_read.error != foundry.proto.ProtobufError.OK:
			return tag_read.error
		offset = tag_read.offset
		var wire_type: int = foundry.proto.Wire.get_wire_type(tag_read.value)
		var field_number: int = foundry.proto.Wire.get_field_number(tag_read.value)
		match field_number:
			1:
				if wire_type != foundry.proto.Wire.WIRE_LENGTH_DELIMITED:
					return foundry.proto.ProtobufError.WIRE_TYPE_MISMATCH
				var length_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)
				if length_read.error != foundry.proto.ProtobufError.OK:
					return length_read.error
				offset = length_read.offset
				if length_read.value < 0 or offset + length_read.value > data.size():
					return foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var string_read: FieldRead[String] = foundry.proto.Wire.decode_string(data, offset, length_read.value)
				if string_read.error != foundry.proto.ProtobufError.OK:
					return string_read.error
				_name = string_read.value
				offset = string_read.offset
			_:
				match wire_type:
					foundry.proto.Wire.WIRE_VARINT:
						var skipped_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)
						if skipped_read.error != foundry.proto.ProtobufError.OK:
							return skipped_read.error
						offset = skipped_read.offset
					foundry.proto.Wire.WIRE_LENGTH_DELIMITED:
						var length_read: FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)
						if length_read.error != foundry.proto.ProtobufError.OK:
							return length_read.error
						offset = length_read.offset
						if length_read.value < 0 or offset + length_read.value > data.size():
							return foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
						offset += length_read.value
					foundry.proto.Wire.WIRE_32BIT:
						if offset + 4 > data.size():
							return foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
						offset += 4
					foundry.proto.Wire.WIRE_64BIT:
						if offset + 8 > data.size():
							return foundry.proto.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
						offset += 8
					_:
						return foundry.proto.ProtobufError.WIRE_TYPE_MISMATCH
	return foundry.proto.ProtobufError.OK
