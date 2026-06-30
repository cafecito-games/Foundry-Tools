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
	require.NoError(t, ValidateNamespace(""))
	require.NoError(t, ValidateNamespace("cafecito.game_v1"))
	require.Error(t, ValidateNamespace("cafecito..game"))
	require.Error(t, ValidateNamespace("cafecito.1game"))
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
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "final class_name Player extends RefCounted uses foundry.proto.Message[Player]")
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var _name: String = \"\"")
	require.Contains(t, files["cafecito/game/v1/PlayerStatus.pb.fs"], "enum_name PlayerStatus")
	require.NoError(t, CheckPublicAPI(files["cafecito/game/v1/Player.pb.fs"]))
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
	require.Contains(t, source, "static func from_bytes(data: PackedByteArray) -> foundry.proto.DecodeResult[Player]:")
	require.NotContains(t, source, "Variant")
}
