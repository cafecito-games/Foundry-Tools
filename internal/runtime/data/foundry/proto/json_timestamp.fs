## Every division below is on two integers and wants the truncated quotient, so
## the decimal-part warning would fire on each one and say nothing. Suppressing
## it here keeps the runtime quiet in the projects that embed it.
@warning_ignore_start("INTEGER_DIVISION")

namespace foundry.proto

## RFC 3339 conversion for the canonical JSON mapping of google.protobuf.Timestamp.
##
## The calendar math is the proleptic Gregorian days-from-civil algorithm, which
## is closed-form in both directions and needs no lookup table. Both functions
## work in seconds since the Unix epoch plus a non-negative nanosecond remainder,
## which is exactly how the message stores them.
class_name JsonTimestamp extends RefCounted

## 0001-01-01T00:00:00Z, the earliest instant the canonical mapping allows.
const MINIMUM_SECONDS: int = -62135596800

## 9999-12-31T23:59:59Z, the latest.
const MAXIMUM_SECONDS: int = 253402300799

const MAXIMUM_NANOS: int = 999999999

const SECONDS_PER_DAY: int = 86400

## The shortest well-formed instant, "0000-01-01T00:00:00Z".
const MINIMUM_TEXT_LENGTH: int = 20

## The longest, a nine-digit fraction and a numeric offset.
const MAXIMUM_TEXT_LENGTH: int = 35

static func format(seconds: int, nanos: int) -> (String, ProtobufError):
	if seconds < MINIMUM_SECONDS or seconds > MAXIMUM_SECONDS:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if nanos < 0 or nanos > MAXIMUM_NANOS:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var days: int = _floor_divide(seconds, SECONDS_PER_DAY)
	var second_of_day: int = seconds - days * SECONDS_PER_DAY
	var (year, month, day) = _civil_from_days(days)

	var text: String = _pad(year, 4) + "-" + _pad(month, 2) + "-" + _pad(day, 2)
	text += "T" + _pad(second_of_day / 3600, 2)
	text += ":" + _pad((second_of_day / 60) % 60, 2)
	text += ":" + _pad(second_of_day % 60, 2)
	text += _format_fraction(nanos)
	return (text + "Z", ProtobufError.OK)

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

static func parse(text: String) -> (int, int, ProtobufError):
	## The grammar bounds the length from both sides, so a pathological input is
	## refused before any of it is scanned rather than being walked to the end.
	if text.length() < MINIMUM_TEXT_LENGTH or text.length() > MAXIMUM_TEXT_LENGTH:
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if text.substr(4, 1) != "-" or text.substr(7, 1) != "-":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	var designator: String = text.substr(10, 1)
	if designator != "T" and designator != "t":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if text.substr(13, 1) != ":" or text.substr(16, 1) != ":":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var (year, year_ok) = _digits(text, 0, 4)
	var (month, month_ok) = _digits(text, 5, 2)
	var (day, day_ok) = _digits(text, 8, 2)
	var (hour, hour_ok) = _digits(text, 11, 2)
	var (minute, minute_ok) = _digits(text, 14, 2)
	var (second, second_ok) = _digits(text, 17, 2)
	if not (year_ok and month_ok and day_ok and hour_ok and minute_ok and second_ok):
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if month < 1 or month > 12 or day < 1 or day > _days_in_month(year, month):
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if hour > 23 or minute > 59 or second > 59:
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var cursor: int = 19
	var nanos: int = 0
	if text.substr(cursor, 1) == ".":
		cursor += 1
		var digit_count: int = 0
		while cursor + digit_count < text.length() and _is_digit(text.substr(cursor + digit_count, 1)):
			digit_count += 1
		if digit_count == 0 or digit_count > 9:
			return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
		var (fraction, fraction_ok) = _digits(text, cursor, digit_count)
		if not fraction_ok:
			return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
		nanos = fraction
		var scale: int = 9 - digit_count
		while scale > 0:
			nanos *= 10
			scale -= 1
		cursor += digit_count

	var (offset_seconds, offset_ok) = _parse_offset(text, cursor)
	if not offset_ok:
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var days: int = _days_from_civil(year, month, day)
	var seconds: int = days * SECONDS_PER_DAY + hour * 3600 + minute * 60 + second
	seconds -= offset_seconds
	if seconds < MINIMUM_SECONDS or seconds > MAXIMUM_SECONDS:
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	return (seconds, nanos, ProtobufError.OK)

## Returns the offset in seconds east of UTC, and whether the suffix was
## well-formed. A `Z` is a zero offset.
static func _parse_offset(text: String, cursor: int) -> (int, bool):
	if cursor >= text.length():
		return (0, false)
	var designator: String = text.substr(cursor, 1)
	if designator == "Z" or designator == "z":
		if cursor + 1 != text.length():
			return (0, false)
		return (0, true)
	if designator != "+" and designator != "-":
		return (0, false)
	if cursor + 6 != text.length() or text.substr(cursor + 3, 1) != ":":
		return (0, false)
	var (hours, hours_ok) = _digits(text, cursor + 1, 2)
	var (minutes, minutes_ok) = _digits(text, cursor + 4, 2)
	if not (hours_ok and minutes_ok) or hours > 23 or minutes > 59:
		return (0, false)
	var total: int = hours * 3600 + minutes * 60
	if designator == "-":
		return (-total, true)
	return (total, true)

static func _digits(text: String, offset: int, length: int) -> (int, bool):
	if offset + length > text.length():
		return (0, false)
	var value: int = 0
	var index: int = 0
	while index < length:
		var character: String = text.substr(offset + index, 1)
		if not _is_digit(character):
			return (0, false)
		value = value * 10 + (character.unicode_at(0) - 48)
		index += 1
	return (value, true)

static func _is_digit(character: String) -> bool:
	return character >= "0" and character <= "9"

static func _pad(value: int, width: int) -> String:
	var text: String = str(value)
	while text.length() < width:
		text = "0" + text
	return text

## A day number a month does not have would otherwise be normalized into the
## next month by the civil-date math rather than being reported as an error.
static func _days_in_month(year: int, month: int) -> int:
	if month == 2:
		if _is_leap_year(year):
			return 29
		return 28
	if month == 4 or month == 6 or month == 9 or month == 11:
		return 30
	return 31

static func _is_leap_year(year: int) -> bool:
	if year % 4 != 0:
		return false
	if year % 100 != 0:
		return true
	return year % 400 == 0

## Integer division that rounds toward negative infinity. The engine truncates
## toward zero, which would put a pre-epoch instant on the wrong day.
static func _floor_divide(numerator: int, denominator: int) -> int:
	var quotient: int = numerator / denominator
	if numerator % denominator != 0 and (numerator < 0) != (denominator < 0):
		quotient -= 1
	return quotient

## Howard Hinnant's civil-from-days, shifted to an era beginning on 0000-03-01
## so that the leap day falls at the end of a year and needs no special case.
static func _civil_from_days(days: int) -> (int, int, int):
	var shifted: int = days + 719468
	var era: int = _floor_divide(shifted, 146097)
	var day_of_era: int = shifted - era * 146097
	var year_of_era: int = (day_of_era - day_of_era / 1460 + day_of_era / 36524 - day_of_era / 146096) / 365
	var year: int = year_of_era + era * 400
	var day_of_year: int = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100)
	var month_prime: int = (5 * day_of_year + 2) / 153
	var day: int = day_of_year - (153 * month_prime + 2) / 5 + 1
	var month: int = month_prime + 3
	if month_prime >= 10:
		month = month_prime - 9
	if month <= 2:
		year += 1
	return (year, month, day)

## The inverse of _civil_from_days.
static func _days_from_civil(year: int, month: int, day: int) -> int:
	var shifted_year: int = year
	if month <= 2:
		shifted_year -= 1
	var era: int = _floor_divide(shifted_year, 400)
	var year_of_era: int = shifted_year - era * 400
	var month_prime: int = month + 9
	if month > 2:
		month_prime = month - 3
	var day_of_year: int = (153 * month_prime + 2) / 5 + day - 1
	var day_of_era: int = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year
	return era * 146097 + day_of_era - 719468
