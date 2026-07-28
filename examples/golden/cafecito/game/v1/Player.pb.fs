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
		match self:
			Tier.TIER_GOLD:
				return 1
			_:
				return 0

	## Returns the case for a protobuf wire value, tolerating unknown values.
	static func from_wire(value: int) -> Tier:
		match value:
			1:
				return Tier.TIER_GOLD
			_:
				return Tier.TIER_UNSPECIFIED

## Nested types keep proto's scoping: Player.Badge, Player.Tier.
class Badge extends RefCounted uses Message:
	## The code protobuf field.
	var code: String = ""

	## Decodes protobuf wire data into a new Badge message.
	static func from_bytes(data: PackedByteArray) -> (Badge?, ProtobufError):
		var message: Badge = Badge.new()
		var error: ProtobufError = message.merge_from_bytes(data)
		if error != ProtobufError.OK:
			var failed: Badge? = null
			return (failed, error)
		return (message, ProtobufError.OK)

	## Serializes this message to protobuf wire data.
	func to_bytes() -> PackedByteArray:
		var result: PackedByteArray = PackedByteArray()
		if code != "":
			result.append_array(Wire.encode_varint(10))
			var code_data: PackedByteArray = Wire.encode_string(code)
			result.append_array(Wire.encode_varint(code_data.size()))
			result.append_array(code_data)
		return result

	## Merges protobuf wire data into this message.
	func merge_from_bytes(data: PackedByteArray) -> ProtobufError:
		var offset: int = 0
		while offset < data.size():
			var tag_read: VarintRead = Wire.decode_varint(data, offset)
			if tag_read.error != ProtobufError.OK:
				return tag_read.error
			offset = tag_read.offset
			var wire_type: int = Wire.get_wire_type(tag_read.value)
			match Wire.get_field_number(tag_read.value):
				1:
					if wire_type != Wire.WIRE_LENGTH_DELIMITED:
						return ProtobufError.WIRE_TYPE_MISMATCH
					var code_length: VarintRead = Wire.decode_varint(data, offset)
					if code_length.error != ProtobufError.OK:
						return code_length.error
					offset = code_length.offset
					if code_length.value < 0 or offset + code_length.value > data.size():
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					var code_read: StringRead = Wire.decode_string(data, offset, code_length.value)
					if code_read.error != ProtobufError.OK:
						return code_read.error
					code = code_read.value
					offset = code_read.offset
				_:
					var skipped: SkipRead = Wire.skip_field(data, offset, wire_type)
					if skipped.error != ProtobufError.OK:
						return skipped.error
					offset = skipped.offset
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

## Decodes protobuf wire data into a new Player message.
static func from_bytes(data: PackedByteArray) -> (Player?, ProtobufError):
	var message: Player = Player.new()
	var error: ProtobufError = message.merge_from_bytes(data)
	if error != ProtobufError.OK:
		var failed: Player? = null
		return (failed, error)
	return (message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	if name != "":
		result.append_array(Wire.encode_varint(10))
		var name_data: PackedByteArray = Wire.encode_string(name)
		result.append_array(Wire.encode_varint(name_data.size()))
		result.append_array(name_data)
	if level != 0:
		result.append_array(Wire.encode_varint(16))
		result.append_array(Wire.encode_varint(level))
	if active:
		result.append_array(Wire.encode_varint(24))
		result.append_array(Wire.encode_varint(1 if active else 0))
	if avatar.size() > 0:
		result.append_array(Wire.encode_varint(34))
		result.append_array(Wire.encode_varint(avatar.size()))
		result.append_array(avatar)
	if nickname is String:
		result.append_array(Wire.encode_varint(42))
		var nickname_data: PackedByteArray = Wire.encode_string(nickname)
		result.append_array(Wire.encode_varint(nickname_data.size()))
		result.append_array(nickname_data)
	for tags_item: String in tags:
		result.append_array(Wire.encode_varint(50))
		var tags_data: PackedByteArray = Wire.encode_string(tags_item)
		result.append_array(Wire.encode_varint(tags_data.size()))
		result.append_array(tags_data)
	if scores.size() > 0:
		var scores_data: PackedByteArray = PackedByteArray()
		for scores_item: int in scores:
			scores_data.append_array(Wire.encode_varint(scores_item))
		result.append_array(Wire.encode_varint(58))
		result.append_array(Wire.encode_varint(scores_data.size()))
		result.append_array(scores_data)
	if status != PlayerStatus.PLAYER_STATUS_UNSPECIFIED:
		result.append_array(Wire.encode_varint(64))
		result.append_array(Wire.encode_varint(status.to_wire()))
	if primary is Slot:
		var primary_data: PackedByteArray = primary.to_bytes()
		result.append_array(Wire.encode_varint(74))
		result.append_array(Wire.encode_varint(primary_data.size()))
		result.append_array(primary_data)
	for slots_item: Slot in slots:
		var slots_data: PackedByteArray = slots_item.to_bytes()
		result.append_array(Wire.encode_varint(82))
		result.append_array(Wire.encode_varint(slots_data.size()))
		result.append_array(slots_data)
	for counts_key: String in counts:
		var counts_entry: PackedByteArray = PackedByteArray()
		counts_entry.append_array(Wire.encode_varint(10))
		var counts_key_data: PackedByteArray = Wire.encode_string(counts_key)
		counts_entry.append_array(Wire.encode_varint(counts_key_data.size()))
		counts_entry.append_array(counts_key_data)
		counts_entry.append_array(Wire.encode_varint(16))
		counts_entry.append_array(Wire.encode_varint(counts[counts_key]))
		result.append_array(Wire.encode_varint(90))
		result.append_array(Wire.encode_varint(counts_entry.size()))
		result.append_array(counts_entry)
	if badge is Badge:
		var badge_data: PackedByteArray = badge.to_bytes()
		result.append_array(Wire.encode_varint(114))
		result.append_array(Wire.encode_varint(badge_data.size()))
		result.append_array(badge_data)
	if tier != Tier.TIER_UNSPECIFIED:
		result.append_array(Wire.encode_varint(120))
		result.append_array(Wire.encode_varint(tier.to_wire()))
	for loadout_key: String in loadout:
		var loadout_entry: PackedByteArray = PackedByteArray()
		loadout_entry.append_array(Wire.encode_varint(10))
		var loadout_key_data: PackedByteArray = Wire.encode_string(loadout_key)
		loadout_entry.append_array(Wire.encode_varint(loadout_key_data.size()))
		loadout_entry.append_array(loadout_key_data)
		var loadout_value_data: PackedByteArray = loadout[loadout_key].to_bytes()
		loadout_entry.append_array(Wire.encode_varint(18))
		loadout_entry.append_array(Wire.encode_varint(loadout_value_data.size()))
		loadout_entry.append_array(loadout_value_data)
		result.append_array(Wire.encode_varint(130))
		result.append_array(Wire.encode_varint(loadout_entry.size()))
		result.append_array(loadout_entry)
	match payload:
		PlayerPayloadCase.Text(var text):
			result.append_array(Wire.encode_varint(98))
			var text_data: PackedByteArray = Wire.encode_string(text)
			result.append_array(Wire.encode_varint(text_data.size()))
			result.append_array(text_data)
		PlayerPayloadCase.Amount(var amount):
			result.append_array(Wire.encode_varint(104))
			result.append_array(Wire.encode_varint(amount))
		_:
			pass
	return result

## Merges protobuf wire data into this message.
func merge_from_bytes(data: PackedByteArray) -> ProtobufError:
	var offset: int = 0
	while offset < data.size():
		var tag_read: VarintRead = Wire.decode_varint(data, offset)
		if tag_read.error != ProtobufError.OK:
			return tag_read.error
		offset = tag_read.offset
		var wire_type: int = Wire.get_wire_type(tag_read.value)
		match Wire.get_field_number(tag_read.value):
			1:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var name_length: VarintRead = Wire.decode_varint(data, offset)
				if name_length.error != ProtobufError.OK:
					return name_length.error
				offset = name_length.offset
				if name_length.value < 0 or offset + name_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var name_read: StringRead = Wire.decode_string(data, offset, name_length.value)
				if name_read.error != ProtobufError.OK:
					return name_read.error
				name = name_read.value
				offset = name_read.offset
			2:
				if wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var level_read: VarintRead = Wire.decode_varint(data, offset)
				if level_read.error != ProtobufError.OK:
					return level_read.error
				level = level_read.value
				offset = level_read.offset
			3:
				if wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var active_read: VarintRead = Wire.decode_varint(data, offset)
				if active_read.error != ProtobufError.OK:
					return active_read.error
				active = active_read.value != 0
				offset = active_read.offset
			4:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var avatar_length: VarintRead = Wire.decode_varint(data, offset)
				if avatar_length.error != ProtobufError.OK:
					return avatar_length.error
				offset = avatar_length.offset
				if avatar_length.value < 0 or offset + avatar_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var avatar_read: BytesRead = Wire.decode_bytes(data, offset, avatar_length.value)
				if avatar_read.error != ProtobufError.OK:
					return avatar_read.error
				avatar = avatar_read.value
				offset = avatar_read.offset
			5:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var nickname_length: VarintRead = Wire.decode_varint(data, offset)
				if nickname_length.error != ProtobufError.OK:
					return nickname_length.error
				offset = nickname_length.offset
				if nickname_length.value < 0 or offset + nickname_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var nickname_read: StringRead = Wire.decode_string(data, offset, nickname_length.value)
				if nickname_read.error != ProtobufError.OK:
					return nickname_read.error
				nickname = nickname_read.value
				offset = nickname_read.offset
			6:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var tags_length: VarintRead = Wire.decode_varint(data, offset)
				if tags_length.error != ProtobufError.OK:
					return tags_length.error
				offset = tags_length.offset
				if tags_length.value < 0 or offset + tags_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var tags_read: StringRead = Wire.decode_string(data, offset, tags_length.value)
				if tags_read.error != ProtobufError.OK:
					return tags_read.error
				tags.append(tags_read.value)
				offset = tags_read.offset
			7:
				if wire_type == Wire.WIRE_LENGTH_DELIMITED:
					var scores_length: VarintRead = Wire.decode_varint(data, offset)
					if scores_length.error != ProtobufError.OK:
						return scores_length.error
					offset = scores_length.offset
					if scores_length.value < 0 or offset + scores_length.value > data.size():
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					var scores_end: int = offset + scores_length.value
					while offset < scores_end:
						var scores_packed: VarintRead = Wire.decode_varint(data, offset)
						if scores_packed.error != ProtobufError.OK:
							return scores_packed.error
						scores.append(scores_packed.value)
						offset = scores_packed.offset
				elif wire_type == Wire.WIRE_VARINT:
					var scores_read: VarintRead = Wire.decode_varint(data, offset)
					if scores_read.error != ProtobufError.OK:
						return scores_read.error
					scores.append(scores_read.value)
					offset = scores_read.offset
				else:
					return ProtobufError.WIRE_TYPE_MISMATCH
			8:
				if wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var status_read: VarintRead = Wire.decode_varint(data, offset)
				if status_read.error != ProtobufError.OK:
					return status_read.error
				status = PlayerStatus.from_wire(status_read.value)
				offset = status_read.offset
			9:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var primary_length: VarintRead = Wire.decode_varint(data, offset)
				if primary_length.error != ProtobufError.OK:
					return primary_length.error
				offset = primary_length.offset
				if primary_length.value < 0 or offset + primary_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var primary_message: Slot = Slot.new()
				var primary_error: ProtobufError = primary_message.merge_from_bytes(data.slice(offset, offset + primary_length.value))
				if primary_error != ProtobufError.OK:
					return primary_error
				primary = primary_message
				offset += primary_length.value
			10:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var slots_length: VarintRead = Wire.decode_varint(data, offset)
				if slots_length.error != ProtobufError.OK:
					return slots_length.error
				offset = slots_length.offset
				if slots_length.value < 0 or offset + slots_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var slots_message: Slot = Slot.new()
				var slots_error: ProtobufError = slots_message.merge_from_bytes(data.slice(offset, offset + slots_length.value))
				if slots_error != ProtobufError.OK:
					return slots_error
				slots.append(slots_message)
				offset += slots_length.value
			11:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var counts_length: VarintRead = Wire.decode_varint(data, offset)
				if counts_length.error != ProtobufError.OK:
					return counts_length.error
				offset = counts_length.offset
				if counts_length.value < 0 or offset + counts_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var counts_end: int = offset + counts_length.value
				var counts_key: String = ""
				var counts_value: int = 0
				while offset < counts_end:
					var counts_entry_tag: VarintRead = Wire.decode_varint(data, offset)
					if counts_entry_tag.error != ProtobufError.OK:
						return counts_entry_tag.error
					offset = counts_entry_tag.offset
					var counts_entry_wire_type: int = Wire.get_wire_type(counts_entry_tag.value)
					match Wire.get_field_number(counts_entry_tag.value):
						1:
							if counts_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var counts_key_length: VarintRead = Wire.decode_varint(data, offset)
							if counts_key_length.error != ProtobufError.OK:
								return counts_key_length.error
							offset = counts_key_length.offset
							if counts_key_length.value < 0 or offset + counts_key_length.value > data.size():
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							var counts_key_read: StringRead = Wire.decode_string(data, offset, counts_key_length.value)
							if counts_key_read.error != ProtobufError.OK:
								return counts_key_read.error
							counts_key = counts_key_read.value
							offset = counts_key_read.offset
						2:
							if counts_entry_wire_type != Wire.WIRE_VARINT:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var counts_value_read: VarintRead = Wire.decode_varint(data, offset)
							if counts_value_read.error != ProtobufError.OK:
								return counts_value_read.error
							counts_value = counts_value_read.value
							offset = counts_value_read.offset
						_:
							var counts_skip: SkipRead = Wire.skip_field(data, offset, counts_entry_wire_type)
							if counts_skip.error != ProtobufError.OK:
								return counts_skip.error
							offset = counts_skip.offset
				counts[counts_key] = counts_value
			12:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var text_length: VarintRead = Wire.decode_varint(data, offset)
				if text_length.error != ProtobufError.OK:
					return text_length.error
				offset = text_length.offset
				if text_length.value < 0 or offset + text_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var text_read: StringRead = Wire.decode_string(data, offset, text_length.value)
				if text_read.error != ProtobufError.OK:
					return text_read.error
				payload = PlayerPayloadCase.Text(text_read.value)
				offset = text_read.offset
			13:
				if wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var amount_read: VarintRead = Wire.decode_varint(data, offset)
				if amount_read.error != ProtobufError.OK:
					return amount_read.error
				payload = PlayerPayloadCase.Amount(amount_read.value)
				offset = amount_read.offset
			14:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var badge_length: VarintRead = Wire.decode_varint(data, offset)
				if badge_length.error != ProtobufError.OK:
					return badge_length.error
				offset = badge_length.offset
				if badge_length.value < 0 or offset + badge_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var badge_message: Badge = Badge.new()
				var badge_error: ProtobufError = badge_message.merge_from_bytes(data.slice(offset, offset + badge_length.value))
				if badge_error != ProtobufError.OK:
					return badge_error
				badge = badge_message
				offset += badge_length.value
			15:
				if wire_type != Wire.WIRE_VARINT:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var tier_read: VarintRead = Wire.decode_varint(data, offset)
				if tier_read.error != ProtobufError.OK:
					return tier_read.error
				tier = Tier.from_wire(tier_read.value)
				offset = tier_read.offset
			16:
				if wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var loadout_length: VarintRead = Wire.decode_varint(data, offset)
				if loadout_length.error != ProtobufError.OK:
					return loadout_length.error
				offset = loadout_length.offset
				if loadout_length.value < 0 or offset + loadout_length.value > data.size():
					return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
				var loadout_end: int = offset + loadout_length.value
				var loadout_key: String = ""
				var loadout_value: Slot = Slot.new()
				while offset < loadout_end:
					var loadout_entry_tag: VarintRead = Wire.decode_varint(data, offset)
					if loadout_entry_tag.error != ProtobufError.OK:
						return loadout_entry_tag.error
					offset = loadout_entry_tag.offset
					var loadout_entry_wire_type: int = Wire.get_wire_type(loadout_entry_tag.value)
					match Wire.get_field_number(loadout_entry_tag.value):
						1:
							if loadout_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var loadout_key_length: VarintRead = Wire.decode_varint(data, offset)
							if loadout_key_length.error != ProtobufError.OK:
								return loadout_key_length.error
							offset = loadout_key_length.offset
							if loadout_key_length.value < 0 or offset + loadout_key_length.value > data.size():
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							var loadout_key_read: StringRead = Wire.decode_string(data, offset, loadout_key_length.value)
							if loadout_key_read.error != ProtobufError.OK:
								return loadout_key_read.error
							loadout_key = loadout_key_read.value
							offset = loadout_key_read.offset
						2:
							if loadout_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var loadout_value_length: VarintRead = Wire.decode_varint(data, offset)
							if loadout_value_length.error != ProtobufError.OK:
								return loadout_value_length.error
							offset = loadout_value_length.offset
							if loadout_value_length.value < 0 or offset + loadout_value_length.value > data.size():
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							var loadout_value_message: Slot = Slot.new()
							var loadout_value_error: ProtobufError = loadout_value_message.merge_from_bytes(data.slice(offset, offset + loadout_value_length.value))
							if loadout_value_error != ProtobufError.OK:
								return loadout_value_error
							loadout_value = loadout_value_message
							offset += loadout_value_length.value
						_:
							var loadout_skip: SkipRead = Wire.skip_field(data, offset, loadout_entry_wire_type)
							if loadout_skip.error != ProtobufError.OK:
								return loadout_skip.error
							offset = loadout_skip.offset
				loadout[loadout_key] = loadout_value
			_:
				var skipped: SkipRead = Wire.skip_field(data, offset, wire_type)
				if skipped.error != ProtobufError.OK:
					return skipped.error
				offset = skipped.offset
	return ProtobufError.OK
