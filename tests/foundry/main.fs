import cafecito.game.v1
import cafecito.inventory.v1
import cafecito.json.v1
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
	check_json_base64()
	check_json_timestamp()
	check_json_duration()
	check_json_uint64()
	check_engine_json_types()
	check_json_round_trip()

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

func max_int64() -> int:
	return 9223372036854775807

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

## The base64 helper behind the canonical JSON mapping of a bytes field: standard
## alphabet on the way out, both alphabets and optional padding on the way in.
func check_json_base64() -> void:
	var base64_source: PackedByteArray = PackedByteArray([0, 1, 250, 255, 16, 32])
	var base64_text: String = JsonBase64.encode(base64_source)
	var (base64_decoded, base64_error) = JsonBase64.decode(base64_text)
	check(base64_error == ProtobufError.OK, "base64 round trip decodes")
	check(base64_decoded == base64_source, "base64 round trip preserves bytes")

	var (base64_padded, base64_padded_error) = JsonBase64.decode("AAH6")
	check(base64_padded_error == ProtobufError.OK, "base64 accepts a full quantum")
	check(base64_padded == PackedByteArray([0, 1, 250]), "base64 decodes a full quantum")

	var (base64_unpadded, base64_unpadded_error) = JsonBase64.decode("AAE")
	check(base64_unpadded_error == ProtobufError.OK, "base64 accepts missing padding")
	check(base64_unpadded == PackedByteArray([0, 1]), "base64 decodes missing padding")

	var (_base64_url_safe, base64_url_safe_error) = JsonBase64.decode("-_8")
	check(base64_url_safe_error == ProtobufError.OK, "base64 accepts the URL-safe alphabet")

	var (_base64_bad, base64_bad_error) = JsonBase64.decode("not base64!")
	check(base64_bad_error == ProtobufError.JSON_TYPE_MISMATCH, "base64 rejects a stray character")

	var (_base64_over_padded, base64_over_padded_error) = JsonBase64.decode("A===")
	check(base64_over_padded_error == ProtobufError.JSON_TYPE_MISMATCH, "base64 rejects a quantum with too much padding")

## The RFC 3339 helper behind the canonical JSON mapping of a Timestamp:
## canonical UTC on the way out, offsets and any fraction width on the way in.
func check_json_timestamp() -> void:
	var (epoch_text, epoch_error) = JsonTimestamp.format(0, 0)
	check(epoch_error == ProtobufError.OK, "epoch formats")
	check(epoch_text == "1970-01-01T00:00:00Z", "epoch formats canonically")

	var (fraction_text, fraction_error) = JsonTimestamp.format(1136214245, 10000000)
	check(fraction_error == ProtobufError.OK, "fractional timestamp formats")
	check(fraction_text == "2006-01-02T15:04:05.010Z", "fraction uses three digits")

	var (micro_text, _micro_error) = JsonTimestamp.format(0, 1000)
	check(micro_text == "1970-01-01T00:00:00.000001Z", "a microsecond uses six digits")

	var (nano_text, _nano_error) = JsonTimestamp.format(0, 1)
	check(nano_text == "1970-01-01T00:00:00.000000001Z", "a single nanosecond uses nine digits")

	var (pre_epoch_text, _pre_epoch_error) = JsonTimestamp.format(-62135596800, 0)
	check(pre_epoch_text == "0001-01-01T00:00:00Z", "the lower bound formats")

	var (upper_text, _upper_error) = JsonTimestamp.format(253402300799, 0)
	check(upper_text == "9999-12-31T23:59:59Z", "the upper bound formats")

	var (_range_text, range_error) = JsonTimestamp.format(253402300800, 0)
	check(range_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "an out-of-range second is refused")

	var (_low_text, low_error) = JsonTimestamp.format(-62135596801, 0)
	check(low_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a second below the lower bound is refused")

	var (_nanos_text, nanos_error) = JsonTimestamp.format(0, 1000000000)
	check(nanos_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "an out-of-range nanosecond is refused")

	var (_negative_text, negative_error) = JsonTimestamp.format(0, -1)
	check(negative_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a negative nanosecond is refused")

	var (parsed_seconds, parsed_nanos, parse_error) = JsonTimestamp.parse("2006-01-02T15:04:05.010Z")
	check(parse_error == ProtobufError.OK, "an RFC 3339 string parses")
	check(parsed_seconds == 1136214245, "parsed seconds")
	check(parsed_nanos == 10000000, "parsed nanos")

	var (_one_digit_seconds, one_digit_nanos, one_digit_error) = JsonTimestamp.parse("1970-01-01T00:00:00.5Z")
	check(one_digit_error == ProtobufError.OK, "a one-digit fraction parses")
	check(one_digit_nanos == 500000000, "a one-digit fraction scales to nanos")

	var (_nine_digit_seconds, nine_digit_nanos, nine_digit_error) = JsonTimestamp.parse("1970-01-01T00:00:00.123456789Z")
	check(nine_digit_error == ProtobufError.OK, "a nine-digit fraction parses")
	check(nine_digit_nanos == 123456789, "a nine-digit fraction keeps every digit")

	var (offset_seconds, _offset_nanos, offset_error) = JsonTimestamp.parse("2006-01-02T16:04:05+01:00")
	check(offset_error == ProtobufError.OK, "an offset parses")
	check(offset_seconds == 1136214245, "an offset is folded into UTC")

	var (west_seconds, _west_nanos, west_error) = JsonTimestamp.parse("2006-01-02T14:04:05-01:00")
	check(west_error == ProtobufError.OK, "a western offset parses")
	check(west_seconds == 1136214245, "a western offset is folded into UTC")

	var (lowercase_seconds, _lowercase_nanos, lowercase_error) = JsonTimestamp.parse("1970-01-01t00:00:00z")
	check(lowercase_error == ProtobufError.OK, "lowercase designators parse")
	check(lowercase_seconds == 0, "lowercase designators give the epoch")

	var (_bad_seconds, _bad_nanos, bad_error) = JsonTimestamp.parse("2006-01-02")
	check(bad_error == ProtobufError.JSON_TYPE_MISMATCH, "a date alone is refused")

	var (_no_zone_seconds, _no_zone_nanos, no_zone_error) = JsonTimestamp.parse("2006-01-02T15:04:05")
	check(no_zone_error == ProtobufError.JSON_TYPE_MISMATCH, "a missing zone designator is refused")

	var (_trailing_seconds, _trailing_nanos, trailing_error) = JsonTimestamp.parse("1970-01-01T00:00:00Zjunk")
	check(trailing_error == ProtobufError.JSON_TYPE_MISMATCH, "trailing text after the designator is refused")

	var (_empty_fraction_seconds, _empty_fraction_nanos, empty_fraction_error) = JsonTimestamp.parse("1970-01-01T00:00:00.Z")
	check(empty_fraction_error == ProtobufError.JSON_TYPE_MISMATCH, "an empty fraction is refused")

	var (_wide_fraction_seconds, _wide_fraction_nanos, wide_fraction_error) = JsonTimestamp.parse("1970-01-01T00:00:00.1234567890Z")
	check(wide_fraction_error == ProtobufError.JSON_TYPE_MISMATCH, "a ten-digit fraction is refused")

	var (_alpha_seconds, _alpha_nanos, alpha_error) = JsonTimestamp.parse("20x6-01-02T15:04:05Z")
	check(alpha_error == ProtobufError.JSON_TYPE_MISMATCH, "a non-digit in the date is refused")

	var (_month_seconds, _month_nanos, month_error) = JsonTimestamp.parse("2006-13-02T15:04:05Z")
	check(month_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a thirteenth month is refused")

	var (_day_seconds, _day_nanos, day_error) = JsonTimestamp.parse("2006-02-30T15:04:05Z")
	check(day_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a day past the end of the month is refused")

	## A pre-epoch instant is where a truncating divide would land on the wrong
	## day, so it is round-tripped through both directions.
	var (before_text, before_error) = JsonTimestamp.format(-1, 0)
	check(before_error == ProtobufError.OK, "a pre-epoch instant formats")
	check(before_text == "1969-12-31T23:59:59Z", "a pre-epoch instant formats on the right day")

	var (before_seconds, before_nanos, before_parse_error) = JsonTimestamp.parse(before_text)
	check(before_parse_error == ProtobufError.OK, "a pre-epoch instant parses")
	check(before_seconds == -1 and before_nanos == 0, "a pre-epoch instant round trips")

	var (deep_text, _deep_error) = JsonTimestamp.format(-2208988800, 0)
	check(deep_text == "1900-01-01T00:00:00Z", "a pre-epoch century boundary formats")
	var (deep_seconds, _deep_nanos, _deep_parse_error) = JsonTimestamp.parse(deep_text)
	check(deep_seconds == -2208988800, "a pre-epoch century boundary round trips")

## The duration text helper behind the canonical JSON mapping of a Duration:
## the sign is carried once, even when it can only be read from the nanos,
## because the whole seconds are zero.
func check_json_duration() -> void:
	var (duration_text, duration_error) = JsonDuration.format(3, 1)
	check(duration_error == ProtobufError.OK, "a duration formats")
	check(duration_text == "3.000000001s", "a duration uses nine digits when it must")

	var (micro_text, _micro_error) = JsonDuration.format(3, 10000)
	check(micro_text == "3.000010s", "a duration uses six digits when it must")

	var (whole_text, _whole_error) = JsonDuration.format(3, 0)
	check(whole_text == "3s", "a whole duration has no fraction")

	var (zero_text, _zero_error) = JsonDuration.format(0, 0)
	check(zero_text == "0s", "a zero duration formats")

	var (negative_whole_text, _negative_whole_error) = JsonDuration.format(-3, -1)
	check(negative_whole_text == "-3.000000001s", "a negative duration keeps one leading sign")

	## The case to get right: the seconds are zero, so only the nanos carry the
	## sign, and a check that reads the sign off seconds alone would miss it.
	var (negative_text, _negative_error) = JsonDuration.format(0, -500000000)
	check(negative_text == "-0.500s", "a sub-second negative duration keeps its sign")

	var (_mixed_text, mixed_error) = JsonDuration.format(1, -1)
	check(mixed_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "disagreeing signs are refused")

	var (_mixed_negative_text, mixed_negative_error) = JsonDuration.format(-1, 1)
	check(mixed_negative_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "disagreeing signs are refused either way round")

	var (_over_range_text, over_range_error) = JsonDuration.format(315576000001, 0)
	check(over_range_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "seconds past the upper bound are refused")

	var (_under_range_text, under_range_error) = JsonDuration.format(-315576000001, 0)
	check(under_range_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "seconds past the lower bound are refused")

	var (_wide_nanos_text, wide_nanos_error) = JsonDuration.format(0, 1000000000)
	check(wide_nanos_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "an out-of-range nanosecond is refused")

	var (duration_seconds, duration_nanos, duration_parse_error) = JsonDuration.parse("-0.5s")
	check(duration_parse_error == ProtobufError.OK, "a negative duration parses")
	check(duration_seconds == 0, "a sub-second duration has zero seconds")
	check(duration_nanos == -500000000, "a sub-second duration carries the sign in nanos")

	var (positive_seconds, positive_nanos, positive_parse_error) = JsonDuration.parse("3.000000001s")
	check(positive_parse_error == ProtobufError.OK, "a nine-digit fraction parses")
	check(positive_seconds == 3, "parsed whole seconds")
	check(positive_nanos == 1, "a nine-digit fraction keeps every digit")

	var (one_digit_seconds, one_digit_nanos, one_digit_error) = JsonDuration.parse("3.5s")
	check(one_digit_error == ProtobufError.OK, "a one-digit fraction parses")
	check(one_digit_seconds == 3, "a one-digit fraction keeps the whole part")
	check(one_digit_nanos == 500000000, "a one-digit fraction scales to nanos")

	var (whole_only_seconds, whole_only_nanos, whole_only_error) = JsonDuration.parse("3s")
	check(whole_only_error == ProtobufError.OK, "a whole duration parses")
	check(whole_only_seconds == 3, "a whole duration has no fraction to parse")
	check(whole_only_nanos == 0, "a whole duration parses to zero nanos")

	var (_no_suffix_seconds, _no_suffix_nanos, no_suffix_error) = JsonDuration.parse("3")
	check(no_suffix_error == ProtobufError.JSON_TYPE_MISMATCH, "a missing suffix is refused")

	var (_empty_seconds, _empty_nanos, empty_error) = JsonDuration.parse("")
	check(empty_error == ProtobufError.JSON_TYPE_MISMATCH, "an empty string is refused")

	## Digits on only one side of the point are legal, matching what a
	## compatible protobuf implementation may emit.
	var (trailing_point_seconds, trailing_point_nanos, trailing_point_error) = JsonDuration.parse("3.s")
	check(trailing_point_error == ProtobufError.OK, "an empty fraction after the point parses")
	check(trailing_point_seconds == 3 and trailing_point_nanos == 0, "an empty fraction after the point has zero nanos")

	var (leading_point_seconds, leading_point_nanos, leading_point_error) = JsonDuration.parse(".5s")
	check(leading_point_error == ProtobufError.OK, "an empty whole part before the point parses")
	check(leading_point_seconds == 0 and leading_point_nanos == 500000000, "an empty whole part before the point has zero seconds")

	var (negative_leading_point_seconds, negative_leading_point_nanos, negative_leading_point_error) = JsonDuration.parse("-.5s")
	check(negative_leading_point_error == ProtobufError.OK, "a negative empty whole part parses")
	check(negative_leading_point_seconds == 0 and negative_leading_point_nanos == -500000000, "a negative empty whole part carries its sign in nanos")

	var (_wide_fraction_seconds, _wide_fraction_nanos, wide_fraction_error) = JsonDuration.parse("3.1234567890s")
	check(wide_fraction_error == ProtobufError.JSON_TYPE_MISMATCH, "a ten-digit fraction is refused")

	var (_alpha_seconds, _alpha_nanos, alpha_error) = JsonDuration.parse("3xs")
	check(alpha_error == ProtobufError.JSON_TYPE_MISMATCH, "a non-digit in the whole part is refused")

	var (_leading_zero_seconds, _leading_zero_nanos, leading_zero_error) = JsonDuration.parse("01s")
	check(leading_zero_error == ProtobufError.JSON_TYPE_MISMATCH, "a zero-padded whole part is refused")

	var (bare_zero_seconds, bare_zero_nanos, bare_zero_error) = JsonDuration.parse("0s")
	check(bare_zero_error == ProtobufError.OK, "a bare zero parses")
	check(bare_zero_seconds == 0 and bare_zero_nanos == 0, "a bare zero parses to zero")

	var (_over_range_seconds, _over_range_nanos, over_range_parse_error) = JsonDuration.parse("315576000001s")
	check(over_range_parse_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a parsed second past the upper bound is refused")

	## Syntactically valid but numerically out of range, so the error names the
	## range rather than the shape.
	var (_too_wide_seconds, _too_wide_nanos, too_wide_error) = JsonDuration.parse("9999999999999s")
	check(too_wide_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a whole part wider than the range can hold is out of range")

	## A run too wide to accumulate into a 64-bit integer without overflowing
	## is refused before it is ever computed.
	var (_unsafe_wide_seconds, _unsafe_wide_nanos, unsafe_wide_error) = JsonDuration.parse("9999999999999999999s")
	check(unsafe_wide_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a whole part too wide to accumulate safely is out of range")

	## Splitting the empty string on "," naively yields one empty element rather
	## than none, so an empty mask is checked before anything with paths in it.
	var (empty_paths, empty_paths_error) = JsonFieldMask.from_json("")
	check(empty_paths_error == ProtobufError.OK, "an empty mask parses")
	check(empty_paths.size() == 0, "an empty mask has no paths")

	var (mask_text, mask_error) = JsonFieldMask.to_json(["foo_bar.baz_qux", "user"])
	check(mask_error == ProtobufError.OK, "a field mask converts to JSON")
	check(mask_text == "fooBar.bazQux,user", "a field mask camelCases each segment")

	var (_upper_text, upper_error) = JsonFieldMask.to_json(["fooBar"])
	check(upper_error == ProtobufError.JSON_TYPE_MISMATCH, "an uppercase path is refused")

	var (mask_paths, mask_paths_error) = JsonFieldMask.from_json("fooBar.bazQux,user")
	check(mask_paths_error == ProtobufError.OK, "a field mask parses")
	check(mask_paths == ["foo_bar.baz_qux", "user"], "a field mask restores snake_case")

	## A trailing or doubled underscore is swallowed by the camelCase mapping
	## rather than reproduced coming back, so it cannot round-trip either.
	var (_trailing_text, trailing_error) = JsonFieldMask.to_json(["foo_"])
	check(trailing_error == ProtobufError.JSON_TYPE_MISMATCH, "a trailing underscore is refused")

	var (_doubled_text, doubled_error) = JsonFieldMask.to_json(["foo__bar"])
	check(doubled_error == ProtobufError.JSON_TYPE_MISMATCH, "a doubled underscore is refused")

	## The canonical form never carries an underscore, so a JSON path with one
	## is not something this helper could have produced.
	var (_underscore_paths, underscore_error) = JsonFieldMask.from_json("foo_bar")
	check(underscore_error == ProtobufError.JSON_TYPE_MISMATCH, "a JSON path with an underscore is refused")

	## An underscore immediately before a digit has no visible effect once
	## capitalized, so the conversion would lose it silently.
	var (_digit_text, digit_error) = JsonFieldMask.to_json(["foo_1"])
	check(digit_error == ProtobufError.JSON_TYPE_MISMATCH, "an underscore before a digit is refused")

	## An empty segment -- a leading, trailing, or doubled dot -- has nothing
	## to convert and is refused in both directions.
	var (_leading_dot_text, leading_dot_error) = JsonFieldMask.to_json([".foo"])
	check(leading_dot_error == ProtobufError.JSON_TYPE_MISMATCH, "a leading dot is refused")

	var (_double_dot_text, double_dot_error) = JsonFieldMask.to_json(["foo..bar"])
	check(double_dot_error == ProtobufError.JSON_TYPE_MISMATCH, "a doubled dot is refused")

	## An underscore at the start of a segment is different from one in the
	## middle: it round-trips cleanly, becoming a capitalized first letter.
	var (leading_underscore_text, leading_underscore_error) = JsonFieldMask.to_json(["_foo.bar._baz"])
	check(leading_underscore_error == ProtobufError.OK, "a segment-initial underscore converts")
	check(leading_underscore_text == "Foo.bar.Baz", "a segment-initial underscore becomes a capital")

	var (leading_underscore_paths, leading_underscore_parse_error) = JsonFieldMask.from_json("Foo.bar.Baz")
	check(leading_underscore_parse_error == ProtobufError.OK, "a capitalized segment start parses")
	check(leading_underscore_paths == ["_foo.bar._baz"], "a capitalized segment start restores its underscore")

	## A comma inside a path would be indistinguishable from the delimiter
	## that joins paths, so it is refused rather than accepted and later
	## split into paths that were never there.
	var (_comma_text, comma_error) = JsonFieldMask.to_json(["foo,bar"])
	check(comma_error == ProtobufError.JSON_TYPE_MISMATCH, "a comma inside a path is refused")

	var (_comma_paths, comma_parse_error) = JsonFieldMask.from_json("foo!bar")
	check(comma_parse_error == ProtobufError.JSON_TYPE_MISMATCH, "a non-identifier character is refused")

	## A protobuf identifier may never start with a digit, in the whole path
	## or in a later segment, so neither direction accepts one that does.
	var (_leading_digit_text, leading_digit_error) = JsonFieldMask.to_json(["1foo"])
	check(leading_digit_error == ProtobufError.JSON_TYPE_MISMATCH, "a path opening on a digit is refused")

	var (_leading_digit_segment_text, leading_digit_segment_error) = JsonFieldMask.to_json(["foo.1bar"])
	check(leading_digit_segment_error == ProtobufError.JSON_TYPE_MISMATCH, "a segment opening on a digit is refused")

	var (_leading_digit_paths, leading_digit_parse_error) = JsonFieldMask.from_json("1foo")
	check(leading_digit_parse_error == ProtobufError.JSON_TYPE_MISMATCH, "a JSON path opening on a digit is refused")

## The unsigned helper behind the canonical JSON mapping of a uint64 or fixed64
## field, and of an unsigned 64-bit map key.
##
## A Foundry int is signed, so half the unsigned range is carried as a negative
## bit pattern that str() would print with a minus sign. Both directions here
## are checked at the two boundaries that distinguishes: the widest signed value,
## which is the last one str() would get right, and the one above it.
func check_json_uint64() -> void:
	check(JsonUint64.format(0) == "0", "zero formats unsigned")
	check(JsonUint64.format(1) == "1", "a small value formats unsigned")
	check(JsonUint64.format(max_int64()) == "9223372036854775807", "the widest signed value formats unsigned")
	## min_int64() is the bit pattern of 2^63, the first value a signed int
	## cannot state and the first one str() would print as a negative.
	check(JsonUint64.format(min_int64()) == "9223372036854775808", "the first unsigned-only value formats unsigned")
	check(JsonUint64.format(-1) == "18446744073709551615", "the widest unsigned value formats unsigned")

	var (zero_value, zero_error) = JsonUint64.parse("0")
	check(zero_error == ProtobufError.OK, "zero parses")
	check(zero_value == 0, "zero parses to zero")

	var (widest_signed_value, widest_signed_error) = JsonUint64.parse("9223372036854775807")
	check(widest_signed_error == ProtobufError.OK, "the widest signed value parses")
	check(widest_signed_value == max_int64(), "the widest signed value parses exactly")

	var (unsigned_only_value, unsigned_only_error) = JsonUint64.parse("9223372036854775808")
	check(unsigned_only_error == ProtobufError.OK, "the first unsigned-only value parses")
	check(unsigned_only_value == min_int64(), "the first unsigned-only value parses to its bit pattern")

	var (widest_value, widest_error) = JsonUint64.parse("18446744073709551615")
	check(widest_error == ProtobufError.OK, "the widest unsigned value parses")
	check(widest_value == -1, "the widest unsigned value parses to its bit pattern")

	var (_empty_value, empty_error) = JsonUint64.parse("")
	check(empty_error == ProtobufError.JSON_TYPE_MISMATCH, "an empty string is refused as unsigned")

	var (_negative_value, negative_error) = JsonUint64.parse("-1")
	check(negative_error == ProtobufError.JSON_TYPE_MISMATCH, "a negative is refused as unsigned")

	## A spelling that does not come back the way it went in is refused rather
	## than normalized, matching the signed reader.
	var (_signed_value, signed_error) = JsonUint64.parse("+5")
	check(signed_error == ProtobufError.JSON_TYPE_MISMATCH, "an explicit plus is refused as unsigned")

	var (_padded_value, padded_error) = JsonUint64.parse("007")
	check(padded_error == ProtobufError.JSON_TYPE_MISMATCH, "a zero-padded value is refused as unsigned")

	var (_exponent_value, exponent_error) = JsonUint64.parse("1e3")
	check(exponent_error == ProtobufError.JSON_TYPE_MISMATCH, "an exponent is refused as unsigned")

	var (_fraction_value, fraction_error) = JsonUint64.parse("1.5")
	check(fraction_error == ProtobufError.JSON_TYPE_MISMATCH, "a fraction is refused as unsigned")

	var (_spaced_value, spaced_error) = JsonUint64.parse(" 1")
	check(spaced_error == ProtobufError.JSON_TYPE_MISMATCH, "leading space is refused as unsigned")

	## Syntactically a canonical decimal, so the error names the range rather
	## than the shape. Both are twenty digits, the only width whose digits have
	## to be compared against the maximum.
	var (_over_value, over_error) = JsonUint64.parse("18446744073709551616")
	check(over_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "one past the widest unsigned value is out of range")

	var (_far_over_value, far_over_error) = JsonUint64.parse("99999999999999999999")
	check(far_over_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a twenty-digit value past the range is refused")

	var (_wider_value, wider_error) = JsonUint64.parse("184467440737095516150")
	check(wider_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "a twenty-one-digit value is out of range")

## The generated JSON surface, over a schema carrying every field kind the
## canonical mapping has a rule for. examples/golden-json pins what the emitter
## writes; this proves the engine agrees -- that the text it produces parses,
## that decoding it rebuilds the same message, and that the parts a Foundry int
## cannot state directly survive the trip as text.
func check_json_round_trip() -> void:
	var suite: JsonSuite = populated_json_suite()
	var text: String = JSON.stringify(suite, "", false)

	## The three mappings a signed int would get wrong on its own, spelled out
	## rather than only compared field by field after the trip.
	check(text.find('"int64Value":"-9223372036854775808"') >= 0, "an int64 is written as a signed decimal string")
	check(text.find('"uint64Value":"18446744073709551615"') >= 0, "a uint64 is written as an unsigned decimal string")
	check(text.find('"uint64Keyed":{"18446744073709551615":"widest"}') >= 0, "an unsigned 64-bit map key is written unsigned")
	check(text.find('"floatValue":"Infinity"') >= 0, "a non-finite float is written as its string form")
	check(text.find('"nickname":""') >= 0, "an explicitly present empty string is written")
	check(text.find('"flavor":"FLAVOR_BITTER"') >= 0, "an enum is written as its declared name")

	var parsed: JsonResult[JsonNode] = JSON.parse_to_node(text)
	check(parsed.is_ok(), "the emitted JSON document parses")
	if not parsed.is_ok():
		return

	var decoded_result: JsonResult[JsonSuite] = JsonSuite.from_json(parsed.value)
	check(decoded_result.is_ok(), "the emitted JSON document decodes")
	if not (decoded_result.value is JsonSuite):
		if decoded_result.error != null:
			printerr("FAIL: JSON decode reported ", decoded_result.error.message, " at ", decoded_result.error.path)
		return
	var decoded: JsonSuite = decoded_result.value

	## Re-encoding what was decoded is the strongest statement available: it
	## covers every member at once, ordering included, without depending on how
	## any one of them compares.
	check(JSON.stringify(decoded, "", false) == text, "a JSON round trip is lossless")

	check(decoded.double_value == -2.5, "a double round trips through JSON")
	check(is_inf(decoded.float_value) and decoded.float_value > 0.0, "positive infinity round trips through JSON")
	check(decoded.int32_value == -2147483648, "an int32 round trips through JSON")
	check(decoded.int64_value == min_int64(), "an int64 round trips through JSON")
	check(decoded.uint32_value == 4294967295, "a uint32 round trips through JSON")
	check(decoded.uint64_value == -1, "the widest uint64 round trips through JSON")
	check(decoded.sint32_value == -2147483648, "a sint32 round trips through JSON")
	check(decoded.sint64_value == min_int64(), "a sint64 round trips through JSON")
	check(decoded.fixed32_value == 4294967295, "a fixed32 round trips through JSON")
	check(decoded.fixed64_value == -1, "a fixed64 round trips through JSON")
	check(decoded.sfixed32_value == -2147483648, "a sfixed32 round trips through JSON")
	check(decoded.sfixed64_value == max_int64(), "a sfixed64 round trips through JSON")
	check(decoded.bool_value, "a bool round trips through JSON")
	check(decoded.string_value == "a \" quote and a \\ backslash", "a string round trips through JSON")
	check(decoded.bytes_value == PackedByteArray([0, 1, 250, 255]), "bytes round trip through JSON as base64")
	check(decoded.two_word_name == "camel", "a camelCase JSON name round trips")
	check(decoded.flavor == Flavor.FLAVOR_BITTER, "an enum round trips through JSON")
	check(decoded.primary is Reference and decoded.primary.label == "primary", "a message field round trips through JSON")
	check(decoded.primary is Reference and decoded.primary.weight == max_int64(), "a message field's int64 round trips through JSON")
	check(decoded.nickname is String and decoded.nickname == "", "an explicitly present empty string round trips through JSON")
	check(decoded.tags == ["first", "second"], "a repeated string round trips through JSON")
	check(decoded.tallies.size() == 3, "a repeated int64 keeps every element through JSON")
	check(decoded.tallies.size() == 3 and decoded.tallies[0] == min_int64() and decoded.tallies[2] == max_int64(),
		"a repeated int64 round trips through JSON")
	check(decoded.flavors.size() == 2, "a repeated enum keeps every element through JSON")
	check(decoded.flavors.size() == 2 and decoded.flavors[1] == Flavor.FLAVOR_BITTER, "a repeated enum round trips through JSON")
	check(decoded.references.size() == 2, "a repeated message keeps every element through JSON")
	check(decoded.references.size() == 2 and decoded.references[1].label == "second", "a repeated message element round trips through JSON")
	check(decoded.counts == {"ore": 7}, "a string-keyed map round trips through JSON")
	check(decoded.int32_keyed.has(-3) and decoded.int32_keyed[-3] == "negative", "an int32-keyed map round trips through JSON")
	check(decoded.int64_keyed.has(min_int64()) and decoded.int64_keyed[min_int64()] == "narrowest",
		"an int64-keyed map round trips through JSON")
	check(decoded.uint64_keyed.has(-1) and decoded.uint64_keyed[-1] == "widest",
		"a uint64-keyed map round trips through JSON")
	check(decoded.bool_keyed.has(true) and decoded.bool_keyed[true].label == "flagged", "a bool-keyed map round trips through JSON")
	check(decoded.inner is JsonSuite.Inner and decoded.inner.code == "nested", "a nested message round trips through JSON")

	match decoded.choice:
		JsonSuiteChoiceCase.Tally(var tally):
			check(tally == min_int64(), "a 64-bit oneof member round trips through JSON")
		_:
			printerr("FAIL: JSON oneof case did not round trip")
			failures += 1

	## The well-known types carry their own JSON form, so a reference to one is
	## the only place the generated surface delegates rather than writes.
	check(decoded.occurred_at is foundry.proto.wkt.Timestamp and decoded.occurred_at.seconds == 1136214245,
		"a Timestamp round trips through JSON")
	check(decoded.occurred_at is foundry.proto.wkt.Timestamp and decoded.occurred_at.nanos == 10000000,
		"a Timestamp fraction round trips through JSON")
	check(decoded.elapsed is Duration and decoded.elapsed.seconds == 3 and decoded.elapsed.nanos == 1,
		"a Duration round trips through JSON")
	check(decoded.label is StringValue and decoded.label.value == "wrapped", "a wrapper round trips through JSON")
	check(decoded.setting is Value, "a Value round trips through JSON")
	if decoded.setting is Value:
		match decoded.setting.kind:
			ValueKindCase.NumberValue(var number):
				## A whole number in a Value is written as 1.0, not 1: the kind
				## has no integral case, so it goes out as a double.
				check(number == 1.0, "a whole Value number round trips through JSON")
				check(text.find('"setting":1.0') >= 0, "a whole Value number is written with a fraction")
			_:
				printerr("FAIL: Value kind did not round trip through JSON")
				failures += 1
	if decoded.attributes is Struct:
		var decoded_name: Value? = decoded.attributes.fields.get("name")
		if decoded_name is Value:
			match decoded_name.kind:
				ValueKindCase.StringValue(var member):
					check(member == "axe", "a Struct member round trips through JSON")
				_:
					printerr("FAIL: Struct member kind did not round trip through JSON")
					failures += 1
		else:
			printerr("FAIL: Struct member did not round trip through JSON")
			failures += 1
	else:
		printerr("FAIL: Struct field did not round trip through JSON")
		failures += 1

	## An unknown member is refused rather than ignored, and the failure comes
	## back as a value carrying the path that named it.
	var unknown_result: JsonResult[JsonSuite] = JsonSuite.from_json(JsonNode.object_of({"nope": JsonNode.Int(1)}))
	check(not unknown_result.is_ok(), "an unknown JSON member is refused")
	if unknown_result.error != null:
		check(unknown_result.error.message.find("JSON_UNKNOWN_FIELD") >= 0, "an unknown member names its error case")

	## A JSON value of the wrong shape is refused with the case that names it.
	var mistyped_result: JsonResult[JsonSuite] = JsonSuite.from_json(JsonNode.object_of({"int32Value": JsonNode.Str("nope")}))
	check(not mistyped_result.is_ok(), "a mistyped JSON member is refused")
	if mistyped_result.error != null:
		check(mistyped_result.error.message.find("JSON_TYPE_MISMATCH") >= 0, "a mistyped member names its error case")

## The fixture behind the JSON round trip, populated so that every field is
## written: the mapping omits a field at its default, so a default-valued one
## would leave its rule unexercised.
func populated_json_suite() -> JsonSuite:
	var suite: JsonSuite = JsonSuite.new()
	suite.double_value = -2.5
	suite.float_value = INF
	suite.int32_value = -2147483648
	suite.int64_value = min_int64()
	suite.uint32_value = 4294967295
	suite.uint64_value = -1
	suite.sint32_value = -2147483648
	suite.sint64_value = min_int64()
	suite.fixed32_value = 4294967295
	suite.fixed64_value = -1
	suite.sfixed32_value = -2147483648
	suite.sfixed64_value = max_int64()
	suite.bool_value = true
	suite.string_value = "a \" quote and a \\ backslash"
	suite.bytes_value = PackedByteArray([0, 1, 250, 255])
	suite.two_word_name = "camel"
	suite.flavor = Flavor.FLAVOR_BITTER

	var primary: Reference = Reference.new()
	primary.label = "primary"
	primary.weight = max_int64()
	suite.primary = primary

	## Present and empty, which the mapping distinguishes from absent.
	suite.nickname = ""

	suite.tags = ["first", "second"]
	suite.tallies.append(min_int64())
	suite.tallies.append(0)
	suite.tallies.append(max_int64())
	suite.flavors.append(Flavor.FLAVOR_SWEET)
	suite.flavors.append(Flavor.FLAVOR_BITTER)

	var first_element: Reference = Reference.new()
	first_element.label = "first"
	suite.references.append(first_element)
	var second_element: Reference = Reference.new()
	second_element.label = "second"
	suite.references.append(second_element)

	suite.counts["ore"] = 7
	suite.int32_keyed[-3] = "negative"
	suite.int64_keyed[min_int64()] = "narrowest"
	suite.uint64_keyed[-1] = "widest"
	var flagged: Reference = Reference.new()
	flagged.label = "flagged"
	suite.bool_keyed[true] = flagged

	suite.choice = JsonSuiteChoiceCase.Tally(min_int64())

	var occurred_at: foundry.proto.wkt.Timestamp = foundry.proto.wkt.Timestamp.new()
	occurred_at.seconds = 1136214245
	occurred_at.nanos = 10000000
	suite.occurred_at = occurred_at

	var elapsed: Duration = Duration.new()
	elapsed.seconds = 3
	elapsed.nanos = 1
	suite.elapsed = elapsed

	var name: Value = Value.new()
	name.kind = ValueKindCase.StringValue("axe")
	var attributes: Struct = Struct.new()
	attributes.fields["name"] = name
	suite.attributes = attributes

	var setting: Value = Value.new()
	setting.kind = ValueKindCase.NumberValue(1.0)
	suite.setting = setting

	var label: StringValue = StringValue.new()
	label.value = "wrapped"
	suite.label = label

	var inner: JsonSuite.Inner = JsonSuite.Inner.new()
	inner.code = "nested"
	suite.inner = inner

	return suite

## A JsonNode on its own has no route to JSON text: the native marshaller fires
## for objects that conform to JsonSerializable, and stringifying a bare node
## yields its raw tagged array instead. So the encoder assertions below go
## through a conforming carrier, which is also the shape the generated JSON
## surface will take.
class JsonCarrier extends RefCounted uses JsonSerializable:
	var node: JsonNode = JsonNode.Null

	func to_json() -> JsonNode:
		return node

	static func from_json(parsed: JsonNode) -> JsonResult[JsonCarrier]:
		var carrier: JsonCarrier = JsonCarrier.new()
		carrier.node = parsed
		return JsonResult[JsonCarrier].ok(carrier)

## The engine owns JsonNode, JsonSerializable, JsonResult and JsonDecodeError,
## so the JSON emitters are built on behavior this repository does not implement
## and cannot see change. These pin the parts they depend on, so an engine bump
## that moves any of them fails here rather than in someone's generated bindings.
func check_engine_json_types() -> void:
	## Directly self-recursive construction: an object holding an array holding
	## an object. Key order survives because sort_keys is passed false, which is
	## what will keep emitted output in field declaration order.
	var item: JsonNode = JsonNode.object_of({"name": JsonNode.Str("axe"), "quantity": JsonNode.Int(2)})
	var document: JsonNode = JsonNode.object_of({"items": JsonNode.array_of([item]), "count": JsonNode.Int(1)})
	var document_text: String = json_text(document)
	check(document_text == '{"items":[{"name":"axe","quantity":2}],"count":1}', "a nested JSON document encodes in declaration order")

	var reparsed: JsonResult[JsonNode] = JSON.parse_to_node(document_text)
	check(reparsed.is_ok(), "a nested JSON document parses back")
	check(json_text(reparsed.value) == document_text, "a nested JSON document round trips")
	check(json_entry_name(reparsed.value) == "axe", "a nested JSON document keeps its innermost object")

	## int32 must render as 1 and double as 1.0, so the two cases have to stay
	## distinguishable through the encoder rather than collapsing to one number.
	check(json_text(JsonNode.Int(1)) == "1", "an Int encodes without a fraction")
	check(json_text(JsonNode.Float(1.0)) == "1.0", "a Float encodes with a fraction")

	## A 64-bit integer is emitted as a string because that is the only form
	## that survives: sent as a bare JSON number it comes back in a different
	## case entirely, so a decoder cannot assume Int for a 64-bit field.
	var widest: int = 9223372036854775807
	check(json_text(JsonNode.Str(str(widest))) == '"9223372036854775807"', "a 64-bit integer encodes as a string")

	var parsed_string: JsonResult[JsonNode] = JSON.parse_to_node('"9223372036854775807"')
	check(parsed_string.is_ok(), "a 64-bit integer sent as a string parses")
	match parsed_string.value:
		JsonNode.Str(var text):
			check(text.to_int() == widest, "a 64-bit integer sent as a string survives exactly")
		_:
			printerr("FAIL: a 64-bit integer sent as a string did not parse as Str")
			failures += 1

	var parsed_number: JsonResult[JsonNode] = JSON.parse_to_node("9223372036854775807")
	check(parsed_number.is_ok(), "a bare 64-bit JSON number parses")
	match parsed_number.value:
		JsonNode.Float(_value):
			pass
		_:
			printerr("FAIL: a bare 64-bit JSON number did not parse as Float")
			failures += 1

	## Handing a non-finite float to JsonNode.Float produces output that is not
	## canonical proto3 and, for NaN, not even valid JSON. That is why the
	## emitter substitutes the string forms itself rather than passing the value
	## through. The NaN case logs an engine warning, which is expected.
	check(json_text(JsonNode.Float(NAN)) == "null", "NaN is mangled to null by the encoder")
	check(json_text(JsonNode.Float(INF)) == "1e99999", "positive infinity is mangled by the encoder")
	check(json_text(JsonNode.Float(-INF)) == "-1e99999", "negative infinity is mangled by the encoder")

	## A malformed document reports through JsonResult rather than raising, so
	## the decoders can surface it as a value instead of aborting their caller.
	var malformed: JsonResult[JsonNode] = JSON.parse_to_node("{")
	check(not malformed.is_ok(), "a malformed JSON document does not parse")
	check(malformed.value == null, "a failed parse carries no value")
	check(malformed.error != null, "a failed parse carries a decode error")
	if malformed.error != null:
		check(malformed.error.message != "", "a decode error explains itself")

func json_text(node: JsonNode) -> String:
	var carrier: JsonCarrier = JsonCarrier.new()
	carrier.node = node
	return JSON.stringify(carrier, "", false)

func json_entry_name(document: JsonNode) -> String:
	var items: Array[JsonNode] = json_items(json_member(document, "items"))
	if items.is_empty():
		return ""
	return json_string(json_member(items[0], "name"))

func json_member(node: JsonNode, key: String) -> JsonNode:
	match node:
		JsonNode.Object(var entries):
			if entries.has(key):
				return entries[key]
		_:
			pass
	return JsonNode.Null

func json_items(node: JsonNode) -> Array[JsonNode]:
	match node:
		JsonNode.Array(var items):
			return items
		_:
			pass
	var empty: Array[JsonNode] = []
	return empty

func json_string(node: JsonNode) -> String:
	match node:
		JsonNode.Str(var text):
			return text
		_:
			pass
	return ""
