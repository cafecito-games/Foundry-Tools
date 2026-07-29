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
	var _unknown_fields: PackedByteArray = PackedByteArray()

	## Decodes protobuf wire data into a new Badge message.
	static func from_bytes(data: PackedByteArray) -> (Badge?, ProtobufError):
		var _message: Badge = Badge.new()
		var _error: ProtobufError = _message.merge_from_bytes(data)
		if _error != ProtobufError.OK:
			var _failed: Badge? = null
			return (_failed, _error)
		return (_message, ProtobufError.OK)

	## Serializes this message to protobuf wire data.
	func to_bytes() -> PackedByteArray:
		var _result: PackedByteArray = PackedByteArray()
		_result.append_array(_unknown_fields)
		if code != "":
			_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
			var _code_data: PackedByteArray = Wire.encode_string(code)
			_result.append_array(Wire.encode_varint(_code_data.size()))
			_result.append_array(_code_data)
		return _result

	## Merges protobuf wire data into this message.
	func merge_from_bytes(_data: PackedByteArray) -> ProtobufError:
		var _offset: int = 0
		while _offset < _data.size():
			var _tag: VarintRead = Wire.decode_varint(_data, _offset)
			if _tag.error != ProtobufError.OK:
				return _tag.error
			_offset = _tag.offset
			var _wire_type: int = Wire.get_wire_type(_tag.value)
			match Wire.get_field_number(_tag.value):
				1:
					if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
						return ProtobufError.WIRE_TYPE_MISMATCH
					var _code_read: StringRead = Wire.read_string(_data, _offset)
					if _code_read.error != ProtobufError.OK:
						return _code_read.error
					code = _code_read.value
					_offset = _code_read.offset
				_:
					var _skipped: SkipRead = Wire.capture_field(_data, _offset, _tag.value, _wire_type, _unknown_fields)
					if _skipped.error != ProtobufError.OK:
						return _skipped.error
					_offset = _skipped.offset
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
var status: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED

## The primary protobuf field.
var primary: Slot? = null

## The slots protobuf field.
var slots: Array[Slot] = []

## The counts protobuf field.
var counts: Dictionary[String, int] = {}

## The badge protobuf field.
var badge: Badge? = null

## The tier protobuf field.
var tier: Tier = Tier.TIER_UNSPECIFIED

## The loadout protobuf field.
var loadout: Dictionary[String, Slot] = {}

## The payload protobuf oneof; null when no case is set.
var payload: PlayerPayloadCase? = null

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Player message.
static func from_bytes(data: PackedByteArray) -> (Player?, ProtobufError):
	var _message: Player = Player.new()
	var _error: ProtobufError = _message.merge_from_bytes(data)
	if _error != ProtobufError.OK:
		var _failed: Player? = null
		return (_failed, _error)
	return (_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _result: PackedByteArray = PackedByteArray()
	_result.append_array(_unknown_fields)
	if name != "":
		_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _name_data: PackedByteArray = Wire.encode_string(name)
		_result.append_array(Wire.encode_varint(_name_data.size()))
		_result.append_array(_name_data)
	if level != 0:
		_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_result.append_array(Wire.encode_varint(level))
	if active:
		_result.append_array(Wire.encode_varint(Wire.make_tag(3, Wire.WIRE_VARINT)))
		_result.append_array(Wire.encode_varint(1 if active else 0))
	if avatar.size() > 0:
		_result.append_array(Wire.encode_varint(Wire.make_tag(4, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(avatar.size()))
		_result.append_array(avatar)
	if nickname is String:
		_result.append_array(Wire.encode_varint(Wire.make_tag(5, Wire.WIRE_LENGTH_DELIMITED)))
		var _nickname_data: PackedByteArray = Wire.encode_string(nickname)
		_result.append_array(Wire.encode_varint(_nickname_data.size()))
		_result.append_array(_nickname_data)
	for _tags_item: String in tags:
		_result.append_array(Wire.encode_varint(Wire.make_tag(6, Wire.WIRE_LENGTH_DELIMITED)))
		var _tags_data: PackedByteArray = Wire.encode_string(_tags_item)
		_result.append_array(Wire.encode_varint(_tags_data.size()))
		_result.append_array(_tags_data)
	if scores.size() > 0:
		var _scores_data: PackedByteArray = PackedByteArray()
		for _scores_item: int in scores:
			_scores_data.append_array(Wire.encode_varint(_scores_item))
		_result.append_array(Wire.encode_varint(Wire.make_tag(7, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(_scores_data.size()))
		_result.append_array(_scores_data)
	if status != PlayerStatus.PLAYER_STATUS_UNSPECIFIED:
		_result.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
		_result.append_array(Wire.encode_varint(status.to_wire()))
	if primary is Slot:
		var _primary_data: PackedByteArray = primary.to_bytes()
		_result.append_array(Wire.encode_varint(Wire.make_tag(9, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(_primary_data.size()))
		_result.append_array(_primary_data)
	for _slots_item: Slot in slots:
		var _slots_data: PackedByteArray = _slots_item.to_bytes()
		_result.append_array(Wire.encode_varint(Wire.make_tag(10, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(_slots_data.size()))
		_result.append_array(_slots_data)
	for _counts_key: String in counts:
		var _counts_entry: PackedByteArray = PackedByteArray()
		_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _counts_key_data: PackedByteArray = Wire.encode_string(_counts_key)
		_counts_entry.append_array(Wire.encode_varint(_counts_key_data.size()))
		_counts_entry.append_array(_counts_key_data)
		_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))
		_counts_entry.append_array(Wire.encode_varint(counts[_counts_key]))
		_result.append_array(Wire.encode_varint(Wire.make_tag(11, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(_counts_entry.size()))
		_result.append_array(_counts_entry)
	match payload:
		PlayerPayloadCase.Text(var _payload_text):
			_result.append_array(Wire.encode_varint(Wire.make_tag(12, Wire.WIRE_LENGTH_DELIMITED)))
			var _payload_text_data: PackedByteArray = Wire.encode_string(_payload_text)
			_result.append_array(Wire.encode_varint(_payload_text_data.size()))
			_result.append_array(_payload_text_data)
		PlayerPayloadCase.Amount(var _payload_amount):
			_result.append_array(Wire.encode_varint(Wire.make_tag(13, Wire.WIRE_VARINT)))
			_result.append_array(Wire.encode_varint(_payload_amount))
		_:
			pass
	if badge is Badge:
		var _badge_data: PackedByteArray = badge.to_bytes()
		_result.append_array(Wire.encode_varint(Wire.make_tag(14, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(_badge_data.size()))
		_result.append_array(_badge_data)
	if tier != Tier.TIER_UNSPECIFIED:
		_result.append_array(Wire.encode_varint(Wire.make_tag(15, Wire.WIRE_VARINT)))
		_result.append_array(Wire.encode_varint(tier.to_wire()))
	for _loadout_key: String in loadout:
		var _loadout_entry: PackedByteArray = PackedByteArray()
		_loadout_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _loadout_key_data: PackedByteArray = Wire.encode_string(_loadout_key)
		_loadout_entry.append_array(Wire.encode_varint(_loadout_key_data.size()))
		_loadout_entry.append_array(_loadout_key_data)
		var _loadout_value_data: PackedByteArray = loadout[_loadout_key].to_bytes()
		_loadout_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_loadout_entry.append_array(Wire.encode_varint(_loadout_value_data.size()))
		_loadout_entry.append_array(_loadout_value_data)
		_result.append_array(Wire.encode_varint(Wire.make_tag(16, Wire.WIRE_LENGTH_DELIMITED)))
		_result.append_array(Wire.encode_varint(_loadout_entry.size()))
		_result.append_array(_loadout_entry)
	return _result

## Merges protobuf wire data into this message.
func merge_from_bytes(_data: PackedByteArray) -> ProtobufError:
	var _offset: int = 0
	while _offset < _data.size():
		var _tag: VarintRead = Wire.decode_varint(_data, _offset)
		if _tag.error != ProtobufError.OK:
			return _tag.error
		_offset = _tag.offset
		var _wire_type: int = Wire.get_wire_type(_tag.value)
		match Wire.get_field_number(_tag.value):
			1:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _name_read: StringRead = Wire.read_string(_data, _offset)
				if _name_read.error != ProtobufError.OK:
					return _name_read.error
				name = _name_read.value
				_offset = _name_read.offset
			2:
				if _wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _level_read: VarintRead = Wire.decode_varint(_data, _offset)
				if _level_read.error != ProtobufError.OK:
					return _level_read.error
				level = _level_read.value
				_offset = _level_read.offset
			3:
				if _wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _active_read: VarintRead = Wire.decode_varint(_data, _offset)
				if _active_read.error != ProtobufError.OK:
					return _active_read.error
				active = _active_read.value != 0
				_offset = _active_read.offset
			4:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _avatar_read: BytesRead = Wire.read_bytes(_data, _offset)
				if _avatar_read.error != ProtobufError.OK:
					return _avatar_read.error
				avatar = _avatar_read.value
				_offset = _avatar_read.offset
			5:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _nickname_read: StringRead = Wire.read_string(_data, _offset)
				if _nickname_read.error != ProtobufError.OK:
					return _nickname_read.error
				nickname = _nickname_read.value
				_offset = _nickname_read.offset
			6:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _tags_read: StringRead = Wire.read_string(_data, _offset)
				if _tags_read.error != ProtobufError.OK:
					return _tags_read.error
				tags.append(_tags_read.value)
				_offset = _tags_read.offset
			7:
				if _wire_type == Wire.WIRE_LENGTH_DELIMITED:
					var _scores_length: VarintRead = Wire.read_length(_data, _offset)
					if _scores_length.error != ProtobufError.OK:
						return _scores_length.error
					var _scores_end: int = _scores_length.offset + _scores_length.value
					_offset = _scores_length.offset
					while _offset < _scores_end:
						var _scores_packed: VarintRead = Wire.decode_varint(_data, _offset)
						if _scores_packed.error != ProtobufError.OK:
							return _scores_packed.error
						scores.append(_scores_packed.value)
						_offset = _scores_packed.offset
				elif _wire_type == Wire.WIRE_VARINT:
					var _scores_read: VarintRead = Wire.decode_varint(_data, _offset)
					if _scores_read.error != ProtobufError.OK:
						return _scores_read.error
					scores.append(_scores_read.value)
					_offset = _scores_read.offset
				else:
					return ProtobufError.WIRE_TYPE_MISMATCH
			8:
				if _wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _status_read: VarintRead = Wire.decode_varint(_data, _offset)
				if _status_read.error != ProtobufError.OK:
					return _status_read.error
				var _status_case: PlayerStatus? = PlayerStatus.from_wire(_status_read.value)
				if _status_case is PlayerStatus:
					status = _status_case
				else:
					_unknown_fields.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
					_unknown_fields.append_array(_data.slice(_offset, _status_read.offset))
				_offset = _status_read.offset
			9:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (primary is Slot):
					primary = Slot.new()
				var _primary_read: SkipRead = Wire.read_message(_data, _offset, primary)
				if _primary_read.error != ProtobufError.OK:
					return _primary_read.error
				_offset = _primary_read.offset
			10:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _slots_message: Slot = Slot.new()
				var _slots_read: SkipRead = Wire.read_message(_data, _offset, _slots_message)
				if _slots_read.error != ProtobufError.OK:
					return _slots_read.error
				slots.append(_slots_message)
				_offset = _slots_read.offset
			11:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _counts_length: VarintRead = Wire.read_length(_data, _offset)
				if _counts_length.error != ProtobufError.OK:
					return _counts_length.error
				var _counts_end: int = _counts_length.offset + _counts_length.value
				_offset = _counts_length.offset
				var _counts_key: String = ""
				var _counts_value: int = 0
				while _offset < _counts_end:
					var _counts_entry_tag: VarintRead = Wire.decode_varint(_data, _offset)
					if _counts_entry_tag.error != ProtobufError.OK:
						return _counts_entry_tag.error
					_offset = _counts_entry_tag.offset
					var _counts_entry_wire_type: int = Wire.get_wire_type(_counts_entry_tag.value)
					match Wire.get_field_number(_counts_entry_tag.value):
						1:
							if _counts_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _counts_key_read: StringRead = Wire.read_string(_data, _offset)
							if _counts_key_read.error != ProtobufError.OK:
								return _counts_key_read.error
							_counts_key = _counts_key_read.value
							_offset = _counts_key_read.offset
						2:
							if _counts_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _counts_value_read: VarintRead = Wire.decode_varint(_data, _offset)
							if _counts_value_read.error != ProtobufError.OK:
								return _counts_value_read.error
							_counts_value = _counts_value_read.value
							_offset = _counts_value_read.offset
						_:
							var _counts_skip: SkipRead = Wire.skip_field(_data, _offset, _counts_entry_wire_type)
							if _counts_skip.error != ProtobufError.OK:
								return _counts_skip.error
							_offset = _counts_skip.offset
				counts[_counts_key] = _counts_value
			12:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _payload_text_read: StringRead = Wire.read_string(_data, _offset)
				if _payload_text_read.error != ProtobufError.OK:
					return _payload_text_read.error
				payload = PlayerPayloadCase.Text(_payload_text_read.value)
				_offset = _payload_text_read.offset
			13:
				if _wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _payload_amount_read: VarintRead = Wire.decode_varint(_data, _offset)
				if _payload_amount_read.error != ProtobufError.OK:
					return _payload_amount_read.error
				payload = PlayerPayloadCase.Amount(_payload_amount_read.value)
				_offset = _payload_amount_read.offset
			14:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (badge is Badge):
					badge = Badge.new()
				var _badge_read: SkipRead = Wire.read_message(_data, _offset, badge)
				if _badge_read.error != ProtobufError.OK:
					return _badge_read.error
				_offset = _badge_read.offset
			15:
				if _wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _tier_read: VarintRead = Wire.decode_varint(_data, _offset)
				if _tier_read.error != ProtobufError.OK:
					return _tier_read.error
				var _tier_case: Tier? = Tier.from_wire(_tier_read.value)
				if _tier_case is Tier:
					tier = _tier_case
				else:
					_unknown_fields.append_array(Wire.encode_varint(Wire.make_tag(15, Wire.WIRE_VARINT)))
					_unknown_fields.append_array(_data.slice(_offset, _tier_read.offset))
				_offset = _tier_read.offset
			16:
				if _wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _loadout_length: VarintRead = Wire.read_length(_data, _offset)
				if _loadout_length.error != ProtobufError.OK:
					return _loadout_length.error
				var _loadout_end: int = _loadout_length.offset + _loadout_length.value
				_offset = _loadout_length.offset
				var _loadout_key: String = ""
				var _loadout_value: Slot = Slot.new()
				while _offset < _loadout_end:
					var _loadout_entry_tag: VarintRead = Wire.decode_varint(_data, _offset)
					if _loadout_entry_tag.error != ProtobufError.OK:
						return _loadout_entry_tag.error
					_offset = _loadout_entry_tag.offset
					var _loadout_entry_wire_type: int = Wire.get_wire_type(_loadout_entry_tag.value)
					match Wire.get_field_number(_loadout_entry_tag.value):
						1:
							if _loadout_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _loadout_key_read: StringRead = Wire.read_string(_data, _offset)
							if _loadout_key_read.error != ProtobufError.OK:
								return _loadout_key_read.error
							_loadout_key = _loadout_key_read.value
							_offset = _loadout_key_read.offset
						2:
							if _loadout_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _loadout_value_read: SkipRead = Wire.read_message(_data, _offset, _loadout_value)
							if _loadout_value_read.error != ProtobufError.OK:
								return _loadout_value_read.error
							_offset = _loadout_value_read.offset
						_:
							var _loadout_skip: SkipRead = Wire.skip_field(_data, _offset, _loadout_entry_wire_type)
							if _loadout_skip.error != ProtobufError.OK:
								return _loadout_skip.error
							_offset = _loadout_skip.offset
				loadout[_loadout_key] = _loadout_value
			_:
				var _skipped: SkipRead = Wire.capture_field(_data, _offset, _tag.value, _wire_type, _unknown_fields)
				if _skipped.error != ProtobufError.OK:
					return _skipped.error
				_offset = _skipped.offset
	return ProtobufError.OK
