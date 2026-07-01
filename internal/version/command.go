// Package version defines the anvil version command.
package version

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Version is set by release builds through ldflags.
var Version = "dev"

// NewCommand builds the version command.
func NewCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(stdout, "anvil %s\n", Version)
			return err
		},
	}
}
