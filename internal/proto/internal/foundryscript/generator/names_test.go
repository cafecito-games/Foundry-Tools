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
		{Options: map[string]any{NamespaceOptionKey: "game"}},
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
			require.Equal(t, test.reason, got.Escape.description())
			require.Equal(t, test.generated, FieldName(test.raw))
		})
	}
}

func TestPlanMemberNameEscapesReachableInheritedEngineMembers(t *testing.T) {
	for _, test := range []struct {
		name   string
		reason string
	}{
		{name: "reference", reason: `inherited method "RefCounted.reference"`},
		{name: "unreference", reason: `inherited method "RefCounted.unreference"`},
		{name: "get_class", reason: `inherited method "Object.get_class"`},
		{name: "script_changed", reason: `inherited signal "Object.script_changed"`},
		{name: "NOTIFICATION_PREDELETE", reason: `inherited constant "Object.NOTIFICATION_PREDELETE"`},
		{name: "ConnectFlags", reason: `inherited enum "Object.ConnectFlags"`},
		{name: "CONNECT_DEFERRED", reason: `inherited enum value "Object.CONNECT_DEFERRED"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			planned := planMemberName(test.name)
			require.Equal(t, test.name+"_", planned.Generated)
			require.Equal(t, test.reason, planned.Escape.description())
			require.Equal(t, test.name+"_", FieldName(test.name))
		})
	}

	// Native member lookup is exact-case, just like Foundry Script lookup.
	require.Equal(t, "Reference", FieldName("Reference"))
	require.Equal(t, "REFERENCE", FieldName("REFERENCE"))

	// These declarations do not inherit RefCounted and keep their own policy.
	require.Equal(t, "reference", planOneofAlternativeName("reference").Generated)
	require.Equal(t, "reference", EnumValueName("reference"))
}

func TestPlanMemberNameKeepsExistingEscapePolicies(t *testing.T) {
	tests := []struct {
		raw       string
		generated string
		kind      memberEscapeKind
		reason    string
	}{
		{raw: "var", generated: "var_", kind: memberEscapeKeyword, reason: "Foundry keyword"},
		{raw: "to_bytes", generated: "to_bytes_", kind: memberEscapeGenerated, reason: "generated member"},
		{raw: "merge_from_bytes", generated: "merge_from_bytes_", kind: memberEscapeGenerated, reason: "generated member"},
		{raw: "plain", generated: "plain", kind: memberEscapeNone},
	}
	for _, test := range tests {
		got := planMemberName(test.raw)
		require.Equal(t, test.generated, got.Generated)
		require.Equal(t, test.kind, got.Escape.Kind)
		require.Equal(t, test.reason, got.Escape.description())
	}
}

// The engine's JSON builtins are script classes, so extension_api.json does not
// describe them and the generated engine type table cannot carry them. A schema
// type spelled the same way would resolve ahead of the global class inside the
// very file that names it, so the escape has to be listed by hand.
func TestEngineJSONBuiltinTypeNamesAreEscaped(t *testing.T) {
	for _, name := range []string{"JsonNode", "JsonResult", "JsonDecodeError", "JsonSerializable"} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, name+"_", TypeName(name))
			require.Equal(t, name+"_", TypeReference(name))
			require.NotContains(t, foundryEngineReservedTypes, name,
				"the generated engine table now carries this name, so the hand-written escape is redundant")
		})
	}

	// A schema name that merely starts the same way keeps its spelling.
	require.Equal(t, "JsonNodeList", TypeName("JsonNodeList"))
}

// A field named after one of the JSON builtins shadows the global class inside
// the generated class body, exactly as a field named after a native class does.
func TestPlanMemberNameEscapesEngineJSONBuiltins(t *testing.T) {
	for _, name := range []string{"JsonNode", "JsonResult", "JsonDecodeError", "JsonSerializable"} {
		t.Run(name, func(t *testing.T) {
			got := planMemberName(name)
			require.Equal(t, name+"_", got.Generated)
			require.Equal(t, memberEscapeEngineBuiltin, got.Escape.Kind)
			require.Equal(t, `built-in type "`+name+`"`, got.Escape.description())
			require.Equal(t, name+"_", FieldName(name))
		})
	}
}

func TestGeneratedMethodNamesAreFreshAndReserved(t *testing.T) {
	want := []string{
		"create_message", "protobuf_type_name", "type_name",
		"from_bytes", "to_bytes", "merge_from_bytes", "to_json", "from_json",
	}
	require.Equal(t, "create_message", createMessageMethod)
	require.Equal(t, "protobuf_type_name", protobufTypeNameMethod)
	require.Equal(t, "type_name", typeNameMethod)
	require.Equal(t, "from_bytes", fromBytesMethod)
	require.Equal(t, "to_bytes", toBytesMethod)
	require.Equal(t, "merge_from_bytes", mergeFromBytesMethod)
	require.Equal(t, "to_json", toJSONMethod)
	require.Equal(t, "from_json", fromJSONMethod)
	require.Equal(t, want, generatedMethodNames())

	mutated := generatedMethodNames()
	mutated[0] = "changed"
	require.Equal(t, want, generatedMethodNames())

	for _, methodName := range want {
		require.True(t, generatedMemberNames[methodName])
	}
}

func TestPlanMemberNamePrefersExistingEscapePolicyOverEngineType(t *testing.T) {
	const name = "var"
	previous, existed := foundryEngineReservedTypes[name]
	foundryEngineReservedTypes[name] = engineTypeEntry{kind: engineTypeBuiltin}
	t.Cleanup(func() {
		if existed {
			foundryEngineReservedTypes[name] = previous
		} else {
			delete(foundryEngineReservedTypes, name)
		}
	})

	got := planMemberName(name)
	require.Equal(t, "var_", got.Generated)
	require.Equal(t, memberEscapeKeyword, got.Escape.Kind)
	require.Equal(t, "Foundry keyword", got.Escape.description())
}

func TestJSONFieldNameDerivesCamelCaseFromProtoName(t *testing.T) {
	tests := []struct {
		protoName string
		want      string
	}{
		{protoName: "player", want: "player"},
		{protoName: "player_id", want: "playerId"},
		{protoName: "player_health_total", want: "playerHealthTotal"},
		{protoName: "_leading", want: "Leading"},
		{protoName: "trailing_", want: "trailing"},
		{protoName: "double__underscore", want: "doubleUnderscore"},
		{protoName: "already_2fast", want: "already2fast"},
	}
	for _, test := range tests {
		t.Run(test.protoName, func(t *testing.T) {
			require.Equal(t, test.want, jsonFieldName(test.protoName, nil))
		})
	}
}

func TestJSONFieldNameHonoursExplicitJSONNameOption(t *testing.T) {
	options := map[string]any{"json_name": "customName"}
	require.Equal(t, "customName", jsonFieldName("player_id", options))

	// A field whose explicit json_name disagrees with the derived spelling
	// still wins with the explicit spelling.
	require.NotEqual(t, jsonFieldName("player_id", nil), jsonFieldName("player_id", options))
}

func TestJSONFieldNameIgnoresNonStringJSONNameOption(t *testing.T) {
	options := map[string]any{"json_name": int64(3)}
	require.Equal(t, jsonFieldName("player_id", nil), jsonFieldName("player_id", options))
}

func TestJSONFieldNameHonoursExplicitlyEmptyJSONNameOption(t *testing.T) {
	// An explicitly present json_name, even an empty one, still wins over
	// derivation: presence is what "wins when present" means, not non-emptiness.
	options := map[string]any{"json_name": ""}
	require.Equal(t, "", jsonFieldName("player_id", options))
}

func TestGeneratedEnumMethodNamesAreFreshAndReserved(t *testing.T) {
	want := []string{"to_wire", "from_wire", "to_json_name", "from_json_name"}
	require.Equal(t, "to_wire", toWireMethod)
	require.Equal(t, "from_wire", fromWireMethod)
	require.Equal(t, "to_json_name", toJSONNameMethod)
	require.Equal(t, "from_json_name", fromJSONNameMethod)
	require.Equal(t, want, generatedEnumMethodNames())

	mutated := generatedEnumMethodNames()
	mutated[0] = "changed"
	require.Equal(t, want, generatedEnumMethodNames())

	for _, methodName := range want {
		require.True(t, generatedEnumValueNames[methodName])
	}
}

func TestPlanEnumValueNameEscapesHostedFunctionNames(t *testing.T) {
	tests := []struct {
		raw       string
		generated string
		kind      memberEscapeKind
		reason    string
	}{
		{raw: "to_wire", generated: "to_wire_", kind: memberEscapeGenerated, reason: "generated member"},
		{raw: "from_wire", generated: "from_wire_", kind: memberEscapeGenerated, reason: "generated member"},
		{raw: "to_json_name", generated: "to_json_name_", kind: memberEscapeGenerated, reason: "generated member"},
		{raw: "from_json_name", generated: "from_json_name_", kind: memberEscapeGenerated, reason: "generated member"},
		{raw: "TRANSPORT_UNSPECIFIED", generated: "TRANSPORT_UNSPECIFIED", kind: memberEscapeNone},
		{raw: "var", generated: "var", kind: memberEscapeNone},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got := planEnumValueName(test.raw)
			require.Equal(t, test.generated, got.Generated)
			require.Equal(t, test.kind, got.Escape.Kind)
			require.Equal(t, test.reason, got.Escape.description())
			require.Equal(t, test.generated, EnumValueName(test.raw))
		})
	}
}

func TestPlanOneofAlternativeNameSkipsEngineTypes(t *testing.T) {
	require.Equal(t, "Image", planOneofAlternativeName("Image").Generated)
	require.Equal(t, memberEscapeNone, planOneofAlternativeName("Image").Escape.Kind)
	require.Equal(t, "var_", planOneofAlternativeName("var").Generated)
	require.Equal(t, memberEscapeKeyword, planOneofAlternativeName("var").Escape.Kind)
	require.Equal(t, "to_bytes_", planOneofAlternativeName("to_bytes").Generated)
	require.Equal(t, memberEscapeGenerated, planOneofAlternativeName("to_bytes").Escape.Kind)
}
