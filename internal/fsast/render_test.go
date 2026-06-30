package fsast

import (
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/stretchr/testify/require"
)

func TestFileRendering(t *testing.T) {
	file := File{
		Namespace: "cafecito.game.v1",
		Imports:   []string{"foundry.proto"},
		Declarations: []Node{
			Class{
				Final:   true,
				Name:    "Player",
				Extends: "RefCounted",
				Uses:    []string{"foundry.proto.Message[Player]"},
				Members: []Node{
					Var{Name: "_name", Type: fstypes.Named("String"), Value: `""`},
					Func{
						Name:       "get_name",
						ReturnType: fstypes.Named("String"),
						Body:       []Node{Return{Value: "_name"}},
					},
				},
			},
		},
	}

	require.Equal(t, "namespace cafecito.game.v1\nimport foundry.proto\n\nfinal class_name Player extends RefCounted uses foundry.proto.Message[Player]\n\nvar _name: String = \"\"\n\nfunc get_name() -> String:\n\treturn _name\n", file.Render())
}

func TestExprRendering(t *testing.T) {
	require.Equal(t, "\t\tdo_work()\n", Expr{Code: "do_work()"}.RenderAt(2))
}
