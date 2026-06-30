// Package main provides the protoc-gen-foundryscript plugin command.
package main

import (
	"fmt"
	"os"

	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
	"github.com/cafecito-games/foundry-tools/internal/plugin"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--print-options-proto" {
			if _, err := os.Stdout.Write(foundrytoolspb.Bytes()); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}
	if err := plugin.Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
