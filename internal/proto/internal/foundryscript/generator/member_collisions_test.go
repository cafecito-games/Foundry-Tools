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
	}, nil), "members.proto", nil)

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
	}, nil), "members.proto", nil)

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
	}}, nil), "members.proto", nil)

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
	}}, nil), "members.proto", nil)

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
