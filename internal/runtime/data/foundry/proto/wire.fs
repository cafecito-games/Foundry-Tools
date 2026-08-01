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

## Encodes the low four bytes of value, little-endian. fixed32 and sfixed32
## put the same bytes on the wire and differ only in how a reader interprets
## the top bit, so both encode through here.
static func encode_fixed32(value: int) -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	result.append(value & 0xFF)
	result.append((value >> 8) & 0xFF)
	result.append((value >> 16) & 0xFF)
	result.append((value >> 24) & 0xFF)
	return result

## Encodes all eight bytes of value, little-endian. As with the 32-bit pair,
## fixed64 and sfixed64 share this and differ only on the way back out.
static func encode_fixed64(value: int) -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	var remaining: int = value
	var index: int = 0
	while index < 8:
		result.append(remaining & 0xFF)
		# Masking each byte makes the arithmetic shift's sign extension moot.
		remaining = remaining >> 8
		index += 1
	return result

## Encodes value as IEEE-754 binary32. Foundry's float is 64-bit, so this
## narrows: a proto float field cannot hold more precision than binary32, and
## rounding here rather than at the reader is what every implementation does.
static func encode_float(value: float) -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	result.resize(4)
	result.encode_float(0, value)
	return result

## Encodes value as IEEE-754 binary64, which is Foundry's float exactly.
static func encode_double(value: float) -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	result.resize(8)
	result.encode_double(0, value)
	return result

## Reports whether value is the default proto3 writes nothing for.
##
## For a float that default is +0.0 specifically. -0.0 is a distinct value that
## protobuf puts on the wire, and `value != 0.0` answers false for both of
## them, so the sign is recovered from a division: it is the one operation that
## still tells the two zeroes apart. The division only ever runs for a zero.
static func is_default_float(value: float) -> bool:
	if value != 0.0:
		return false
	return 1.0 / value > 0.0

## The same question for a proto float, which is binary32 while Foundry's float
## is binary64. A value too small for binary32 narrows to +0.0 when it is
## written, so presence has to be decided on what actually goes on the wire;
## deciding it on the wider value would emit a field protobuf omits.
static func is_default_float32(value: float) -> bool:
	if is_default_float(value):
		return true
	return is_default_float(narrow_float32(value))

## Narrows value to the binary32 a proto float actually carries.
##
## A member holding a proto float is a Foundry float, which is binary64, so it
## can carry precision the field cannot. The encoder drops that precision on
## the way to the wire; anything else that reports the field's value -- the
## canonical JSON mapping above all -- has to drop it the same way, or two
## renderings of one field disagree about what it holds.
static func narrow_float32(value: float) -> float:
	var narrowed: PackedByteArray = PackedByteArray()
	narrowed.resize(4)
	narrowed.encode_float(0, value)
	return narrowed.decode_float(0)

## Encodes value as a zig-zag varint over 32 bits. Zig-zag maps small negatives
## onto small unsigned numbers, so -1 costs one byte instead of the ten a plain
## varint spends sign-extending it.
static func encode_sint32(value: int) -> PackedByteArray:
	return encode_varint(zigzag_encode_32(value))

## Encodes value as a zig-zag varint over 64 bits.
static func encode_sint64(value: int) -> PackedByteArray:
	return encode_varint(zigzag_encode_64(value))

static func zigzag_encode_32(value: int) -> int:
	return ((value << 1) ^ (value >> 31)) & 0xFFFFFFFF

static func zigzag_encode_64(value: int) -> int:
	return (value << 1) ^ (value >> 63)

static func zigzag_decode_32(value: int) -> int:
	var encoded: int = value & 0xFFFFFFFF
	return (encoded >> 1) ^ -(encoded & 1)

static func zigzag_decode_64(value: int) -> int:
	# The mask stands in for a logical shift; Foundry's >> is arithmetic, and
	# the top bit is a value bit here rather than a sign.
	return ((value >> 1) & 0x7FFFFFFFFFFFFFFF) ^ -(value & 1)

## Reads four bytes as an unsigned 32-bit value, so fixed32 spans 0 to 2^32-1
## rather than wrapping negative.
static func read_fixed32(data: PackedByteArray, offset: int) -> FixedRead:
	if offset + 4 > data.size():
		return FixedRead(0, offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return FixedRead(data.decode_u32(offset), offset + 4, ProtobufError.OK)

## Reads four bytes as a signed 32-bit value.
static func read_sfixed32(data: PackedByteArray, offset: int) -> FixedRead:
	if offset + 4 > data.size():
		return FixedRead(0, offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return FixedRead(data.decode_s32(offset), offset + 4, ProtobufError.OK)

## Reads eight bytes. fixed64 and sfixed64 both land here: Foundry's int is
## signed 64-bit, so the bits are preserved either way and only the printed
## value of the top half of the fixed64 range differs from its proto meaning.
static func read_fixed64(data: PackedByteArray, offset: int) -> FixedRead:
	if offset + 8 > data.size():
		return FixedRead(0, offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return FixedRead(data.decode_s64(offset), offset + 8, ProtobufError.OK)

## Reads four bytes as IEEE-754 binary32, widened into Foundry's 64-bit float.
static func read_float(data: PackedByteArray, offset: int) -> FloatRead:
	if offset + 4 > data.size():
		return FloatRead(0.0, offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return FloatRead(data.decode_float(offset), offset + 4, ProtobufError.OK)

## Reads eight bytes as IEEE-754 binary64.
static func read_double(data: PackedByteArray, offset: int) -> FloatRead:
	if offset + 8 > data.size():
		return FloatRead(0.0, offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	return FloatRead(data.decode_double(offset), offset + 8, ProtobufError.OK)

## Reads a zig-zag varint over 32 bits.
static func read_sint32(data: PackedByteArray, offset: int) -> VarintRead:
	var read: VarintRead = decode_varint(data, offset)
	if read.error != ProtobufError.OK:
		return read
	return VarintRead(zigzag_decode_32(read.value), read.offset, ProtobufError.OK)

## Reads a zig-zag varint over 64 bits.
static func read_sint64(data: PackedByteArray, offset: int) -> VarintRead:
	var read: VarintRead = decode_varint(data, offset)
	if read.error != ProtobufError.OK:
		return read
	return VarintRead(zigzag_decode_64(read.value), read.offset, ProtobufError.OK)

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
