package fsgenerator

import (
	"strings"
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestGenerateAggregatesCurrentFileCollisions(t *testing.T) {
	file := namespacedFile(
		[]*protoast.Message{{
			Position: protoast.Position{Line: 4, Column: 1},
			Name:     "Node",
			NestedMessages: []*protoast.Message{{
				Position: protoast.Position{Line: 6, Column: 3},
				Name:     "Timer",
			}},
		}},
		[]*protoast.Enum{{
			Position: protoast.Position{Line: 9, Column: 1},
			Name:     "String",
			Values:   []*protoast.EnumValue{{Name: "STRING_UNSPECIFIED", Number: 0}},
		}},
	)

	files, err := Generate(file, "types.proto", nil)
	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	node := `types.proto:4:1: message cafecito.game.v1.Node generates Foundry type "Node", which conflicts with native class "Node"`
	timer := `types.proto:6:3: message cafecito.game.v1.Node.Timer generates Foundry type "Timer", which conflicts with native class "Timer"`
	stringType := `types.proto:9:1: enum cafecito.game.v1.String generates Foundry type "String", which conflicts with built-in type "String"`
	require.Contains(t, diagnostic, node)
	require.Contains(t, diagnostic, timer)
	require.Contains(t, diagnostic, stringType)
	require.Less(t, strings.Index(diagnostic, node), strings.Index(diagnostic, timer))
	require.Less(t, strings.Index(diagnostic, timer), strings.Index(diagnostic, stringType))
	require.Contains(t, diagnostic, "set a non-empty file option such as:")
	require.Contains(t, diagnostic, `option (foundrytools.type_prefix) = "Game";`)
}

func TestPrefixResolvesEngineTypeCollisions(t *testing.T) {
	files, err := Generate(prefixedFile(
		"Game",
		[]*protoast.Message{{Name: "Node"}},
		[]*protoast.Enum{{
			Name:   "String",
			Values: []*protoast.EnumValue{{Name: "STRING_UNSPECIFIED", Number: 0}},
		}},
	), "types.proto", nil)
	require.NoError(t, err)

	require.Contains(t, files, "cafecito/game/v1/GameNode.pb.fs")
	require.Contains(t, files["cafecito/game/v1/GameNode.pb.fs"], "class_name GameNode")
	require.Contains(t, files, "cafecito/game/v1/GameString.pb.fs")
	require.Contains(t, files["cafecito/game/v1/GameString.pb.fs"], "enum_name GameString")
}

func TestPrefixCanStillProduceEngineTypeCollision(t *testing.T) {
	files, err := Generate(prefixedFile(
		"Animation",
		[]*protoast.Message{{Name: "Node"}},
		nil,
	), "types.proto", nil)
	require.Error(t, err)
	require.Nil(t, files)
	require.Contains(t, err.Error(),
		`message cafecito.game.v1.Node generates Foundry type "AnimationNode", which conflicts with native class "AnimationNode"`)
	require.Contains(t, err.Error(), `current prefix "Animation" still produces reserved Foundry type names`)
	require.Contains(t, err.Error(), "change it")
}

func TestInternalNonExposedClassRemainsLegal(t *testing.T) {
	files, err := Generate(namespacedFile(
		[]*protoast.Message{{Name: "FSNativeClass"}},
		nil,
	), "types.proto", nil)
	require.NoError(t, err)
	require.Contains(t, files, "cafecito/game/v1/FSNativeClass.pb.fs")
	require.Contains(t, files["cafecito/game/v1/FSNativeClass.pb.fs"], "class_name FSNativeClass")
}

func TestReferencedDependencyCollisionIsDeduplicatedAndLazy(t *testing.T) {
	inventory := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name: "Node",
		}},
	}
	local := namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Node", Name: "first", Number: 1, SourceFile: "inventory.proto"},
			{FieldType: "Node", Name: "second", Number: 2, SourceFile: "inventory.proto"},
		},
	}}, nil)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}})
	require.Error(t, err)
	require.Nil(t, files)
	require.Equal(t, 1, strings.Count(err.Error(),
		`inventory.proto: message cafecito.inventory.v1.Node generates Foundry type "Node"`))
	require.Contains(t, err.Error(), "set or change (foundrytools.type_prefix) in inventory.proto")

	unreferenced := namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}},
	}}, nil)
	files, err = Generate(unreferenced, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}})
	require.NoError(t, err)
	require.Contains(t, files, "cafecito/game/v1/Player.pb.fs")
}

func TestReferencedDependencyCollisionsSortAcrossSourceFiles(t *testing.T) {
	inventory := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name: "Node",
		}},
	}
	local := namespacedFile(
		[]*protoast.Message{
			{
				Name:   "Player",
				Fields: []*protoast.Field{{FieldType: "Node", Name: "node", Number: 1, SourceFile: "inventory.proto"}},
			},
			{Name: "Timer"},
		},
		nil,
	)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}})
	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	imported := `inventory.proto: message cafecito.inventory.v1.Node generates Foundry type "Node"`
	localCollision := `player.proto: message cafecito.game.v1.Timer generates Foundry type "Timer"`
	require.Contains(t, diagnostic, imported)
	require.Contains(t, diagnostic, localCollision)
	require.Less(t, strings.Index(diagnostic, imported), strings.Index(diagnostic, localCollision))
}

func TestGenerateReportsBuiltInCollisionWithoutUnknownCoordinates(t *testing.T) {
	files, err := Generate(namespacedFile(nil, []*protoast.Enum{{
		Name:   "AsyncCallable",
		Values: []*protoast.EnumValue{{Name: "ASYNC_CALLABLE_UNSPECIFIED", Number: 0}},
	}}), "types.proto", nil)
	require.Error(t, err)
	require.Nil(t, files)
	require.Contains(t, err.Error(),
		`types.proto: enum cafecito.game.v1.AsyncCallable generates Foundry type "AsyncCallable", which conflicts with built-in type "AsyncCallable"`)
	require.NotContains(t, err.Error(), "types.proto:0:0")
}
