package foundrytest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultBinaryCandidates(t *testing.T) {
	candidates := BinaryCandidates("")
	require.Contains(t, candidates[0], ".foundry/bin/foundry.macos.editor.dev.arm64")
}

func TestEnvBinaryCandidateWins(t *testing.T) {
	candidates := BinaryCandidates("/tmp/foundry")
	require.Equal(t, "/tmp/foundry", candidates[0])
}
