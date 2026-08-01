namespace foundry.proto.wkt
import foundry.proto

## `Any` contains an arbitrary serialized protocol buffer message along with a
## URL that describes the type of the serialized message.
## Protobuf library provides support to pack/unpack Any values in the form
## of utility functions or additional generated methods of the Any type.
## Example 1: Pack and unpack a message in C++.
## Foo foo = ...;
## Any any;
## any.PackFrom(foo);
## ...
## if (any.UnpackTo(&foo)) {
## ...
## }
## Example 2: Pack and unpack a message in Java.
## Foo foo = ...;
## Any any = Any.pack(foo);
## ...
## if (any.is(Foo.class)) {
## foo = any.unpack(Foo.class);
## }
## // or ...
## if (any.isSameTypeAs(Foo.getDefaultInstance())) {
## foo = any.unpack(Foo.getDefaultInstance());
## }
## Example 3: Pack and unpack a message in Python.
## foo = Foo(...)
## any = Any()
## any.Pack(foo)
## ...
## if any.Is(Foo.DESCRIPTOR):
## any.Unpack(foo)
## ...
## Example 4: Pack and unpack a message in Go
## foo := &pb.Foo{...}
## any, err := anypb.New(foo)
## if err != nil {
## ...
## }
## ...
## foo := &pb.Foo{}
## if err := any.UnmarshalTo(foo); err != nil {
## ...
## }
## The pack methods provided by protobuf library will by default use
## 'type.googleapis.com/full.type.name' as the type URL and the unpack
## methods only use the fully qualified type name after the last '/'
## in the type URL, for example "foo.bar.com/x/y.z" will yield type
## name "y.z".
## JSON
## ====
## The JSON representation of an `Any` value uses the regular
## representation of the deserialized, embedded message, with an
## additional field `@type` which contains the type URL. Example:
## package google.profile;
## message Person {
## string first_name = 1;
## string last_name = 2;
## }
## {
## "@type": "type.googleapis.com/google.profile.Person",
## "firstName": <string>,
## "lastName": <string>
## }
## If the embedded message type is well-known and has a custom JSON
## representation, that representation will be embedded adding a field
## `value` which holds the custom JSON in addition to the `@type`
## field. Example (for message [google.protobuf.Duration][]):
## {
## "@type": "type.googleapis.com/google.protobuf.Duration",
## "value": "1.212s"
## }
final class_name Any extends RefCounted uses Message, JsonSerializable

## A URL/resource name that uniquely identifies the type of the serialized
## protocol buffer message. This string must contain at least
## one "/" character. The last segment of the URL's path must represent
## the fully qualified name of the type (as in
## `path/google.protobuf.Duration`). The name should be in a canonical form
## (e.g., leading "." is not accepted).
## In practice, teams usually precompile into the binary all types that they
## expect it to use in the context of Any. However, for URLs which use the
## scheme `http`, `https`, or no scheme, one can optionally set up a type
## server that maps type URLs to message definitions as follows:
## * If no scheme is provided, `https` is assumed.
## * An HTTP GET on the URL must yield a [google.protobuf.Type][]
## value in binary format, or produce an error.
## * Applications are allowed to cache lookup results based on the
## URL, or have them precompiled into a binary to avoid any
## lookup. Therefore, binary compatibility needs to be preserved
## on changes to types. (Use versioned type names to manage
## breaking changes.)
## Note: this functionality is not currently available in the official
## protobuf release, and it is not used for type URLs beginning with
## type.googleapis.com. As of May 2023, there are no widely used type server
## implementations and no plans to implement one.
## Schemes other than `http`, `https` (or the empty scheme) might be
## used with implementation specific semantics.
var type_url: String = ""

## Must be a valid serialized protocol buffer of the above specified type.
var value: PackedByteArray = PackedByteArray()

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Any message.
static func from_bytes(_pb_data: PackedByteArray) -> (Any?, ProtobufError):
	var _pb_message: Any = Any.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Any? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if type_url != "":
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_type_url_data: PackedByteArray = Wire.encode_string(type_url)
		_pb_result.append_array(Wire.encode_varint(_pb_type_url_data.size()))
		_pb_result.append_array(_pb_type_url_data)
	if value.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(value.size()))
		_pb_result.append_array(value)
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
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_type_url_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_type_url_read.error != ProtobufError.OK:
					return _pb_type_url_read.error
				type_url = _pb_type_url_read.value
				_pb_offset = _pb_type_url_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_value_read: BytesRead = Wire.read_bytes(_pb_data, _pb_offset)
				if _pb_value_read.error != ProtobufError.OK:
					return _pb_value_read.error
				value = _pb_value_read.value
				_pb_offset = _pb_value_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK

## Returns this message as a proto3 canonical JSON document.
##
## JSON.stringify(message, "", false) renders it as text; the third argument
## turns off key sorting, which keeps members in field declaration order.
func to_json() -> JsonNode:
	push_error("JSON_ANY_UNSUPPORTED: google.protobuf.Any has no JSON form until the type-URL registry lands")
	return JsonNode.Null

## Decodes a proto3 canonical JSON document into a new Any message.
##
## Not generated yet: this reports a failure for every document. The
## conformance it completes is what makes to_json reachable through
## JSON.stringify, which is why the member exists ahead of the decoder.
static func from_json(_pb_node: JsonNode) -> JsonResult[Any]:
	return JsonResult[Any].fail("JSON_PARSE_FAILED: Any cannot be decoded from JSON yet", "$")
