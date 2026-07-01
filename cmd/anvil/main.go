// Package main provides the anvil command.
package main

import (
	"os"

	"github.com/cafecito-games/foundry-tools/internal/anvil"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager"
	"github.com/cafecito-games/foundry-tools/internal/proto"
	"github.com/cafecito-games/foundry-tools/internal/version"
)

func main() {
	stdout := os.Stdout
	stderr := os.Stderr
	root := anvil.NewRootCommand(stdout, stderr)
	root.AddCommand(version.NewCommand(stdout))
	root.AddCommand(proto.NewCommand(stdout))
	root.AddCommand(packagemanager.NewCommand(stdout, stderr))
	root.SetArgs(os.Args[1:])
	os.Exit(anvil.Execute(root, stdout, stderr))
}
