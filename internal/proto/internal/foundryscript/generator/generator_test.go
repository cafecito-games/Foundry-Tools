package fsgenerator

import (
	"strconv"
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
	}), "player.proto", nil, Options{})

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
	files, err := Generate(file, "player.proto", nil, Options{})
	require.NoError(t, err)
	return files
}

func playerSource(t *testing.T, fields []*protoast.Field, alsoDeclared ...*protoast.Message) string {
	t.Helper()
	messages := append([]*protoast.Message{{Name: "Player", Fields: fields}}, alsoDeclared...)
	files := generate(t, namespacedFile(messages, nil))
	return files["cafecito/game/v1/Player.pb.fs"]
}

func TestMessagesExposeExactProtobufIdentityAndConstruction(t *testing.T) {
	file := namespacedFile([]*protoast.Message{{
		Name: "Player",
		NestedMessages: []*protoast.Message{{
			Name: "Badge",
		}},
	}}, nil)
	files := generate(t, file)
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, `static func create_message() -> Player:
	return Player.new()

static func protobuf_type_name() -> String:
	return "cafecito.game.v1.Player"

func type_name() -> String:
	return Player.protobuf_type_name()`)
	require.Contains(t, source, `	static func create_message() -> Badge:
		return Badge.new()

	static func protobuf_type_name() -> String:
		return "cafecito.game.v1.Player.Badge"

	func type_name() -> String:
		return Badge.protobuf_type_name()`)

	file.Options = map[string]any{
		NamespaceOptionKey:  "custom.bindings",
		typePrefixOptionKey: "Game",
	}
	files = generate(t, file)
	source = files["custom/bindings/GamePlayer.pb.fs"]
	require.Contains(t, source, `static func create_message() -> GamePlayer:`)
	require.Contains(t, source, `return "cafecito.game.v1.Player"`)
	require.Contains(t, source, `return "cafecito.game.v1.Player.Badge"`)
	require.NotContains(t, source, `return "custom.bindings`)
	require.NotContains(t, source, `return "cafecito.game.v1.Game`)
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

func TestGenerateCanonicalizesAbsoluteLocalReferencesBeforeApplyingPrefix(t *testing.T) {
	files := generate(t, prefixedFile("Game", []*protoast.Message{
		{
			Name: "Outer",
			Fields: []*protoast.Field{
				{FieldType: "Inner", Name: "lexical", Number: 1},
				{FieldType: "Outer.Inner", Name: "qualified", Number: 2},
				{
					FieldType:    "Inner",
					FullTypePath: "cafecito.game.v1.Outer.Inner",
					SourceFile:   "player.proto",
					Name:         "descriptor_lexical",
					Number:       3,
				},
			},
			NestedMessages: []*protoast.Message{{Name: "Inner"}},
		},
		{
			Name: "Holder",
			Fields: []*protoast.Field{
				{
					FieldType: ".cafecito.game.v1.Outer.Inner",
					Name:      "absolute",
					Number:    1,
				},
				{
					FieldType: "cafecito.game.v1.Outer.Inner",
					Name:      "package_qualified",
					Number:    2,
				},
				{
					FieldType: "cafecito.game.v1.Node",
					Name:      "matching_relative",
					Number:    3,
				},
			},
		},
		{
			Name: "cafecito",
			NestedMessages: []*protoast.Message{{
				Name: "game",
				NestedMessages: []*protoast.Message{{
					Name: "v1",
					NestedMessages: []*protoast.Message{{
						Name: "Node",
					}},
				}},
			}},
		},
	}, nil))

	outer := files["cafecito/game/v1/GameOuter.pb.fs"]
	require.Contains(t, outer, "var lexical: GameInner? = null")
	require.Contains(t, outer, "var qualified: GameOuter.GameInner? = null")
	require.Contains(t, outer, "var descriptor_lexical: GameInner? = null")

	holder := files["cafecito/game/v1/GameHolder.pb.fs"]
	require.Contains(t, holder, "var absolute: GameOuter.GameInner? = null")
	require.Contains(t, holder, "var package_qualified: GameOuter.GameInner? = null")
	require.Contains(t, holder,
		"var matching_relative: GameCafecito.GameGame.GameV1.GameNode? = null")
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
	), "player.proto", nil, Options{})

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
	), "player.proto", nil, Options{})

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
	), "player.proto", nil, Options{})

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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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
	}}, Options{})

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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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

// The JSON surface names the engine's builtins unqualified, so a message that
// claims one of those spellings has to be renamed rather than emitted verbatim.
func TestMessageNamedAfterAnEngineJSONBuiltinIsEscaped(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name:   "JsonNode",
		Fields: []*protoast.Field{{FieldType: "string", Name: "label", Number: 1}},
	}}, nil))

	source := files["cafecito/game/v1/JsonNode_.pb.fs"]
	require.NotEmpty(t, source, "expected the escaped path, got %v", files)
	require.Contains(t, source, "final class_name JsonNode_ extends RefCounted uses Message\n")
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

// A field's proto default is its enum's zero-valued case; when that case
// itself is spelled the same as a hosted function, the default expression has
// to follow the escaped case name too, or it would reference a declaration
// that does not exist.
func TestGenerateEscapesEnumZeroValueInFieldDefault(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name:   "Player",
			Fields: []*protoast.Field{{FieldType: "Transport", Name: "transport", Number: 1}},
		}},
		[]*protoast.Enum{{
			Name: "Transport",
			Values: []*protoast.EnumValue{
				{Name: "to_wire", Number: 0},
				{Name: "TRANSPORT_ALT", Number: 1},
			},
		}},
	))

	enumSource := files["cafecito/game/v1/Transport.pb.fs"]
	require.Contains(t, enumSource, "\tto_wire_ = 0\n")

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, messageSource, "var transport: Transport = Transport.to_wire_")
	require.NotContains(t, messageSource, "Transport.to_wire\n")
	require.Contains(t, messageSource, "transport = Transport.to_wire_")
}

// An enum value spelled the same as one of the functions the enum hosts
// beside it -- to_wire, from_wire, to_json_name, from_json_name -- would
// otherwise declare that name twice: once as a case, once as a function. The
// case is escaped exactly as a colliding message member is, and every
// reference to it -- the declaration, from_wire's match arm, and
// from_json_name's match arm -- follows the escaped spelling. The JSON name
// itself is unaffected: canonical JSON still reads and writes the value's
// original proto spelling.
func TestGenerateEscapesEnumValueNamedForAHostedFunction(t *testing.T) {
	reservedNames := []string{"to_wire", "from_wire", "to_json_name", "from_json_name"}

	for _, reserved := range reservedNames {
		t.Run(reserved, func(t *testing.T) {
			files := generateJSON(t, namespacedFile(nil, []*protoast.Enum{{
				Name: "Transport",
				Values: []*protoast.EnumValue{
					{Name: "TRANSPORT_UNSPECIFIED", Number: 0},
					{Name: reserved, Number: 1},
				},
			}}), "transport.proto")

			enumSource := files["cafecito/game/v1/Transport.pb.fs"]
			escaped := reserved + "_"

			// The case is declared under the escaped spelling, not the raw one
			// that collides with the function.
			require.Contains(t, enumSource, "\t"+escaped+" = 1\n")
			require.NotContains(t, enumSource, "\t"+reserved+" = 1\n")

			// The functions the collision is against are still emitted intact.
			require.Contains(t, enumSource, "func to_wire() -> int:")
			require.Contains(t, enumSource, "static func from_wire(value: int) -> Self?:")
			require.Contains(t, enumSource, "func to_json_name() -> String:")
			require.Contains(t, enumSource, "static func from_json_name(name: String) -> Self?:")

			// from_wire resolves to the escaped case.
			require.Contains(t, enumSource, "return Transport."+escaped)

			// to_json_name still writes the value's original proto spelling.
			require.Contains(t, enumSource, "return "+strconv.Quote(reserved))
			// from_json_name still reads that same original spelling back, into
			// the escaped case.
			require.Contains(t, enumSource,
				"\t\t\t"+strconv.Quote(reserved)+":\n\t\t\t\treturn Transport."+escaped)
		})
	}
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

// Each of the eight non-varint scalars frames its own way. The generated
// source has to name that framing in the tag and reach for the codec that
// matches it, or the bytes are wrong in a way only a foreign decoder notices.
func TestGenerateFixedWidthScalars(t *testing.T) {
	cases := map[string]struct {
		declared string
		wireType string
		encode   string
		carrier  string
		read     string
	}{
		"fixed32":  {"var score: int = 0", "Wire.WIRE_32BIT", "Wire.encode_fixed32(score)", "FixedRead", "Wire.read_fixed32"},
		"sfixed32": {"var score: int = 0", "Wire.WIRE_32BIT", "Wire.encode_fixed32(score)", "FixedRead", "Wire.read_sfixed32"},
		"float":    {"var score: float = 0.0", "Wire.WIRE_32BIT", "Wire.encode_float(score)", "FloatRead", "Wire.read_float"},
		"fixed64":  {"var score: int = 0", "Wire.WIRE_64BIT", "Wire.encode_fixed64(score)", "FixedRead", "Wire.read_fixed64"},
		"sfixed64": {"var score: int = 0", "Wire.WIRE_64BIT", "Wire.encode_fixed64(score)", "FixedRead", "Wire.read_fixed64"},
		"double":   {"var score: float = 0.0", "Wire.WIRE_64BIT", "Wire.encode_double(score)", "FloatRead", "Wire.read_double"},
	}

	for scalar, want := range cases {
		t.Run(scalar, func(t *testing.T) {
			source := playerSource(t, []*protoast.Field{{FieldType: scalar, Name: "score", Number: 1}})

			require.Contains(t, source, want.declared)
			require.Contains(t, source, "Wire.make_tag(1, "+want.wireType+")")
			require.Contains(t, source, "_pb_result.append_array("+want.encode+")")
			require.Contains(t, source, "if _pb_wire_type != "+want.wireType+":")
			require.Contains(t, source, "var _pb_score_read: "+want.carrier+" = "+want.read+"(_pb_data, _pb_offset)")
			require.Contains(t, source, "score = _pb_score_read.value")

			// A fixed-width value is never framed as a varint.
			require.NotContains(t, source, "Wire.encode_varint(score)")
		})
	}
}

// sint32 and sint64 stay varints, but zig-zag first, so a small negative costs
// one byte rather than ten.
func TestGenerateZigZagScalars(t *testing.T) {
	for _, scalar := range []string{"sint32", "sint64"} {
		t.Run(scalar, func(t *testing.T) {
			source := playerSource(t, []*protoast.Field{{FieldType: scalar, Name: "score", Number: 1}})

			require.Contains(t, source, "var score: int = 0")
			require.Contains(t, source, "Wire.make_tag(1, Wire.WIRE_VARINT)")
			require.Contains(t, source, "_pb_result.append_array(Wire.encode_"+scalar+"(score))")
			require.Contains(t, source, "var _pb_score_read: VarintRead = Wire.read_"+scalar+"(_pb_data, _pb_offset)")
			require.Contains(t, source, "score = _pb_score_read.value")

			// The plain varint codec would put a negative on the wire as ten bytes.
			require.NotContains(t, source, "Wire.encode_varint(score)")
		})
	}
}

// proto3 packs every numeric scalar by default, fixed-width ones included, so
// the tag carries the length-delimited wire type and the elements do not.
func TestGenerateRepeatedFixedWidthScalarsPack(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "double", Name: "ratios", Number: 1, Repeated: true},
		{FieldType: "sint32", Name: "deltas", Number: 2, Repeated: true},
	})

	require.Contains(t, source, "var ratios: Array[float] = []")
	require.Contains(t, source, "Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)")
	require.Contains(t, source, "for _pb_ratios_item: float in ratios:")
	require.Contains(t, source, "_pb_ratios_data.append_array(Wire.encode_double(_pb_ratios_item))")

	require.Contains(t, source, "var deltas: Array[int] = []")
	require.Contains(t, source, "_pb_deltas_data.append_array(Wire.encode_sint32(_pb_deltas_item))")

	// A packed field must still decode the unpacked encoding, which for a
	// fixed-width element arrives under its own wire type rather than a varint.
	require.Contains(t, source, "elif _pb_wire_type == Wire.WIRE_64BIT:")
	require.Contains(t, source, "elif _pb_wire_type == Wire.WIRE_VARINT:")
}

// A packed run declares its length, and an element must not read past it. The
// element readers bound themselves against the whole buffer, so without a
// check against the run's own end a payload whose length is not a whole number
// of elements would take bytes from the field after it and accept the message.
func TestGeneratePackedRunRejectsElementsCrossingItsEnd(t *testing.T) {
	for _, scalar := range []string{"sfixed32", "double", "sint64", "int32"} {
		t.Run(scalar, func(t *testing.T) {
			source := playerSource(t, []*protoast.Field{
				{FieldType: scalar, Name: "values", Number: 1, Repeated: true},
			})

			require.Contains(t, source, "if _pb_values_packed.offset > _pb_values_end:")
			require.Contains(t, source, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH")
		})
	}
}

// `[packed = false]` asks for one tagged record per element, and the encoder
// has to honour it: a schema author who writes it is usually encoding for a
// peer that reads no other form.
func TestGeneratePackedOptionGovernsTheEncoding(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "int32", Name: "packed_int32", Number: 1, Repeated: true, Options: map[string]any{"packed": true}},
		{FieldType: "int32", Name: "unpacked_int32", Number: 2, Repeated: true, Options: map[string]any{"packed": false}},
		{FieldType: "int32", Name: "default_int32", Number: 3, Repeated: true},
	})

	require.Contains(t, source, "var _pb_packed_int32_data: PackedByteArray = PackedByteArray()")
	require.Contains(t, source, "Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)")
	require.Contains(t, source, "var _pb_default_int32_data: PackedByteArray = PackedByteArray()")
	require.Contains(t, source, "Wire.make_tag(3, Wire.WIRE_LENGTH_DELIMITED)")

	// The unpacked field writes each element under its own varint tag, with no
	// run buffer to collect them into.
	require.Contains(t, source, "for _pb_unpacked_int32_item: int in unpacked_int32:")
	require.Contains(t, source, "Wire.make_tag(2, Wire.WIRE_VARINT)")
	require.NotContains(t, source, "var _pb_unpacked_int32_data: PackedByteArray = PackedByteArray()")
	require.NotContains(t, source, "Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)")
}

// The option binds the encoder alone. A decoder must take either encoding for
// any packable repeated field, so `[packed = false]` keeps the permissive read.
func TestGeneratePackedOptionLeavesDecodingPermissive(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "int32", Name: "unpacked_int32", Number: 1, Repeated: true, Options: map[string]any{"packed": false}},
	})

	require.Contains(t, source, "if _pb_wire_type == Wire.WIRE_LENGTH_DELIMITED:")
	require.Contains(t, source, "if _pb_unpacked_int32_packed.offset > _pb_unpacked_int32_end:")
	require.Contains(t, source, "elif _pb_wire_type == Wire.WIRE_VARINT:")
}

// A length-delimited element has no packed form, so `[packed = true]` on one
// is a schema error rather than something to silently drop.
func TestGeneratePackedTrueOnUnpackableFieldIsRejected(t *testing.T) {
	for _, fieldType := range []string{"string", "bytes", "Slot"} {
		t.Run(fieldType, func(t *testing.T) {
			_, err := Generate(namespacedFile([]*protoast.Message{
				{Name: "Player", Fields: []*protoast.Field{{
					FieldType: fieldType,
					Name:      "values",
					Number:    1,
					Repeated:  true,
					Options:   map[string]any{"packed": true},
				}}},
				slotMessage(),
			}, nil), "player.proto", nil, Options{})

			require.Error(t, err)
			require.Contains(t, err.Error(), "field Player.values")
			require.Contains(t, err.Error(), "[packed = true] is only valid on a repeated numeric or enum field")
		})
	}
}

// `[packed = false]` on a length-delimited element only restates what the wire
// format already requires, so it is accepted and changes nothing.
func TestGeneratePackedFalseOnUnpackableFieldIsAccepted(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "tags", Number: 1, Repeated: true, Options: map[string]any{"packed": false}},
	})

	require.Contains(t, source, "for _pb_tags_item: String in tags:")
	require.Contains(t, source, "Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)")
	require.NotContains(t, source, "var _pb_tags_data: PackedByteArray = PackedByteArray()")
}

// A repeated enum packs like the varint it is, and takes the option the same way.
func TestGeneratePackedOptionOnRepeatedEnum(t *testing.T) {
	status := &protoast.Enum{
		Name: "Status",
		Values: []*protoast.EnumValue{
			{Name: "STATUS_UNSPECIFIED", Number: 0},
			{Name: "STATUS_READY", Number: 1},
		},
	}
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Status", Name: "history", Number: 1, Repeated: true, IsEnum: true, Options: map[string]any{"packed": false}},
		},
	}}, []*protoast.Enum{status}))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "for _pb_history_item: Status in history:")
	require.Contains(t, source, "Wire.make_tag(1, Wire.WIRE_VARINT)")
	require.NotContains(t, source, "var _pb_history_data: PackedByteArray = PackedByteArray()")
	// Decoding still takes the packed run.
	require.Contains(t, source, "if _pb_history_packed.offset > _pb_history_end:")
}

// proto3 omits an implicit-presence field holding the default, and for a float
// the default is +0.0 specifically. -0.0 is a distinct value that protobuf puts
// on the wire, and `!= 0.0` reports the two as equal, so a float cannot use the
// plain zero comparison the integral types do.
func TestGenerateFloatPresenceDistinguishesNegativeZero(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "double", Name: "ratio", Number: 1},
		{FieldType: "float", Name: "accuracy", Number: 2},
		{FieldType: "int32", Name: "level", Number: 3},
	})

	require.Contains(t, source, "if not Wire.is_default_float(ratio):")
	// A proto float is binary32, so presence is decided on the narrowed value:
	// a double too small for binary32 becomes the default on the way out.
	require.Contains(t, source, "if not Wire.is_default_float32(accuracy):")
	require.NotContains(t, source, "if ratio != 0.0:")
	// An integer has one zero, so it keeps the direct comparison.
	require.Contains(t, source, "if level != 0:")
}

// A map entry is a length-delimited submessage. Its key and value readers bound
// themselves against the whole buffer, so a truncated entry would otherwise
// read the field that follows it and report success.
func TestGenerateMapEntryRejectsReadsCrossingItsEnd(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Maps: []*protoast.MapField{{KeyType: "sfixed64", ValueType: "float", Name: "ratios", Number: 1}},
	}}, nil))
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "if _pb_ratios_key_read.offset > _pb_ratios_end:")
	require.Contains(t, source, "if _pb_ratios_value_read.offset > _pb_ratios_end:")
	require.Contains(t, source, "return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH")

	// The entry's own tag and the skip of a field the entry does not recognize
	// read from the same buffer, so they need the same bound: either one can
	// otherwise run off the end of the entry and into the next field.
	require.Contains(t, source, "if _pb_ratios_entry_tag.offset > _pb_ratios_end:")
	require.Contains(t, source, "if _pb_ratios_skip.offset > _pb_ratios_end:")
}

// The same framing has to hold where the value is not a plain field: map keys
// and values carry their own tags inside the entry, and a oneof member carries
// the tag of the field it stands for.
func TestGenerateFixedWidthScalarsInMapsAndOneofs(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Maps: []*protoast.MapField{
			{KeyType: "sfixed64", ValueType: "float", Name: "ratios", Number: 1},
		},
		Oneofs: []*protoast.Oneof{{
			Name: "payload",
			Fields: []*protoast.Field{
				{FieldType: "double", Name: "ratio", Number: 2},
				{FieldType: "sint64", Name: "delta", Number: 3},
			},
		}},
	}}, nil))
	message := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, message, "var ratios: Dictionary[int, float] = {}")
	// Inside the entry, key and value carry field numbers 1 and 2 with their
	// own framing rather than the map field's.
	require.Contains(t, message, "Wire.encode_fixed64(_pb_ratios_key)")
	require.Contains(t, message, "Wire.encode_float(ratios[_pb_ratios_key])")
	require.Contains(t, message, "var _pb_ratios_key_read: FixedRead = Wire.read_fixed64(_pb_data, _pb_offset)")
	require.Contains(t, message, "var _pb_ratios_value_read: FloatRead = Wire.read_float(_pb_data, _pb_offset)")

	require.Contains(t, message, "Wire.encode_double(_pb_payload_ratio)")
	require.Contains(t, message, "Wire.encode_sint64(_pb_payload_delta)")
	require.Contains(t, message, "Wire.make_tag(2, Wire.WIRE_64BIT)")
	require.Contains(t, message, "Wire.make_tag(3, Wire.WIRE_VARINT)")
}

// Every proto3 scalar generates; nothing in the schema language is refused for
// want of a wire encoding any more.
func TestGenerateAcceptsEveryProto3Scalar(t *testing.T) {
	scalars := []string{
		"double", "float", "int32", "int64", "uint32", "uint64",
		"sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64",
		"bool", "string", "bytes",
	}
	fields := make([]*protoast.Field, 0, len(scalars))
	for i, scalar := range scalars {
		fields = append(fields, &protoast.Field{FieldType: scalar, Name: scalar + "_field", Number: i + 1})
	}

	_, err := Generate(namespacedFile([]*protoast.Message{{Name: "Player", Fields: fields}}, nil), "player.proto", nil, Options{})
	require.NoError(t, err)
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
	}}, nil), "player.proto", nil, Options{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "the _pb_ prefix is reserved")

	_, err = Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name:   "_pb_result",
			Fields: []*protoast.Field{{FieldType: "int32", Name: "amount", Number: 1}},
		}},
	}}, nil), "player.proto", nil, Options{})

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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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

// A payload nested in the very message that declares the oneof is ordinary
// protobuf. The union is hoisted to file level, so it names the payload by its
// scoped reference through the declaring class.
func TestOneofCarriesPayloadNestedInDeclaringMessage(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		NestedMessages: []*protoast.Message{{
			Name:   "Badge",
			Fields: []*protoast.Field{{FieldType: "string", Name: "code", Number: 1}},
		}},
		NestedEnums: []*protoast.Enum{{
			Name:   "Tier",
			Values: []*protoast.EnumValue{{Name: "TIER_UNSPECIFIED", Number: 0}},
		}},
		Oneofs: []*protoast.Oneof{{
			Name: "payload",
			Fields: []*protoast.Field{
				{FieldType: "Badge", Name: "badge", Number: 1},
				{FieldType: "Tier", Name: "tier", Number: 2},
			},
		}},
	}}, nil))

	require.Contains(t, files["cafecito/game/v1/PlayerPayloadCase.pb.fs"], "\tBadge(badge: Player.Badge)\n")
	require.Contains(t, files["cafecito/game/v1/PlayerPayloadCase.pb.fs"], "\tTier(tier: Player.Tier)\n")
	// Inside the class the lexical spelling is still what Foundry resolves.
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var _pb_payload_badge_message: Badge = Badge.new()")
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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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
	}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}}, Options{})

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
	}}, nil), "player.proto", nil, Options{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "generated Foundry member names collide:")
	require.Contains(t, err.Error(), `Foundry member "var_"`)
	require.Contains(t, err.Error(), "rename one protobuf declaration")
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
	}}, nil), "player.proto", nil, Options{})

	require.Error(t, err)
	require.Contains(t, err.Error(), `Foundry member "payload"`)
	require.Contains(t, err.Error(), "field cafecito.game.v1.Player.payload")
	require.Contains(t, err.Error(), "oneof cafecito.game.v1.Player.payload")
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
	), "player.proto", nil, Options{})

	require.Error(t, err)
	diagnostic := err.Error()
	require.Equal(t, 1, strings.Count(
		diagnostic,
		`retained enum companion cafecito.game.v1.Player.pick_kind generates Foundry member "_pb_pick_kind_unknown"`,
	))
	require.Equal(t, 1, strings.Count(
		diagnostic,
		`retained enum companion cafecito.game.v1.Player.kind generates Foundry member "_pb_pick_kind_unknown"`,
	))
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
	}, Options{})

	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "var held: cafecito.inventory.v1.Item? = null")
	require.Contains(t, source, "var listed: cafecito.catalog.v1.Item? = null")
	require.Contains(t, source, "import cafecito.catalog.v1")
	require.Contains(t, source, "import cafecito.inventory.v1")
}

func TestGenerateEscapesEngineTypeMessageMembersEverywhere(t *testing.T) {
	files := generate(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "int32", Name: "Node", Number: 1},
			{FieldType: "string", Name: "String", Number: 2, Optional: true},
			{FieldType: "int32", Name: "Timer", Number: 3, Repeated: true},
		},
		Maps: []*protoast.MapField{{
			KeyType: "string", ValueType: "string", Name: "Resource", Number: 4,
		}},
		Oneofs: []*protoast.Oneof{{
			Name: "Object",
			Fields: []*protoast.Field{{
				FieldType: "string", Name: "Image", Number: 5,
			}},
		}},
		NestedMessages: []*protoast.Message{{
			Name: "Child",
			Fields: []*protoast.Field{{
				FieldType: "int32", Name: "Node", Number: 1,
			}},
		}},
	}}, nil))

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	for _, declaration := range []string{
		"var Node_: int = 0",
		"var String_: String? = null",
		"var Timer_: Array[int] = []",
		"var Resource_: Dictionary[String, String] = {}",
		"var Object_: PlayerObjectCase? = null",
	} {
		require.Contains(t, messageSource, declaration)
	}
	require.Equal(t, 2, strings.Count(messageSource, "var Node_: int = 0"))

	for _, expression := range []string{
		"if Node_ != 0:",
		"Wire.encode_varint(Node_)",
		"if String_ is String:",
		"Wire.encode_string(String_)",
		"if Timer_.size() > 0:",
		"for _pb_Timer_item: int in Timer_:",
		"for _pb_Resource_key: String in Resource_:",
		"Resource_[_pb_Resource_key]",
		"match Object_:",
		"Node_ = _pb_Node_read.value",
		"String_ = _pb_String_read.value",
		"Timer_.append(_pb_Timer_read.value)",
		"Resource_[_pb_Resource_key] = _pb_Resource_value",
		"Object_ = PlayerObjectCase.Image(_pb_Object_Image_read.value)",
	} {
		require.Contains(t, messageSource, expression)
	}

	require.Contains(t, messageSource, "## The Node protobuf field.")
	require.Contains(t, messageSource, "## The Object protobuf oneof; null when no case is set.")
	require.NotContains(t, messageSource, "The Node_ protobuf field.")
	require.NotContains(t, messageSource, "The Object_ protobuf oneof")

	oneofSource := files["cafecito/game/v1/PlayerObjectCase.pb.fs"]
	require.Contains(t, oneofSource, "## Cases of the Object protobuf oneof.")
	require.Contains(t, oneofSource, "\tImage(Image: String)")
	require.NotContains(t, oneofSource, "Image_")
}

func TestGenerateEscapesEngineNamedEnumAndMessageMembersEverywhere(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{
			{
				Name: "Envelope",
				Fields: []*protoast.Field{
					{FieldType: "DeliveryState", Name: "String", Number: 1},
					{FieldType: "Payload", Name: "Node", Number: 2},
				},
			},
			{Name: "Payload"},
		},
		[]*protoast.Enum{{
			Name: "DeliveryState",
			Values: []*protoast.EnumValue{
				{Name: "Node", Number: 0},
				{Name: "READY", Number: 1},
			},
		}},
	))
	messageSource := files["cafecito/game/v1/Envelope.pb.fs"]

	require.Contains(t, messageSource,
		"## The String protobuf field.\n"+
			"var String_: DeliveryState = DeliveryState.Node:\n"+
			"\tset(_pb_value):\n"+
			"\t\t_pb_String_unknown = PackedByteArray()\n"+
			"\t\tString_ = _pb_value\n")
	require.Contains(t, messageSource,
		"## Raw bytes of an unrecognized String value, kept so a re-encode is lossless.\n"+
			"var _pb_String_unknown: PackedByteArray = PackedByteArray()\n")
	require.Contains(t, messageSource,
		"\tif _pb_String_unknown.size() > 0:\n"+
			"\t\t_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))\n"+
			"\t\t_pb_result.append_array(_pb_String_unknown)\n"+
			"\telif String_ != DeliveryState.Node:\n"+
			"\t\t_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_VARINT)))\n"+
			"\t\t_pb_result.append_array(Wire.encode_varint(String_.to_wire()))\n")
	require.Contains(t, messageSource,
		"\t\t\t\tvar _pb_String_case: DeliveryState? = DeliveryState.from_wire(_pb_String_read.value)\n"+
			"\t\t\t\tif _pb_String_case is DeliveryState:\n"+
			"\t\t\t\t\tString_ = _pb_String_case\n"+
			"\t\t\t\telse:\n"+
			"\t\t\t\t\tString_ = DeliveryState.Node\n"+
			"\t\t\t\t\t_pb_String_unknown = _pb_data.slice(_pb_offset, _pb_String_read.offset)\n")

	require.Contains(t, messageSource,
		"## The Node protobuf field.\n"+
			"var Node_: Payload? = null\n")
	require.Contains(t, messageSource,
		"\tif Node_ is Payload:\n"+
			"\t\tvar _pb_Node_data: PackedByteArray = Node_.to_bytes()\n"+
			"\t\t_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))\n")
	require.Contains(t, messageSource,
		"\t\t\t\tif not (Node_ is Payload):\n"+
			"\t\t\t\t\tNode_ = Payload.new()\n"+
			"\t\t\t\tvar _pb_Node_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, Node_)\n")

	require.NotContains(t, messageSource, "The String_ protobuf field.")
	require.NotContains(t, messageSource, "The Node_ protobuf field.")

	enumSource := files["cafecito/game/v1/DeliveryState.pb.fs"]
	require.Contains(t, enumSource, "\tNode = 0\n")
	require.NotContains(t, enumSource, "\tNode_ = 0\n")
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
	}}, nil), "player.proto", []FileEntry{{File: imported, Filename: "loose.proto"}}, Options{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "no usable namespace")
}

func TestGenerateAcceptsOptions(t *testing.T) {
	file := namespacedFile([]*protoast.Message{{
		Name:   "Probe",
		Fields: []*protoast.Field{{FieldType: "string", Name: "label", Number: 1}},
	}}, nil)

	withJSON, err := Generate(file, "probe.proto", nil, Options{JSON: true})
	require.NoError(t, err)

	withoutJSON, err := Generate(file, "probe.proto", nil, Options{})
	require.NoError(t, err)

	// The option is what the JSON surface is gated on, so the two runs differ
	// by exactly that surface and by nothing else.
	require.NotEqual(t, withoutJSON, withJSON)
	require.NotContains(t, withoutJSON["cafecito/game/v1/Probe.pb.fs"], "to_json")
	require.Contains(t, withJSON["cafecito/game/v1/Probe.pb.fs"], "func to_json() -> JsonNode:")
}

func generateJSON(t *testing.T, file *protoast.ProtoFile, sourceName string) GeneratedFiles {
	t.Helper()
	files, err := Generate(file, sourceName, nil, Options{JSON: true})
	require.NoError(t, err)
	return files
}

func jsonPlayerSource(t *testing.T, fields ...*protoast.Field) string {
	t.Helper()
	files := generateJSON(t, namespacedFile([]*protoast.Message{{Name: "Player", Fields: fields}}, nil), "player.proto")
	return files["cafecito/game/v1/Player.pb.fs"]
}

// Conforming to the engine's trait is what teaches JSON.stringify to lower a
// message, so a binding without it has no route to JSON text at all.
func TestJSONOptionAddsTheSerializableConformance(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "string", Name: "name", Number: 1})

	require.Contains(t, source, "final class_name Player extends RefCounted uses Message, JsonSerializable\n")
	require.Contains(t, source, "func to_json() -> JsonNode:")
	// Text conversion is JSON.stringify(msg, "", false); no method is emitted.
	require.NotContains(t, source, "to_json_string")
}

// The option is opt-in because it roughly doubles a generated file, so the
// off-path has to stay exactly what it was.
func TestJSONSurfaceIsAbsentWithoutTheOption(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})

	require.NotContains(t, source, "JsonSerializable")
	require.NotContains(t, source, "to_json")
	require.NotContains(t, source, "JsonNode")
}

func TestJSONScalarsFollowTheCanonicalTable(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "int32", Name: "level", Number: 1},
		&protoast.Field{FieldType: "sfixed32", Name: "offset", Number: 2},
		&protoast.Field{FieldType: "int64", Name: "score", Number: 3},
		&protoast.Field{FieldType: "uint64", Name: "seed", Number: 4},
		&protoast.Field{FieldType: "double", Name: "ratio", Number: 5},
		&protoast.Field{FieldType: "float", Name: "speed", Number: 6},
		&protoast.Field{FieldType: "bool", Name: "active", Number: 7},
		&protoast.Field{FieldType: "string", Name: "name", Number: 8},
		&protoast.Field{FieldType: "bytes", Name: "blob", Number: 9},
	)

	// A 32-bit integer renders as a JSON number.
	require.Contains(t, source, `_pb_json["level"] = JsonNode.Int(level)`)
	require.Contains(t, source, `_pb_json["offset"] = JsonNode.Int(offset)`)
	// A 64-bit one renders as a string: a bare number past 2^53 does not
	// survive the engine's parser.
	require.Contains(t, source, `_pb_json["score"] = JsonNode.Str(str(score))`)
	// An unsigned one is held in a signed host int, so its decimal digits come
	// from the runtime helper rather than from str().
	require.Contains(t, source, `_pb_json["seed"] = JsonNode.Str(JsonUint64.format(seed))`)
	require.Contains(t, source, `_pb_json["ratio"] = _pb_json_float(ratio)`)
	// A proto float is binary32; the member holding it is not.
	require.Contains(t, source, `_pb_json["speed"] = _pb_json_float(Wire.narrow_float32(speed))`)
	require.Contains(t, source, `_pb_json["active"] = JsonNode.Bool(active)`)
	require.Contains(t, source, `_pb_json["name"] = JsonNode.Str(name)`)
	require.Contains(t, source, `_pb_json["blob"] = JsonNode.Str(JsonBase64.encode(blob))`)
	require.Contains(t, source, "return JsonNode.object_of(_pb_json)")
}

// A host int is signed, so an unsigned 64-bit value at or above 2^63 is held as
// a negative bit pattern and str() would print it with a minus sign. The
// canonical mapping wants its unsigned decimal digits -- 18446744073709551615,
// not -1 -- so the unsigned types go through the runtime helper and the signed
// ones keep str().
func TestJSONUnsigned64BitScalarsAreWrittenAsUnsignedDecimal(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "uint64", Name: "seed", Number: 1},
		&protoast.Field{FieldType: "fixed64", Name: "handle", Number: 2},
		&protoast.Field{FieldType: "int64", Name: "score", Number: 3},
		&protoast.Field{FieldType: "sint64", Name: "delta", Number: 4},
		&protoast.Field{FieldType: "sfixed64", Name: "offset", Number: 5},
	)

	require.Contains(t, source, `_pb_json["seed"] = JsonNode.Str(JsonUint64.format(seed))`)
	require.Contains(t, source, `_pb_json["handle"] = JsonNode.Str(JsonUint64.format(handle))`)
	require.Contains(t, source, `_pb_json["score"] = JsonNode.Str(str(score))`)
	require.Contains(t, source, `_pb_json["delta"] = JsonNode.Str(str(delta))`)
	require.Contains(t, source, `_pb_json["offset"] = JsonNode.Str(str(offset))`)
}

// A map key is stringified into the object's member name, so an unsigned 64-bit
// key has the same problem the value does.
func TestJSONUnsigned64BitMapKeysAreWrittenAsUnsignedDecimal(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Maps: []*protoast.MapField{
			{KeyType: "uint64", ValueType: "string", Name: "labels", Number: 1},
			{KeyType: "sfixed64", ValueType: "string", Name: "notes", Number: 2},
		},
	}}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source,
		"_pb_labels_fields[JsonUint64.format(_pb_labels_key)] = JsonNode.Str(labels[_pb_labels_key])")
	require.Contains(t, source, "_pb_notes_fields[str(_pb_notes_key)] = JsonNode.Str(notes[_pb_notes_key])")
}

// JSON.stringify turns NaN into null and the infinities into ±1e99999, none of
// which is canonical proto3, so a non-finite float never reaches the Float case.
func TestJSONNonFiniteFloatsAreEmittedAsTheirStringForms(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "double", Name: "ratio", Number: 1})

	require.Contains(t, source, "static func _pb_json_float(_pb_value: float) -> JsonNode:")
	require.Contains(t, source, "if is_nan(_pb_value):")
	require.Contains(t, source, `return JsonNode.Str("NaN")`)
	require.Contains(t, source, "if is_inf(_pb_value):")
	require.Contains(t, source, `return JsonNode.Str("Infinity")`)
	require.Contains(t, source, `return JsonNode.Str("-Infinity")`)
	require.Contains(t, source, "return JsonNode.Float(_pb_value)")
}

func TestJSONFloatHelperIsEmittedOnlyWhereAFloatIsWritten(t *testing.T) {
	require.NotContains(t, jsonPlayerSource(t,
		&protoast.Field{FieldType: "string", Name: "name", Number: 1},
	), "_pb_json_float")
	require.Contains(t, jsonPlayerSource(t,
		&protoast.Field{FieldType: "float", Name: "speed", Number: 1, Repeated: true},
	), "static func _pb_json_float(")
}

// proto3 has no presence for a plain scalar, so its default is indistinguishable
// from unset and canonical JSON leaves it out.
func TestJSONOmitsProto3ZeroValues(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "int32", Name: "level", Number: 1},
		&protoast.Field{FieldType: "string", Name: "name", Number: 2},
		&protoast.Field{FieldType: "bool", Name: "active", Number: 3},
		&protoast.Field{FieldType: "bytes", Name: "blob", Number: 4},
		&protoast.Field{FieldType: "double", Name: "ratio", Number: 5},
	)

	require.Contains(t, source, "if level != 0:")
	require.Contains(t, source, "if name != \"\":")
	require.Contains(t, source, "if active:")
	require.Contains(t, source, "if blob.size() > 0:")
	// A float has two zeroes and protobuf treats only one as the default.
	require.Contains(t, source, "if not Wire.is_default_float(ratio):")
}

// An explicit-presence field carries its own default, so the zero-value rule
// does not apply to it: only nullness decides.
func TestJSONWritesPresentMembersOnly(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "int32", Name: "level", Number: 1, Optional: true},
		&protoast.Field{FieldType: "Slot", Name: "slot", Number: 2},
	)

	require.Contains(t, source, "if level is int:")
	require.Contains(t, source, `_pb_json["level"] = JsonNode.Int(level)`)
	require.Contains(t, source, "if slot is Slot:")
	require.Contains(t, source, `_pb_json["slot"] = slot.to_json()`)
}

// The JSON name comes from the field model, which honours an explicit
// [json_name] and derives camelCase otherwise.
func TestJSONWritesTheResolvedJSONFieldName(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "string", Name: "display_name", Number: 1},
		&protoast.Field{FieldType: "string", Name: "raw_label", Number: 2, Options: map[string]any{"json_name": "wire"}},
	)

	require.Contains(t, source, `_pb_json["displayName"] = JsonNode.Str(display_name)`)
	require.Contains(t, source, `_pb_json["wire"] = JsonNode.Str(raw_label)`)
	require.NotContains(t, source, `_pb_json["rawLabel"]`)
}

// Canonical JSON writes an enum as its declared case name, so the conversion is
// hosted on the enum for the same reason to_wire is.
func TestJSONEnumsAreWrittenAsCaseNames(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Tier", Name: "tier", Number: 1}},
	}}, []*protoast.Enum{{
		Name: "Tier",
		Values: []*protoast.EnumValue{
			{Name: "TIER_BRONZE", Number: 0},
			{Name: "TIER_GOLD", Number: 1},
		},
	}}), "player.proto")

	enumSource := files["cafecito/game/v1/Tier.pb.fs"]
	require.Contains(t, enumSource, "func to_json_name() -> String:")
	require.Contains(t, enumSource, `return "TIER_BRONZE"`)
	require.Contains(t, enumSource, `return "TIER_GOLD"`)

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "if tier != Tier.TIER_BRONZE:")
	require.Contains(t, source, `_pb_json["tier"] = JsonNode.Str(tier.to_json_name())`)
}

func TestEnumJSONNameIsAbsentWithoutTheOption(t *testing.T) {
	files := generate(t, namespacedFile(nil, []*protoast.Enum{{
		Name:   "Tier",
		Values: []*protoast.EnumValue{{Name: "TIER_BRONZE", Number: 0}},
	}}))

	require.NotContains(t, files["cafecito/game/v1/Tier.pb.fs"], "to_json_name")
}

func TestJSONRepeatedFieldsBecomeArrays(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "string", Name: "tags", Number: 1, Repeated: true},
	)

	require.Contains(t, source, "if tags.size() > 0:")
	require.Contains(t, source, "var _pb_tags_items: Array[JsonNode] = []")
	require.Contains(t, source, "for _pb_tags_item: String in tags:")
	require.Contains(t, source, "_pb_tags_items.append(JsonNode.Str(_pb_tags_item))")
	require.Contains(t, source, `_pb_json["tags"] = JsonNode.array_of(_pb_tags_items)`)
}

// JSON object keys are strings, so a non-string map key is stringified per the
// specification rather than left to the encoder.
func TestJSONMapsBecomeObjectsWithStringifiedKeys(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Maps: []*protoast.MapField{
			{KeyType: "string", ValueType: "int32", Name: "counts", Number: 1},
			{KeyType: "int64", ValueType: "string", Name: "labels", Number: 2},
			{KeyType: "bool", ValueType: "string", Name: "flags", Number: 3},
		},
	}}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "if counts.size() > 0:")
	require.Contains(t, source, "var _pb_counts_fields: Dictionary[String, JsonNode] = {}")
	require.Contains(t, source, "for _pb_counts_key: String in counts:")
	require.Contains(t, source, "_pb_counts_fields[_pb_counts_key] = JsonNode.Int(counts[_pb_counts_key])")
	require.Contains(t, source, `_pb_json["counts"] = JsonNode.object_of(_pb_counts_fields)`)
	require.Contains(t, source, "_pb_labels_fields[str(_pb_labels_key)] = JsonNode.Str(labels[_pb_labels_key])")
	require.Contains(t, source, `_pb_flags_fields["true" if _pb_flags_key else "false"] = JsonNode.Str(flags[_pb_flags_key])`)
}

// A oneof writes exactly the member that is set, and nothing when it is unset.
func TestJSONOneofWritesTheSetMemberOnly(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name: "payload",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "text", Number: 1},
				{FieldType: "int32", Name: "amount", Number: 2},
			},
		}},
	}}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "match payload:")
	require.Contains(t, source, "PlayerPayloadCase.Text(var _pb_payload_text):")
	require.Contains(t, source, `_pb_json["text"] = JsonNode.Str(_pb_payload_text)`)
	require.Contains(t, source, "PlayerPayloadCase.Amount(var _pb_payload_amount):")
	require.Contains(t, source, `_pb_json["amount"] = JsonNode.Int(_pb_payload_amount)`)
}

// A message field recurses through the trait, which is what makes the whole
// document one dispatch from the root.
func TestJSONNestedMessagesRecurseThroughTheTrait(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Slot", Name: "slots", Number: 1, Repeated: true},
		},
	}, slotMessage()}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "_pb_slots_items.append(_pb_slots_item.to_json())")
	require.Contains(t, files["cafecito/game/v1/Slot.pb.fs"], "func to_json() -> JsonNode:")
}

// Conformance is all-or-nothing: the engine's analyzer rejects a class that
// declares the trait and implements only half of it, so the two halves are
// emitted together.
func TestJSONConformanceCarriesBothHalves(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "string", Name: "name", Number: 1})

	require.Contains(t, source, "func to_json() -> JsonNode:")
	require.Contains(t, source, "static func from_json(_pb_node: JsonNode) -> JsonResult[Player]:")
}

// wellKnownSource generates one google/protobuf file's messages under its own
// import path, which is what selects the canonical JSON form: the type name
// alone cannot, since a schema of the caller's own may declare a Timestamp.
func wellKnownSource(t *testing.T, importPath, typeName string, messages ...*protoast.Message) string {
	t.Helper()
	files := generateJSON(t, namespacedFile(messages, nil), importPath)
	return files["cafecito/game/v1/"+typeName+".pb.fs"]
}

func TestWellKnownTimestampAndDurationSerializeThroughTheirRuntimeHelpers(t *testing.T) {
	secondsAndNanos := []*protoast.Field{
		{FieldType: "int64", Name: "seconds", Number: 1},
		{FieldType: "int32", Name: "nanos", Number: 2},
	}

	timestamp := wellKnownSource(t, "google/protobuf/timestamp.proto", "Timestamp",
		&protoast.Message{Name: "Timestamp", Fields: secondsAndNanos})
	require.Contains(t, timestamp, "var (_pb_text, _pb_error) = JsonTimestamp.format(seconds, nanos)")
	require.Contains(t, timestamp, "return JsonNode.Str(_pb_text)")
	// to_json has no error channel, so a value the helper refuses has to be
	// reported rather than quietly written as a null the caller cannot tell
	// from a successful one.
	require.Contains(t, timestamp,
		`push_error("JSON_VALUE_OUT_OF_RANGE: Timestamp cannot be written as canonical JSON")`)
	// The seconds field is not written as a member of its own.
	require.NotContains(t, timestamp, `_pb_json["seconds"]`)

	duration := wellKnownSource(t, "google/protobuf/duration.proto", "Duration",
		&protoast.Message{Name: "Duration", Fields: secondsAndNanos})
	require.Contains(t, duration, "var (_pb_text, _pb_error) = JsonDuration.format(seconds, nanos)")
}

func TestWellKnownFieldMaskSerializesAsOneCommaJoinedString(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/field_mask.proto", "FieldMask",
		&protoast.Message{Name: "FieldMask", Fields: []*protoast.Field{
			{FieldType: "string", Name: "paths", Number: 1, Repeated: true},
		}})

	require.Contains(t, source, "var (_pb_text, _pb_error) = JsonFieldMask.to_json(paths)")
	require.Contains(t, source, "return JsonNode.Str(_pb_text)")
}

func TestWellKnownEmptySerializesAsAnEmptyObject(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/empty.proto", "Empty",
		&protoast.Message{Name: "Empty"})

	require.Contains(t, source, "var _pb_json: Dictionary[String, JsonNode] = {}\n\treturn JsonNode.object_of(_pb_json)")
}

// A wrapper exists to give a scalar explicit presence, so its value is written
// whatever it is; the message's own presence was already decided by the member
// that held it.
func TestWellKnownWrappersSerializeAsTheBareScalar(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/wrappers.proto", "Int32Value",
		&protoast.Message{Name: "Int32Value", Fields: []*protoast.Field{
			{FieldType: "int32", Name: "value", Number: 1},
		}})
	require.Contains(t, source, "func to_json() -> JsonNode:\n\treturn JsonNode.Int(value)")
	require.NotContains(t, source, "if value != 0:\n\t\t_pb_json")

	wide := wellKnownSource(t, "google/protobuf/wrappers.proto", "Int64Value",
		&protoast.Message{Name: "Int64Value", Fields: []*protoast.Field{
			{FieldType: "int64", Name: "value", Number: 1},
		}})
	require.Contains(t, wide, "return JsonNode.Str(str(value))")
}

func TestWellKnownStructAndListValueSerializeAsPlainJSON(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{
		{Name: "Struct", Maps: []*protoast.MapField{
			{KeyType: "string", ValueType: "Value", Name: "fields", Number: 1},
		}},
		{Name: "ListValue", Fields: []*protoast.Field{
			{FieldType: "Value", Name: "values", Number: 1, Repeated: true},
		}},
		{Name: "Value", Oneofs: []*protoast.Oneof{{
			Name: "kind",
			Fields: []*protoast.Field{
				{FieldType: "NullValue", Name: "null_value", Number: 1},
				{FieldType: "double", Name: "number_value", Number: 2},
				{FieldType: "string", Name: "string_value", Number: 3},
				{FieldType: "Struct", Name: "struct_value", Number: 5},
			},
		}}},
	}, []*protoast.Enum{{
		Name:   "NullValue",
		Values: []*protoast.EnumValue{{Name: "NULL_VALUE", Number: 0}},
	}}), "google/protobuf/struct.proto")

	structSource := files["cafecito/game/v1/Struct.pb.fs"]
	require.Contains(t, structSource, "for _pb_fields_key: String in fields:")
	require.Contains(t, structSource, "_pb_json[_pb_fields_key] = fields[_pb_fields_key].to_json()")

	listSource := files["cafecito/game/v1/ListValue.pb.fs"]
	require.Contains(t, listSource, "_pb_values_items.append(_pb_values_item.to_json())")
	require.Contains(t, listSource, "return JsonNode.array_of(_pb_values_items)")

	// A Value is whichever JSON value its kind names. NullValue is the one
	// place the general enum rule does not apply: a null renders as null, not
	// as the case name, and an unset Value is null too.
	valueSource := files["cafecito/game/v1/Value.pb.fs"]
	require.Contains(t, valueSource, "ValueKindCase.NullValue(var _pb_kind_null_value):\n\t\t\treturn JsonNode.Null")
	require.Contains(t, valueSource, "return _pb_json_float(_pb_kind_number_value)")
	require.Contains(t, valueSource, "return JsonNode.Str(_pb_kind_string_value)")
	require.Contains(t, valueSource, "return _pb_kind_struct_value.to_json()")
	require.Contains(t, valueSource, "_:\n\t\t\treturn JsonNode.Null")
}

// Any needs its type URL resolved to a generated binding, which needs a runtime
// type registry that does not exist yet.
func TestWellKnownAnyReportsThatItHasNoJSONForm(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/any.proto", "Any",
		&protoast.Message{Name: "Any", Fields: []*protoast.Field{
			{FieldType: "string", Name: "type_url", Number: 1},
			{FieldType: "bytes", Name: "value", Number: 2},
		}})

	require.Contains(t, source, `push_error("JSON_ANY_UNSUPPORTED: `)
	require.Contains(t, source, "return JsonNode.Null")
	require.NotContains(t, source, `_pb_json["typeUrl"]`)
}

// The table is keyed on the import path because the type name is not an
// identity: a schema of the caller's own may declare a Timestamp of its own,
// and it serializes as an ordinary message.
func TestWellKnownFormsAreKeyedOnTheImportPath(t *testing.T) {
	source := wellKnownSource(t, "cafecito/game/v1/player.proto", "Timestamp",
		&protoast.Message{Name: "Timestamp", Fields: []*protoast.Field{
			{FieldType: "int64", Name: "seconds", Number: 1},
		}})

	require.NotContains(t, source, "JsonTimestamp")
	require.Contains(t, source, `_pb_json["seconds"] = JsonNode.Str(str(seconds))`)
}

// A field named after a generated member would replace it rather than sit
// beside it, so the JSON members are reserved for the same reason to_bytes is.
// The escaping does not depend on the option: a member that changed name when
// JSON was switched on would break every caller of the binding.
func TestJSONMethodNamesAreReservedForGeneratedMembers(t *testing.T) {
	fields := []*protoast.Field{
		{FieldType: "string", Name: "to_json", Number: 1},
		{FieldType: "string", Name: "from_json", Number: 2},
	}

	source := jsonPlayerSource(t, fields...)
	require.Contains(t, source, "var to_json_: String")
	require.Contains(t, source, "var from_json_: String")
	require.Contains(t, source, "func to_json() -> JsonNode:")
	require.Contains(t, source, `_pb_json["toJson"] = JsonNode.Str(to_json_)`)

	require.Contains(t, playerSource(t, fields), "var to_json_: String")
}

// Foundry's float is binary64 while a proto float is binary32, so a member may
// carry precision the field cannot. The encoder drops it on the way to the
// wire; canonical JSON has to drop it too, or a wire round trip and a JSON one
// disagree about the same value.
func TestJSONNarrowsAProtoFloatToBinary32(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "float", Name: "speed", Number: 1},
		&protoast.Field{FieldType: "double", Name: "ratio", Number: 2},
	)

	require.Contains(t, source, `_pb_json["speed"] = _pb_json_float(Wire.narrow_float32(speed))`)
	// A double is already binary64; narrowing it would destroy the value.
	require.Contains(t, source, `_pb_json["ratio"] = _pb_json_float(ratio)`)
}

// The decode surface is the engine's JsonSerializable half that #73 could only
// stub: from_json constructs and merges, so repeated, map, and oneof members
// are decoded once rather than twice.
func TestJSONDecodeSurfaceIsConstructThenMerge(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "string", Name: "name", Number: 1})

	require.Contains(t, source, "static func from_json(_pb_node: JsonNode) -> JsonResult[Player]:")
	require.Contains(t, source, "var _pb_message: Player = Player.new()")
	require.Contains(t, source, "var _pb_error: JsonDecodeError? = _pb_message._pb_merge_from_json(_pb_node)")
	require.Contains(t, source, "return JsonResult[Player].fail(_pb_error.message, _pb_error.path)")
	require.Contains(t, source, "return JsonResult[Player].ok(_pb_message)")
	require.Contains(t, source, "func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:")
	// The seam #73 emitted is gone; two from_json declarations would not compile.
	require.NotContains(t, source, "cannot be decoded from JSON yet")
	// The caller passes JSON.parse_to_node(text).value, so no text entry point.
	require.NotContains(t, source, "from_json_string")
}

// A document that is not an object cannot carry members, so the shape is
// checked before anything is read out of it.
func TestJSONDecodeRejectsANonObjectDocument(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "string", Name: "name", Number: 1})

	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_TYPE_MISMATCH: Player expects a JSON object", "$")`)
}

// Both spellings are accepted on input; the specification's camelCase form and
// the original proto name name the same field.
func TestJSONDecodeAcceptsBothFieldNameSpellings(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "string", Name: "display_name", Number: 1},
		&protoast.Field{FieldType: "string", Name: "name", Number: 2},
	)

	require.Contains(t, source, "\t\t\t\"displayName\", \"display_name\":\n")
	// A field whose two spellings agree is listed once, not twice.
	require.Contains(t, source, "\t\t\t\"name\":\n")
}

// JSON has no unknown-field preservation, so a member the schema does not know
// is an error rather than something silently dropped.
func TestJSONDecodeRejectsAnUnknownMember(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "string", Name: "name", Number: 1})

	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_UNKNOWN_FIELD: Player has no field named " + _pb_key, _pb_member_path)`)
	require.Contains(t, source, `var _pb_member_path: String = "$." + _pb_key`)
}

// The readers follow the specification's accept column, which is wider than the
// case the emitter writes: a 32-bit field takes a string and an integral float
// as well as a number.
func TestJSONDecodeAcceptsTheSpecifiedCasesPer32BitField(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "int32", Name: "level", Number: 1})

	require.Contains(t, source,
		"static func _pb_json_read_int32(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):")
	require.Contains(t, source, "JsonNode.Int(var _pb_int):")
	require.Contains(t, source, "JsonNode.Float(var _pb_float):")
	require.Contains(t, source, "if _pb_float != floor(_pb_float):")
	require.Contains(t, source, "if not _pb_text.is_valid_int():")
	require.Contains(t, source, "if _pb_value < -2147483648 or _pb_value > 2147483647:")
	require.Contains(t, source,
		`JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: a signed 32-bit integer field cannot hold this value", _pb_path)`)
	require.Contains(t, source, "var (_pb_level_value, _pb_level_error) = _pb_json_read_int32(_pb_member, _pb_member_path)")
	require.Contains(t, source, "level = _pb_level_value")
}

// An unsigned 32-bit field spans a range the signed one does not, so it gets a
// reader of its own rather than sharing the signed bounds.
func TestJSONDecodeBoundsAnUnsigned32BitFieldOnItsOwnRange(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "uint32", Name: "count", Number: 1})

	require.Contains(t, source,
		"static func _pb_json_read_uint32(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):")
	require.Contains(t, source, "if _pb_value < 0 or _pb_value > 4294967295:")
}

// A large bare number arrives as a Float, never an Int, because the engine's
// parser produces a double; the string form is the one that stays exact.
func TestJSONDecodeAcceptsAllThreeCasesForA64BitField(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "int64", Name: "score", Number: 1},
		&protoast.Field{FieldType: "uint64", Name: "seed", Number: 2},
	)

	require.Contains(t, source,
		"static func _pb_json_read_int64(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):")
	require.Contains(t, source, "_pb_value = _pb_text.to_int()")
	// The largest int64 arrives as exactly 2^63, because the parser produces a
	// double; refusing it would refuse a conformant document, so it is the
	// documented lossy case instead.
	require.Contains(t, source, "if _pb_float > 9223372036854775808.0 or _pb_float < -9223372036854775808.0:")
	// A string the host int cannot hold wraps rather than reporting, so the
	// text has to be exactly what the value prints as.
	require.Contains(t, source, "if str(_pb_value) != _pb_text:")
	// An unsigned 64-bit field spans a range the signed reader cannot express,
	// so it gets a reader of its own, matching what the serializer writes.
	require.Contains(t, source, "var (_pb_seed_value, _pb_seed_error) = _pb_json_read_uint64(_pb_member, _pb_member_path)")
}

// The top half of the uint64 range has no signed spelling, so the reader parses
// the decimal text through the runtime helper instead of String.to_int(), which
// wraps to int64 min there.
func TestJSONDecodeReadsAnUnsigned64BitFieldThroughItsOwnReader(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "uint64", Name: "seed", Number: 1},
		&protoast.Field{FieldType: "fixed64", Name: "handle", Number: 2},
	)

	require.Contains(t, source,
		"static func _pb_json_read_uint64(_pb_node: JsonNode, _pb_path: String) -> (int, JsonDecodeError?):")
	require.Contains(t, source, "var (_pb_unsigned, _pb_unsigned_error) = JsonUint64.parse(_pb_text)")
	// The widest value has no double of its own and arrives rounded to 2^64, so
	// that bound is the documented lossy edge rather than a value to refuse; a
	// negative number is not a uint64 at all.
	require.Contains(t, source, "if _pb_float > 18446744073709551616.0 or _pb_float < 0.0:")
	require.Contains(t, source, "if _pb_float == 18446744073709551616.0:")
	require.Contains(t, source, "_pb_value = JsonUint64.WIDEST_BITS")
	require.Contains(t, source, "if _pb_int < 0:")
	// The signed reader is not emitted when no signed 64-bit field asks for it.
	require.NotContains(t, source, "_pb_json_read_int64")
}

// The three non-finite strings are the only spelling canonical JSON has for
// them, so a float field has to read them back.
func TestJSONDecodeReadsTheNonFiniteFloatStrings(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "double", Name: "ratio", Number: 1},
		&protoast.Field{FieldType: "float", Name: "speed", Number: 2},
	)

	require.Contains(t, source,
		"static func _pb_json_read_float(_pb_node: JsonNode, _pb_path: String) -> (float, JsonDecodeError?):")
	require.Contains(t, source, `if _pb_text == "NaN":`)
	require.Contains(t, source, `if _pb_text == "Infinity":`)
	require.Contains(t, source, `if _pb_text == "-Infinity":`)
	require.Contains(t, source, "ratio = _pb_ratio_value")
	// A proto float is binary32, so the wider value a document may carry is
	// narrowed exactly as the encoder narrows it on the way out.
	require.Contains(t, source, "speed = Wire.narrow_float32(_pb_speed_value)")
}

func TestJSONDecodeReadsBoolStringAndBytes(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "bool", Name: "active", Number: 1},
		&protoast.Field{FieldType: "string", Name: "name", Number: 2},
		&protoast.Field{FieldType: "bytes", Name: "blob", Number: 3},
	)

	require.Contains(t, source, "static func _pb_json_read_bool(_pb_node: JsonNode, _pb_path: String) -> (bool, JsonDecodeError?):")
	require.Contains(t, source, "static func _pb_json_read_string(_pb_node: JsonNode, _pb_path: String) -> (String, JsonDecodeError?):")
	require.Contains(t, source, "static func _pb_json_read_bytes(_pb_node: JsonNode, _pb_path: String) -> (PackedByteArray, JsonDecodeError?):")
	require.Contains(t, source, "var (_pb_bytes, _pb_bytes_error) = JsonBase64.decode(_pb_text)")
}

// A reader a message never calls would be dead code in every generated file.
func TestJSONDecodeReadersAreEmittedOnlyWhereTheyAreCalled(t *testing.T) {
	source := jsonPlayerSource(t, &protoast.Field{FieldType: "string", Name: "name", Number: 1})

	require.Contains(t, source, "_pb_json_read_string")
	require.NotContains(t, source, "_pb_json_read_int32")
	require.NotContains(t, source, "_pb_json_read_bytes")
}

// An explicit-presence field distinguishes unset from its default, so a JSON
// null clears it rather than writing the default over it.
func TestJSONDecodeClearsANullableFieldOnNull(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "int32", Name: "level", Number: 1, Optional: true},
		&protoast.Field{FieldType: "Slot", Name: "slot", Number: 2},
	)

	require.Contains(t, source, "JsonNode.Null:\n\t\t\t\t\t\tlevel = null")
	require.Contains(t, source, "JsonNode.Null:\n\t\t\t\t\t\tslot = null")
}

// A nested message decodes through the trait, and its failure is re-rooted at
// the document root so the reported path reads from there.
func TestJSONDecodeRerootsANestedFailure(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Slot", Name: "slot", Number: 1}},
	}, slotMessage()}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "var _pb_slot_result: JsonResult[Slot] = Slot.from_json(_pb_member)")
	require.Contains(t, source, "return JsonResult[Player].nested(_pb_slot_error, _pb_key).error")
}

// A repeated member is an array, and each element reports its own index in the
// path so a failure names the element rather than the field.
func TestJSONDecodeReadsRepeatedFieldsWithIndexedPaths(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "string", Name: "tags", Number: 1, Repeated: true},
	)

	require.Contains(t, source, "JsonNode.Array(var _pb_tags_items):")
	require.Contains(t, source, "tags = []")
	require.Contains(t, source, "while _pb_tags_index < _pb_tags_items.size():")
	require.Contains(t, source, `_pb_json_read_string(_pb_tags_items[_pb_tags_index], _pb_member_path + "." + str(_pb_tags_index))`)
	require.Contains(t, source, "tags.append(_pb_tags_element_value)")
	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_TYPE_MISMATCH: tags expects a JSON array", _pb_member_path)`)
}

// JSON object keys are strings, so a non-string map key is parsed back out of
// the key text rather than read as a JSON value.
func TestJSONDecodeParsesMapKeysOutOfTheirText(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Maps: []*protoast.MapField{
			{KeyType: "string", ValueType: "int32", Name: "counts", Number: 1},
			{KeyType: "int64", ValueType: "string", Name: "labels", Number: 2},
			{KeyType: "bool", ValueType: "string", Name: "flags", Number: 3},
		},
	}}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "JsonNode.Object(var _pb_counts_entries):")
	require.Contains(t, source, "for _pb_counts_key: String in _pb_counts_entries:")
	require.Contains(t, source, "counts[_pb_counts_key] = _pb_counts_value")
	require.Contains(t, source, "_pb_json_read_int64(JsonNode.Str(_pb_labels_key)")
	require.Contains(t, source, `if _pb_flags_key == "true":`)
	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_TYPE_MISMATCH: a bool map key takes \"true\" or \"false\"", _pb_flags_key_path)`)
}

// proto3 JSON has no merge semantics for a repeated key, so a document naming
// one logical field twice is refused rather than resolved by member order.
func TestJSONDecodeRefusesOneFieldGivenTwice(t *testing.T) {
	source := jsonPlayerSource(t,
		&protoast.Field{FieldType: "string", Name: "display_name", Number: 1},
		&protoast.Field{FieldType: "string", Name: "name", Number: 2},
	)

	require.Contains(t, source, "var _pb_display_name_seen: bool = false")
	require.Contains(t, source, "if _pb_display_name_seen:")
	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_PARSE_FAILED: display_name was given more than once", _pb_member_path)`)
	require.Contains(t, source, "_pb_display_name_seen = true")
	// A field whose two spellings agree cannot be named twice by one object.
	require.NotContains(t, source, "_pb_name_seen")
}

// A oneof holds one member, so a document setting two of them has no reading
// that does not silently discard one.
func TestJSONDecodeRefusesTwoMembersOfOneOneof(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name: "payload",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "text", Number: 1},
				{FieldType: "int32", Name: "amount", Number: 2},
			},
		}},
	}}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "var _pb_payload_seen: bool = false")
	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_PARSE_FAILED: payload has more than one member set", _pb_member_path)`)
}

// A oneof member is an ordinary JSON member; reading it sets the union case.
func TestJSONDecodeSetsTheOneofCaseItReads(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name: "Player",
		Oneofs: []*protoast.Oneof{{
			Name: "payload",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "text", Number: 1},
				{FieldType: "int32", Name: "amount", Number: 2},
			},
		}},
	}}, nil), "player.proto")
	source := files["cafecito/game/v1/Player.pb.fs"]

	require.Contains(t, source, "payload = PlayerPayloadCase.Text(_pb_payload_text_value)")
	require.Contains(t, source, "payload = PlayerPayloadCase.Amount(_pb_payload_amount_value)")
}

// Canonical JSON writes an enum as its case name, so the enum hosts the
// conversion back for the same reason it hosts from_wire.
func TestJSONDecodeReadsAnEnumFromANameOrANumber(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "Tier", Name: "tier", Number: 1}},
	}}, []*protoast.Enum{{
		Name: "Tier",
		Values: []*protoast.EnumValue{
			{Name: "TIER_BRONZE", Number: 0},
			{Name: "TIER_GOLD", Number: 1},
		},
	}}), "player.proto")

	enumSource := files["cafecito/game/v1/Tier.pb.fs"]
	require.Contains(t, enumSource, "static func from_json_name(name: String) -> Self?:")
	require.Contains(t, enumSource, "\t\t\t\"TIER_GOLD\":\n\t\t\t\treturn Tier.TIER_GOLD")

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "var _pb_tier_case: Tier? = Tier.from_json_name(_pb_tier_name)")
	// An unrecognized number takes the default; an unrecognized name does not,
	// because a name that no case answers to is a typo, not a newer schema.
	require.Contains(t, source, "var _pb_tier_wire: Tier? = Tier.from_wire(_pb_tier_number)")
	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: Tier has no case with this JSON name", _pb_member_path)`)
}

func TestEnumJSONNameDecodeIsAbsentWithoutTheOption(t *testing.T) {
	files := generate(t, namespacedFile(nil, []*protoast.Enum{{
		Name:   "Tier",
		Values: []*protoast.EnumValue{{Name: "TIER_BRONZE", Number: 0}},
	}}))

	require.NotContains(t, files["cafecito/game/v1/Tier.pb.fs"], "from_json_name")
}

func TestWellKnownTimestampAndDurationDecodeThroughTheirRuntimeHelpers(t *testing.T) {
	secondsAndNanos := []*protoast.Field{
		{FieldType: "int64", Name: "seconds", Number: 1},
		{FieldType: "int32", Name: "nanos", Number: 2},
	}

	timestamp := wellKnownSource(t, "google/protobuf/timestamp.proto", "Timestamp",
		&protoast.Message{Name: "Timestamp", Fields: secondsAndNanos})
	require.Contains(t, timestamp, "var (_pb_seconds, _pb_nanos, _pb_error) = JsonTimestamp.parse(_pb_text)")
	require.Contains(t, timestamp, "seconds = _pb_seconds")
	require.Contains(t, timestamp, "nanos = _pb_nanos")
	require.Contains(t, timestamp,
		`return JsonDecodeError.create("JSON_VALUE_OUT_OF_RANGE: Timestamp cannot be decoded from this JSON string", "$")`)
	require.Contains(t, timestamp,
		`return JsonDecodeError.create("JSON_TYPE_MISMATCH: Timestamp expects a JSON string", "$")`)

	duration := wellKnownSource(t, "google/protobuf/duration.proto", "Duration",
		&protoast.Message{Name: "Duration", Fields: secondsAndNanos})
	require.Contains(t, duration, "var (_pb_seconds, _pb_nanos, _pb_error) = JsonDuration.parse(_pb_text)")
}

func TestWellKnownFieldMaskDecodesThroughItsRuntimeHelper(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/field_mask.proto", "FieldMask",
		&protoast.Message{Name: "FieldMask", Fields: []*protoast.Field{
			{FieldType: "string", Name: "paths", Number: 1, Repeated: true},
		}})

	require.Contains(t, source, "var (_pb_paths, _pb_error) = JsonFieldMask.from_json(_pb_text)")
	require.Contains(t, source, "paths = _pb_paths")
}

// A wrapper's JSON form is the bare scalar, so the whole document is the value.
func TestWellKnownWrappersDecodeTheBareScalar(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/wrappers.proto", "Int32Value",
		&protoast.Message{Name: "Int32Value", Fields: []*protoast.Field{
			{FieldType: "int32", Name: "value", Number: 1},
		}})

	require.Contains(t, source, `var (_pb_value_value, _pb_value_error) = _pb_json_read_int32(_pb_node, "$")`)
	require.Contains(t, source, "value = _pb_value_value")
}

// Empty has no fields, so every member of its document is unknown and there is
// no key table to match against.
func TestWellKnownEmptyDecodesAnEmptyObject(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/empty.proto", "Empty",
		&protoast.Message{Name: "Empty"})

	require.Contains(t, source, "func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?:")
	require.Contains(t, source,
		`return JsonDecodeError.create("JSON_UNKNOWN_FIELD: Empty has no field named " + _pb_key, _pb_member_path)`)
	require.NotContains(t, source, "match _pb_key:")
}

func TestWellKnownStructListValueAndValueDecodeFromPlainJSON(t *testing.T) {
	files := generateJSON(t, namespacedFile([]*protoast.Message{
		{Name: "Struct", Maps: []*protoast.MapField{
			{KeyType: "string", ValueType: "Value", Name: "fields", Number: 1},
		}},
		{Name: "ListValue", Fields: []*protoast.Field{
			{FieldType: "Value", Name: "values", Number: 1, Repeated: true},
		}},
		{Name: "Value", Oneofs: []*protoast.Oneof{{
			Name: "kind",
			Fields: []*protoast.Field{
				{FieldType: "NullValue", Name: "null_value", Number: 1},
				{FieldType: "double", Name: "number_value", Number: 2},
				{FieldType: "string", Name: "string_value", Number: 3},
				{FieldType: "bool", Name: "bool_value", Number: 4},
				{FieldType: "Struct", Name: "struct_value", Number: 5},
				{FieldType: "ListValue", Name: "list_value", Number: 6},
			},
		}}},
	}, []*protoast.Enum{{
		Name:   "NullValue",
		Values: []*protoast.EnumValue{{Name: "NULL_VALUE", Number: 0}},
	}}), "google/protobuf/struct.proto")

	structSource := files["cafecito/game/v1/Struct.pb.fs"]
	require.Contains(t, structSource, "for _pb_fields_key: String in _pb_entries:")
	require.Contains(t, structSource, "fields[_pb_fields_key] = _pb_fields_value")

	listSource := files["cafecito/game/v1/ListValue.pb.fs"]
	require.Contains(t, listSource, "JsonNode.Array(var _pb_values_items):")
	require.Contains(t, listSource, "values.append(_pb_values_element_value)")

	// A Value maps case for case onto the engine's JsonNode, which is what
	// makes the two agree about what a JSON value is.
	valueSource := files["cafecito/game/v1/Value.pb.fs"]
	require.Contains(t, valueSource, "JsonNode.Null:\n\t\t\tkind = ValueKindCase.NullValue(NullValue.NULL_VALUE)")
	require.Contains(t, valueSource, "kind = ValueKindCase.BoolValue(_pb_bool)")
	// Value.number_value is a double, so a whole number lands there too.
	require.Contains(t, valueSource, "JsonNode.Int(var _pb_int):\n\t\t\tkind = ValueKindCase.NumberValue(_pb_int)")
	require.Contains(t, valueSource, "JsonNode.Float(var _pb_float):\n\t\t\tkind = ValueKindCase.NumberValue(_pb_float)")
	require.Contains(t, valueSource, "kind = ValueKindCase.StringValue(_pb_text)")
	require.Contains(t, valueSource, "ListValue.from_json(JsonNode.array_of(_pb_items))")
	require.Contains(t, valueSource, "Struct.from_json(JsonNode.object_of(_pb_object))")
}

// Any needs its type URL resolved to a generated binding, which needs a runtime
// type registry that does not exist yet, so the decode half says so too.
func TestWellKnownAnyReportsThatItCannotBeDecoded(t *testing.T) {
	source := wellKnownSource(t, "google/protobuf/any.proto", "Any",
		&protoast.Message{Name: "Any", Fields: []*protoast.Field{
			{FieldType: "string", Name: "type_url", Number: 1},
			{FieldType: "bytes", Name: "value", Number: 2},
		}})

	require.Contains(t, source, `return JsonDecodeError.create("JSON_ANY_UNSUPPORTED: `)
	require.NotContains(t, source, "_pb_json_read_bytes")
}
