package fsgenerator

import (
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestNewTypeNamerValidatesLiteralPrefix(t *testing.T) {
	position := protoast.Position{Line: 4, Column: 1}
	file := &protoast.ProtoFile{
		Options:         map[string]any{typePrefixOptionKey: "Game_"},
		OptionPositions: map[string]protoast.Position{typePrefixOptionKey: position},
	}

	namer, err := newTypeNamer(file, "types.proto")
	require.NoError(t, err)
	require.Equal(t, "Game_Node", namer.Name("node"))

	for _, raw := range []any{"", "game-tools", "game tools", "game.tools", "2D", int64(3)} {
		t.Run("invalid", func(t *testing.T) {
			file.Options[typePrefixOptionKey] = raw

			_, err := newTypeNamer(file, "types.proto")
			require.Error(t, err)
			require.ErrorContains(t, err, "types.proto:4:1")
			require.ErrorContains(t, err, typePrefixOptionKey)
		})
	}
}

func TestNewTypeNamerAllowsAbsentAndIdentifierFragmentPrefixes(t *testing.T) {
	for _, file := range []*protoast.ProtoFile{
		nil,
		{},
		{Options: map[string]any{namespaceOptionKey: "game"}},
	} {
		namer, err := newTypeNamer(file, "types.proto")
		require.NoError(t, err)
		require.Equal(t, "Node", namer.Name("node"))
	}

	for _, prefix := range []string{"_", "A2_"} {
		namer, err := newTypeNamer(&protoast.ProtoFile{
			Options: map[string]any{typePrefixOptionKey: prefix},
		}, "types.proto")
		require.NoError(t, err)
		require.Equal(t, prefix+"Node", namer.Name("node"))
	}
}

func TestNewTypeNamerOmitsUnknownOptionPosition(t *testing.T) {
	file := &protoast.ProtoFile{
		Options: map[string]any{typePrefixOptionKey: ""},
	}

	_, err := newTypeNamer(file, "types.proto")
	require.Error(t, err)
	require.ErrorContains(t, err, "types.proto: error: "+typePrefixOptionKey)
	require.NotContains(t, err.Error(), ":0:0")
}

func TestTypeNamerPrefixesBeforeEscaping(t *testing.T) {
	namer := typeNamer{prefix: "Game"}

	require.Equal(t, "GameClass", namer.Name("class"))
	require.Equal(t, "GameMessage", namer.Name("message"))
	require.Equal(t, "GameOuter.GameInner", namer.Reference("outer.inner"))
	require.Equal(t, "GameOuter.GameInner", namer.Reference(".outer.inner"))
	require.Equal(t, "Class_", TypeName("class"))
	require.Equal(t, "Message_", TypeName("message"))
}
