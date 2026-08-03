namespace foundry.proto

enum_name ProtobufError:
	OK = 0
	VARINT_NOT_FOUND = 1
	VARINT_TOO_LONG = 2
	WIRE_TYPE_MISMATCH = 3
	LENGTH_DELIMITED_SIZE_NOT_FOUND = 4
	LENGTH_DELIMITED_SIZE_MISMATCH = 5
	UNKNOWN_REQUIRED_FEATURE = 6
	## The document is not well-formed JSON.
	JSON_PARSE_FAILED = 7
	## A well-formed JSON value has the wrong shape for the field it is being
	## read into: an object where a string belongs, a string where a number does.
	JSON_TYPE_MISMATCH = 8
	## A JSON object member matches no field. Canonical JSON has no unknown-field
	## buffer, so there is nothing to preserve it in and it is refused instead.
	JSON_UNKNOWN_FIELD = 9
	## A number falls outside its field's domain, which includes a timestamp
	## outside the range RFC 3339 can express.
	JSON_VALUE_OUT_OF_RANGE = 10
	## Deprecated: retained for numeric compatibility. New Any JSON paths use
	## ANY_JSON_UNSUPPORTED.
	JSON_ANY_UNSUPPORTED = 11
	## A Dictionary presented as a Struct has a key that is not a String.
	STRUCT_KEY_NOT_STRING = 12
	## A native value presented as a protobuf Value has no representation.
	STRUCT_VALUE_UNREPRESENTABLE = 13
	## A native float is non-finite or normalizes outside the protobuf
	## Timestamp or Duration range.
	WELL_KNOWN_TIME_OUT_OF_RANGE = 14
	## A registered Message handle reports an invalid protobuf full name.
	ANY_TYPE_NAME_INVALID = 15
	## A different Message handle already owns the same protobuf full name.
	ANY_REGISTRY_CONFLICT = 16
	## A type URL has no valid trailing protobuf full name.
	ANY_TYPE_URL_INVALID = 17
	## A valid type URL names a Message handle that has not been registered.
	ANY_TYPE_NOT_REGISTERED = 18
	## A registered Message does not implement canonical protobuf JSON.
	ANY_JSON_UNSUPPORTED = 19
