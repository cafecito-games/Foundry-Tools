namespace foundry.proto.wkt
import foundry.proto

## A Timestamp represents a point in time independent of any time zone or local
## calendar, encoded as a count of seconds and fractions of seconds at
## nanosecond resolution. The count is relative to an epoch at UTC midnight on
## January 1, 1970, in the proleptic Gregorian calendar which extends the
## Gregorian calendar backwards to year one.
## All minutes are 60 seconds long. Leap seconds are "smeared" so that no leap
## second table is needed for interpretation, using a [24-hour linear
## smear](https://developers.google.com/time/smear).
## The range is from 0001-01-01T00:00:00Z to 9999-12-31T23:59:59.999999999Z. By
## restricting to that range, we ensure that we can convert to and from [RFC
## 3339](https://www.ietf.org/rfc/rfc3339.txt) date strings.
## # Examples
## Example 1: Compute Timestamp from POSIX `time()`.
## Timestamp timestamp;
## timestamp.set_seconds(time(NULL));
## timestamp.set_nanos(0);
## Example 2: Compute Timestamp from POSIX `gettimeofday()`.
## struct timeval tv;
## gettimeofday(&tv, NULL);
## Timestamp timestamp;
## timestamp.set_seconds(tv.tv_sec);
## timestamp.set_nanos(tv.tv_usec * 1000);
## Example 3: Compute Timestamp from Win32 `GetSystemTimeAsFileTime()`.
## FILETIME ft;
## GetSystemTimeAsFileTime(&ft);
## UINT64 ticks = (((UINT64)ft.dwHighDateTime) << 32) | ft.dwLowDateTime;
## // A Windows tick is 100 nanoseconds. Windows epoch 1601-01-01T00:00:00Z
## // is 11644473600 seconds before Unix epoch 1970-01-01T00:00:00Z.
## Timestamp timestamp;
## timestamp.set_seconds((INT64) ((ticks / 10000000) - 11644473600LL));
## timestamp.set_nanos((INT32) ((ticks % 10000000) * 100));
## Example 4: Compute Timestamp from Java `System.currentTimeMillis()`.
## long millis = System.currentTimeMillis();
## Timestamp timestamp = Timestamp.newBuilder().setSeconds(millis / 1000)
## .setNanos((int) ((millis % 1000) * 1000000)).build();
## Example 5: Compute Timestamp from Java `Instant.now()`.
## Instant now = Instant.now();
## Timestamp timestamp =
## Timestamp.newBuilder().setSeconds(now.getEpochSecond())
## .setNanos(now.getNano()).build();
## Example 6: Compute Timestamp from current time in Python.
## timestamp = Timestamp()
## timestamp.GetCurrentTime()
## # JSON Mapping
## In JSON format, the Timestamp type is encoded as a string in the
## [RFC 3339](https://www.ietf.org/rfc/rfc3339.txt) format. That is, the
## format is "{year}-{month}-{day}T{hour}:{min}:{sec}[.{frac_sec}]Z"
## where {year} is always expressed using four digits while {month}, {day},
## {hour}, {min}, and {sec} are zero-padded to two digits each. The fractional
## seconds, which can go up to 9 digits (i.e. up to 1 nanosecond resolution),
## are optional. The "Z" suffix indicates the timezone ("UTC"); the timezone
## is required. A ProtoJSON serializer should always use UTC (as indicated by
## "Z") when printing the Timestamp type and a ProtoJSON parser should be
## able to accept both UTC and other timezones (as indicated by an offset).
## For example, "2017-01-15T01:30:15.01Z" encodes 15.01 seconds past
## 01:30 UTC on January 15, 2017.
## In JavaScript, one can convert a Date object to this format using the
## standard
## [toISOString()](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Date/toISOString)
## method. In Python, a standard `datetime.datetime` object can be converted
## to this format using
## [`strftime`](https://docs.python.org/2/library/time.html#time.strftime) with
## the time format spec '%Y-%m-%dT%H:%M:%S.%fZ'. Likewise, in Java, one can use
## the Joda Time's [`ISODateTimeFormat.dateTime()`](
## http://joda-time.sourceforge.net/apidocs/org/joda/time/format/ISODateTimeFormat.html#dateTime()
## ) to obtain a formatter capable of generating timestamps in this format.
final class_name Timestamp extends RefCounted uses Message, JsonSerializable

## Represents seconds of UTC time since Unix epoch 1970-01-01T00:00:00Z. Must
## be between -62135596800 and 253402300799 inclusive (which corresponds to
## 0001-01-01T00:00:00Z to 9999-12-31T23:59:59Z).
var seconds: int = 0

## Non-negative fractions of a second at nanosecond resolution. This field is
## the nanosecond portion of the duration, not an alternative to seconds.
## Negative second values with fractions must still have non-negative nanos
## values that count forward in time. Must be between 0 and 999,999,999
## inclusive.
var nanos: int = 0

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Timestamp message.
static func from_bytes(_pb_data: PackedByteArray) -> (Timestamp?, ProtobufError):
	var _pb_message: Timestamp = Timestamp.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Timestamp? = null
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

## Converts unix seconds into a normalized Timestamp.
##
## The seconds field is floored so a negative timestamp keeps nonnegative
## nanos. The finite input rounds to the nearest nanosecond; a full
## billion carries into seconds. Invalid inputs return no Timestamp.
static func from_unix_time(_pb_value: float) -> (Timestamp?, ProtobufError):
	var _pb_failed: Timestamp? = null
	if is_nan(_pb_value) or is_inf(_pb_value):
		return (_pb_failed, ProtobufError.WELL_KNOWN_TIME_OUT_OF_RANGE)
	if _pb_value < -62135596801.0 or _pb_value > 253402300800.0:
		return (_pb_failed, ProtobufError.WELL_KNOWN_TIME_OUT_OF_RANGE)
	var _pb_result: Timestamp = Timestamp._from_valid_unix_time(_pb_value)
	if _pb_result.seconds < -62135596800 or _pb_result.seconds > 253402300799 or _pb_result.nanos < 0 or _pb_result.nanos > 999999999:
		return (_pb_failed, ProtobufError.WELL_KNOWN_TIME_OUT_OF_RANGE)
	return (_pb_result, ProtobufError.OK)

static func _from_valid_unix_time(_pb_value: float) -> Timestamp:
	var _pb_result: Timestamp = Timestamp.new()
	var _pb_whole: int = floori(_pb_value)
	_pb_result.seconds = _pb_whole
	_pb_result.nanos = roundi((_pb_value - float(_pb_whole)) * 1000000000.0)
	if _pb_result.nanos >= 1000000000:
		_pb_result.nanos -= 1000000000
		_pb_result.seconds += 1
	return _pb_result

## Returns the current system time as a Timestamp.
static func now() -> Timestamp:
	return Timestamp._from_valid_unix_time(Time.get_unix_time_from_system())

## Returns this Timestamp as unix seconds.
##
## This direction is lossy: a float near the present epoch resolves to about
## 238 nanoseconds. Read seconds and nanos directly when full precision matters.
func to_unix_time() -> float:
	return float(seconds) + float(nanos) / 1000000000.0

## Returns this message as a proto3 canonical JSON document.
##
## JSON.stringify(message, "", false) renders it as text; the third argument
## turns off key sorting, which keeps members in field declaration order.
func to_json() -> JsonNode:
	var (_pb_text, _pb_error) = JsonTimestamp.format(seconds, nanos)
	if _pb_error != ProtobufError.OK:
		push_error("JSON_VALUE_OUT_OF_RANGE: Timestamp cannot be written as canonical JSON")
		return JsonNode.Null
	return JsonNode.Str(_pb_text)

## Decodes a proto3 canonical JSON document into a new Timestamp message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[Timestamp]:
	var _pb_message: Timestamp = Timestamp.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[Timestamp].fail(_pb_error.message, _pb_error.path)
	return JsonResult[Timestamp].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Str(var _pb_text):
			var (_pb_seconds, _pb_nanos, _pb_error) = JsonTimestamp.parse(_pb_text)
			if _pb_error != ProtobufError.OK:
				return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: Timestamp cannot be decoded from this JSON string", "$")
			seconds = _pb_seconds
			nanos = _pb_nanos
		_:
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: Timestamp expects a JSON string", "$")
	return null
