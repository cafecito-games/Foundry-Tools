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
				Uses:    []string{"Message"},
				Members: []Node{
					Var{Name: "_name", Type: fstypes.Named("String"), Value: `""`},
					Func{
						Doc:        []string{"Returns the name protobuf field."},
						Name:       "get_name",
						ReturnType: fstypes.Named("String"),
						Body:       []Node{Return{Value: "_name"}},
					},
					Raw{Code: "const VERSION: int = 1\n"},
				},
			},
		},
	}

	require.Equal(t, "namespace cafecito.game.v1\nimport foundry.proto\n\n## Generated protobuf message binding for Player.\nfinal class_name Player extends RefCounted uses Message\n\nvar _name: String = \"\"\n\n## Returns the name protobuf field.\nfunc get_name() -> String:\n\treturn _name\n\nconst VERSION: int = 1\n", file.Render())
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

// An inner class uses the plain form with an indented body, unlike the
// file-level class_name form whose members are unindented.
func TestInnerClassRendering(t *testing.T) {
	inner := Class{
		Inner:   true,
		Name:    "Badge",
		Extends: "RefCounted",
		Uses:    []string{"Message"},
		Members: []Node{Var{Name: "code", Type: fstypes.Named("String"), Value: `""`}},
	}

	require.Equal(t, "class Badge extends RefCounted uses Message:\n\tvar code: String = \"\"\n", inner.RenderAt(0))
	require.Equal(t, "class Empty:\n\tpass\n", Class{Inner: true, Name: "Empty"}.RenderAt(0))
}

// Tagged-union cases are ordinal by declaration order and take no value.
func TestTaggedUnionRendering(t *testing.T) {
	union := Enum{
		Name: "PayloadCase",
		Values: []EnumValue{
			{Name: "Text", Payload: []Parameter{{Name: "text", Type: fstypes.Named("String")}}},
			{Name: "Amount", Payload: []Parameter{{Name: "amount", Type: fstypes.Named("int")}}},
		},
	}

	require.Equal(t, "enum_name PayloadCase:\n\tText(text: String)\n\tAmount(amount: int)\n", union.RenderAt(0))
}

func TestEnumHostedFunctionRendering(t *testing.T) {
	enum := Enum{
		Inner:   true,
		Name:    "Tier",
		Values:  []EnumValue{{Name: "TIER_GOLD", Number: 1}},
		Members: []Node{Func{Name: "to_wire", ReturnType: fstypes.Named("int"), Body: []Node{Return{Value: "1"}}}},
	}

	require.Equal(t, "enum Tier:\n\tTIER_GOLD = 1\n\n\tfunc to_wire() -> int:\n\t\treturn 1\n", enum.RenderAt(0))
}

func TestLineRendersAtRelativeDepth(t *testing.T) {
	require.Equal(t, "\t\t\tdeep()\n", Line{Depth: 2, Code: "deep()"}.RenderAt(1))
}

func TestExprRendering(t *testing.T) {
	require.Equal(t, "\t\tdo_work()\n", Expr{Code: "do_work()"}.RenderAt(2))
}
