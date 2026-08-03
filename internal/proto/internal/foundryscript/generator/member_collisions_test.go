package fsgenerator

import (
	"strings"
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestGenerateAggregatesEscapedMemberCollisions(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{
		{
			Name: "Alpha",
			Fields: []*protoast.Field{
				{
					Position:  protoast.Position{Line: 5, Column: 3},
					FieldType: "int32",
					Name:      "Node",
					Number:    1,
				},
				{
					Position:  protoast.Position{Line: 6, Column: 3},
					FieldType: "int32",
					Name:      "Node_",
					Number:    2,
				},
			},
		},
		{
			Name: "Beta",
			Fields: []*protoast.Field{{
				Position:  protoast.Position{Line: 10, Column: 3},
				FieldType: "string",
				Name:      "String",
				Number:    1,
			}},
			Oneofs: []*protoast.Oneof{{
				Position: protoast.Position{Line: 11, Column: 3},
				Name:     "String_",
				Fields: []*protoast.Field{{
					FieldType: "string",
					Name:      "text",
					Number:    2,
				}},
			}},
		},
	}, nil), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.EqualError(t, err, `generated Foundry member names collide:
  members.proto:5:3: field cafecito.game.v1.Alpha.Node generates Foundry member "Node_" after escaping native class "Node"
  members.proto:6:3: field cafecito.game.v1.Alpha.Node_ generates Foundry member "Node_"
  rename one protobuf declaration in cafecito.game.v1.Alpha
  members.proto:10:3: field cafecito.game.v1.Beta.String generates Foundry member "String_" after escaping built-in type "String"
  members.proto:11:3: oneof cafecito.game.v1.Beta.String_ generates Foundry member "String_"
  rename one protobuf declaration in cafecito.game.v1.Beta`)
}

func TestGenerateAggregatesKeywordAndEscapedOneofCollisions(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{
		{
			Name: "Keyword",
			Fields: []*protoast.Field{
				{FieldType: "int32", Name: "var", Number: 1},
				{FieldType: "int32", Name: "var_", Number: 2},
			},
		},
		{
			Name:   "Escaped",
			Fields: []*protoast.Field{{FieldType: "int32", Name: "Node_", Number: 1}},
			Oneofs: []*protoast.Oneof{{
				Name:   "Node",
				Fields: []*protoast.Field{{FieldType: "string", Name: "text", Number: 2}},
			}},
		},
	}, nil), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.Error(t, err)
	diagnostic := err.Error()
	require.Contains(t, diagnostic, "field cafecito.game.v1.Keyword.var")
	require.Contains(t, diagnostic, `after escaping Foundry keyword`)
	require.Contains(t, diagnostic, "field cafecito.game.v1.Keyword.var_")
	require.Contains(t, diagnostic, "field cafecito.game.v1.Escaped.Node_")
	require.Contains(t, diagnostic, "oneof cafecito.game.v1.Escaped.Node")
	require.Contains(t, diagnostic, `after escaping native class "Node"`)
	require.Less(t,
		strings.Index(diagnostic, "cafecito.game.v1.Escaped"),
		strings.Index(diagnostic, "cafecito.game.v1.Keyword"),
	)
}

func TestGenerateReportsTypeCollisionsBeforeMemberCollisions(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Node",
		Fields: []*protoast.Field{
			{FieldType: "int32", Name: "String", Number: 1},
			{FieldType: "int32", Name: "String_", Number: 2},
		},
	}}, nil), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.Error(t, err)
	require.Contains(t, err.Error(), "generated Foundry type names conflict with reserved engine types:")
	require.NotContains(t, err.Error(), "generated Foundry member names collide:")
}

func TestGenerateCollectsNestedMessageMemberCollisions(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Outer",
		NestedMessages: []*protoast.Message{{
			Name: "Inner",
			Fields: []*protoast.Field{
				{FieldType: "int32", Name: "Timer", Number: 1},
				{FieldType: "int32", Name: "Timer_", Number: 2},
			},
		}},
	}}, nil), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.Error(t, err)
	require.Contains(t, err.Error(), "field cafecito.game.v1.Outer.Inner.Timer")
	require.Contains(t, err.Error(), "field cafecito.game.v1.Outer.Inner.Timer_")
	require.Contains(t, err.Error(), `Foundry member "Timer_"`)
}

func TestMemberCollisionCollectorIncludesGeneratedNamespaceClaims(t *testing.T) {
	collector := newMemberCollisionCollector()
	collector.addMessage("members.proto", "cafecito.game.v1.Player", []fieldPlan{
		{
			Position: protoast.Position{Line: 7, Column: 3},
			Kind:     "field",
			Name:     "_pb_pick_kind_unknown",
			RawName:  "_pb_pick_kind_unknown",
		},
		{
			Position: protoast.Position{Line: 8, Column: 3},
			Kind:     "field",
			Name:     "pick_kind",
			RawName:  "pick_kind",
			Value:    valuePlan{Kind: kindEnum},
		},
		{
			Kind:    "field",
			Name:    unknownFieldsMember,
			RawName: unknownFieldsMember,
		},
	}, nil)

	err := collector.err()
	require.Error(t, err)
	diagnostic := err.Error()
	require.Contains(t, diagnostic, "members.proto:7:3")
	require.Contains(t, diagnostic, "field cafecito.game.v1.Player._pb_pick_kind_unknown")
	require.Contains(t, diagnostic, "retained enum companion cafecito.game.v1.Player.pick_kind")
	require.Contains(t, diagnostic, "generated unknown-field buffer cafecito.game.v1.Player."+unknownFieldsMember)
	require.Equal(t, 2, strings.Count(diagnostic, "rename one protobuf declaration"))
}

func TestMemberCollisionCollectorIncludesGeneratedMethodClaims(t *testing.T) {
	collector := newMemberCollisionCollector()
	collector.addMessage("members.proto", "cafecito.game.v1.Player", []fieldPlan{{
		Position: protoast.Position{Line: 7, Column: 3},
		Kind:     "field",
		Name:     "to_bytes",
		RawName:  "wire_value",
	}}, nil)

	require.EqualError(t, collector.err(), `generated Foundry member names collide:
  members.proto:7:3: field cafecito.game.v1.Player.wire_value generates Foundry member "to_bytes"
  members.proto: generated method cafecito.game.v1.Player.to_bytes generates Foundry member "to_bytes"
  rename one protobuf declaration in cafecito.game.v1.Player`)
}

func TestGenerateMethodEscapeCollidesOnlyAtEscapedSpelling(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{Position: protoast.Position{Line: 5, Column: 3}, FieldType: "bytes", Name: "to_bytes", Number: 1},
			{Position: protoast.Position{Line: 6, Column: 3}, FieldType: "bytes", Name: "to_bytes_", Number: 2},
		},
	}}, nil), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.EqualError(t, err, `generated Foundry member names collide:
  members.proto:5:3: field cafecito.game.v1.Player.to_bytes generates Foundry member "to_bytes_" after escaping generated member
  members.proto:6:3: field cafecito.game.v1.Player.to_bytes_ generates Foundry member "to_bytes_"
  rename one protobuf declaration in cafecito.game.v1.Player`)
	require.NotContains(t, err.Error(), "generated method")
}

func TestGeneratedMemberIdentityNamesAreEscaped(t *testing.T) {
	fields := make([]*protoast.Field, 0, 3)
	for number, name := range []string{"create_message", "protobuf_type_name", "type_name"} {
		fields = append(fields, &protoast.Field{
			FieldType: "string",
			Name:      name,
			Number:    number + 1,
		})
	}

	source := playerSource(t, fields)
	for _, name := range []string{"create_message", "protobuf_type_name", "type_name"} {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, source, "var "+name+"_: String")
			require.NotContains(t, source, "var "+name+"__: String")
			require.Equal(t, "generated member", planMemberName(name).Escape.description())
		})
	}
}

func TestAnyValueMetadataUsesTheReservedGeneratedPrefix(t *testing.T) {
	require.Equal(t, generatedPrefix+"any_uses_value", anyUsesValueMethod)
	require.True(t, generatedMemberNames[anyUsesValueMethod])
}

func TestGenerateInheritedMemberEscapeCollidesOnlyAtEscapedSpelling(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{Position: protoast.Position{Line: 5, Column: 3}, FieldType: "string", Name: "reference", Number: 1},
			{Position: protoast.Position{Line: 6, Column: 3}, FieldType: "string", Name: "reference_", Number: 2},
		},
	}}, nil), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.EqualError(t, err, `generated Foundry member names collide:
  members.proto:5:3: field cafecito.game.v1.Player.reference generates Foundry member "reference_" after escaping inherited method "RefCounted.reference"
  members.proto:6:3: field cafecito.game.v1.Player.reference_ generates Foundry member "reference_"
  rename one protobuf declaration in cafecito.game.v1.Player`)
}

// Escaping an enum value that collides with a hosted function is not
// injective: a value already spelled `to_wire_` collides with one spelled
// `to_wire` once the latter is escaped to the same identifier. This is caught
// as a collision rather than silently generating two cases with the same
// name.
func TestGenerateReportsEnumValueEscapeCollisions(t *testing.T) {
	files, err := Generate(namespacedFile(nil, []*protoast.Enum{{
		Name: "Transport",
		Values: []*protoast.EnumValue{
			{Position: protoast.Position{Line: 4, Column: 3}, Name: "to_wire", Number: 0},
			{Position: protoast.Position{Line: 5, Column: 3}, Name: "to_wire_", Number: 1},
		},
	}}), "members.proto", nil, Options{})

	require.Nil(t, files)
	require.EqualError(t, err, `generated Foundry member names collide:
  members.proto:4:3: enum value cafecito.game.v1.Transport.to_wire generates Foundry member "to_wire_" after escaping generated member
  members.proto:5:3: enum value cafecito.game.v1.Transport.to_wire_ generates Foundry member "to_wire_"
  rename one protobuf declaration in cafecito.game.v1.Transport`)
}

func TestMemberCollisionCollectorReportsPositionlessClaims(t *testing.T) {
	collector := newMemberCollisionCollector()
	collector.addMessage("descriptor.proto", "cafecito.game.v1.Outer.Inner", []fieldPlan{
		{Kind: "field", Name: "Node_", RawName: "Node", Escape: planMemberName("Node").Escape},
		{Kind: "field", Name: "Node_", RawName: "Node_"},
	}, nil)

	err := collector.err()
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto: field cafecito.game.v1.Outer.Inner.Node")
	require.Contains(t, err.Error(), "descriptor.proto: field cafecito.game.v1.Outer.Inner.Node_")
}

func TestFieldPlanUsesSeparateRawIdentityAndLocalStem(t *testing.T) {
	plan := fieldPlan{
		RawName:   "kind",
		LocalStem: "pick_kind",
	}

	require.Equal(t, "kind", plan.RawName)
	require.Equal(t, "_pb_pick_kind", plan.Local())
	require.Equal(t, "_pb_pick_kind_read", plan.Local("read"))
	require.Equal(t, "_pb_pick_kind_unknown", plan.UnknownMember())
}
