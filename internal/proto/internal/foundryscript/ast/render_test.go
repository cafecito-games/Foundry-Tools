package fsast

import (
	"testing"

	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
	"github.com/stretchr/testify/require"
)

func TestFileRendering(t *testing.T) {
	file := File{
		Namespace: "cafecito.game.v1",
		Imports:   []string{"foundry.proto"},
		Declarations: []Node{
			Class{
				Doc:     []string{"Generated protobuf message binding for Player."},
				Final:   true,
				Name:    "Player",
				Extends: "RefCounted",
				Uses:    []string{"foundry.proto.Message[Player]"},
				Members: []Node{
					Var{Name: "_name", Type: fstypes.Named("String"), Value: `""`},
					Func{
						Doc:        []string{"Returns the name protobuf field."},
						Name:       "get_name",
						ReturnType: fstypes.Named("String"),
						Body:       []Node{Return{Value: "_name"}},
					},
					Doc{Lines: []string{"Generated protobuf enum binding for PlayerStatus."}, Node: Raw{Code: "enum_name PlayerStatus {}\n"}},
				},
			},
		},
	}

	require.Equal(t, "namespace cafecito.game.v1\nimport foundry.proto\n\n## Generated protobuf message binding for Player.\nfinal class_name Player extends RefCounted uses foundry.proto.Message[Player]\n\nvar _name: String = \"\"\n\n## Returns the name protobuf field.\nfunc get_name() -> String:\n\treturn _name\n\n## Generated protobuf enum binding for PlayerStatus.\nenum_name PlayerStatus {}\n", file.Render())
}

func TestEnumRendering(t *testing.T) {
	enum := Enum{
		Doc:  []string{"Generated protobuf enum binding for PlayerStatus."},
		Name: "PlayerStatus",
		Values: []EnumValue{
			{Doc: []string{"Unknown status."}, Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0},
			{Name: "PLAYER_STATUS_ONLINE", Number: 1},
		},
	}

	require.Equal(t, "## Generated protobuf enum binding for PlayerStatus.\nenum_name PlayerStatus:\n\t## Unknown status.\n\tPLAYER_STATUS_UNSPECIFIED = 0\n\tPLAYER_STATUS_ONLINE = 1\n", enum.RenderAt(0))
}

func TestEnumWithoutValuesRendersPass(t *testing.T) {
	require.Equal(t, "enum_name Empty:\n\tpass\n", Enum{Name: "Empty"}.RenderAt(0))
}

func TestExprRendering(t *testing.T) {
	require.Equal(t, "\t\tdo_work()\n", Expr{Code: "do_work()"}.RenderAt(2))
}
