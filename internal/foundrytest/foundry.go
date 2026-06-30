package foundrytest

import (
	"os"
	"path/filepath"
)

// BinaryCandidates returns possible Foundry editor binary paths in priority order.
func BinaryCandidates(envValue string) []string {
	var out []string
	if envValue != "" {
		out = append(out, envValue)
	}
	home, err := os.UserHomeDir()
	if err == nil {
		out = append(out, filepath.Join(home, ".foundry/bin/foundry.macos.editor.dev.arm64"))
	}
	out = append(out, filepath.Join(".cache", "foundry", "foundry"))
	return out
}
