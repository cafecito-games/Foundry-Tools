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

static func decode_varint(data: PackedByteArray, offset: int) -> VarintRead:
	var result_value: int = 0
	var shift: int = 0
	var cursor: int = offset
	while cursor < data.size():
		var byte: int = data[cursor]
		result_value |= (byte & 0x7F) << shift
		cursor += 1
		if (byte & 0x80) == 0:
			return VarintRead(result_value, cursor, ProtobufError.OK)
		shift += 7
		if shift > 63:
			return VarintRead(0, cursor, ProtobufError.VARINT_TOO_LONG)
	return VarintRead(0, cursor, ProtobufError.VARINT_NOT_FOUND)

static func encode_string(value: String) -> PackedByteArray:
	return value.to_utf8_buffer()

static func decode_string(data: PackedByteArray, offset: int, length: int) -> StringRead:
	if offset + length > data.size():
		return StringRead("", offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	var slice: PackedByteArray = data.slice(offset, offset + length)
	return StringRead(slice.get_string_from_utf8(), offset + length, ProtobufError.OK)

static func decode_bytes(data: PackedByteArray, offset: int, length: int) -> BytesRead:
	if offset + length > data.size():
		return BytesRead(PackedByteArray(), offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return BytesRead(data.slice(offset, offset + length), offset + length, ProtobufError.OK)

## Reads a length prefix and bounds-checks the payload it announces. The
## returned value is the payload length and the returned offset is where the
## payload starts, so every length-delimited read shares one bounds policy.
static func read_length(data: PackedByteArray, offset: int) -> VarintRead:
	var length: VarintRead = decode_varint(data, offset)
	if length.error != ProtobufError.OK:
		return VarintRead(0, offset, length.error)
	if length.value < 0 or length.offset + length.value > data.size():
		return VarintRead(0, offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return length

## Reads a length-prefixed string field, prefix and payload together.
static func read_string(data: PackedByteArray, offset: int) -> StringRead:
	var length: VarintRead = read_length(data, offset)
	if length.error != ProtobufError.OK:
		return StringRead("", offset, length.error)
	return decode_string(data, length.offset, length.value)

## Reads a length-prefixed bytes field, prefix and payload together.
static func read_bytes(data: PackedByteArray, offset: int) -> BytesRead:
	var length: VarintRead = read_length(data, offset)
	if length.error != ProtobufError.OK:
		return BytesRead(PackedByteArray(), offset, length.error)
	return decode_bytes(data, length.offset, length.value)

## Merges a length-prefixed submessage into target and reports the new offset.
## Merging rather than replacing is what protobuf requires when a singular
## message field appears more than once in the same stream.
static func read_message(data: PackedByteArray, offset: int, target: Message) -> SkipRead:
	var length: VarintRead = read_length(data, offset)
	if length.error != ProtobufError.OK:
		return SkipRead(offset, length.error)
	var end: int = length.offset + length.value
	return SkipRead(end, target.merge_from_bytes(data.slice(length.offset, end)))

## Copies one unrecognized field, tag included, into sink and advances past it.
## proto3 requires unknown fields to survive a decode/re-encode round trip, so
## a peer on a newer schema does not lose data passing through an older binding.
static func capture_field(data: PackedByteArray, offset: int, tag: int, wire_type: int, sink: PackedByteArray) -> SkipRead:
	var skipped: SkipRead = skip_field(data, offset, wire_type)
	if skipped.error != ProtobufError.OK:
		return skipped
	sink.append_array(encode_varint(tag))
	sink.append_array(data.slice(offset, skipped.offset))
	return skipped

## Advances past one unknown field of wire_type and returns the new offset.
## Every generated binding shares this, so the skip policy lives in one place.
static func skip_field(data: PackedByteArray, offset: int, wire_type: int) -> SkipRead:
	match wire_type:
		WIRE_VARINT:
			var skipped: VarintRead = decode_varint(data, offset)
			return SkipRead(skipped.offset, skipped.error)
		WIRE_LENGTH_DELIMITED:
			var length: VarintRead = decode_varint(data, offset)
			if length.error != ProtobufError.OK:
				return SkipRead(offset, length.error)
			if length.value < 0 or length.offset + length.value > data.size():
				return SkipRead(offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
			return SkipRead(length.offset + length.value, ProtobufError.OK)
		WIRE_32BIT:
			if offset + 4 > data.size():
				return SkipRead(offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
			return SkipRead(offset + 4, ProtobufError.OK)
		WIRE_64BIT:
			if offset + 8 > data.size():
				return SkipRead(offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
			return SkipRead(offset + 8, ProtobufError.OK)
		_:
			return SkipRead(offset, ProtobufError.WIRE_TYPE_MISMATCH)
