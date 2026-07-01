package fsgenerator

import (
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
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
	file := &protoast.ProtoFile{
		Syntax: "proto3",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				FieldType: "string",
				Name:      "name",
				Number:    1,
			}},
		}},
	}

	_, err := Generate(file, "player.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "namespace is required")
}

func TestTypeName(t *testing.T) {
	require.Equal(t, "PlayerState", TypeName("player_state"))
	require.Equal(t, "PlayerState", TypeName("player-state"))
	require.Equal(t, "OuterInner", TypeName("outer.inner"))
	require.Equal(t, "Class_", TypeName("class"))
}

func TestScalarTypeMapping(t *testing.T) {
	require.Equal(t, "int", ScalarType("int32").Render())
	require.Equal(t, "float", ScalarType("double").Render())
	require.Equal(t, "String", ScalarType("string").Render())
	require.Equal(t, "PackedByteArray", ScalarType("bytes").Render())
}

func TestGenerateMessageAndEnumSkeletons(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				FieldType: "string",
				Name:      "name",
				Number:    1,
			}},
		}},
		Enums: []*protoast.Enum{{
			Name: "PlayerStatus",
			Values: []*protoast.EnumValue{
				{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0},
				{Name: "PLAYER_STATUS_ONLINE", Number: 1},
			},
		}},
	}

	files, err := Generate(file, "examples/example.proto", nil)
	require.NoError(t, err)
	require.Len(t, files, 2)
	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	enumSource := files["cafecito/game/v1/PlayerStatus.pb.fs"]
	require.Contains(t, messageSource, "## Generated protobuf message binding for Player.\nfinal class_name Player extends RefCounted")
	require.Contains(t, enumSource, "## Generated protobuf enum binding for PlayerStatus.\nenum_name PlayerStatus")
	require.NotContains(t, messageSource, "## var _name")
	require.Contains(t, messageSource, "final class_name Player extends RefCounted")
	require.NotContains(t, messageSource, " uses ")
	require.Contains(t, messageSource, "var _name: String = \"\"")
	require.Contains(t, enumSource, "enum_name PlayerStatus")
	require.NoError(t, CheckPublicAPI(messageSource))
}

func TestGeneratePrefersSchemaDocs(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Doc:  []string{"Schema-authored player docs.", "Shown in generated API docs."},
			Name: "Player",
			Fields: []*protoast.Field{{
				Doc:       []string{"Display name.", "Stored player name."},
				FieldType: "string",
				Name:      "name",
				Number:    1,
			}},
		}},
		Enums: []*protoast.Enum{{
			Doc:  []string{"Schema-authored status docs."},
			Name: "PlayerStatus",
			Values: []*protoast.EnumValue{{
				Doc:    []string{"Unknown status."},
				Name:   "PLAYER_STATUS_UNSPECIFIED",
				Number: 0,
			}},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)

	messageSource := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, messageSource, "## Schema-authored player docs.\n## Shown in generated API docs.\nfinal class_name Player extends RefCounted")
	require.Contains(t, messageSource, "## Display name.\n## Stored player name.\nfunc set_name(value: String) -> void:")
	require.Contains(t, messageSource, "## Display name.\n## Stored player name.\nfunc get_name() -> String:")
	require.NotContains(t, messageSource, "Generated protobuf message binding for Player.")
	require.NotContains(t, messageSource, "Sets the name protobuf field.")

	enumSource := files["cafecito/game/v1/PlayerStatus.pb.fs"]
	require.Contains(t, enumSource, "## Schema-authored status docs.\nenum_name PlayerStatus")
	require.Contains(t, enumSource, "\t## Unknown status.\n\tPLAYER_STATUS_UNSPECIFIED = 0,")
	require.NotContains(t, enumSource, "Generated protobuf enum binding for PlayerStatus.")
}

func TestGenerateTypedAccessorsAndDecodeFactory(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "name", Number: 1},
				{FieldType: "int32", Name: "level", Number: 2},
			},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "func set_name(value: String) -> void:")
	require.Contains(t, source, "func get_name() -> String:")
	require.Contains(t, source, "func set_level(value: int) -> void:")
	require.Contains(t, source, "static func from_bytes(data: PackedByteArray) -> DecodeResult[Player]:")
	require.Contains(t, source, "## Sets the name protobuf field.\nfunc set_name(value: String) -> void:")
	require.Contains(t, source, "## Returns the name protobuf field.\nfunc get_name() -> String:")
	require.Contains(t, source, "## Sets the level protobuf field.\nfunc set_level(value: int) -> void:")
	require.Contains(t, source, "## Decodes protobuf wire data into a new Player message.\nstatic func from_bytes(data: PackedByteArray) -> DecodeResult[Player]:")
	require.Contains(t, source, "var tag_read: FieldRead[int] =")
	require.Contains(t, source, "var string_read: FieldRead[String] =")
	require.NotContains(t, source, "-> foundry.proto.DecodeResult[")
	require.NotContains(t, source, ": foundry.proto.FieldRead[")
	require.NotContains(t, source, "Variant")
}

func TestGenerateScalarSerialization(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "name", Number: 1},
				{FieldType: "int32", Name: "level", Number: 2},
			},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "foundry.proto.Wire.encode_varint(10)")
	require.Contains(t, source, "foundry.proto.Wire.encode_string(_name)")
	require.Contains(t, source, "foundry.proto.Wire.encode_varint(16)")
	require.Contains(t, source, "match field_number:")
	require.Contains(t, source, "if wire_type != foundry.proto.Wire.WIRE_LENGTH_DELIMITED:")
	require.Contains(t, source, "if wire_type != foundry.proto.Wire.WIRE_VARINT:")
	require.Contains(t, source, "if length_read.value < 0 or offset + length_read.value > data.size():")
	require.Contains(t, source, "_name = string_read.value")
	require.Contains(t, source, "_level = value_read.value")
	require.Contains(t, source, "## Serializes this message to protobuf wire data.\nfunc to_bytes() -> PackedByteArray:")
	require.Contains(t, source, "## Merges protobuf wire data into this message.\nfunc merge_from_bytes(data: PackedByteArray) -> ProtobufError:")
	require.NotContains(t, source, "func merge_from_bytes(data: PackedByteArray) -> foundry.proto.ProtobufError:")
}

func TestGenerateBoolAndBytesWireCode(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{
				{FieldType: "bool", Name: "active", Number: 1},
				{FieldType: "bytes", Name: "avatar", Number: 2},
			},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "var _active: bool = false")
	require.Contains(t, source, "result.append_array(foundry.proto.Wire.encode_varint(1 if _active else 0))")
	require.Contains(t, source, "if wire_type != foundry.proto.Wire.WIRE_VARINT:")
	require.Contains(t, source, "_active = value_read.value != 0")
	require.NotContains(t, source, "_active = value_read.value\n")
	require.Contains(t, source, "var _avatar: PackedByteArray = PackedByteArray()")
	require.Contains(t, source, "result.append_array(foundry.proto.Wire.encode_varint(_avatar.size()))")
	require.Contains(t, source, "if wire_type != foundry.proto.Wire.WIRE_LENGTH_DELIMITED:")
	require.Contains(t, source, "if length_read.value < 0 or offset + length_read.value > data.size():")
	require.Contains(t, source, "var bytes_read: FieldRead[PackedByteArray] = foundry.proto.Wire.decode_bytes(data, offset, length_read.value)")
	require.Contains(t, source, "_avatar = bytes_read.value")
	require.NotContains(t, source, "_avatar = value_read.value")
}

func TestGenerateUnknownFieldSkipsByWireType(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				FieldType: "string",
				Name:      "name",
				Number:    1,
			}},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "var wire_type: int = foundry.proto.Wire.get_wire_type(tag_read.value)")
	require.Contains(t, source, "match wire_type:")
	require.Contains(t, source, "foundry.proto.Wire.WIRE_VARINT:")
	require.Contains(t, source, "foundry.proto.Wire.WIRE_LENGTH_DELIMITED:")
	require.Contains(t, source, "if length_read.value < 0 or offset + length_read.value > data.size():")
	require.Contains(t, source, "foundry.proto.Wire.WIRE_32BIT:")
	require.Contains(t, source, "foundry.proto.Wire.WIRE_64BIT:")
	require.Contains(t, source, "offset += 4")
	require.Contains(t, source, "offset += 8")
	require.NotContains(t, source, "_:\n\t\t\t\treturn foundry.proto.ProtobufError.UNKNOWN_REQUIRED_FEATURE")
}

func TestGenerateUnsupportedWireScalarsReturnsError(t *testing.T) {
	for _, scalar := range []string{"float", "double", "fixed32", "fixed64", "sfixed32", "sfixed64", "sint32", "sint64"} {
		t.Run(scalar, func(t *testing.T) {
			file := &protoast.ProtoFile{
				Syntax:  "proto3",
				Package: "cafecito.game.v1",
				Messages: []*protoast.Message{{
					Name: "Player",
					Fields: []*protoast.Field{{
						FieldType: scalar,
						Name:      "score",
						Number:    1,
					}},
				}},
			}

			_, err := Generate(file, "player.proto", nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unsupported scalar type "+scalar+" for wire generation")
		})
	}
}
