## Both divisions below are on two integers and want the truncated quotient, so
## the decimal-part warning would fire on each one and say nothing. Suppressing
## it here keeps the runtime quiet in the projects that embed it.
@warning_ignore_start("INTEGER_DIVISION")

namespace foundry.proto

## Unsigned decimal text for the canonical JSON mapping of a uint64 or fixed64
## field.
##
## A ulong carries the full unsigned value directly, so formatting is str() and
## parsing is digit-by-digit assembly. The widest value is 18446744073709551615
## (2^64-1), and a checked multiply/add in the loop refuses anything larger.
class_name JsonUint64 extends RefCounted

## The largest value this can carry, and the only text length whose digits have
## to be compared against it.
const MAXIMUM_TEXT: String = "18446744073709551615"

## Returns the unsigned decimal the given value stands for.
static func format(value: ulong) -> String:
	return str(value)

## Returns the value the given unsigned decimal text stands for.
##
## The text has to be exactly what format() would print: digits only, no sign,
## and no leading zero. Anything else either is not a canonical JSON integer or
## would come back as different text, and a value past the unsigned range is
## refused rather than wrapped.
static func parse(text: String) -> (ulong, ProtobufError):
	if not _is_canonical_decimal(text):
		return (0UL, ProtobufError.JSON_TYPE_MISMATCH)
	if not _is_in_range(text):
		return (0UL, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	var value: ulong = 0UL
	var index: int = 0
	while index < text.length():
		value = value * 10UL + (text.substr(index, 1).to_int() as ulong)
		index += 1
	return (value, ProtobufError.OK)

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
