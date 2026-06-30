package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// NewRootCommand returns the root foundry-tools command.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "foundry-tools",
		Short:         "Tooling for Foundry Engine projects",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.AddCommand(newVersionCommand(stdout))
	cmd.AddCommand(newProtoCommand(stdout))
	return cmd
}

func newVersionCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(stdout, "foundry-tools %s\n", Version)
			return err
		},
	}
}

func newProtoCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proto",
		Short: "Protocol Buffers tools",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "print-options-proto",
		Short: "Print foundrytools/options.proto",
		RunE: func(_ *cobra.Command, _ []string) error {
			_ = stdout
			return fmt.Errorf("options proto support is not wired")
		},
	})
	return cmd
}
