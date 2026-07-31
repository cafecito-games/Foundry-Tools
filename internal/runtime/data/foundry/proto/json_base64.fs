namespace foundry.proto

## Base64 for the canonical JSON mapping of a bytes field.
##
## Output is always the standard alphabet with padding, which is what the
## canonical form specifies. Input is looser on purpose: the mapping requires a
## parser to accept the URL-safe alphabet as well, and padding is optional in
## practice, so both are normalized before the engine decoder sees them.
class_name JsonBase64 extends RefCounted

static func encode(value: PackedByteArray) -> String:
	return Marshalls.raw_to_base64(value)

static func decode(text: String) -> (PackedByteArray, ProtobufError):
	var normalized: String = text.replace("-", "+").replace("_", "/")
	var remainder: int = normalized.length() % 4
	if remainder == 1:
		return (PackedByteArray(), ProtobufError.JSON_TYPE_MISMATCH)
	if remainder == 2:
		normalized += "=="
	elif remainder == 3:
		normalized += "="
	if not _is_base64(normalized):
		return (PackedByteArray(), ProtobufError.JSON_TYPE_MISMATCH)
	return (Marshalls.base64_to_raw(normalized), ProtobufError.OK)

## The engine decoder does not report a bad character or a bad padding count,
## so both are checked here rather than being told about a silently truncated
## or silently emptied result. A quantum has at most two padding characters,
## and once padding starts nothing but more padding may follow.
static func _is_base64(text: String) -> bool:
	var index: int = 0
	var padding_count: int = 0
	while index < text.length():
		var character: String = text.substr(index, 1)
		if character == "=":
			padding_count += 1
		elif padding_count > 0:
			return false
		elif not _is_base64_character(character):
			return false
		index += 1
	return padding_count <= 2

static func _is_base64_character(character: String) -> bool:
	if character >= "A" and character <= "Z":
		return true
	if character >= "a" and character <= "z":
		return true
	if character >= "0" and character <= "9":
		return true
	return character == "+" or character == "/"
