// Package main provides the anvil command.
package main

import (
	"os"

	"github.com/cafecito-games/foundry-tools/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
