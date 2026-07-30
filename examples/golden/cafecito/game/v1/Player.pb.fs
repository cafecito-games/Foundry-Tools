namespace cafecito.game.v1
import foundry.proto

## A player and everything the wire format has to round trip.
final class_name Player extends RefCounted uses Message

## Generated protobuf enum binding for Tier.
enum Tier:
	TIER_UNSPECIFIED = 0
	TIER_GOLD = 1

	## Returns the protobuf wire value for this case.
	func to_wire() -> int:
		return self as int

	## Returns the case for a protobuf wire value, or null if it names none.
	static func from_wire(value: int) -> Self?:
		match value:
			0:
				return Tier.TIER_UNSPECIFIED
			1:
				return Tier.TIER_GOLD
			_:
				return null

## Nested types keep proto's scoping: Player.Badge, Player.Tier.
final class Badge extends RefCounted uses Message:
	## The code protobuf field.
	var code: String = ""

	## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
	var _pb_unknown_fields: PackedByteArray = PackedByteArray()

	## Decodes protobuf wire data into a new Badge message.
	static func from_bytes(_pb_data: PackedByteArray) -> (Badge?, ProtobufError):
		var _pb_message: Badge = Badge.new()
		var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
		if _pb_error != ProtobufError.OK:
			var _pb_failed: Badge? = null
			return (_pb_failed, _pb_error)
		return (_pb_message, ProtobufError.OK)

	## Serializes this message to protobuf wire data.
	func to_bytes() -> PackedByteArray:
		var _pb_result: PackedByteArray = PackedByteArray()
		if code != "":
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
			var _pb_code_data: PackedByteArray = Wire.encode_string(code)
			_pb_result.append_array(Wire.encode_varint(_pb_code_data.size()))
			_pb_result.append_array(_pb_code_data)
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
					var _pb_code_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
					if _pb_code_read.error != ProtobufError.OK:
						return _pb_code_read.error
					code = _pb_code_read.value
					_pb_offset = _pb_code_read.offset
				_:
					var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
					if _pb_skipped.error != ProtobufError.OK:
						return _pb_skipped.error
					_pb_offset = _pb_skipped.offset
		return ProtobufError.OK

## The name protobuf field.
var name: String = ""

## The level protobuf field.
var level: int = 0

## The active protobuf field.
var active: bool = false

## The avatar protobuf field.
var avatar: PackedByteArray = PackedByteArray()

## Explicit presence: an empty nickname is distinct from an absent one.
var nickname: String? = null

## The tags protobuf field.
var tags: Array[String] = []

## The scores protobuf field.
var scores: Array[int] = []

## The status protobuf field.
var status: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED:
	set(_pb_value):
		_pb_status_unknown = PackedByteArray()
		status = _pb_value

## The primary protobuf field.
var primary: Slot? = null

## The slots protobuf field.
var slots: Array[Slot] = []

## The counts protobuf field.
var counts: Dictionary[String, int] = {}

## The badge protobuf field.
var badge: Badge? = null

## The tier protobuf field.
var tier: Tier = Tier.TIER_UNSPECIFIED:
	set(_pb_value):
		_pb_tier_unknown = PackedByteArray()
		tier = _pb_value

## The loadout protobuf field.
var loadout: Dictionary[String, Slot] = {}

## Fixed-width and zig-zag scalars: a float and a double are IEEE-754 rather
## than varints, fixed64 spends eight bytes to spend the same on every value,
## and sint32 zig-zags so a negative costs one byte instead of ten.
var accuracy: float = 0.0

## The play_time_seconds protobuf field.
var play_time_seconds: float = 0.0

## The session_id protobuf field.
var session_id: int = 0

## The rating_delta protobuf field.
var rating_delta: int = 0

## The payload protobuf oneof; null when no case is set.
var payload: PlayerPayloadCase? = null

## Raw bytes of an unrecognized status value, kept so a re-encode is lossless.
var _pb_status_unknown: PackedByteArray = PackedByteArray()

## Raw bytes of an unrecognized tier value, kept so a re-encode is lossless.
var _pb_tier_unknown: PackedByteArray = PackedByteArray()

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Player message.
static func from_bytes(_pb_data: PackedByteArray) -> (Player?, ProtobufError):
	var _pb_message: Player = Player.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Player? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if name != "":
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_name_data: PackedByteArray = Wire.encode_string(name)
		_pb_result.append_array(Wire.encode_varint(_pb_name_data.size()))
		_pb_result.append_array(_pb_name_data)
	if level != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(level))
	if active:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(3, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(1 if active else 0))
	if avatar.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(4, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(avatar.size()))
		_pb_result.append_array(avatar)
	if nickname is String:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(5, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_nickname_data: PackedByteArray = Wire.encode_string(nickname)
		_pb_result.append_array(Wire.encode_varint(_pb_nickname_data.size()))
		_pb_result.append_array(_pb_nickname_data)
	for _pb_tags_item: String in tags:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(6, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_tags_data: PackedByteArray = Wire.encode_string(_pb_tags_item)
		_pb_result.append_array(Wire.encode_varint(_pb_tags_data.size()))
		_pb_result.append_array(_pb_tags_data)
	if scores.size() > 0:
		var _pb_scores_data: PackedByteArray = PackedByteArray()
		for _pb_scores_item: int in scores:
			_pb_scores_data.append_array(Wire.encode_varint(_pb_scores_item))
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(7, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_scores_data.size()))
		_pb_result.append_array(_pb_scores_data)
	if _pb_status_unknown.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
		_pb_result.append_array(_pb_status_unknown)
	elif status != PlayerStatus.PLAYER_STATUS_UNSPECIFIED:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(status.to_wire()))
	if primary is Slot:
		var _pb_primary_data: PackedByteArray = primary.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(9, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_primary_data.size()))
		_pb_result.append_array(_pb_primary_data)
	for _pb_slots_item: Slot in slots:
		var _pb_slots_data: PackedByteArray = _pb_slots_item.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(10, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_slots_data.size()))
		_pb_result.append_array(_pb_slots_data)
	for _pb_counts_key: String in counts:
		var _pb_counts_entry: PackedByteArray = PackedByteArray()
		_pb_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_counts_key_data: PackedByteArray = Wire.encode_string(_pb_counts_key)
		_pb_counts_entry.append_array(Wire.encode_varint(_pb_counts_key_data.size()))
		_pb_counts_entry.append_array(_pb_counts_key_data)
		_pb_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_pb_counts_entry.append_array(Wire.encode_varint(counts[_pb_counts_key]))
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(11, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_counts_entry.size()))
		_pb_result.append_array(_pb_counts_entry)
	match payload:
		PlayerPayloadCase.Text(var _pb_payload_text):
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(12, Wire.WIRE_LENGTH_DELIMITED)))
			var _pb_payload_text_data: PackedByteArray = Wire.encode_string(_pb_payload_text)
			_pb_result.append_array(Wire.encode_varint(_pb_payload_text_data.size()))
			_pb_result.append_array(_pb_payload_text_data)
		PlayerPayloadCase.Amount(var _pb_payload_amount):
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(13, Wire.WIRE_VARINT)))
			_pb_result.append_array(Wire.encode_varint(_pb_payload_amount))
		_:
			pass
	if badge is Badge:
		var _pb_badge_data: PackedByteArray = badge.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(14, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_badge_data.size()))
		_pb_result.append_array(_pb_badge_data)
	if _pb_tier_unknown.size() > 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(15, Wire.WIRE_VARINT)))
		_pb_result.append_array(_pb_tier_unknown)
	elif tier != Tier.TIER_UNSPECIFIED:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(15, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_varint(tier.to_wire()))
	for _pb_loadout_key: String in loadout:
		var _pb_loadout_entry: PackedByteArray = PackedByteArray()
		_pb_loadout_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_loadout_key_data: PackedByteArray = Wire.encode_string(_pb_loadout_key)
		_pb_loadout_entry.append_array(Wire.encode_varint(_pb_loadout_key_data.size()))
		_pb_loadout_entry.append_array(_pb_loadout_key_data)
		var _pb_loadout_value_data: PackedByteArray = loadout[_pb_loadout_key].to_bytes()
		_pb_loadout_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_loadout_entry.append_array(Wire.encode_varint(_pb_loadout_value_data.size()))
		_pb_loadout_entry.append_array(_pb_loadout_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(16, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_loadout_entry.size()))
		_pb_result.append_array(_pb_loadout_entry)
	if not Wire.is_default_float32(accuracy):
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(17, Wire.WIRE_32BIT)))
		_pb_result.append_array(Wire.encode_float(accuracy))
	if not Wire.is_default_float(play_time_seconds):
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(18, Wire.WIRE_64BIT)))
		_pb_result.append_array(Wire.encode_double(play_time_seconds))
	if session_id != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(19, Wire.WIRE_64BIT)))
		_pb_result.append_array(Wire.encode_fixed64(session_id))
	if rating_delta != 0:
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(20, Wire.WIRE_VARINT)))
		_pb_result.append_array(Wire.encode_sint32(rating_delta))
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
				var _pb_name_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_name_read.error != ProtobufError.OK:
					return _pb_name_read.error
				name = _pb_name_read.value
				_pb_offset = _pb_name_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_level_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_level_read.error != ProtobufError.OK:
					return _pb_level_read.error
				level = _pb_level_read.value
				_pb_offset = _pb_level_read.offset
			3:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_active_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_active_read.error != ProtobufError.OK:
					return _pb_active_read.error
				active = _pb_active_read.value != 0
				_pb_offset = _pb_active_read.offset
			4:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_avatar_read: BytesRead = Wire.read_bytes(_pb_data, _pb_offset)
				if _pb_avatar_read.error != ProtobufError.OK:
					return _pb_avatar_read.error
				avatar = _pb_avatar_read.value
				_pb_offset = _pb_avatar_read.offset
			5:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_nickname_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_nickname_read.error != ProtobufError.OK:
					return _pb_nickname_read.error
				nickname = _pb_nickname_read.value
				_pb_offset = _pb_nickname_read.offset
			6:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_tags_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_tags_read.error != ProtobufError.OK:
					return _pb_tags_read.error
				tags.append(_pb_tags_read.value)
				_pb_offset = _pb_tags_read.offset
			7:
				if _pb_wire_type == Wire.WIRE_LENGTH_DELIMITED:
					var _pb_scores_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
					if _pb_scores_length.error != ProtobufError.OK:
						return _pb_scores_length.error
					var _pb_scores_end: int = _pb_scores_length.offset + _pb_scores_length.value
					_pb_offset = _pb_scores_length.offset
					while _pb_offset < _pb_scores_end:
						var _pb_scores_packed: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
						if _pb_scores_packed.error != ProtobufError.OK:
							return _pb_scores_packed.error
						if _pb_scores_packed.offset > _pb_scores_end:
							return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
						scores.append(_pb_scores_packed.value)
						_pb_offset = _pb_scores_packed.offset
				elif _pb_wire_type == Wire.WIRE_VARINT:
					var _pb_scores_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_scores_read.error != ProtobufError.OK:
						return _pb_scores_read.error
					scores.append(_pb_scores_read.value)
					_pb_offset = _pb_scores_read.offset
				else:
					return ProtobufError.WIRE_TYPE_MISMATCH
			8:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_status_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_status_read.error != ProtobufError.OK:
					return _pb_status_read.error
				var _pb_status_case: PlayerStatus? = PlayerStatus.from_wire(_pb_status_read.value)
				if _pb_status_case is PlayerStatus:
					status = _pb_status_case
				else:
					status = PlayerStatus.PLAYER_STATUS_UNSPECIFIED
					_pb_status_unknown = _pb_data.slice(_pb_offset, _pb_status_read.offset)
				_pb_offset = _pb_status_read.offset
			9:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (primary is Slot):
					primary = Slot.new()
				var _pb_primary_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, primary)
				if _pb_primary_read.error != ProtobufError.OK:
					return _pb_primary_read.error
				_pb_offset = _pb_primary_read.offset
			10:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_slots_message: Slot = Slot.new()
				var _pb_slots_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_slots_message)
				if _pb_slots_read.error != ProtobufError.OK:
					return _pb_slots_read.error
				slots.append(_pb_slots_message)
				_pb_offset = _pb_slots_read.offset
			11:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_counts_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_counts_length.error != ProtobufError.OK:
					return _pb_counts_length.error
				var _pb_counts_end: int = _pb_counts_length.offset + _pb_counts_length.value
				_pb_offset = _pb_counts_length.offset
				var _pb_counts_key: String = ""
				var _pb_counts_value: int = 0
				while _pb_offset < _pb_counts_end:
					var _pb_counts_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_counts_entry_tag.error != ProtobufError.OK:
						return _pb_counts_entry_tag.error
					if _pb_counts_entry_tag.offset > _pb_counts_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_counts_entry_tag.offset
					var _pb_counts_entry_wire_type: int = Wire.get_wire_type(_pb_counts_entry_tag.value)
					match Wire.get_field_number(_pb_counts_entry_tag.value):
						1:
							if _pb_counts_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_counts_key_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_counts_key_read.error != ProtobufError.OK:
								return _pb_counts_key_read.error
							if _pb_counts_key_read.offset > _pb_counts_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_counts_key = _pb_counts_key_read.value
							_pb_offset = _pb_counts_key_read.offset
						2:
							if _pb_counts_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_counts_value_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
							if _pb_counts_value_read.error != ProtobufError.OK:
								return _pb_counts_value_read.error
							if _pb_counts_value_read.offset > _pb_counts_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_counts_value = _pb_counts_value_read.value
							_pb_offset = _pb_counts_value_read.offset
						_:
							var _pb_counts_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_counts_entry_wire_type)
							if _pb_counts_skip.error != ProtobufError.OK:
								return _pb_counts_skip.error
							if _pb_counts_skip.offset > _pb_counts_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_counts_skip.offset
				counts[_pb_counts_key] = _pb_counts_value
			12:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_payload_text_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_payload_text_read.error != ProtobufError.OK:
					return _pb_payload_text_read.error
				payload = PlayerPayloadCase.Text(_pb_payload_text_read.value)
				_pb_offset = _pb_payload_text_read.offset
			13:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_payload_amount_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_payload_amount_read.error != ProtobufError.OK:
					return _pb_payload_amount_read.error
				payload = PlayerPayloadCase.Amount(_pb_payload_amount_read.value)
				_pb_offset = _pb_payload_amount_read.offset
			14:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (badge is Badge):
					badge = Badge.new()
				var _pb_badge_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, badge)
				if _pb_badge_read.error != ProtobufError.OK:
					return _pb_badge_read.error
				_pb_offset = _pb_badge_read.offset
			15:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_tier_read: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
				if _pb_tier_read.error != ProtobufError.OK:
					return _pb_tier_read.error
				var _pb_tier_case: Tier? = Tier.from_wire(_pb_tier_read.value)
				if _pb_tier_case is Tier:
					tier = _pb_tier_case
				else:
					tier = Tier.TIER_UNSPECIFIED
					_pb_tier_unknown = _pb_data.slice(_pb_offset, _pb_tier_read.offset)
				_pb_offset = _pb_tier_read.offset
			16:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_loadout_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_loadout_length.error != ProtobufError.OK:
					return _pb_loadout_length.error
				var _pb_loadout_end: int = _pb_loadout_length.offset + _pb_loadout_length.value
				_pb_offset = _pb_loadout_length.offset
				var _pb_loadout_key: String = ""
				var _pb_loadout_value: Slot = Slot.new()
				while _pb_offset < _pb_loadout_end:
					var _pb_loadout_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_loadout_entry_tag.error != ProtobufError.OK:
						return _pb_loadout_entry_tag.error
					if _pb_loadout_entry_tag.offset > _pb_loadout_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_loadout_entry_tag.offset
					var _pb_loadout_entry_wire_type: int = Wire.get_wire_type(_pb_loadout_entry_tag.value)
					match Wire.get_field_number(_pb_loadout_entry_tag.value):
						1:
							if _pb_loadout_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_loadout_key_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_loadout_key_read.error != ProtobufError.OK:
								return _pb_loadout_key_read.error
							if _pb_loadout_key_read.offset > _pb_loadout_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_loadout_key = _pb_loadout_key_read.value
							_pb_offset = _pb_loadout_key_read.offset
						2:
							if _pb_loadout_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_loadout_value_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_loadout_value)
							if _pb_loadout_value_read.error != ProtobufError.OK:
								return _pb_loadout_value_read.error
							if _pb_loadout_value_read.offset > _pb_loadout_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_loadout_value_read.offset
						_:
							var _pb_loadout_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_loadout_entry_wire_type)
							if _pb_loadout_skip.error != ProtobufError.OK:
								return _pb_loadout_skip.error
							if _pb_loadout_skip.offset > _pb_loadout_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_loadout_skip.offset
				loadout[_pb_loadout_key] = _pb_loadout_value
			17:
				if _pb_wire_type != Wire.WIRE_32BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_accuracy_read: FloatRead = Wire.read_float(_pb_data, _pb_offset)
				if _pb_accuracy_read.error != ProtobufError.OK:
					return _pb_accuracy_read.error
				accuracy = _pb_accuracy_read.value
				_pb_offset = _pb_accuracy_read.offset
			18:
				if _pb_wire_type != Wire.WIRE_64BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_play_time_seconds_read: FloatRead = Wire.read_double(_pb_data, _pb_offset)
				if _pb_play_time_seconds_read.error != ProtobufError.OK:
					return _pb_play_time_seconds_read.error
				play_time_seconds = _pb_play_time_seconds_read.value
				_pb_offset = _pb_play_time_seconds_read.offset
			19:
				if _pb_wire_type != Wire.WIRE_64BIT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_session_id_read: FixedRead = Wire.read_fixed64(_pb_data, _pb_offset)
				if _pb_session_id_read.error != ProtobufError.OK:
					return _pb_session_id_read.error
				session_id = _pb_session_id_read.value
				_pb_offset = _pb_session_id_read.offset
			20:
				if _pb_wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_rating_delta_read: VarintRead = Wire.read_sint32(_pb_data, _pb_offset)
				if _pb_rating_delta_read.error != ProtobufError.OK:
					return _pb_rating_delta_read.error
				rating_delta = _pb_rating_delta_read.value
				_pb_offset = _pb_rating_delta_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK
