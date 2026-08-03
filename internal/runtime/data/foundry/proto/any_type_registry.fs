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
