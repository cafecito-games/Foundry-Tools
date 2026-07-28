package foundrytest

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultBinaryCandidatesFallBackToCache(t *testing.T) {
	candidates := BinaryCandidates("")
	require.NotEmpty(t, candidates)
	require.Equal(t, filepath.Join(".cache", "foundry", "foundry"), candidates[len(candidates)-1])
	for _, candidate := range candidates {
		require.NotContains(t, candidate, filepath.Join(".foundry", "bin"))
	}
}

func TestEnvBinaryCandidateWins(t *testing.T) {
	candidates := BinaryCandidates("/tmp/foundry")
	require.Equal(t, "/tmp/foundry", candidates[0])
}
