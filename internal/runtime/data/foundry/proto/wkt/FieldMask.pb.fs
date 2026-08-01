namespace foundry.proto.wkt
import foundry.proto

## `FieldMask` represents a set of symbolic field paths, for example:
## paths: "f.a"
## paths: "f.b.d"
## Here `f` represents a field in some root message, `a` and `b`
## fields in the message found in `f`, and `d` a field found in the
## message in `f.b`.
## Field masks are used to specify a subset of fields that should be
## returned by a get operation or modified by an update operation.
## Field masks also have a custom JSON encoding (see below).
## # Field Masks in Projections
## When used in the context of a projection, a response message or
## sub-message is filtered by the API to only contain those fields as
## specified in the mask. For example, if the mask in the previous
## example is applied to a response message as follows:
## f {
## a : 22
## b {
## d : 1
## x : 2
## }
## y : 13
## }
## z: 8
## The result will not contain specific values for fields x,y and z
## (their value will be set to the default, and omitted in proto text
## output):
## f {
## a : 22
## b {
## d : 1
## }
## }
## A repeated field is not allowed except at the last position of a
## paths string.
## If a FieldMask object is not present in a get operation, the
## operation applies to all fields (as if a FieldMask of all fields
## had been specified).
## Note that a field mask does not necessarily apply to the
## top-level response message. In case of a REST get operation, the
## field mask applies directly to the response, but in case of a REST
## list operation, the mask instead applies to each individual message
## in the returned resource list. In case of a REST custom method,
## other definitions may be used. Where the mask applies will be
## clearly documented together with its declaration in the API.  In
## any case, the effect on the returned resource/resources is required
## behavior for APIs.
## # Field Masks in Update Operations
## A field mask in update operations specifies which fields of the
## targeted resource are going to be updated. The API is required
## to only change the values of the fields as specified in the mask
## and leave the others untouched. If a resource is passed in to
## describe the updated values, the API ignores the values of all
## fields not covered by the mask.
## If a repeated field is specified for an update operation, new values will
## be appended to the existing repeated field in the target resource. Note that
## a repeated field is only allowed in the last position of a `paths` string.
## If a sub-message is specified in the last position of the field mask for an
## update operation, then new value will be merged into the existing sub-message
## in the target resource.
## For example, given the target message:
## f {
## b {
## d: 1
## x: 2
## }
## c: [1]
## }
## And an update message:
## f {
## b {
## d: 10
## }
## c: [2]
## }
## then if the field mask is:
## paths: ["f.b", "f.c"]
## then the result will be:
## f {
## b {
## d: 10
## x: 2
## }
## c: [1, 2]
## }
## An implementation may provide options to override this default behavior for
## repeated and message fields.
## In order to reset a field's value to the default, the field must
## be in the mask and set to the default value in the provided resource.
## Hence, in order to reset all fields of a resource, provide a default
## instance of the resource and set all fields in the mask, or do
## not provide a mask as described below.
## If a field mask is not present on update, the operation applies to
## all fields (as if a field mask of all fields has been specified).
## Note that in the presence of schema evolution, this may mean that
## fields the client does not know and has therefore not filled into
## the request will be reset to their default. If this is unwanted
## behavior, a specific service may require a client to always specify
## a field mask, producing an error if not.
## As with get operations, the location of the resource which
## describes the updated values in the request message depends on the
## operation kind. In any case, the effect of the field mask is
## required to be honored by the API.
## ## Considerations for HTTP REST
## The HTTP kind of an update operation which uses a field mask must
## be set to PATCH instead of PUT in order to satisfy HTTP semantics
## (PUT must only be used for full updates).
## # JSON Encoding of Field Masks
## In JSON, a field mask is encoded as a single string where paths are
## separated by a comma. Fields name in each path are converted
## to/from lower-camel naming conventions.
## As an example, consider the following message declarations:
## message Profile {
## User user = 1;
## Photo photo = 2;
## }
## message User {
## string display_name = 1;
## string address = 2;
## }
## In proto a field mask for `Profile` may look as such:
## mask {
## paths: "user.display_name"
## paths: "photo"
## }
## In JSON, the same mask is represented as below:
## {
## mask: "user.displayName,photo"
## }
## # Field Masks and Oneof Fields
## Field masks treat fields in oneofs just as regular fields. Consider the
## following message:
## message SampleMessage {
## oneof test_oneof {
## string name = 4;
## SubMessage sub_message = 9;
## }
## }
## The field mask can be:
## mask {
## paths: "name"
## }
## Or:
## mask {
## paths: "sub_message"
## }
## Note that oneof type names ("test_oneof" in this case) cannot be used in
## paths.
## ## Field Mask Verification
## The implementation of any API method which has a FieldMask type field in the
## request should verify the included field paths, and return an
## `INVALID_ARGUMENT` error if any path is unmappable.
final class_name FieldMask extends RefCounted uses Message, JsonSerializable

## The set of field mask paths.
var paths: Array[String] = []

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new FieldMask message.
static func from_bytes(_pb_data: PackedByteArray) -> (FieldMask?, ProtobufError):
	var _pb_message: FieldMask = FieldMask.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: FieldMask? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	for _pb_paths_item: String in paths:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_paths_data: PackedByteArray = Wire.encode_string(_pb_paths_item)
		_pb_result.append_array(Wire.encode_varint(_pb_paths_data.size()))
		_pb_result.append_array(_pb_paths_data)
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
				var _pb_paths_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_paths_read.error != ProtobufError.OK:
					return _pb_paths_read.error
				paths.append(_pb_paths_read.value)
				_pb_offset = _pb_paths_read.offset
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
	var (_pb_text, _pb_error) = JsonFieldMask.to_json(paths)
	if _pb_error != ProtobufError.OK:
		push_error("JSON_VALUE_OUT_OF_RANGE: FieldMask cannot be written as canonical JSON")
		return JsonNode.Null
	return JsonNode.Str(_pb_text)

## Decodes a proto3 canonical JSON document into a new FieldMask message.
##
## JSON.parse_to_node(text).value produces the document; a malformed one is
## already reported through that JsonResult, so no text entry point is
## generated here.
static func from_json(_pb_node: JsonNode) -> JsonResult[FieldMask]:
	var _pb_message: FieldMask = FieldMask.new()
	var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)
	if _pb_error is JsonDecodeError:
		return JsonResult[FieldMask].fail(_pb_error.message, _pb_error.path)
	return JsonResult[FieldMask].ok(_pb_message)

## Merges a proto3 canonical JSON document into this message.
##
## A failure is returned rather than raised, matching the wire path, and
## carries the JSONPath of the value that could not be read.
func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:
	match _pb_node:
		JsonNode.Null:
			pass
		JsonNode.Str(var _pb_text):
			var (_pb_paths, _pb_error) = JsonFieldMask.from_json(_pb_text)
			if _pb_error != ProtobufError.OK:
				return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: FieldMask cannot be decoded from this JSON string", "$")
			paths = _pb_paths
		_:
			return JsonDecodeError.create("JSON_TYPE_MISMATCH: FieldMask expects a JSON string", "$")
	return null
