// Package main provides the foundry-tools command.
package main

import (
	"fmt"
	"os"

	"github.com/cafecito-games/foundry-tools/internal/cli"
)

func main() {
	cmd := cli.NewRootCommand(os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
