import cafecito.game.v1
import foundry.proto

extends SceneTree

var failures: int = 0

func check(condition: bool, label: String) -> void:
	if not condition:
		printerr("FAIL: ", label)
		failures += 1

func _init() -> void:
	var player: Player = Player.new()
	player.name = "Ava"
	player.level = 7
	player.active = true
	player.avatar = PackedByteArray([1, 2, 3])
	player.nickname = ""
	player.tags = ["alpha", "beta"]
	player.scores = [10, 20, 30]
	player.status = PlayerStatus.PLAYER_STATUS_ONLINE
	player.counts = {"wins": 3}
	player.tier = Player.Tier.TIER_GOLD
	player.payload = PlayerPayloadCase.Amount(42)

	var primary: Slot = Slot.new()
	primary.label = "sword"
	primary.quantity = 1
	player.primary = primary

	var extra: Slot = Slot.new()
	extra.label = "shield"
	player.slots = [extra]

	var boots: Slot = Slot.new()
	boots.label = "boots"
	boots.quantity = 2
	player.loadout = {"feet": boots}

	var badge: Player.Badge = Player.Badge.new()
	badge.code = "veteran"
	player.badge = badge

	var (decoded, error) = Player.from_bytes(player.to_bytes())
	check(error == ProtobufError.OK, "decode returned an error")
	if not (decoded is Player):
		printerr("FAIL: decoded message was null")
		quit(1)
		return

	check(decoded.name == "Ava", "name")
	check(decoded.level == 7, "level")
	check(decoded.active, "active")
	check(decoded.avatar == PackedByteArray([1, 2, 3]), "avatar")
	## Explicit presence: an empty string must survive as set, not collapse to null.
	check(decoded.nickname == "", "nickname presence")
	check(decoded.tags == ["alpha", "beta"], "repeated string")
	check(decoded.scores == [10, 20, 30], "packed repeated int")
	check(decoded.status == PlayerStatus.PLAYER_STATUS_ONLINE, "enum")
	check(decoded.counts == {"wins": 3}, "map")
	check(decoded.tier == Player.Tier.TIER_GOLD, "nested enum")
	check(decoded.primary is Slot and decoded.primary.label == "sword", "message field")
	check(decoded.slots.size() == 1 and decoded.slots[0].label == "shield", "repeated message")
	check(decoded.badge is Player.Badge and decoded.badge.code == "veteran", "nested message")
	check(decoded.loadout.size() == 1, "message-valued map size")
	check(decoded.loadout.has("feet") and decoded.loadout["feet"].quantity == 2, "message-valued map entry")

	match decoded.payload:
		PlayerPayloadCase.Amount(var amount):
			check(amount == 42, "oneof amount")
		_:
			printerr("FAIL: oneof case did not round trip")
			failures += 1

	## An absent optional must stay absent rather than decoding as "".
	var bare: Player = Player.new()
	var (bare_decoded, bare_error) = Player.from_bytes(bare.to_bytes())
	check(bare_error == ProtobufError.OK, "bare decode returned an error")
	if bare_decoded is Player:
		check(not (bare_decoded.nickname is String), "absent optional stays null")
		check(not (bare_decoded.payload is PlayerPayloadCase), "unset oneof stays null")

	## A map entry may legally omit its value; the default applies rather than null.
	var entry: PackedByteArray = PackedByteArray()
	var entry_key: PackedByteArray = Wire.encode_string("bare")
	entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
	entry.append_array(Wire.encode_varint(entry_key.size()))
	entry.append_array(entry_key)
	var valueless: PackedByteArray = PackedByteArray()
	valueless.append_array(Wire.encode_varint(Wire.make_tag(16, Wire.WIRE_LENGTH_DELIMITED)))
	valueless.append_array(Wire.encode_varint(entry.size()))
	valueless.append_array(entry)
	var sparse: Player = Player.new()
	check(sparse.merge_from_bytes(valueless) == ProtobufError.OK, "valueless map entry decodes")
	check(sparse.loadout.has("bare") and sparse.loadout["bare"] is Slot, "valueless map entry defaults")

	## Unknown fields must be skipped, not rejected.
	var unknown: PackedByteArray = PackedByteArray()
	unknown.append_array(Wire.encode_varint(Wire.make_tag(900, Wire.WIRE_VARINT)))
	unknown.append_array(Wire.encode_varint(1))
	check(Player.new().merge_from_bytes(unknown) == ProtobufError.OK, "unknown field skipped")

	if failures > 0:
		printerr("round trip failed with ", failures, " error(s)")
		quit(1)
		return
	print("round trip ok")
	quit(0)
