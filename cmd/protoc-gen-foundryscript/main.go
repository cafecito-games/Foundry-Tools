// Package main provides the protoc-gen-foundryscript plugin command.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "protoc-gen-foundryscript plugin support is not wired")
	os.Exit(1)
}
