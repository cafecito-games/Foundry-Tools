package foundrytest

import (
	"os/exec"
	"path/filepath"
)

// BinaryCandidates returns possible Foundry binary paths in priority order.
func BinaryCandidates(envValue string) []string {
	var out []string
	if envValue != "" {
		out = append(out, envValue)
	}
	if onPath, err := exec.LookPath("foundry"); err == nil {
		out = append(out, onPath)
	}
	out = append(out, filepath.Join(".cache", "foundry", "foundry"))
	return out
}
