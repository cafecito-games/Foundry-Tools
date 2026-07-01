// Package anvil defines the root command shell for the anvil binary.
package anvil

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager"
)

// NewRootCommand returns the root anvil command.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "anvil",
		Short:         "Tooling for Foundry Engine projects",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	return cmd
}

// Execute runs cmd and returns the mapped process exit code.
func Execute(cmd *cobra.Command, stdout, stderr io.Writer) int {
	executed, err := cmd.ExecuteC()
	if err != nil {
		code := int(packagemanager.CodeFor(err))
		if wantsJSON(executed) {
			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			_ = encoder.Encode(map[string]any{
				"error": err.Error(),
				"code":  code,
			})
		} else {
			_, _ = fmt.Fprintf(stderr, "anvil: %v\n", err)
		}
		return code
	}
	return int(packagemanager.ExitOK)
}

func wantsJSON(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if flag := current.Flags().Lookup("json"); flag != nil {
			return parseBoolFlag(flag.Value.String())
		}
		if flag := current.PersistentFlags().Lookup("json"); flag != nil {
			return parseBoolFlag(flag.Value.String())
		}
		if flag := current.InheritedFlags().Lookup("json"); flag != nil {
			return parseBoolFlag(flag.Value.String())
		}
	}
	return false
}

func parseBoolFlag(raw string) bool {
	value, err := strconv.ParseBool(raw)
	return err == nil && value
}
