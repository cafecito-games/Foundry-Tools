namespace foundry.proto

## Path conversion for the canonical JSON mapping of google.protobuf.FieldMask.
##
## A mask serializes as one string of comma-joined paths, each path a
## dot-separated chain of field names carried in lowerCamelCase. The conversion
## is only reversible when the proto field names are lower_snake_case, so a path
## that already contains an uppercase letter is refused rather than emitted as
## something that would come back different.
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

## A dot separating segments and a digit inside a name are both fine. An
## uppercase letter, an empty segment (a leading, trailing, or doubled dot),
## and a stray underscore all make a path unable to survive the round trip:
## an underscore at either end of a segment, or two in a row, is silently
## swallowed by _to_camel_case rather than reproduced by _to_snake_case, and
## so is an underscore immediately before a digit, because to_upper() on a
## digit is the identity -- capitalize_next fires but leaves no visible mark
## to reverse. Each is refused up front instead of being emitted as a path
## that would come back different.
static func _is_lower_snake_case(path: String) -> bool:
	if path.length() == 0:
		return false
	if path.substr(0, 1) == "." or path.substr(path.length() - 1, 1) == ".":
		return false
	var previous: String = ""
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character >= "A" and character <= "Z":
			return false
		if character == "." and previous == ".":
			return false
		if character == "_":
			if previous == "" or previous == "_" or previous == ".":
				return false
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
## one is not a JSON field mask this helper produced. Neither is one that
## opens or closes a segment with an empty name, or that capitalizes the
## first letter of a segment: _to_snake_case would turn that leading capital
## into a leading or post-dot underscore, which _is_lower_snake_case refuses
## to re-encode, so the mismatch is caught here instead.
static func _is_lower_camel_case(path: String) -> bool:
	if path.length() == 0:
		return false
	var previous: String = ""
	var at_segment_start: bool = true
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character == "_":
			return false
		if character == ".":
			if previous == "" or previous == ".":
				return false
			at_segment_start = true
			previous = character
			index += 1
			continue
		if at_segment_start and character >= "A" and character <= "Z":
			return false
		at_segment_start = false
		previous = character
		index += 1
	return previous != "."
