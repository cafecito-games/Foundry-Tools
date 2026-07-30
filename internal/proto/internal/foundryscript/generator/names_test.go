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

	tests := []struct {
		name string
		raw  any
		got  string
	}{
		{name: "empty", raw: "", got: `got ""`},
		{name: "hyphen", raw: "game-tools", got: `got "game-tools"`},
		{name: "space", raw: "game tools", got: `got "game tools"`},
		{name: "dot", raw: "game.tools", got: `got "game.tools"`},
		{name: "leading digit", raw: "2D", got: `got "2D"`},
		{name: "non-string integer", raw: int64(3), got: "got int64(3)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file.Options[typePrefixOptionKey] = test.raw

			_, err := newTypeNamer(file, "types.proto")
			require.EqualError(t, err,
				"types.proto:4:1: error: "+typePrefixOptionKey+
					" must be a non-empty identifier fragment, "+test.got)
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
	require.EqualError(t, err,
		`types.proto: error: `+typePrefixOptionKey+
			` must be a non-empty identifier fragment, got ""`)
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

func TestTypeNamerReferenceFiltersEmptySegments(t *testing.T) {
	namer := typeNamer{prefix: "Game"}

	tests := map[string]string{
		"":              "",
		".":             "",
		"outer..inner":  "GameOuter.GameInner",
		"..outer.inner": "GameOuter.GameInner",
	}
	for protoType, want := range tests {
		require.Equal(t, want, namer.Reference(protoType))
	}
}

func TestTypeNamerZeroValueMatchesLegacyHelpers(t *testing.T) {
	var namer typeNamer

	for _, name := range []string{"", "class", "message", "player_state", "outer-inner"} {
		require.Equal(t, TypeName(name), namer.Name(name))
	}
	for _, reference := range []string{"", ".", ".outer.inner", "outer..inner", "..outer.inner"} {
		require.Equal(t, TypeReference(reference), namer.Reference(reference))
	}
}
