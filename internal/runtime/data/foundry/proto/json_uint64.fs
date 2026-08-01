## Both divisions below are on two integers and want the truncated quotient, so
## the decimal-part warning would fire on each one and say nothing. Suppressing
## it here keeps the runtime quiet in the projects that embed it.
@warning_ignore_start("INTEGER_DIVISION")

namespace foundry.proto

## Unsigned decimal text for the canonical JSON mapping of a uint64 or fixed64
## field.
##
## A Foundry int is signed, so an unsigned value at or above 2^63 is carried as
## a negative bit pattern and str() would print it with a minus sign: the widest
## uint64 would read as "-1" rather than as its twenty digits. Both directions
## here work on that bit pattern instead, and neither of them ever holds the
## whole unsigned value in an int. A value is split into its quotient by ten and
## its last digit, both of which a signed int holds comfortably, and it is put
## back together by a shift, which moves the top bit into the sign rather than
## overflowing on the way.
class_name JsonUint64 extends RefCounted

## The largest value this can carry, and the only text length whose digits have
## to be compared against it.
const MAXIMUM_TEXT: String = "18446744073709551615"

## Every bit but the sign, which turns Foundry's arithmetic >> into the logical
## shift the unsigned halving needs.
const WITHOUT_SIGN_BIT: int = 0x7FFFFFFFFFFFFFFF

## The bit pattern of 18446744073709551615, the widest value. A bare JSON number
## can only reach it by rounding -- no double holds those digits exactly, and the
## nearest one is 2^64 -- so a reader that accepts a bare number needs the value
## the rounding stands for spelled out.
const WIDEST_BITS: int = -1

## Returns the unsigned decimal the given signed bit pattern stands for.
static func format(value: int) -> String:
	if value >= 0:
		return str(value)
	# floor(unsigned / 2), which always fits: the widest unsigned value halves
	# to exactly the largest signed one.
	var half: int = (value >> 1) & WITHOUT_SIGN_BIT
	var quotient: int = half / 5
	var last_digit: int = 2 * (half - quotient * 5) + (value & 1)
	return str(quotient) + str(last_digit)

## Returns the signed bit pattern the given unsigned decimal text stands for.
##
## The text has to be exactly what format() would print: digits only, no sign,
## and no leading zero. Anything else either is not a canonical JSON integer or
## would come back as different text, and a value past the unsigned range is
## refused rather than wrapped.
static func parse(text: String) -> (int, ProtobufError):
	if not _is_canonical_decimal(text):
		return (0, ProtobufError.JSON_TYPE_MISMATCH)
	if not _is_in_range(text):
		return (0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	# The text without its last digit is floor(unsigned / 10), small enough to
	# parse directly once the range is known.
	var quotient: int = 0
	if text.length() > 1:
		quotient = text.substr(0, text.length() - 1).to_int()
	var last_digit: int = text.substr(text.length() - 1, 1).to_int()
	var half: int = quotient * 5 + last_digit / 2
	return ((half << 1) | (last_digit & 1), ProtobufError.OK)

## A leading zero is refused rather than ignored, matching the signed reader:
## it is text no emitter produces, and accepting it would mean accepting a
## spelling that does not come back the way it went in.
static func _is_canonical_decimal(text: String) -> bool:
	if text.length() == 0:
		return false
	if text.length() > 1 and text.substr(0, 1) == "0":
		return false
	var index: int = 0
	while index < text.length():
		var character: String = text.substr(index, 1)
		if character < "0" or character > "9":
			return false
		index += 1
	return true

## Compares against the widest value digit by digit. The text is canonical, so a
## shorter run of digits is a smaller number and a longer one is out of range
## without anything being parsed.
static func _is_in_range(text: String) -> bool:
	if text.length() < MAXIMUM_TEXT.length():
		return true
	if text.length() > MAXIMUM_TEXT.length():
		return false
	var index: int = 0
	while index < text.length():
		var digit: String = text.substr(index, 1)
		var limit: String = MAXIMUM_TEXT.substr(index, 1)
		if digit != limit:
			return digit < limit
		index += 1
	return true
