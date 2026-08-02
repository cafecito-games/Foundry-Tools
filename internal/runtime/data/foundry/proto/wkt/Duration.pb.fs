namespace foundry.proto.wkt
import foundry.proto

## A Duration represents a signed, fixed-length span of time represented
## as a count of seconds and fractions of seconds at nanosecond
## resolution. It is independent of any calendar and concepts like "day"
## or "month". It is related to Timestamp in that the difference between
## two Timestamp values is a Duration and it can be added or subtracted
## from a Timestamp. Range is approximately +-10,000 years.
## # Examples
## Example 1: Compute Duration from two Timestamps in pseudo code.
## Timestamp start = ...;
## Timestamp end = ...;
## Duration duration = ...;
## duration.seconds = end.seconds - start.seconds;
## duration.nanos = end.nanos - start.nanos;
## if (duration.seconds < 0 && duration.nanos > 0) {
## duration.seconds += 1;
## duration.nanos -= 1000000000;
## } else if (duration.seconds > 0 && duration.nanos < 0) {
## duration.seconds -= 1;
## duration.nanos += 1000000000;
## }
## Example 2: Compute Timestamp from Timestamp + Duration in pseudo code.
## Timestamp start = ...;
## Duration duration = ...;
## Timestamp end = ...;
## end.seconds = start.seconds + duration.seconds;
## end.nanos = start.nanos + duration.nanos;
## if (end.nanos < 0) {
## end.seconds -= 1;
## end.nanos += 1000000000;
## } else if (end.nanos >= 1000000000) {
## end.seconds += 1;
## end.nanos -= 1000000000;
## }
## Example 3: Compute Duration from datetime.timedelta in Python.
## td = datetime.timedelta(days=3, minutes=10)
## duration = Duration()
## duration.FromTimedelta(td)
## # JSON Mapping
## In JSON format, the Duration type is encoded as a string rather than an
## object, where the string ends in the suffix "s" (indicating seconds) and
## is preceded by the number of seconds, with nanoseconds expressed as
## fractional seconds. For example, 3 seconds with 0 nanoseconds should be
## encoded in JSON format as "3s", while 3 seconds and 1 nanosecond should
## be expressed in JSON format as "3.000000001s", and 3 seconds and 1
## microsecond should be expressed in JSON format as "3.000001s".
final class_name Duration extends RefCounted uses Message, JsonSerializable

## Signed seconds of the span of time. Must be from -315,576,000,000
## to +315,576,000,000 inclusive. Note: these bounds are computed from:
## 60 sec/min * 60 min/hr * 24 hr/day * 365.25 days/year * 10000 years
var seconds: int = 0

## Signed fractions of a second at nanosecond resolution of the span
## of time. Durations less than one second are represented with a 0
## `seconds` field and a positive or negative `nanos` field. For durations
## of one second or more, a non-zero value for the `nanos` field must be
## of the same sign as the `seconds` field. Must be from -999,999,999
## to +999,999,999 inclusive.
var nanos: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Duration message.
static func from_bytes(_pb_data: PackedByteArray) -> (Duration?, ProtobufError):
	var _pb_message: Duration = Duration.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Duration? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if seconds != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(seconds))
	if nanos != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(nanos))
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
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_seconds_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_seconds_read.error != ProtobufError.OK:
					return _pb_seconds_read.error
				seconds = _pb_seconds_read.value
				_pb_offset = _pb_seconds_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_nanos_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_nanos_read.error != ProtobufError.OK:
					return _pb_nanos_read.error
				nanos = _pb_nanos_read.value
				_pb_offset = _pb_nanos_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK

## Converts float seconds into a normalized Duration.
##
## Whole seconds truncate toward zero so seconds and nanos keep compatible
## signs. The finite input rounds to the nearest nanosecond; a full
## billion carries in either direction. Invalid inputs return no Duration.
static func from_seconds(_pb_value: float) -> (Duration?, ProtobufError):
	var _pb_failed: Duration? = null
	if is_nan(_pb_value) or is_inf(_pb_value):
		return (_pb_failed, ProtobufError.WELL_KNOWN_TIME_OUT_OF_RANGE)
	if _pb_value < -315576000001.0 or _pb_value > 315576000001.0:
		return (_pb_failed, ProtobufError.WELL_KNOWN_TIME_OUT_OF_RANGE)
	var _pb_result: Duration = Duration._from_valid_seconds(_pb_value)
	if _pb_result.seconds < -315576000000 or _pb_result.seconds > 315576000000 or _pb_result.nanos < -999999999 or _pb_result.nanos > 999999999 or (_pb_result.seconds > 0 and _pb_result.nanos < 0) or (_pb_result.seconds < 0 and _pb_result.nanos > 0):
		return (_pb_failed, ProtobufError.WELL_KNOWN_TIME_OUT_OF_RANGE)
	return (_pb_result, ProtobufError.OK)

static func _from_valid_seconds(_pb_value: float) -> Duration:
	var _pb_result: Duration = Duration.new()
	var _pb_whole: int = int(_pb_value)
	_pb_result.seconds = _pb_whole
	_pb_result.nanos = roundi((_pb_value - float(_pb_whole)) * 1000000000.0)
	if _pb_result.nanos >= 1000000000:
		_pb_result.nanos -= 1000000000
		_pb_result.seconds += 1
	elif _pb_result.nanos <= -1000000000:
		_pb_result.nanos += 1000000000
		_pb_result.seconds -= 1
	return _pb_result

## Returns this Duration as float seconds.
## Read seconds and nanos directly when nanosecond precision matters.
func to_seconds() -> float:
	return float(seconds) + float(nanos) / 1000000000.0

## Returns this message as a proto3 canonical JSON document.
##
## JSON.stringify(message, "", false) renders it as text; the third argument
## turns off key sorting, which keeps members in field declaration order.
func to_json() -> JsonNode:
	var (_pb_text, _pb_error) = JsonDuration.format(seconds, nanos)
	if _pb_error != ProtobufError.OK:
		push_error("JSON_VALUE_OUT_OF_RANGE: Duration cannot be written as canonical JSON")
		return JsonNode.Null
	return JsonNode.Str(_pb_text)

## Decodes a proto3 canonical JSON document into a new Duration message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Duration]:
	var _pb_message: Duration = Duration.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Duration].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Duration].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Str(var _pb_text):
			var (_pb_seconds, _pb_nanos, _pb_error) = JsonDuration.parse(_pb_text)
			if _pb_error != ProtobufError.OK:
				return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: Duration cannot be decoded from this JSON string", "$")
			seconds = _pb_seconds
			nanos = _pb_nanos
		_:
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Duration expects a JSON string", "$")
	return null
