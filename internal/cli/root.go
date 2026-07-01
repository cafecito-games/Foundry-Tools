package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
	"github.com/cafecito-games/foundry-tools/internal/fsgenerator"
	"github.com/cafecito-games/foundry-tools/internal/packagemanager"
	"github.com/cafecito-games/foundry-tools/internal/protoparse"
	"github.com/cafecito-games/foundry-tools/internal/protovalidate"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
	"github.com/spf13/cobra"
)

// NewRootCommand returns the root anvil command.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	return newRootCommand(stdout, stderr, &packageCLIOptions{})
}

func newRootCommand(stdout, stderr io.Writer, pkgOpts *packageCLIOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "anvil",
		Short:         "Tooling for Foundry Engine projects",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.AddCommand(newVersionCommand(stdout))
	cmd.AddCommand(newProtoCommand(stdout))
	cmd.AddCommand(newPkgCommand(pkgOpts))
	return cmd
}

// Execute runs anvil with explicit streams and returns the mapped process exit
// code. It is used by main and tests so package-manager error rendering stays
// consistent.
func Execute(args []string, stdout, stderr io.Writer) int {
	pkgOpts := &packageCLIOptions{}
	cmd := newRootCommand(stdout, stderr, pkgOpts)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		code := int(packagemanager.CodeFor(err))
		if pkgOpts.JSON {
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

func newVersionCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(stdout, "anvil %s\n", Version)
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
			_, err := stdout.Write(foundrytoolspb.Bytes())
			return err
		},
	})
	cmd.AddCommand(newProtoGenerateCommand(stdout))
	return cmd
}

func newProtoGenerateCommand(stdout io.Writer) *cobra.Command {
	var opts struct {
		outDir     string
		importPath []string
	}

	cmd := &cobra.Command{
		Use:   "generate [flags] <proto files...>",
		Short: "Generate Foundry Script from protobuf files",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one .proto file is required")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			parsedFiles, err := protoparse.ParseFiles(args, opts.importPath)
			if err != nil {
				return err
			}
			for _, parsed := range parsedFiles {
				if validationErrors := protovalidate.Validate(parsed.File, parsed.Filename); len(validationErrors) != 0 {
					return validationErrorList(validationErrors)
				}
				files, err := fsgenerator.Generate(parsed.File, parsed.Filename, nil)
				if err != nil {
					return err
				}
				if err := writeFiles(opts.outDir, files); err != nil {
					return err
				}
			}
			if err := writeFiles(opts.outDir, runtime.Files()); err != nil {
				return err
			}
			_, err = fmt.Fprintf(stdout, "generated Foundry Script for %d proto file(s)\n", len(args))
			return err
		},
	}
	cmd.Flags().StringVarP(&opts.outDir, "out", "o", ".", "output directory")
	cmd.Flags().StringArrayVarP(&opts.importPath, "proto_path", "I", nil, "proto import path")
	return cmd
}

type validationErrorList []protovalidate.ValidationError

func (l validationErrorList) Error() string {
	messages := make([]string, 0, len(l))
	for i := range l {
		messages = append(messages, (&l[i]).Error())
	}
	return strings.Join(messages, "\n")
}

func writeFiles(outDir string, files map[string]string) error {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		path := filepath.Join(outDir, filepath.FromSlash(name))
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // Generated source directories should be project-readable.
			return err
		}
		if err := chmodGeneratedDirs(outDir, filepath.FromSlash(name)); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(files[name]), 0o644); err != nil { //nolint:gosec // Generated source files should be project-readable.
			return err
		}
		if err := os.Chmod(path, 0o644); err != nil { //nolint:gosec // Generated source files should be project-readable.
			return err
		}
	}
	return nil
}

func chmodGeneratedDirs(outDir, generatedPath string) error {
	dir := filepath.Dir(generatedPath)
	if dir == "." {
		return nil
	}
	current := outDir
	for _, part := range strings.Split(dir, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if err := os.Chmod(current, 0o755); err != nil { //nolint:gosec // Generated source directories should be project-readable.
			return err
		}
	}
	return nil
}
