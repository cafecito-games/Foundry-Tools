package fsgenerator

import (
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

func generate(t *testing.T, file *protoast.ProtoFile) GeneratedFiles {
	t.Helper()
	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	return files
}

func playerSource(t *testing.T, fields []*protoast.Field) string {
	t.Helper()
	files := generate(t, namespacedFile([]*protoast.Message{{Name: "Player", Fields: fields}}, nil))
	return files["cafecito/game/v1/Player.pb.fs"]
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
	require.NotContains(t, source, "var _name")
}

func TestGenerateDecodeFactoryReturnsTuple(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})
	require.Contains(t, source, "static func from_bytes(data: PackedByteArray) -> (Player?, ProtobufError):")
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
	require.Contains(t, source, "var tag_read: VarintRead = Wire.decode_varint(data, offset)")
	require.Contains(t, source, "var name_read: StringRead = Wire.decode_string(")
	require.Contains(t, source, "var avatar_read: BytesRead = Wire.decode_bytes(")
}

// Unknown-field skipping is invariant across every message, so it lives in the
// runtime rather than being inlined into each binding.
func TestGenerateDelegatesUnknownFieldSkipToRuntime(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}})
	require.Contains(t, source, "var skipped: SkipRead = Wire.skip_field(data, offset, wire_type)")
	require.NotContains(t, source, "Wire.WIRE_32BIT:")
	require.NotContains(t, source, "offset += 8")
}

func TestGenerateRepeatedFields(t *testing.T) {
	source := playerSource(t, []*protoast.Field{
		{FieldType: "string", Name: "tags", Number: 1, Repeated: true},
		{FieldType: "int32", Name: "scores", Number: 2, Repeated: true},
	})

	require.Contains(t, source, "var tags: Array[String] = []")
	require.Contains(t, source, "var scores: Array[int] = []")
	require.Contains(t, source, "for tags_item: String in tags:")
	require.Contains(t, source, "tags.append(tags_read.value)")

	// Varint scalars pack; length-delimited ones cannot.
	require.Contains(t, source, "var scores_data: PackedByteArray = PackedByteArray()")
	require.Contains(t, source, "for scores_item: int in scores:")
	// A packed field must still decode the unpacked encoding.
	require.Contains(t, source, "if wire_type == Wire.WIRE_LENGTH_DELIMITED:")
	require.Contains(t, source, "elif wire_type == Wire.WIRE_VARINT:")
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
	})
	require.NotContains(t, source, "!= null")
	require.Contains(t, source, "if nickname is String:")
	require.Contains(t, source, "if primary is Slot:")
}

// A message-typed field is a length-delimited submessage, not a varint.
func TestGenerateMessageTypedFields(t *testing.T) {
	source := playerSource(t, []*protoast.Field{{FieldType: "Slot", Name: "primary", Number: 1}})

	require.Contains(t, source, "var primary: Slot? = null")
	require.NotContains(t, source, "var primary: Slot = 0")
	require.Contains(t, source, "var primary_data: PackedByteArray = primary.to_bytes()")
	// Field 1, wire type 2.
	require.Contains(t, source, "Wire.encode_varint(10)")
	require.Contains(t, source, "var primary_message: Slot = Slot.new()")
	require.Contains(t, source, "merge_from_bytes(data.slice(offset, offset + primary_length.value))")
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
	require.Contains(t, enumSource, "\tfunc to_wire() -> int:")
	require.Contains(t, enumSource, "\tstatic func from_wire(value: int) -> PlayerStatus:")
	require.Contains(t, enumSource, "\t\t\treturn PlayerStatus.PLAYER_STATUS_ONLINE")

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, messageSource, "var status: PlayerStatus = PlayerStatus.PLAYER_STATUS_UNSPECIFIED")
	require.Contains(t, messageSource, "Wire.encode_varint(status.to_wire())")
	require.Contains(t, messageSource, "status = PlayerStatus.from_wire(status_read.value)")
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
	require.Contains(t, source, "for counts_key: String in counts:")
	// A map entry is a submessage of key = 1, value = 2.
	require.Contains(t, source, "counts_entry.append_array(Wire.encode_varint(10))")
	require.Contains(t, source, "counts_entry.append_array(Wire.encode_varint(16))")
	require.Contains(t, source, "counts[counts_key] = counts_value")
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
	require.Contains(t, source, "PlayerPayloadCase.Text(var text):")
	require.Contains(t, source, "payload = PlayerPayloadCase.Amount(amount_read.value)")
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

	require.Contains(t, source, "class Badge extends RefCounted uses Message:")
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
