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

func TestPlanMemberNameEscapesEngineTypes(t *testing.T) {
	tests := []struct {
		raw       string
		generated string
		kind      memberEscapeKind
		reason    string
	}{
		{raw: "String", generated: "String_", kind: memberEscapeEngineBuiltin, reason: `built-in type "String"`},
		{raw: "AsyncCallable", generated: "AsyncCallable_", kind: memberEscapeEngineBuiltin, reason: `built-in type "AsyncCallable"`},
		{raw: "Node", generated: "Node_", kind: memberEscapeEngineNative, reason: `native class "Node"`},
		{raw: "Timer", generated: "Timer_", kind: memberEscapeEngineNative, reason: `native class "Timer"`},
		{raw: "node", generated: "node", kind: memberEscapeNone},
		{raw: "string", generated: "string", kind: memberEscapeNone},
		{raw: "Node_", generated: "Node_", kind: memberEscapeNone},
	}

	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got := planMemberName(test.raw)
			require.Equal(t, test.generated, got.Generated)
			require.Equal(t, test.kind, got.Escape.Kind)
			require.Equal(t, test.reason, got.Escape.Description())
			require.Equal(t, test.generated, FieldName(test.raw))
		})
	}
}

func TestPlanMemberNameKeepsExistingEscapePolicies(t *testing.T) {
	tests := []struct {
		raw       string
		generated string
		kind      memberEscapeKind
	}{
		{raw: "var", generated: "var_", kind: memberEscapeKeyword},
		{raw: "to_bytes", generated: "to_bytes_", kind: memberEscapeGenerated},
		{raw: "merge_from_bytes", generated: "merge_from_bytes_", kind: memberEscapeGenerated},
		{raw: "plain", generated: "plain", kind: memberEscapeNone},
	}
	for _, test := range tests {
		got := planMemberName(test.raw)
		require.Equal(t, test.generated, got.Generated)
		require.Equal(t, test.kind, got.Escape.Kind)
	}
}

func TestPlanOneofAlternativeNameSkipsEngineTypes(t *testing.T) {
	require.Equal(t, "Image", planOneofAlternativeName("Image").Generated)
	require.Equal(t, memberEscapeNone, planOneofAlternativeName("Image").Escape.Kind)
	require.Equal(t, "var_", planOneofAlternativeName("var").Generated)
	require.Equal(t, memberEscapeKeyword, planOneofAlternativeName("var").Escape.Kind)
}
