package fstypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeRendering(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want string
	}{
		{name: "simple", typ: Named("Player"), want: "Player"},
		{name: "namespace", typ: Named("cafecito.game.v1.Player"), want: "cafecito.game.v1.Player"},
		{name: "nullable", typ: Nullable(Named("Player")), want: "Player?"},
		{name: "array", typ: Array(Named("String")), want: "Array[String]"},
		{name: "dictionary", typ: Dictionary(Named("String"), Named("int")), want: "Dictionary[String, int]"},
		{name: "generic", typ: Generic("foundry.proto.DecodeResult", Named("Player")), want: "foundry.proto.DecodeResult[Player]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.typ.Render())
		})
	}
}

func TestNoVariantPublicTypeByDefault(t *testing.T) {
	require.False(t, Named("String").IsVariant())
	require.True(t, Named("Variant").IsVariant())
}
