namespace foundry.proto

class_name Wire extends RefCounted

const WIRE_VARINT: int = 0
const WIRE_64BIT: int = 1
const WIRE_LENGTH_DELIMITED: int = 2
const WIRE_32BIT: int = 5

static func make_tag(field_number: int, wire_type: int) -> int:
	return (field_number << 3) | wire_type

static func get_wire_type(tag: int) -> int:
	return tag & 0x7

static func get_field_number(tag: int) -> int:
	return tag >> 3

static func encode_varint(value: int) -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	var unsigned_value: int = value
	while unsigned_value > 0x7F or unsigned_value < 0:
		result.append((unsigned_value & 0x7F) | 0x80)
		unsigned_value = (unsigned_value >> 7) & 0x01FFFFFFFFFFFFFF
	result.append(unsigned_value & 0x7F)
	return result

static func decode_varint(data: PackedByteArray, offset: int) -> FieldRead[int]:
	var result_value: int = 0
	var shift: int = 0
	var cursor: int = offset
	while cursor < data.size():
		var byte: int = data[cursor]
		result_value |= (byte & 0x7F) << shift
		cursor += 1
		if (byte & 0x80) == 0:
			return FieldRead[int].from(result_value, cursor, ProtobufError.OK)
		shift += 7
		if shift > 63:
			return FieldRead[int].from(0, cursor, ProtobufError.VARINT_TOO_LONG)
	return FieldRead[int].from(0, cursor, ProtobufError.VARINT_NOT_FOUND)

static func encode_string(value: String) -> PackedByteArray:
	return value.to_utf8_buffer()

static func decode_string(data: PackedByteArray, offset: int, length: int) -> FieldRead[String]:
	if offset + length > data.size():
		return FieldRead[String].from("", offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	var slice: PackedByteArray = data.slice(offset, offset + length)
	return FieldRead[String].from(slice.get_string_from_utf8(), offset + length, ProtobufError.OK)
