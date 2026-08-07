## Every division below is on two integers and wants the truncated quotient, so
## the decimal-part warning would fire on each one and say nothing. Suppressing
## it here keeps the runtime quiet in the projects that embed it.
@warning_ignore_start("INTEGER_DIVISION")

namespace foundry.proto

## Text conversion for the canonical JSON mapping of google.protobuf.Duration.
##
## A Duration stores a signed second count and a signed nanosecond remainder
## whose signs must agree, so the sign is pulled out once and both components
## are formatted from their magnitudes. That is also why a sub-second negative
## duration needs care: its seconds are zero, and only the nanos carry the sign.
class_name JsonDuration extends RefCounted

## The largest magnitude either component may carry, matching the range a
## well-formed Duration message can hold.
const MAXIMUM_SECONDS: long = 315576000000

const MAXIMUM_NANOS: int = 999999999

## The widest whole-seconds run that fits in a signed 64-bit integer without
## overflowing while it is accumulated. A run this wide or narrower is safe to
## parse into a number and then compare against MAXIMUM_SECONDS; a wider one is
## reported as out of range without ever being parsed into a value.
const MAXIMUM_SAFE_DIGITS: int = 18

## The longest well-formed text: a sign, eighteen digits of seconds (matching
## MAXIMUM_SAFE_DIGITS), a point, a nine-digit fraction, and the suffix.
const MAXIMUM_TEXT_LENGTH: int = 30

static func format(seconds: long, nanos: int) -> (String, ProtobufError):
	if seconds > MAXIMUM_SECONDS or seconds < -MAXIMUM_SECONDS:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if nanos > MAXIMUM_NANOS or nanos < -MAXIMUM_NANOS:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if (seconds > 0 and nanos < 0) or (seconds < 0 and nanos > 0):
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var sign_text: String = ""
	if seconds < 0 or nanos < 0:
		sign_text = "-"
	var whole: long = seconds
	if whole < 0:
		whole = -whole
	var fraction: int = nanos
	if fraction < 0:
		fraction = -fraction
	return (sign_text + str(whole) + _format_fraction(fraction) + "s", ProtobufError.OK)

## The canonical form uses whichever of 0, 3, 6, or 9 fractional digits
## represents the value exactly.
static func _format_fraction(nanos: int) -> String:
	if nanos == 0:
		return ""
	if nanos % 1000000 == 0:
		return "." + _pad(nanos / 1000000, 3)
	if nanos % 1000 == 0:
		return "." + _pad(nanos / 1000, 6)
	return "." + _pad(nanos, 9)

static func parse(text: String) -> (long, int, ProtobufError):
	## The grammar bounds the length from both sides, so a pathological input is
	## refused before any of it is scanned rather than being walked to the end.
	if text.length() < 2 or text.length() > MAXIMUM_TEXT_LENGTH:
		return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if text.substr(text.length() - 1, 1) != "s":
		return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
	var body: String = text.substr(0, text.length() - 1)

	var negative: bool = false
	if body.substr(0, 1) == "-":
		negative = true
		body = body.substr(1, body.length() - 1)
	elif body.substr(0, 1) == "+":
		body = body.substr(1, body.length() - 1)

	## Canonical output always carries a whole part, but a parser has to accept
	## either side of the point being omitted -- ".5s" and "3.s" both come from
	## compatible protobuf implementations -- as long as at least one of the two
	## is present.
	var point: int = body.find(".")
	var whole_text: String = body
	var fraction_text: String = ""
	var has_point: bool = point >= 0
	if has_point:
		whole_text = body.substr(0, point)
		fraction_text = body.substr(point + 1, body.length() - point - 1)
		if fraction_text.length() > 9:
			return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if whole_text.length() == 0 and (not has_point or fraction_text.length() == 0):
		return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var whole: long = 0
	if whole_text.length() > 0:
		## A whole-seconds run of more than one digit may not start with a zero:
		## the reference decoder reads a single leading zero and then refuses
		## whatever digit follows it, so "01s" is malformed rather than a
		## zero-padded "1s".
		if whole_text.length() > 1 and whole_text.substr(0, 1) == "0":
			return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
		if not _is_digit_run(whole_text):
			return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
		## A syntactically valid run wider than fits safely in a 64-bit integer
		## is numerically out of range rather than malformed, so it is reported
		## that way without ever being accumulated into a value.
		if whole_text.length() > MAXIMUM_SAFE_DIGITS:
			return (0L, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
		var (whole_value, whole_ok) = _digits(whole_text)
		if not whole_ok:
			return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
		whole = whole_value
	if whole > MAXIMUM_SECONDS:
		return (0L, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var nanos: int = 0
	if fraction_text.length() > 0:
		var (fraction, fraction_ok) = _digits(fraction_text)
		if not fraction_ok:
			return (0L, 0, ProtobufError.JSON_TYPE_MISMATCH)
		nanos = fraction as int
		var scale: int = 9 - fraction_text.length()
		while scale > 0:
			nanos *= 10
			scale -= 1

	if negative:
		return (-whole, -nanos, ProtobufError.OK)
	return (whole, nanos, ProtobufError.OK)

static func _is_digit_run(text: String) -> bool:
	var index: int = 0
	while index < text.length():
		var character: String = text.substr(index, 1)
		if character < "0" or character > "9":
			return false
		index += 1
	return true

static func _digits(text: String) -> (long, bool):
	if text.length() == 0:
		return (0, false)
	var value: long = 0
	var index: int = 0
	while index < text.length():
		var character: String = text.substr(index, 1)
		if character < "0" or character > "9":
			return (0, false)
		value = value * 10 + (character.unicode_at(0) - 48)
		index += 1
	return (value, true)

static func _pad(value: int, width: int) -> String:
	var text: String = str(value)
	while text.length() < width:
		text = "0" + text
	return text
