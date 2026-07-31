namespace foundry.proto

## Path conversion for the canonical JSON mapping of google.protobuf.FieldMask.
##
## A mask serializes as one string of comma-joined paths, each path a
## dot-separated chain of field names carried in lowerCamelCase. Only a path
## built from lower_snake_case field names round-trips through that mapping
## without changing meaning, so anything else -- an out-of-alphabet character,
## an empty segment, or an underscore placed where it cannot survive the
## conversion -- is refused rather than emitted as something that would come
## back different.
class_name JsonFieldMask extends RefCounted

static func to_json(paths: Array[String]) -> (String, ProtobufError):
	var converted: Array[String] = []
	for path in paths:
		if not _is_lower_snake_case(path):
			return ("", ProtobufError.JSON_TYPE_MISMATCH)
		converted.append(_to_camel_case(path))
	return (",".join(converted), ProtobufError.OK)

static func from_json(text: String) -> (Array[String], ProtobufError):
	var paths: Array[String] = []
	if text.length() == 0:
		return (paths, ProtobufError.OK)
	for entry in text.split(","):
		if not _is_lower_camel_case(entry):
			return ([], ProtobufError.JSON_TYPE_MISMATCH)
		paths.append(_to_snake_case(entry))
	return (paths, ProtobufError.OK)

static func _to_camel_case(path: String) -> String:
	var result: String = ""
	var capitalize_next: bool = false
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character == "_":
			capitalize_next = true
		elif capitalize_next:
			result += character.to_upper()
			capitalize_next = false
		else:
			result += character
		index += 1
	return result

static func _to_snake_case(path: String) -> String:
	var result: String = ""
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character >= "A" and character <= "Z":
			result += "_" + character.to_lower()
		else:
			result += character
		index += 1
	return result

## Only a lower_snake_case identifier character set survives the round trip.
## A leading underscore, or one right after a dot, is fine -- "_foo" and
## "Foo" convert to each other cleanly -- but a trailing underscore, a
## doubled one, or one immediately before a digit or a dot is swallowed by
## _to_camel_case without leaving a mark _to_snake_case can recover, and
## anything outside [a-z0-9_.], a comma above all, is not a protobuf
## identifier character to begin with. Each is refused up front instead of
## being emitted as a path that would come back different.
static func _is_lower_snake_case(path: String) -> bool:
	if path.length() == 0:
		return false
	if path.substr(0, 1) == "." or path.substr(path.length() - 1, 1) == ".":
		return false
	var previous: String = ""
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		var is_lowercase_letter: bool = character >= "a" and character <= "z"
		var is_digit: bool = character >= "0" and character <= "9"
		if not (is_lowercase_letter or is_digit or character == "_" or character == "."):
			return false
		if character == "." and previous == ".":
			return false
		if character == "_":
			if index + 1 >= path.length():
				return false
			var next_character: String = path.substr(index + 1, 1)
			if next_character == "_" or next_character == "." or (next_character >= "0" and next_character <= "9"):
				return false
		previous = character
		index += 1
	return true

## The canonical form never carries an underscore -- that is the marker that
## survives the round trip in the other direction -- so a path that contains
## one is not a JSON field mask this helper produced. Neither is a path with
## an empty segment (a leading, trailing, or doubled dot) or a character
## outside a protobuf identifier's alphabet; a capitalized segment start such
## as "Foo" is fine, because that is exactly what _to_camel_case produces
## from a leading underscore.
static func _is_lower_camel_case(path: String) -> bool:
	if path.length() == 0:
		return false
	if path.substr(0, 1) == "." or path.substr(path.length() - 1, 1) == ".":
		return false
	var previous: String = ""
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		var is_letter: bool = (character >= "a" and character <= "z") or (character >= "A" and character <= "Z")
		var is_digit: bool = character >= "0" and character <= "9"
		if not (is_letter or is_digit or character == "."):
			return false
		if character == "." and previous == ".":
			return false
		previous = character
		index += 1
	return true
