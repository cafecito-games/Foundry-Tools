namespace foundry.proto

## Explicit runtime mapping from protobuf full names to generated Message
## class handles. Applications populate this process-wide registry before
## asking Any operations to resolve a type URL.
class_name AnyTypeRegistry extends RefCounted

static var _types: Dictionary[String, Type[Message]] = {}

static func register(message_type: Type[Message]) -> ProtobufError:
	var name: String = message_type.protobuf_type_name()
	if not _is_valid_type_name(name):
		return ProtobufError.ANY_TYPE_NAME_INVALID
	if _types.has(name):
		if _types[name] == message_type:
			return ProtobufError.OK
		return ProtobufError.ANY_REGISTRY_CONFLICT
	_types[name] = message_type
	return ProtobufError.OK

static func clear() -> void:
	_types = {}

static func _resolve(type_url: String) -> (Type[Message]?, ProtobufError):
	var missing: Type[Message]? = null
	var (name, error) = _type_name_from_url(type_url)
	if error != ProtobufError.OK:
		return (missing, error)
	if not _types.has(name):
		return (missing, ProtobufError.ANY_TYPE_NOT_REGISTERED)
	return (_types[name], ProtobufError.OK)

static func _type_name_from_url(type_url: String) -> (String, ProtobufError):
	var last_slash: int = -1
	var index: int = 0
	while index < type_url.length():
		if type_url.substr(index, 1) == "/":
			last_slash = index
		index += 1
	var name: String = type_url
	if last_slash >= 0:
		name = type_url.substr(last_slash + 1, type_url.length() - last_slash - 1)
	if not _is_valid_type_name(name):
		return ("", ProtobufError.ANY_TYPE_URL_INVALID)
	return (name, ProtobufError.OK)

static func _any_to_json(type_url: String, bytes: PackedByteArray) -> JsonNode:
	if type_url.is_empty() and bytes.is_empty():
		return JsonNode.object_of({})
	var (message_type, resolve_error) = _resolve(type_url)
	if resolve_error != ProtobufError.OK or message_type == null:
		push_error(_error_name(resolve_error) + ": google.protobuf.Any cannot resolve @type")
		return JsonNode.Null

	## Message and JsonSerializable are independent traits. Keep this checked
	## dynamic bridge private and discard it as soon as the handle is narrowed.
	var message_type_value: Variant = message_type
	if not (message_type_value is Type[JsonSerializable]):
		push_error("ANY_JSON_UNSUPPORTED: registered Any type has no canonical JSON form")
		return JsonNode.Null
	var _json_message_type: Type[JsonSerializable] = message_type_value

	var message: Message = message_type.create_message()
	var merge_error: ProtobufError = message.merge_from_bytes(bytes)
	if merge_error != ProtobufError.OK:
		push_error(_error_name(merge_error) + ": google.protobuf.Any payload cannot be decoded")
		return JsonNode.Null

	var message_value: Variant = message
	if not (message_value is JsonSerializable):
		push_error("ANY_JSON_UNSUPPORTED: registered Any value has no canonical JSON form")
		return JsonNode.Null
	var json_message: JsonSerializable = message_value
	var embedded: JsonNode = json_message.to_json()
	match embedded:
		JsonNode.Object(var entries):
			var result: Dictionary[String, JsonNode] = {}
			result["@type"] = JsonNode.Str(type_url)
			for key: String in entries:
				result[key] = entries[key]
			return JsonNode.object_of(result)
		_:
			push_error("ANY_JSON_UNSUPPORTED: ordinary Any payload must encode as a JSON object")
			return JsonNode.Null

static func _any_from_json(node: JsonNode) -> (String, PackedByteArray, JsonDecodeError?):
	var no_error: JsonDecodeError? = null
	match node:
		JsonNode.Null:
			return ("", PackedByteArray(), no_error)
		JsonNode.Object(var entries):
			if entries.is_empty():
				return ("", PackedByteArray(), no_error)
			if not entries.has("@type"):
				return ("", PackedByteArray(), JsonDecodeError.create("JSON_PARSE_FAILED: google.protobuf.Any requires @type", "$[\"@type\"]"))
			match entries["@type"]:
				JsonNode.Str(var type_url):
					var (message_type, resolve_error) = _resolve(type_url)
					if resolve_error != ProtobufError.OK or message_type == null:
						return ("", PackedByteArray(), JsonDecodeError.create(_error_name(resolve_error) + ": google.protobuf.Any cannot resolve @type", "$[\"@type\"]"))

					## The registry retains Type[Message]. Cross to the unrelated JSON
					## trait only through this checked, private dynamic seam.
					var message_type_value: Variant = message_type
					if not (message_type_value is Type[JsonSerializable]):
						return ("", PackedByteArray(), JsonDecodeError.create("ANY_JSON_UNSUPPORTED: registered Any type has no canonical JSON form", "$[\"@type\"]"))
					var json_message_type: Type[JsonSerializable] = message_type_value

					var payload: Dictionary[String, JsonNode] = {}
					for key: String in entries:
						if key != "@type":
							payload[key] = entries[key]
					var decoded: JsonResult[JsonSerializable] = json_message_type.from_json(JsonNode.object_of(payload))
					if not decoded.is_ok():
						return ("", PackedByteArray(), decoded.error)
					var decoded_value: Variant = decoded.value
					if not (decoded_value is Message):
						return ("", PackedByteArray(), JsonDecodeError.create("ANY_JSON_UNSUPPORTED: decoded Any value is not a protobuf Message", "$[\"@type\"]"))
					var message: Message = decoded_value
					var packed: foundry.proto.wkt.Any = foundry.proto.wkt.Any.pack(message)
					packed.type_url = type_url
					return (packed.type_url, packed.value, no_error)
				_:
					return ("", PackedByteArray(), JsonDecodeError.create("JSON_TYPE_MISMATCH: google.protobuf.Any @type expects a string", "$[\"@type\"]"))
		_:
			return ("", PackedByteArray(), JsonDecodeError.create("JSON_TYPE_MISMATCH: google.protobuf.Any expects a JSON object", "$"))

static func _error_name(error: ProtobufError) -> String:
	match error:
		ProtobufError.OK:
			return "OK"
		ProtobufError.VARINT_NOT_FOUND:
			return "VARINT_NOT_FOUND"
		ProtobufError.VARINT_TOO_LONG:
			return "VARINT_TOO_LONG"
		ProtobufError.WIRE_TYPE_MISMATCH:
			return "WIRE_TYPE_MISMATCH"
		ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND:
			return "LENGTH_DELIMITED_SIZE_NOT_FOUND"
		ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH:
			return "LENGTH_DELIMITED_SIZE_MISMATCH"
		ProtobufError.UNKNOWN_REQUIRED_FEATURE:
			return "UNKNOWN_REQUIRED_FEATURE"
		ProtobufError.ANY_TYPE_URL_INVALID:
			return "ANY_TYPE_URL_INVALID"
		ProtobufError.ANY_TYPE_NOT_REGISTERED:
			return "ANY_TYPE_NOT_REGISTERED"
		ProtobufError.ANY_JSON_UNSUPPORTED:
			return "ANY_JSON_UNSUPPORTED"
		_:
			return "UNKNOWN_REQUIRED_FEATURE"

static func _is_valid_type_name(name: String) -> bool:
	if name.length() == 0:
		return false
	var at_segment_start: bool = true
	var index: int = 0
	while index < name.length():
		var character: String = name.substr(index, 1)
		if character == ".":
			if at_segment_start:
				return false
			at_segment_start = true
			index += 1
			continue
		var is_letter: bool = (character >= "A" and character <= "Z") or (character >= "a" and character <= "z")
		var is_digit: bool = character >= "0" and character <= "9"
		if not (is_letter or character == "_" or (is_digit and not at_segment_start)):
			return false
		at_segment_start = false
		index += 1
	return not at_segment_start
