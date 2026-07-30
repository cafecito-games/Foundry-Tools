import cafecito.game.v1
import cafecito.inventory.v1
import foundry.proto
import foundry.proto.wkt
import probe.collisions.v1
import probe.dependency.v1
import probe.packing.v1
import probe.scalars.v1
import probe.wellknown.common.v1
import probe.wellknown.v1

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
	collision.Node_ = "native node"
	collision.String_ = "built-in string"
	collision.Timer_ = ["first", "second"]
	collision.Resource_ = {"ore": 7}
	collision.Object_ = GameNodeObjectCase.Image("image case")

	var (collision_decoded, collision_error) = GameNode.from_bytes(collision.to_bytes())
	check(collision_error == ProtobufError.OK, "prefixed collision fixture decodes")
	check(collision_decoded is GameNode, "prefixed collision fixture has the renamed type")
	if collision_decoded is GameNode:
		check(collision_decoded.nested is GameNode.GameTimer, "prefixed nested type")
		check(collision_decoded.imported is DependencyTimer, "dependency prefix")
		check(collision_decoded.state == GameString.STRING_READY, "prefixed built-in collision")
		check(collision_decoded.Node_ == "native node", "native class member collision")
		check(collision_decoded.String_ == "built-in string", "built-in type member collision")
		check(collision_decoded.Timer_ == ["first", "second"], "repeated native class member collision")
		check(collision_decoded.Resource_ == {"ore": 7}, "map native class member collision")
		match collision_decoded.payload:
			GameNodePayloadCase.Text(var text):
				check(text == "safe", "prefixed oneof")
			_:
				printerr("FAIL: prefixed oneof case did not round trip")
				failures += 1
		match collision_decoded.Object_:
			GameNodeObjectCase.Image(var image):
				check(image == "image case", "oneof native class member collision")
			_:
				printerr("FAIL: escaped oneof case did not round trip")
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

	## A oneof payload declared inside the message that declares the oneof: the
	## hoisted union names it as Player.Badge, through the class it lives in.
	var chosen: Player.Badge = Player.Badge.new()
	chosen.code = "champion"
	var nested_payload: Player = Player.new()
	nested_payload.payload = PlayerPayloadCase.ChosenBadge(chosen)
	var (nested_decoded, nested_error) = Player.from_bytes(nested_payload.to_bytes())
	check(nested_error == ProtobufError.OK, "nested oneof payload decodes")
	if nested_decoded is Player:
		match nested_decoded.payload:
			PlayerPayloadCase.ChosenBadge(var badge_case):
				check(badge_case.code == "champion", "oneof carries a message nested in its own message")
			_:
				printerr("FAIL: nested message oneof case did not round trip")
				failures += 1

	## The same for an enum nested in the declaring message.
	var tier_payload: Player = Player.new()
	tier_payload.payload = PlayerPayloadCase.ChosenTier(Player.Tier.TIER_GOLD)
	var (tier_decoded, tier_error) = Player.from_bytes(tier_payload.to_bytes())
	check(tier_error == ProtobufError.OK, "nested enum oneof payload decodes")
	if tier_decoded is Player:
		match tier_decoded.payload:
			PlayerPayloadCase.ChosenTier(var tier_case):
				check(tier_case == Player.Tier.TIER_GOLD, "oneof carries an enum nested in its own message")
			_:
				printerr("FAIL: nested enum oneof case did not round trip")
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

	check_scalars()
	check_packing()
	check_well_known()
	check_well_known_name_collision()

	if failures > 0:
		printerr("round trip failed with ", failures, " error(s)")
		quit(1)
		return
	print("round trip ok")
	quit(0)

## The eight scalars that are not plain varints, checked against bytes a
## reference protobuf implementation produced for the same values. Lint proves
## the bindings typecheck; only this proves the framing is right, and comparing
## against another implementation is what keeps it from proving our encoder
## agrees with our decoder and nothing more.
func check_scalars() -> void:
	var reference: PackedByteArray = read_reference_bytes()
	if reference.is_empty():
		return

	var suite: ScalarSuite = populated_scalar_suite()
	check(suite.to_bytes() == reference, "scalars encode to the reference bytes")

	var (decoded, decode_error) = ScalarSuite.from_bytes(reference)
	check(decode_error == ProtobufError.OK, "reference bytes decode")
	if not (decoded is ScalarSuite):
		return

	## Re-encoding what we decoded is the strongest statement available here: it
	## covers every bit pattern in the vector at once, sign of zero included,
	## without depending on how any one value compares.
	check(decoded.to_bytes() == reference, "decoding and re-encoding is lossless")

	check(decoded.double_value == -2.5, "double round trips")
	## 0.1 is not representable in binary32, so a proto float narrows on the way
	## out. Narrowing an already-narrowed value must be a no-op, and the result
	## must differ from the binary64 original.
	check(Wire.encode_float(decoded.float_value) == Wire.encode_float(0.1), "float narrowing is idempotent")
	check(decoded.float_value != 0.1, "float does not keep binary64 precision")
	check(decoded.fixed32_value == 4294967295, "fixed32 spans the unsigned range")
	## Foundry's int is signed, so the top half of the fixed64 range arrives as
	## a negative with the same bits rather than as an unrepresentable positive.
	check(decoded.fixed64_value == -1, "fixed64 keeps every bit of the unsigned range")
	check(decoded.sfixed32_value == -2147483648, "sfixed32 holds its minimum")
	check(decoded.sfixed64_value == min_int64(), "sfixed64 holds its minimum")
	check(decoded.sint32_value == -2147483648, "sint32 holds its minimum")
	check(decoded.sint64_value == min_int64(), "sint64 holds its minimum")

	check(decoded.sint32_list == [0, -1, 1, -2147483648, 2147483647], "packed sint32 round trips")
	check(decoded.sfixed32_list == [-1, 0, 2147483647], "packed sfixed32 round trips")
	check(decoded.double_list.size() == 4, "packed double keeps every element")
	if decoded.double_list.size() == 4:
		check(decoded.double_list[1] == 1.5, "packed double round trips")
		check(decoded.double_list[2] == INF, "positive infinity round trips")
		check(decoded.double_list[3] == -INF, "negative infinity round trips")

	match decoded.choice:
		ScalarSuiteChoiceCase.ChoiceDelta(var delta):
			check(delta == -4096, "sint64 oneof member round trips")
		_:
			printerr("FAIL: scalar oneof case did not round trip")
			failures += 1

	check_scalar_edges()

## Cases the reference vector deliberately leaves out: NaN, whose bit pattern is
## not unique enough to compare across implementations, and the map, whose entry
## order is not fixed.
func check_scalar_edges() -> void:
	var suite: ScalarSuite = ScalarSuite.new()
	suite.double_value = NAN
	suite.float_value = NAN
	suite.ratios = {min_int64(): 0.5, 7: -1.5}

	var (decoded, decode_error) = ScalarSuite.from_bytes(suite.to_bytes())
	check(decode_error == ProtobufError.OK, "NaN and map bytes decode")
	if not (decoded is ScalarSuite):
		return
	check(is_nan(decoded.double_value), "NaN survives a double round trip")
	check(is_nan(decoded.float_value), "NaN survives a float round trip")
	check(decoded.ratios == {min_int64(): 0.5, 7: -1.5}, "sfixed64 keys and float values round trip")

	check_truncated_packed_run()

## A packed run declares a byte length. If that length is not a whole number of
## elements the message is malformed, and reading the last element would take
## bytes from whatever follows rather than failing.
func check_truncated_packed_run() -> void:
	## Field 11 is repeated sfixed32, so four bytes per element. Six bytes
	## announces one element and half of another.
	var truncated: PackedByteArray = PackedByteArray()
	truncated.append_array(Wire.encode_varint(Wire.make_tag(11, Wire.WIRE_LENGTH_DELIMITED)))
	truncated.append_array(Wire.encode_varint(6))
	truncated.append_array(PackedByteArray([1, 0, 0, 0, 2, 0]))
	## A trailing field the overrun would otherwise eat into.
	truncated.append_array(Wire.encode_varint(Wire.make_tag(7, Wire.WIRE_VARINT)))
	truncated.append_array(Wire.encode_sint32(9))

	var (decoded, decode_error) = ScalarSuite.from_bytes(truncated)
	check(decode_error == ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH, "a packed run that overruns its length is rejected")
	check(not (decoded is ScalarSuite), "a rejected message yields no value")

	check_truncated_map_entry()
	check_negative_zero_presence()

## A map entry is length-delimited too, and a value that reads past the entry
## would take bytes from the field after it.
func check_truncated_map_entry() -> void:
	## Field 12 is map<sfixed64, float>. The entry announces six bytes but holds
	## a key tag plus only two of the key's eight.
	var entry: PackedByteArray = PackedByteArray()
	entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_64BIT)))
	entry.append_array(PackedByteArray([1, 0]))

	var truncated: PackedByteArray = PackedByteArray()
	truncated.append_array(Wire.encode_varint(Wire.make_tag(12, Wire.WIRE_LENGTH_DELIMITED)))
	truncated.append_array(Wire.encode_varint(entry.size()))
	truncated.append_array(entry)
	truncated.append_array(Wire.encode_varint(Wire.make_tag(7, Wire.WIRE_VARINT)))
	truncated.append_array(Wire.encode_sint32(9))

	var (decoded, decode_error) = ScalarSuite.from_bytes(truncated)
	check(decode_error == ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH, "a map entry that overruns its length is rejected")
	check(not (decoded is ScalarSuite), "a rejected map entry yields no value")

## proto3 omits a float field holding the default, and the default is +0.0.
## -0.0 is a distinct value protobuf writes, so it has to survive a decode and
## re-encode rather than being folded onto the default and dropped.
func check_negative_zero_presence() -> void:
	## Built from bytes: a -0.0 written as a literal cannot be relied on here.
	## See cafecito-games/Foundry#1371.
	var negative_zero: PackedByteArray = PackedByteArray()
	negative_zero.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_64BIT)))
	negative_zero.append_array(PackedByteArray([0, 0, 0, 0, 0, 0, 0, 128]))

	var (decoded, decode_error) = ScalarSuite.from_bytes(negative_zero)
	check(decode_error == ProtobufError.OK, "negative zero decodes")
	if decoded is ScalarSuite:
		check(decoded.double_value == 0.0, "negative zero compares equal to zero")
		check(decoded.to_bytes() == negative_zero, "negative zero is written rather than treated as the default")

	## Positive zero is the default and stays off the wire.
	var positive: ScalarSuite = ScalarSuite.new()
	positive.double_value = 0.0
	check(positive.to_bytes().is_empty(), "positive zero is omitted as the default")

	## A proto float is binary32. A double too small to survive that narrowing
	## is the default once written, so it is omitted the way protobuf omits it,
	## rather than written as four bytes of zero.
	var underflowing: ScalarSuite = ScalarSuite.new()
	underflowing.float_value = 1e-60
	check(underflowing.to_bytes().is_empty(), "a float that underflows binary32 is omitted")
	## The same magnitude in a double survives, so it is still written.
	var representable: ScalarSuite = ScalarSuite.new()
	representable.double_value = 1e-60
	check(not representable.to_bytes().is_empty(), "a double of the same magnitude is written")

func populated_scalar_suite() -> ScalarSuite:
	var suite: ScalarSuite = ScalarSuite.new()
	suite.double_value = -2.5
	suite.float_value = 0.1
	suite.fixed32_value = 4294967295
	suite.fixed64_value = -1
	suite.sfixed32_value = -2147483648
	suite.sfixed64_value = min_int64()
	suite.sint32_value = -2147483648
	suite.sint64_value = min_int64()
	suite.sint32_list = [0, -1, 1, -2147483648, 2147483647]
	## No -0.0 here: this engine cannot hold one written as a literal, and a
	## -0.0 anywhere in the script would take the sign of every 0.0 with it.
	## See cafecito-games/Foundry#1371.
	suite.double_list = [0.0, 1.5, INF, -INF]
	suite.sfixed32_list = [-1, 0, 2147483647]
	suite.choice = ScalarSuiteChoiceCase.ChoiceDelta(-4096)
	return suite

## Written as a subtraction because the literal for it does not fit the positive
## range the parser folds a unary minus over.
func min_int64() -> int:
	return -9223372036854775807 - 1

func read_reference_bytes() -> PackedByteArray:
	var file: FileAccess? = FileAccess.open("res://scalars_reference.bin", FileAccess.READ)
	if not (file is FileAccess):
		printerr("FAIL: could not open res://scalars_reference.bin")
		failures += 1
		return PackedByteArray()
	var data: PackedByteArray = file.get_buffer(file.get_length())
	file.close()
	return data

## `[packed = false]` binds the encoder: the field goes out as one tagged
## record per element even though the same value type packs by default. What it
## must not change is the decoder, which protobuf requires to take either form
## for any packable repeated field.
func check_packing() -> void:
	var values: Array[int] = [1, 300]
	var suite: PackingSuite = PackingSuite.new()
	suite.packed_int32 = values
	suite.unpacked_int32 = values
	suite.default_int32 = values

	var expected: PackedByteArray = PackedByteArray()
	expected.append_array(packed_run(1, values))
	expected.append_array(unpacked_records(2, values))
	expected.append_array(packed_run(3, values))
	check(suite.to_bytes() == expected, "the packed option decides how each field is written")

	var (decoded, decode_error) = PackingSuite.from_bytes(suite.to_bytes())
	check(decode_error == ProtobufError.OK, "packing fixture decodes")
	if not (decoded is PackingSuite):
		return
	check(decoded.packed_int32 == values, "packed field round trips")
	check(decoded.unpacked_int32 == values, "unpacked field round trips")
	check(decoded.default_int32 == values, "default field round trips")

	## Each field fed the encoding the other one writes.
	var swapped: PackedByteArray = PackedByteArray()
	swapped.append_array(unpacked_records(1, values))
	swapped.append_array(packed_run(2, values))
	var (swapped_decoded, swapped_error) = PackingSuite.from_bytes(swapped)
	check(swapped_error == ProtobufError.OK, "the opposite encoding decodes")
	if not (swapped_decoded is PackingSuite):
		return
	check(swapped_decoded.packed_int32 == values, "a packed field accepts one record per element")
	check(swapped_decoded.unpacked_int32 == values, "an unpacked field accepts a packed run")

func packed_run(number: int, values: Array[int]) -> PackedByteArray:
	var run: PackedByteArray = PackedByteArray()
	for value: int in values:
		run.append_array(Wire.encode_varint(value))
	var record: PackedByteArray = PackedByteArray()
	record.append_array(Wire.encode_varint(Wire.make_tag(number, Wire.WIRE_LENGTH_DELIMITED)))
	record.append_array(Wire.encode_varint(run.size()))
	record.append_array(run)
	return record

func unpacked_records(number: int, values: Array[int]) -> PackedByteArray:
	var records: PackedByteArray = PackedByteArray()
	for value: int in values:
		records.append_array(Wire.encode_varint(Wire.make_tag(number, Wire.WIRE_VARINT)))
		records.append_array(Wire.encode_varint(value))
	return records

## A schema that references the well-known types, which the runtime ships rather
## than the project generating them. probe.wellknown.v1 declares a Timestamp of
## its own, so both spellings have to be qualified here -- this file imports the
## two namespaces that export the name.
func check_well_known() -> void:
	var local: probe.wellknown.v1.Timestamp = probe.wellknown.v1.Timestamp.new()
	local.label = "local"

	var occurred_at: foundry.proto.wkt.Timestamp = foundry.proto.wkt.Timestamp.new()
	occurred_at.seconds = 1700000000
	occurred_at.nanos = 500

	var name: Value = Value.new()
	name.kind = ValueKindCase.StringValue("player")
	var attributes: Struct = Struct.new()
	attributes.fields["name"] = name

	var attachment: Any = Any.new()
	attachment.type_url = "type.googleapis.com/probe.wellknown.v1.Timestamp"
	attachment.value = local.to_bytes()

	var reading: Reading = Reading.new()
	reading.local = local
	reading.occurred_at = occurred_at
	reading.attributes = attributes
	reading.attachments.append(attachment)
	reading.checkpoints["start"] = occurred_at
	reading.detail = ReadingDetailCase.Payload(attachment)

	var (decoded, decode_error) = Reading.from_bytes(reading.to_bytes())
	check(decode_error == ProtobufError.OK, "well-known fixture decodes")
	if not (decoded is Reading):
		return
	check(decoded.local is probe.wellknown.v1.Timestamp and decoded.local.label == "local",
		"a schema type keeps a well-known name")
	check(decoded.occurred_at is foundry.proto.wkt.Timestamp and decoded.occurred_at.seconds == 1700000000,
		"a well-known message field round trips")
	check(decoded.occurred_at is foundry.proto.wkt.Timestamp and decoded.occurred_at.nanos == 500,
		"a well-known message field round trips in full")
	check(decoded.checkpoints.has("start") and decoded.checkpoints["start"].seconds == 1700000000,
		"a well-known map value round trips")
	check(decoded.attachments.size() == 1 and decoded.attachments[0].type_url == attachment.type_url,
		"a repeated well-known element round trips")
	if decoded.attributes is Struct:
		var decoded_name: Value? = decoded.attributes.fields.get("name")
		if decoded_name is Value:
			match decoded_name.kind:
				ValueKindCase.StringValue(var text):
					check(text == "player", "a well-known oneof case round trips")
				_:
					printerr("FAIL: Struct value case did not round trip")
					failures += 1
		else:
			printerr("FAIL: Struct field did not round trip")
			failures += 1
	else:
		printerr("FAIL: well-known Struct field did not round trip")
		failures += 1
	match decoded.detail:
		ReadingDetailCase.Payload(var payload):
			check(payload.value == local.to_bytes(), "a well-known oneof member round trips")
		_:
			printerr("FAIL: well-known oneof case did not round trip")
			failures += 1

## A schema importing a well-known type also has every other well-known name in
## scope, so an Empty of another schema's must be qualified to resolve at all.
func check_well_known_name_collision() -> void:
	var local_empty: probe.wellknown.common.v1.Empty = probe.wellknown.common.v1.Empty.new()
	local_empty.x = 7

	var timeout: Duration = Duration.new()
	timeout.seconds = 30

	var holder: Holder = Holder.new()
	holder.local_empty = local_empty
	holder.timeout = timeout

	var (decoded, decode_error) = Holder.from_bytes(holder.to_bytes())
	check(decode_error == ProtobufError.OK, "well-known collision fixture decodes")
	if not (decoded is Holder):
		return
	check(decoded.local_empty is probe.wellknown.common.v1.Empty and decoded.local_empty.x == 7,
		"an imported type named like a well-known one round trips")
	check(decoded.timeout is Duration and decoded.timeout.seconds == 30,
		"the well-known reference that pulled the namespace in round trips")
