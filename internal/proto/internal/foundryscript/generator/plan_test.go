package fsgenerator

import (
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func newTestResolver() *resolver {
	file := &protoast.ProtoFile{Package: "test"}
	return newResolver(file, "test.proto", nil, typeNamer{})
}

func TestPlanFieldDerivesJSONNameFromProtoName(t *testing.T) {
	resolve := newTestResolver()
	field := &protoast.Field{
		FieldType: "string",
		Name:      "player_id",
		Number:    1,
	}

	plan, err := planField(field, "Player", "Player", resolve)
	require.NoError(t, err)
	require.Equal(t, "playerId", plan.JSONName)
}

func TestPlanFieldPrefersExplicitJSONNameOption(t *testing.T) {
	resolve := newTestResolver()
	field := &protoast.Field{
		FieldType: "string",
		Name:      "player_id",
		Number:    1,
		Options:   map[string]any{"json_name": "playerIdentifier"},
	}

	plan, err := planField(field, "Player", "Player", resolve)
	require.NoError(t, err)
	require.Equal(t, "playerIdentifier", plan.JSONName)
	require.NotEqual(t, plan.JSONName, deriveJSONName(field.Name))
}

func TestPlanMapFieldDerivesJSONNameFromProtoName(t *testing.T) {
	resolve := newTestResolver()
	mapField := &protoast.MapField{
		KeyType:   "string",
		ValueType: "int32",
		Name:      "player_scores",
		Number:    1,
	}

	plan, err := planMapField(mapField, "Player", "Player", resolve)
	require.NoError(t, err)
	require.Equal(t, "playerScores", plan.JSONName)
}
