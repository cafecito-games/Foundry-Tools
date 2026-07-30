import cafecito.game.v1
import cafecito.inventory.v1
import foundry.proto
import probe.collisions.v1
import probe.dependency.v1

extends SceneTree

var failures: int = 0

func check(condition: bool, label: String) -> void:
	if not condition:
		printerr("FAIL: ", label)
		failures += 1

func _init() -> void:
	var collision: GameNode = GameNode.new()
	var nested_timer: GameNode.GameTimer = GameNode.GameTimer.new()
	nested_timer.label = "nested"
	collision.nested = nested_timer

	var dependency_timer: DependencyTimer = DependencyTimer.new()
	dependency_timer.label = "imported"
	collision.imported = dependency_timer
	collision.state = GameString.STRING_READY
	collision.payload = GameNodePayloadCase.Text("safe")

	var (collision_decoded, collision_error) = GameNode.from_bytes(collision.to_bytes())
	check(collision_error == ProtobufError.OK, "prefixed collision fixture decodes")
	check(collision_decoded is GameNode, "prefixed collision fixture has the renamed type")
	if collision_decoded is GameNode:
		check(collision_decoded.nested is GameNode.GameTimer, "prefixed nested type")
		check(collision_decoded.imported is DependencyTimer, "dependency prefix")
		check(collision_decoded.state == GameString.STRING_READY, "prefixed built-in collision")
		match collision_decoded.payload:
			GameNodePayloadCase.Text(var text):
				check(text == "safe", "prefixed oneof")
			_:
				printerr("FAIL: prefixed oneof case did not round trip")
				failures += 1

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

	## A type declared in another proto file, reached through the generated import.
	var held: Item = Item.new()
	held.sku = "torch"
	held.rarity = Rarity.RARITY_LEGENDARY
	player.held = held
	player.rarity = Rarity.RARITY_COMMON

	## Field names that collide with the emitter's own locals or a keyword.
	player.offset = 11
	player.data = "payload"
	player.result = PackedByteArray([9])
	player.var_ = 5

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
	check(decoded.held is Item and decoded.held.sku == "torch", "imported message field")
	check(decoded.held is Item and decoded.held.rarity == Rarity.RARITY_LEGENDARY, "imported nested enum")
	check(decoded.rarity == Rarity.RARITY_COMMON, "imported enum field")
	check(decoded.offset == 11, "field named offset")
	check(decoded.data == "payload", "field named data")
	check(decoded.result == PackedByteArray([9]), "field named result")
	check(decoded.var_ == 5, "field named after a keyword")

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

	## An unknown field must be accepted and must survive a re-encode, or a peer
	## on a newer schema loses data passing through this binding.
	var unknown: PackedByteArray = PackedByteArray()
	unknown.append_array(Wire.encode_varint(Wire.make_tag(900, Wire.WIRE_VARINT)))
	unknown.append_array(Wire.encode_varint(7))
	var carrier: Player = Player.new()
	check(carrier.merge_from_bytes(unknown) == ProtobufError.OK, "unknown field accepted")
	check(carrier.to_bytes() == unknown, "unknown field survives a re-encode")

	## An enum value this schema has no case for is preserved the same way,
	## rather than collapsing onto the zero case and then being dropped.
	var future: PackedByteArray = PackedByteArray()
	future.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
	future.append_array(Wire.encode_varint(99))
	var forward: Player = Player.new()
	check(forward.merge_from_bytes(future) == ProtobufError.OK, "unknown enum value accepted")
	check(forward.status == PlayerStatus.PLAYER_STATUS_UNSPECIFIED, "unknown enum leaves the field default")
	check(forward.to_bytes() == future, "unknown enum value survives a re-encode")

	## A retained value stands in for the field, in the field's own position, so
	## what a reader takes as the last record for that number is what the sender
	## wrote. Assigning the field supersedes the retained value outright rather
	## than being written alongside it, which for a singular field would leave
	## whichever record happened to come last deciding the value.
	var known: PackedByteArray = PackedByteArray()
	known.append_array(Wire.encode_varint(Wire.make_tag(8, Wire.WIRE_VARINT)))
	known.append_array(Wire.encode_varint(1))

	forward.status = PlayerStatus.PLAYER_STATUS_ONLINE
	check(forward.to_bytes() == known, "assigning the field supersedes a retained unknown value")

	## Recognized then unrecognized: the unrecognized value came last on the
	## wire, so it is what survives.
	var known_then_unknown: Player = Player.new()
	var pair: PackedByteArray = PackedByteArray()
	pair.append_array(known)
	pair.append_array(future)
	check(known_then_unknown.merge_from_bytes(pair) == ProtobufError.OK, "known then unknown decodes")
	check(known_then_unknown.to_bytes() == future, "the last record wins when it is unrecognized")

	## Unrecognized then recognized: the recognized value came last, so the
	## retained copy has to be dropped rather than re-emitted after it.
	var unknown_then_known: Player = Player.new()
	var reversed: PackedByteArray = PackedByteArray()
	reversed.append_array(future)
	reversed.append_array(known)
	check(unknown_then_known.merge_from_bytes(reversed) == ProtobufError.OK, "unknown then known decodes")
	check(unknown_then_known.status == PlayerStatus.PLAYER_STATUS_ONLINE, "the last record wins when it is recognized")
	check(unknown_then_known.to_bytes() == known, "a recognized value drops the retained copy")

	## A singular message field split across two records must merge, not replace.
	var first: PackedByteArray = PackedByteArray()
	var first_slot: Slot = Slot.new()
	first_slot.label = "axe"
	var first_bytes: PackedByteArray = first_slot.to_bytes()
	first.append_array(Wire.encode_varint(Wire.make_tag(9, Wire.WIRE_LENGTH_DELIMITED)))
	first.append_array(Wire.encode_varint(first_bytes.size()))
	first.append_array(first_bytes)
	var second_slot: Slot = Slot.new()
	second_slot.quantity = 4
	var second_bytes: PackedByteArray = second_slot.to_bytes()
	first.append_array(Wire.encode_varint(Wire.make_tag(9, Wire.WIRE_LENGTH_DELIMITED)))
	first.append_array(Wire.encode_varint(second_bytes.size()))
	first.append_array(second_bytes)
	var merged: Player = Player.new()
	check(merged.merge_from_bytes(first) == ProtobufError.OK, "split message field decodes")
	check(merged.primary is Slot and merged.primary.label == "axe", "split message field keeps the first record")
	check(merged.primary is Slot and merged.primary.quantity == 4, "split message field merges the second record")

	if failures > 0:
		printerr("round trip failed with ", failures, " error(s)")
		quit(1)
		return
	print("round trip ok")
	quit(0)
