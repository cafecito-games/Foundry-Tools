package fsgenerator

import (
	"strings"
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestNamespaceFromPackageAndOption(t *testing.T) {
	require.Equal(t, "cafecito.game.v1", NamespaceFor(&protoast.ProtoFile{Package: "cafecito.game.v1"}))
	require.Equal(t, "custom.ns", NamespaceFor(&protoast.ProtoFile{
		Package: "ignored",
		Options: map[string]any{"(foundrytools.namespace)": "custom.ns"},
	}))
}

func TestValidateNamespace(t *testing.T) {
	require.Error(t, ValidateNamespace(""))
	require.NoError(t, ValidateNamespace("cafecito.game_v1"))
	require.Error(t, ValidateNamespace("cafecito..game"))
	require.Error(t, ValidateNamespace("cafecito.1game"))
}

func TestGenerateRequiresNamespace(t *testing.T) {
	_, err := Generate(messageFile(&protoast.Message{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}},
	}), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "namespace is required")
}

func TestTypeName(t *testing.T) {
	require.Equal(t, "PlayerState", TypeName("player_state"))
	require.Equal(t, "PlayerState", TypeName("player-state"))
	require.Equal(t, "OuterInner", TypeName("outer.inner"))
	require.Equal(t, "Class_", TypeName("class"))
}

// A dotted proto path keeps its scoping, unlike TypeName which flattens.
func TestTypeReferenceKeepsNesting(t *testing.T) {
	require.Equal(t, "Outer.Inner", TypeReference("outer.inner"))
	require.Equal(t, "Outer.Inner", TypeReference(".outer.inner"))
	require.Equal(t, "PlayerStatus", TypeReference("PlayerStatus"))
}

func TestScalarTypeMapping(t *testing.T) {
	require.Equal(t, "int", ScalarType("int32").Render())
	require.Equal(t, "float", ScalarType("double").Render())
	require.Equal(t, "String", ScalarType("string").Render())
	require.Equal(t, "PackedByteArray", ScalarType("bytes").Render())
}

func messageFile(messages ...*protoast.Message) *protoast.ProtoFile {
	return &protoast.ProtoFile{Syntax: "proto3", Messages: messages}
}

func namespacedFile(messages []*protoast.Message, enums []*protoast.Enum) *protoast.ProtoFile {
	return &protoast.ProtoFile{
		Syntax:   "proto3",
		Package:  "cafecito.game.v1",
		Messages: messages,
		Enums:    enums,
	}
}

func prefixedFile(prefix string, messages []*protoast.Message, enums []*protoast.Enum) *protoast.ProtoFile {
	file := namespacedFile(messages, enums)
	file.Options = map[string]any{typePrefixOptionKey: prefix}
	return file
}

func generate(t *testing.T, file *protoast.ProtoFile) GeneratedFiles {
	t.Helper()
	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	return files
}

func playerSource(t *testing.T, fields []*protoast.Field, alsoDeclared ...*protoast.Message) string {
	t.Helper()
	messages := append([]*protoast.Message{{Name: "Player", Fields: fields}}, alsoDeclared...)
	files := generate(t, namespacedFile(messages, nil))
	return files["cafecito/game/v1/Player.pb.fs"]
}

func slotMessage() *protoast.Message {
	return &protoast.Message{
		Name:   "Slot",
		Fields: []*protoast.Field{{FieldType: "string", Name: "label", Number: 1}},
	}
}

func TestGeneratePrefixesDeclarationsReferencesAndPaths(t *testing.T) {
	files := generate(t, prefixedFile("Game", []*protoast.Message{{
		Name: "Outer",
		Fields: []*protoast.Field{
			{FieldType: "Inner", Name: "inner", Number: 1},
			{FieldType: "Outer.Inner", Name: "qualified_inner", Number: 2},
			{FieldType: "State", Name: "state", Number: 3},
			{FieldType: "Kind", Name: "kind", Number: 4},
		},
		NestedMessages: []*protoast.Message{{
			Name: "Inner",
		}},
		NestedEnums: []*protoast.Enum{{
			Name:   "Kind",
			Values: []*protoast.EnumValue{{Name: "KIND_UNSPECIFIED", Number: 0}},
		}},
		Oneofs: []*protoast.Oneof{{
			Name:   "choice",
			Fields: []*protoast.Field{{FieldType: "string", Name: "text", Number: 5}},
		}},
	}}, []*protoast.Enum{{
		Name:   "State",
		Values: []*protoast.EnumValue{{Name: "STATE_UNSPECIFIED", Number: 0}},
	}}))

	messageSource := files["cafecito/game/v1/GameOuter.pb.fs"]
	require.Contains(t, messageSource, "class_name GameOuter")
	require.Contains(t, messageSource, "final class GameInner")
	require.Contains(t, messageSource, "var inner: GameInner?")
	require.Contains(t, messageSource, "var qualified_inner: GameOuter.GameInner?")
	require.Contains(t, messageSource, "var state: GameState = GameState.STATE_UNSPECIFIED")
	require.Contains(t, messageSource, "enum GameKind:")
	require.Contains(t, messageSource, "var kind: GameKind = GameKind.KIND_UNSPECIFIED")

	oneofSource := files["cafecito/game/v1/GameOuterChoiceCase.pb.fs"]
	require.Contains(t, oneofSource, "enum_name GameOuterChoiceCase")

	enumSource := files["cafecito/game/v1/GameState.pb.fs"]
	require.Contains(t, enumSource, "enum_name GameState")

	require.NotContains(t, files, "cafecito/game/v1/Outer.pb.fs")
	require.NotContains(t, files, "cafecito/game/v1/OuterChoiceCase.pb.fs")
	require.NotContains(t, files, "cafecito/game/v1/State.pb.fs")
}

func TestNestedOneofFlattensPrefixedOwnerSegments(t *testing.T) {
	files := generate(t, prefixedFile("Game", []*protoast.Message{{
		Name: "Outer",
		NestedMessages: []*protoast.Message{{
			Name: "Inner",
			Oneofs: []*protoast.Oneof{{
				Name:   "choice",
				Fields: []*protoast.Field{{FieldType: "string", Name: "text", Number: 1}},
			}},
		}},
	}}, nil))

	source := files["cafecito/game/v1/GameOuterGameInnerChoiceCase.pb.fs"]
	require.Contains(t, source, "enum_name GameOuterGameInnerChoiceCase")
	require.NotContains(t, files, "cafecito/game/v1/GameOuterInnerChoiceCase.pb.fs")
}

func TestSameFileDescriptorEnumResolvesLocally(t *testing.T) {
	files, err := Generate(namespacedFile(
		[]*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				Name:         "status",
				Number:       1,
				FieldType:    "PlayerStatus",
				FullTypePath: "cafecito.game.v1.PlayerStatus",
				SourceFile:   "player.proto",
				IsEnum:       true,
			}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	), "player.proto", nil)
	require.NoError(t, err)

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source,
		"var status: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED")
	require.Contains(t, source, "status.to_wire()")
}

func TestSameFileDescriptorEnumUsesLocalPrefix(t *testing.T) {
	files, err := Generate(prefixedFile(
		"Game",
		[]*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				Name:         "status",
				Number:       1,
				FieldType:    "PlayerStatus",
				FullTypePath: "cafecito.game.v1.PlayerStatus",
				SourceFile:   "player.proto",
				IsEnum:       true,
			}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	), "player.proto", nil)
	require.NoError(t, err)

	source := files["cafecito/game/v1/GamePlayer.pb.fs"]
	require.Contains(t, source,
		"var status: GamePlayerStatus = GamePlayerStatus.PLAYER_STATUS_UNSPECIFIED")
	require.Contains(t, files["cafecito/game/v1/GamePlayerStatus.pb.fs"],
		"enum_name GamePlayerStatus")
}

func TestSameFileDescriptorMapEnumResolvesLocally(t *testing.T) {
	files, err := Generate(namespacedFile(
		[]*protoast.Message{{
			Name: "Player",
			Maps: []*protoast.MapField{{
				Name:              "seen",
				Number:            1,
				KeyType:           "string",
				ValueType:         "PlayerStatus",
				FullValueTypePath: "cafecito.game.v1.PlayerStatus",
				ValueSourceFile:   "player.proto",
				ValueIsEnum:       true,
			}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	), "player.proto", nil)
	require.NoError(t, err)

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "var seen: Dictionary[String, PlayerStatus] = {}")
	require.Contains(t, source,
		"var _pb_seen_value: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED")
	require.Contains(t, source,
		"_pb_seen_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))")
}

func TestReferenceUsesDeclaringFilesPrefix(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Options: map[string]any{typePrefixOptionKey: "Inventory"},
		Messages: []*protoast.Message{{
			Name: "Item",
			NestedMessages: []*protoast.Message{{
				Name: "Detail",
			}},
			NestedEnums: []*protoast.Enum{{
				Name:   "Rarity",
				Values: []*protoast.EnumValue{{Name: "RARITY_UNSPECIFIED", Number: 0}},
			}},
		}},
	}
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Item", Name: "held", Number: 1, SourceFile: "inventory.proto"},
			{FieldType: "Item.Detail", Name: "detail", Number: 2, SourceFile: "inventory.proto"},
			{
				FieldType: "Item.Rarity", Name: "rarity", Number: 3,
				IsEnum: true, SourceFile: "inventory.proto",
			},
		},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.NoError(t, err)

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "var held: InventoryItem?")
	require.Contains(t, source, "var detail: InventoryItem.InventoryDetail?")
	require.Contains(t, source,
		"var rarity: InventoryItem.InventoryRarity = InventoryItem.InventoryRarity.RARITY_UNSPECIFIED")
}

func TestMissingImportedDeclarationDoesNotResolveToLocalType(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Options: map[string]any{typePrefixOptionKey: "Inventory"},
		// Item is intentionally absent. Descriptor-driven generation can know
		// the field's source and kind without receiving its declaration.
	}
	local := namespacedFile(
		[]*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				FieldType: "Item", Name: "held", Number: 1,
				SourceFile: "inventory.proto",
			}},
			Oneofs: []*protoast.Oneof{{
				Name: "payload",
				Fields: []*protoast.Field{{
					FieldType: "Item", Name: "equipped", Number: 2,
					SourceFile: "inventory.proto",
				}},
			}},
		}},
		[]*protoast.Enum{{
			Name:   "Item",
			Values: []*protoast.EnumValue{{Name: "LOCAL_ITEM_UNSPECIFIED", Number: 0}},
		}},
	)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: imported, Filename: "inventory.proto",
	}})
	require.NoError(t, err)

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, messageSource, "import cafecito.inventory.v1")
	require.Contains(t, messageSource, "var held: InventoryItem? = null")
	require.NotContains(t, messageSource, "LOCAL_ITEM_UNSPECIFIED")

	unionSource := files["cafecito/game/v1/PlayerPayloadCase.pb.fs"]
	require.Contains(t, unionSource, "import cafecito.inventory.v1")
	require.Contains(t, unionSource, "Equipped(equipped: InventoryItem)")
	require.NotContains(t, unionSource, "Equipped(equipped: Item)")
}

func TestReferencedDependencyReportsInvalidPrefix(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Options: map[string]any{typePrefixOptionKey: "bad-prefix"},
		Messages: []*protoast.Message{{
			Name: "Item",
		}},
	}
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Item", Name: "held", Number: 1, SourceFile: "inventory.proto"}},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "inventory.proto")
	require.Contains(t, err.Error(), typePrefixOptionKey)
}

func TestUnreferencedDependencyIgnoresInvalidPrefix(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Options: map[string]any{typePrefixOptionKey: "bad-prefix"},
		Messages: []*protoast.Message{{
			Name: "Item",
		}},
	}
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.NoError(t, err)
	require.Contains(t, files, "cafecito/game/v1/Player.pb.fs")
}

func TestGenerateMessageAndEnumSkeletons(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}},
		}},
		[]*protoast.Enum{{
			Name: "PlayerStatus",
			Values: []*protoast.EnumValue{
				{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0},
				{Name: "PLAYER_STATUS_ONLINE", Number: 1},
			},
		}},
	))
	require.Len(t, files, 2)

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	enumSource := files["cafecito/game/v1/PlayerStatus.pb.fs"]

	require.Contains(t, messageSource, "## Generated protobuf message binding for Player.\nfinal class_name Player extends RefCounted uses Message")
	require.Contains(t, messageSource, "var name: String = \"\"")
	require.NoError(t, CheckPublicAPI(messageSource))
	require.Contains(t, enumSource,
		"## Generated protobuf enum binding for PlayerStatus.\n"+
			"enum_name PlayerStatus:\n"+
			"\tPLAYER_STATUS_UNSPECIFIED = 0\n"+
			"\tPLAYER_STATUS_ONLINE = 1\n")
	require.NotContains(t, enumSource, "{")
}

// The trait is inert unless generated messages actually declare conformance.
func TestGeneratedMessagesConformToMessageTrait(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})
	require.Contains(t, source, "final class_name Player extends RefCounted uses Message\n")
	require.NotContains(t, source, "uses foundry.proto.Message")
}

// An active import makes every runtime reference short.
func TestGeneratedCodeDoesNotRequalifyRuntimeReferences(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})
	require.Contains(t, source, "import foundry.proto")
	require.NotContains(t, source, "foundry.proto.Wire")
	require.NotContains(t, source, "foundry.proto.ProtobufError")
	require.NotContains(t, source, "foundry.proto.FieldRead")
}

// A plain field carries what a get/set pair would, so accessors are noise.
func TestGenerateEmitsPublicFieldsNotAccessorPairs(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "name", Number: 1},
		{FieldType: "int32", Name: "level", Number: 2},
	})
	require.Contains(t, source, "var name: String = \"\"")
	require.Contains(t, source, "var level: int = 0")
	require.NotContains(t, source, "func set_name(")
	require.NotContains(t, source, "func get_name(")
	require.NotContains(t, source, "var _name:")
}

func TestGenerateDecodeFactoryReturnsTuple(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})
	require.Contains(t, source, "static func from_bytes(_pb_data: PackedByteArray) -> (Player?, ProtobufError):")
	require.NotContains(t, source, "DecodeResult[")
	require.NotContains(t, source, "FieldRead[")
	require.NotContains(t, source, "Variant")
}

// The carriers are named tuples now, so the reads name them directly.
func TestGenerateUsesNamedTupleCarriers(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "name", Number: 1},
		{FieldType: "bytes", Name: "avatar", Number: 2},
	})
	require.Contains(t, source, "var _pb_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)")
	require.Contains(t, source, "var _pb_name_read: StringRead = Wire.read_string(")
	require.Contains(t, source, "var _pb_avatar_read: BytesRead = Wire.read_bytes(")
}

// Handling an unrecognized field is invariant across every message, so it lives
// in the runtime rather than being inlined into each binding. proto3 requires
// the bytes to survive a re-encode, so they are captured rather than dropped.
func TestGenerateDelegatesUnknownFieldCaptureToRuntime(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})
	require.Contains(t, source, "var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)")
	require.Contains(t, source, "var _pb_unknown_fields: PackedByteArray = PackedByteArray()")
	require.Contains(t, source, "_pb_result.append_array(_pb_unknown_fields)")
	require.NotContains(t, source, "Wire.WIRE_32BIT:")
	require.NotContains(t, source, "_pb_offset += 8")
}

func TestGenerateRepeatedFields(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "tags", Number: 1, Repeated: true},
		{FieldType: "int32", Name: "scores", Number: 2, Repeated: true},
	})

	require.Contains(t, source, "var tags: Array[String] = []")
	require.Contains(t, source, "var scores: Array[int] = []")
	require.Contains(t, source, "for _pb_tags_item: String in tags:")
	require.Contains(t, source, "tags.append(_pb_tags_read.value)")

	// Varint scalars pack; length-delimited ones cannot.
	require.Contains(t, source, "var _pb_scores_data: PackedByteArray = PackedByteArray()")
	require.Contains(t, source, "for _pb_scores_item: int in scores:")
	// A packed field must still decode the unpacked encoding.
	require.Contains(t, source, "if _pb_wire_type == Wire.WIRE_LENGTH_DELIMITED:")
	require.Contains(t, source, "elif _pb_wire_type == Wire.WIRE_VARINT:")
}

// proto3 explicit presence is the type-level nullable, so an optional string
// explicitly set to "" still goes on the wire.
func TestGenerateOptionalPresence(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "nickname", Number: 1, Optional: true},
		{FieldType: "string", Name: "name", Number: 2},
	})

	require.Contains(t, source, "var nickname: String? = null")
	require.Contains(t, source, "if nickname is String:")
	// Implicit presence keeps the zero-value rule.
	require.Contains(t, source, "if name != \"\":")
}

// Comparing a nullable built-in against null reports the wrong answer on the
// current engine; `is` is correct for every kind.
func TestGeneratePresenceUsesIsNotNullComparison(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "nickname", Number: 1, Optional: true},
		{FieldType: "Slot", Name: "primary", Number: 2},
	}, slotMessage())
	require.NotContains(t, source, "!= null")
	require.Contains(t, source, "if nickname is String:")
	require.Contains(t, source, "if primary is Slot:")
}

// A message-typed field is a length-delimited submessage, not a varint.
func TestGenerateMessageTypedFields(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "Slot", Name: "primary", Number: 1}}, slotMessage())

	require.Contains(t, source, "var primary: Slot? = null")
	require.NotContains(t, source, "var primary: Slot = 0")
	require.Contains(t, source, "var _pb_primary_data: PackedByteArray = primary.to_bytes()")
	require.Contains(t, source, "Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED))")
	// A singular message field merges rather than replaces, which protobuf
	// requires when the same field appears twice in one stream.
	require.Contains(t, source, "if not (primary is Slot):")
	require.Contains(t, source, "primary = Slot.new()")
	require.Contains(t, source, "var _pb_primary_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, primary)")
}

func TestGenerateEnumFieldsUseHostedWireConversion(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "status", Number: 1}},
		}},
		[]*protoast.Enum{{
			Name: "PlayerStatus",
			Values: []*protoast.EnumValue{
				{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0},
				{Name: "PLAYER_STATUS_ONLINE", Number: 1},
			},
		}},
	))

	enumSource := files["cafecito/game/v1/PlayerStatus.pb.fs"]
	// Each case is declared with its wire number, so the conversion out is a cast.
	require.Contains(t, enumSource, "\tfunc to_wire() -> int:\n\t\treturn self as int\n")
	// Self is what resolves to the namespaced enum from inside its own body.
	require.Contains(t, enumSource, "\tstatic func from_wire(value: int) -> Self?:")
	require.Contains(t, enumSource, "\t\t\t\treturn PlayerStatus.PLAYER_STATUS_ONLINE")
	// An open enum reports an unrecognized value rather than folding it to zero.
	require.Contains(t, enumSource, "\t\t\t_:\n\t\t\t\treturn null\n")

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, messageSource, "var status: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED")
	require.Contains(t, messageSource, "Wire.encode_varint(status.to_wire())")
	require.Contains(t, messageSource, "var _pb_status_case: PlayerStatus? = PlayerStatus.from_wire(_pb_status_read.value)")
	// An unrecognized value is kept verbatim rather than destroyed, in a
	// companion of its own so it can stand in for the field on re-encode.
	require.Contains(t, messageSource, "_pb_status_unknown = _pb_data.slice(_pb_offset, _pb_status_read.offset)")
	require.Contains(t, messageSource, "var _pb_status_unknown: PackedByteArray = PackedByteArray()")
}

func TestGenerateMapFields(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Maps: []*protoast.MapField{{
			KeyType:   "string",
			ValueType: "int32",
			Name:      "counts",
			Number:    1,
		}},
	}}, nil))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "var counts: Dictionary[String, int] = {}")
	require.Contains(t, source, "for _pb_counts_key: String in counts:")
	// A map entry is a submessage of key = 1, value = 2.
	require.Contains(t, source, "_pb_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))")
	require.Contains(t, source, "_pb_counts_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_VARINT)))")
	require.Contains(t, source, "counts[_pb_counts_key] = _pb_counts_value")
}

// A oneof becomes a tagged union, which makes "two cases set at once"
// unrepresentable instead of an invariant the binding must maintain.
func TestGenerateOneofAsTaggedUnion(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name: "payload",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "text", Number: 1},
				{FieldType: "int32", Name: "amount", Number: 2},
			},
		}},
	}}, nil))

	unionSource := files["cafecito/game/v1/PlayerPayloadCase.pb.fs"]
	require.Contains(t, unionSource, "enum_name PlayerPayloadCase:\n\tText(text: String)\n\tAmount(amount: int)\n")
	// Tagged-union cases are ordinal; an explicit value is a parse error.
	require.NotContains(t, unionSource, " = 0")

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "var payload: PlayerPayloadCase? = null")
	require.Contains(t, source, "match payload:")
	require.Contains(t, source, "PlayerPayloadCase.Text(var _pb_payload_text):")
	require.Contains(t, source, "payload = PlayerPayloadCase.Amount(_pb_payload_amount_read.value)")
	// No separate per-case fields and no which_case discriminator.
	require.NotContains(t, source, "var text: String")
	require.NotContains(t, source, "which_case")
}

// A oneof union must be file-level: a tagged-union case pattern resolves only
// one level deep, so a nested union could not be matched by any consumer.
func TestOneofUnionIsFileLevel(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name:   "payload",
			Fields: []*protoast.Field{{FieldType: "string", Name: "text", Number: 1}},
		}},
	}}, nil))

	require.Contains(t, files, "cafecito/game/v1/PlayerPayloadCase.pb.fs")
	require.NotContains(t, files["cafecito/game/v1/Player.pb.fs"], "enum PlayerPayloadCase:")
}

// Nested types keep proto's scoping as inner class/enum members.
func TestGenerateNestedTypes(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Badge", Name: "badge", Number: 1}, {FieldType: "Tier", Name: "tier", Number: 2}},
		NestedMessages: []*protoast.Message{{
			Name:   "Badge",
			Fields: []*protoast.Field{{FieldType: "string", Name: "code", Number: 1}},
		}},
		NestedEnums: []*protoast.Enum{{
			Name: "Tier",
			Values: []*protoast.EnumValue{
				{Name: "TIER_UNSPECIFIED", Number: 0},
				{Name: "TIER_GOLD", Number: 1},
			},
		}},
	}}, nil))

	require.Len(t, files, 1)
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "final class Badge extends RefCounted uses Message:")
	require.Contains(t, source, "enum Tier:")
	require.Contains(t, source, "var badge: Badge? = null")
	require.Contains(t, source, "var tier: Tier = Tier.TIER_UNSPECIFIED")
	// An inner type keeps the plain forms, not the file-level _name variants.
	require.NotContains(t, source, "class_name Badge")
	require.NotContains(t, source, "enum_name Tier")
}

// The parser only sets IsEnum for cross-file references, so a same-file enum
// must still be recognised or it is emitted as a message.
func TestSameFileEnumIsNotTreatedAsMessage(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "status", Number: 1}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.NotContains(t, source, "status.to_bytes()")
	require.NotContains(t, source, "PlayerStatus.new()")
	require.Contains(t, source, "status.to_wire()")
}

func TestGenerateUnsupportedWireScalarsReturnsError(t *testing.T) {
	for _, scalar := range []string{"float", "double", "fixed32", "fixed64", "sfixed32", "sfixed64", "sint32", "sint64"} {
		t.Run(scalar, func(t *testing.T) {
			_, err := Generate(namespacedFile([]*protoast.Message{{
				Name:   "Player",
				Fields: []*protoast.Field{{FieldType: scalar, Name: "score", Number: 1}},
			}}, nil), "player.proto", nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported scalar type "+scalar+" for wire generation")
		})
	}
}

// An unsupported scalar must be rejected wherever it appears, not only as a
// top-level field, or the binding silently drops it.
func TestUnsupportedScalarsRejectedInEveryPosition(t *testing.T) {
	cases := map[string]*protoast.Message{
		"nested": {
			Name: "Player",
			NestedMessages: []*protoast.Message{{
				Name:   "Inner",
				Fields: []*protoast.Field{{FieldType: "float", Name: "ratio", Number: 1}},
			}},
		},
		"oneof": {
			Name: "Player",
			Oneofs: []*protoast.Oneof{{
				Name:   "payload",
				Fields: []*protoast.Field{{FieldType: "double", Name: "ratio", Number: 1}},
			}},
		},
		"map value": {
			Name: "Player",
			Maps: []*protoast.MapField{{KeyType: "string", ValueType: "sint32", Name: "ratios", Number: 1}},
		},
	}

	for name, message := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Generate(namespacedFile([]*protoast.Message{message}, nil), "player.proto", nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported scalar type")
		})
	}
}

func TestGeneratePrefersSchemaDocs(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Doc:  []string{"Schema-authored player docs.", "Shown in generated API docs."},
			Name: "Player",
			Fields: []*protoast.Field{{
				Doc:       []string{"Display name.", "Stored player name."},
				FieldType: "string",
				Name:      "name",
				Number:    1,
			}},
		}},
		[]*protoast.Enum{{
			Doc:  []string{"Schema-authored status docs."},
			Name: "PlayerStatus",
			Values: []*protoast.EnumValue{{
				Doc:    []string{"Unknown status."},
				Name:   "PLAYER_STATUS_UNSPECIFIED",
				Number: 0,
			}},
		}},
	))

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, messageSource, "## Schema-authored player docs.\n## Shown in generated API docs.\nfinal class_name Player extends RefCounted")
	require.Contains(t, messageSource, "## Display name.\n## Stored player name.\nvar name: String = \"\"")
	require.NotContains(t, messageSource, "Generated protobuf message binding for Player.")

	enumSource := files["cafecito/game/v1/PlayerStatus.pb.fs"]
	require.Contains(t, enumSource, "## Schema-authored status docs.\nenum_name PlayerStatus")
	require.Contains(t, enumSource, "\t## Unknown status.\n\tPLAYER_STATUS_UNSPECIFIED = 0\n")
	require.NotContains(t, enumSource, "Generated protobuf enum binding for PlayerStatus.")
}

// A field name that is a Foundry Script keyword cannot be emitted verbatim, and
// one that matches an emitter local must not be able to capture it.
func TestGenerateEscapesCollidingFieldNames(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "int32", Name: "var", Number: 1},
		{FieldType: "int32", Name: "offset", Number: 2},
		{FieldType: "string", Name: "data", Number: 3},
		{FieldType: "bytes", Name: "result", Number: 4},
	})

	require.Contains(t, source, "var var_: int = 0")
	// A name that is merely awkward stays readable; only keywords are renamed.
	require.Contains(t, source, "var offset: int = 0")
	require.Contains(t, source, "var data: String = \"\"")
	require.Contains(t, source, "var result: PackedByteArray = PackedByteArray()")

	// Every emitter-introduced name is underscore-prefixed, so none of the
	// fields above is shadowed by the code that decodes it.
	require.Contains(t, source, "func merge_from_bytes(_pb_data: PackedByteArray) -> ProtobufError:")
	require.Contains(t, source, "var _pb_offset: int = 0")
	require.Contains(t, source, "offset = _pb_offset_read.value")
	require.Contains(t, source, "data = _pb_data_read.value")
	require.NotContains(t, source, "var offset: int = 0\n\twhile")
}

// protoc accepts a leading underscore on a field name, so rejecting those would
// refuse schemas that build everywhere else. Only the emitter's own prefix is
// reserved, and it is narrow enough that no real schema reaches it.
func TestGenerateAcceptsLeadingUnderscoreFieldNames(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "int32", Name: "_private", Number: 1},
		{FieldType: "int32", Name: "_offset", Number: 2},
	})
	require.Contains(t, source, "var _private: int = 0")
	require.Contains(t, source, "var _offset: int = 0")
	// The cursor still has a name of its own.
	require.Contains(t, source, "var _pb_offset: int = 0")
	require.Contains(t, source, "_offset = _pb__offset_read.value")
}

// The emitter's prefix is the one spelling a schema may not use, for fields and
// oneofs alike: a oneof named _pb_result would be shadowed by the serializer's
// own buffer.
func TestGenerateRejectsTheGeneratedPrefix(t *testing.T) {
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "int32", Name: "_pb_offset", Number: 1}},
	}}, nil), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "the _pb_ prefix is reserved")

	_, err = Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name:   "_pb_result",
			Fields: []*protoast.Field{{FieldType: "int32", Name: "amount", Number: 1}},
		}},
	}}, nil), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "the _pb_ prefix is reserved")
}

// A dependency whose namespace would not parse cannot be imported, so a
// reference to it is reported rather than emitted as `import bad..ns`.
func TestReferenceToADependencyWithAnInvalidNamespaceIsRejected(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Options: map[string]any{"(foundrytools.namespace)": "bad..ns"},
		Messages: []*protoast.Message{{
			Name:   "Item",
			Fields: []*protoast.Field{{FieldType: "string", Name: "sku", Number: 1}},
		}},
	}
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Item", Name: "held", Number: 1, SourceFile: "inventory.proto"}},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no usable namespace")
}

// The union is hoisted to file level, so a payload declared in another message
// has to be named by its scoped reference rather than lexically.
func TestOneofPayloadUsesScopedReference(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{
		{
			Name: "Player",
			Oneofs: []*protoast.Oneof{{
				Name:   "payload",
				Fields: []*protoast.Field{{FieldType: "Slot.Detail", Name: "detail", Number: 1}},
			}},
		},
		{
			Name: "Slot",
			NestedMessages: []*protoast.Message{{
				Name:   "Detail",
				Fields: []*protoast.Field{{FieldType: "string", Name: "note", Number: 1}},
			}},
		},
	}, nil))

	require.Contains(t, files["cafecito/game/v1/PlayerPayloadCase.pb.fs"], "\tDetail(detail: Slot.Detail)\n")
	// Inside the class the lexical spelling is still what Foundry resolves.
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var _pb_payload_detail_message: Slot.Detail = Slot.Detail.new()")
}

// The hoisted union cannot name a type nested in the class that owns the oneof:
// the class needs the union to declare its member and the union needs the class
// to reach the nested type, and Foundry cannot break that cycle for a class that
// conforms to a trait. Failing loudly beats emitting a file that will not parse.
func TestOneofRejectsPayloadNestedInDeclaringMessage(t *testing.T) {
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		NestedMessages: []*protoast.Message{{
			Name:   "Badge",
			Fields: []*protoast.Field{{FieldType: "string", Name: "code", Number: 1}},
		}},
		Oneofs: []*protoast.Oneof{{
			Name:   "payload",
			Fields: []*protoast.Field{{FieldType: "Badge", Name: "badge", Number: 1}},
		}},
	}}, nil), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot carry a type nested in the message that declares it")
}

// A oneof may still carry the top-level message it is declared in; only a type
// nested inside it closes the cycle.
func TestOneofAcceptsSiblingMessagePayload(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{
		{
			Name: "Player",
			Oneofs: []*protoast.Oneof{{
				Name:   "payload",
				Fields: []*protoast.Field{{FieldType: "Slot", Name: "slot", Number: 1}},
			}},
		},
		slotMessage(),
	}, nil))
	require.Contains(t, files["cafecito/game/v1/PlayerPayloadCase.pb.fs"], "\tSlot(slot: Slot)\n")
}

// An allow_alias enum declares one number twice; the second match arm would be
// unreachable.
func TestEnumFromWireDedupesAliasedNumbers(t *testing.T) {
	files := generate(t, namespacedFile(nil, []*protoast.Enum{{
		Name: "Tier",
		Values: []*protoast.EnumValue{
			{Name: "TIER_UNSPECIFIED", Number: 0},
			{Name: "TIER_GOLD", Number: 1},
			{Name: "TIER_PREMIUM", Number: 1},
		},
	}}))
	source := files["cafecito/game/v1/Tier.pb.fs"]

	require.Contains(t, source, "\t\t\treturn Tier.TIER_GOLD")
	require.NotContains(t, source, "return Tier.TIER_PREMIUM")
}

// Records go out in field-number order, oneofs included, so a hexdump reads
// against the schema.
func TestSerializeWritesFieldsInNumberOrder(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "string", Name: "name", Number: 1},
			{FieldType: "int32", Name: "level", Number: 9},
		},
		Oneofs: []*protoast.Oneof{{
			Name:   "payload",
			Fields: []*protoast.Field{{FieldType: "int32", Name: "amount", Number: 5}},
		}},
	}}, nil))
	source := files["cafecito/game/v1/Player.pb.fs"]

	nameAt := strings.Index(source, "if name != \"\":")
	oneofAt := strings.Index(source, "match payload:")
	levelAt := strings.Index(source, "if level != 0:")
	require.Positive(t, nameAt)
	require.Greater(t, oneofAt, nameAt)
	require.Greater(t, levelAt, oneofAt)
}

// A type from another proto file resolves to its own namespace, which the
// generated file has to import and, for an enum, take its default from.
func TestCrossFileReferencesImportTheirNamespace(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Enums: []*protoast.Enum{{
			Name: "Rarity",
			Values: []*protoast.EnumValue{
				{Name: "RARITY_UNSPECIFIED", Number: 0},
				{Name: "RARITY_COMMON", Number: 1},
			},
		}},
		Messages: []*protoast.Message{{
			Name:   "Item",
			Fields: []*protoast.Field{{FieldType: "string", Name: "sku", Number: 1}},
		}},
	}
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Item", Name: "held", Number: 1, SourceFile: "inventory.proto"},
			{FieldType: "Rarity", Name: "rarity", Number: 2, IsEnum: true, SourceFile: "inventory.proto"},
		},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "import cafecito.inventory.v1")
	require.Contains(t, source, "var held: Item? = null")
	// The zero case comes from the imported declaration; guessing it is what
	// used to emit an unparseable `Rarity.`.
	require.Contains(t, source, "var rarity: Rarity = Rarity.RARITY_UNSPECIFIED")
	// Only the message being generated is emitted; the import is someone else's.
	require.NotContains(t, files, "cafecito/inventory/v1/Item.pb.fs")
}

// The parser rewrites a cross-file reference to its short name, so a schema
// that declares Slot locally and also imports one leaves two declarations
// competing for the spelling. Only the field's source file says which was meant,
// and only the namespace-qualified reference is unambiguous in the output.
func TestImportedTypeIsNotCapturedByALocalOfTheSameName(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name:   "Slot",
			Fields: []*protoast.Field{{FieldType: "string", Name: "sku", Number: 1}},
		}},
		Enums: []*protoast.Enum{{
			Name: "Tier",
			Values: []*protoast.EnumValue{
				{Name: "TIER_NONE", Number: 0},
				{Name: "TIER_GOLD", Number: 1},
			},
		}},
	}
	files, err := Generate(namespacedFile([]*protoast.Message{
		{
			Name: "Player",
			Fields: []*protoast.Field{
				{FieldType: "Slot", Name: "held", Number: 1, SourceFile: "inventory.proto"},
				{FieldType: "Slot", Name: "mine", Number: 2},
				{FieldType: "Tier", Name: "tier", Number: 3, IsEnum: true, SourceFile: "inventory.proto"},
			},
		},
		slotMessage(),
	}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "import cafecito.inventory.v1")
	require.Contains(t, source, "var held: cafecito.inventory.v1.Slot? = null")
	require.Contains(t, source, "if held is cafecito.inventory.v1.Slot:")
	// The local declaration keeps the short spelling.
	require.Contains(t, source, "var mine: Slot? = null")
	// The enum's default comes from the imported declaration, qualified the
	// same way even though only the message name actually collides, because
	// the collision is decided on the outermost segment.
	require.Contains(t, source, "var tier: Tier = Tier.TIER_NONE")
}

// An unqualified imported reference stays short when nothing shadows it.
func TestImportedTypeKeepsTheShortSpellingWhenUnambiguous(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name:   "Item",
			Fields: []*protoast.Field{{FieldType: "string", Name: "sku", Number: 1}},
		}},
	}
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Item", Name: "held", Number: 1, SourceFile: "inventory.proto"}},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.NoError(t, err)

	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var held: Item? = null")
}

// Escaping a keyword appends an underscore, which can land on a name the schema
// already uses. Two fields mapping to one member would conflate them.
func TestGenerateRejectsCollidingMemberNames(t *testing.T) {
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "int32", Name: "var", Number: 1},
			{FieldType: "string", Name: "var_", Number: 2},
		},
	}}, nil), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "both map to the member var_")
}

// A oneof member and a plain field can collide the same way.
func TestGenerateRejectsOneofCollidingWithAField(t *testing.T) {
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "int32", Name: "payload", Number: 1}},
		Oneofs: []*protoast.Oneof{{
			Name:   "payload",
			Fields: []*protoast.Field{{FieldType: "string", Name: "text", Number: 2}},
		}},
	}}, nil), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "both map to the member payload")
}

// Every record in the shared buffer carries a field number this schema has no
// member for, so it competes with nothing and trails the live fields. A value
// that does share a field number with a member is kept by that member instead.
func TestSerializeWritesTheSharedBufferLast(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})

	nameAt := strings.Index(source, "if name != \"\":")
	bufferAt := strings.Index(source, "_pb_result.append_array(_pb_unknown_fields)")
	require.Positive(t, nameAt)
	require.Greater(t, bufferAt, nameAt)
}

// proto3 enums are open, so a number this schema has no case for has to survive
// a re-encode. It cannot go in the shared unknown-field buffer: it carries a
// field number the schema does know, and protobuf takes the last record for a
// singular field, so a copy on either side of the member would decide the value
// rather than stand in for it. It is kept per field and written in the field's
// own position instead, and the member's setter drops it the moment anything
// assigns a value the schema can represent.
func TestUnknownEnumValueIsRetainedPerFieldAndSuperseded(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "status", Number: 1}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "var status: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED:\n"+
		"\tset(_pb_value):\n"+
		"\t\t_pb_status_unknown = PackedByteArray()\n"+
		"\t\tstatus = _pb_value\n")
	// The retained value takes the field's own position, and the two are
	// mutually exclusive rather than both written.
	require.Contains(t, source, "\tif _pb_status_unknown.size() > 0:\n"+
		"\t\t_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))\n"+
		"\t\t_pb_result.append_array(_pb_status_unknown)\n"+
		"\telif status != PlayerStatus.PLAYER_STATUS_UNSPECIFIED:\n")
	// Retaining clears the member, which runs the setter and so discards any
	// value retained for an earlier record of the same field.
	require.Contains(t, source, "\t\t\t\t\tstatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED\n"+
		"\t\t\t\t\t_pb_status_unknown = _pb_data.slice(_pb_offset, _pb_status_read.offset)")
}

// A repeated enum has no single member to attach raw bytes to, and the shared
// buffer is emitted as one run, so moving an element into it would reorder the
// sequence. The value is folded onto the default instead, which loses it
// visibly rather than silently permuting data.
func TestRepeatedEnumFoldsUnrecognizedValues(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "history", Number: 1, Repeated: true}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "history.append(PlayerStatus.PLAYER_STATUS_UNSPECIFIED)")
	require.NotContains(t, source, "_pb_history_unknown")
	require.NotContains(t, source, "_pb_unknown_fields.append_array(_pb_data.slice(")
}

// A map-valued enum cannot be retained either: an entry moved into the shared
// buffer changes which of two records for the same key protobuf takes as the
// last one, so duplicate-key precedence would flip.
func TestMapValuedEnumFoldsUnrecognizedValues(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name: "Player",
			Maps: []*protoast.MapField{{
				KeyType:     "string",
				ValueType:   "PlayerStatus",
				ValueIsEnum: true,
				Name:        "seen",
				Number:      1,
			}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "_pb_seen_value = PlayerStatus.PLAYER_STATUS_UNSPECIFIED")
	require.Contains(t, source, "seen[_pb_seen_key] = _pb_seen_value")
	require.NotContains(t, source, "_pb_seen_unknown")
}

// A member and its setter parameter share a scope, so a field named `value`
// with a parameter named `value` would leave the member unwritten -- silently,
// with no diagnostic from the analyzer.
func TestSetterParameterCannotCollideWithTheField(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "value", Number: 1}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "\tset(_pb_value):\n\t\t_pb_value_unknown = PackedByteArray()\n\t\tvalue = _pb_value\n")
	require.NotContains(t, source, "value = value")
}

// A retained-value companion is a member too, and its name is built by joining
// names with underscores, so two different declarations can reach it.
func TestGenerateRejectsCollidingRetentionMembers(t *testing.T) {
	_, err := Generate(namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "pick_kind", Number: 1}},
			Oneofs: []*protoast.Oneof{{
				Name:   "pick",
				Fields: []*protoast.Field{{FieldType: "PlayerStatus", Name: "kind", Number: 2}},
			}},
		}},
		[]*protoast.Enum{{
			Name:   "PlayerStatus",
			Values: []*protoast.EnumValue{{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0}},
		}},
	), "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "both map to the member _pb_pick_kind_unknown")
}

// Two imported namespaces declaring the same short name make it ambiguous even
// though nothing local competes. Foundry rejects such a reference outright, so
// both have to be named by their namespace.
func TestSameNameFromTwoImportedNamespacesIsQualified(t *testing.T) {
	first := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name:   "Item",
			Fields: []*protoast.Field{{FieldType: "string", Name: "sku", Number: 1}},
		}},
	}
	second := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.catalog.v1",
		Messages: []*protoast.Message{{
			Name:   "Item",
			Fields: []*protoast.Field{{FieldType: "string", Name: "label", Number: 1}},
		}},
	}
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Item", Name: "held", Number: 1, SourceFile: "inventory.proto"},
			{FieldType: "Item", Name: "listed", Number: 2, SourceFile: "catalog.proto"},
		},
	}}, nil), "player.proto", []FileEntry{
		{File: first, Filename: "inventory.proto"},
		{File: second, Filename: "catalog.proto"},
	})
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "var held: cafecito.inventory.v1.Item? = null")
	require.Contains(t, source, "var listed: cafecito.catalog.v1.Item? = null")
	require.Contains(t, source, "import cafecito.catalog.v1")
	require.Contains(t, source, "import cafecito.inventory.v1")
}

// Every generated file imports foundry.proto, so a schema type sharing a name
// with one of its exports would make that name ambiguous and break the runtime
// calls the binding makes through it.
func TestSchemaTypesDoNotCollideWithRuntimeNames(t *testing.T) {
	require.Equal(t, "Wire_", TypeName("Wire"))
	require.Equal(t, "Message_", TypeName("message"))
	require.Equal(t, "ProtobufError_", TypeName("ProtobufError"))
	require.Equal(t, "VarintRead_", TypeName("varint_read"))

	files := generate(t, namespacedFile([]*protoast.Message{
		{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "Wire", Name: "wire", Number: 1}},
		},
		{
			Name:   "Wire",
			Fields: []*protoast.Field{{FieldType: "string", Name: "kind", Number: 1}},
		},
	}, nil))

	// The declaration and the reference are renamed together, so the runtime's
	// own Wire keeps answering to the unqualified name the calls use.
	require.Contains(t, files, "cafecito/game/v1/Wire_.pb.fs")
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var wire: Wire_? = null")
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "Wire.encode_varint(")
}

// A dependency with neither a package nor a namespace option has no name its
// types can be reached by, so referencing one is reported rather than emitted
// as a reference that cannot resolve.
func TestReferenceToAnUnnamespacedDependencyIsRejected(t *testing.T) {
	imported := &protoast.ProtoFile{
		Syntax: "proto3",
		Messages: []*protoast.Message{{
			Name:   "Item",
			Fields: []*protoast.Field{{FieldType: "string", Name: "sku", Number: 1}},
		}},
	}
	_, err := Generate(namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Item", Name: "held", Number: 1, SourceFile: "loose.proto"}},
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "loose.proto"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no usable namespace")
}
