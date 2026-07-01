package proto

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

// NewCommand builds the protobuf command tree.
func NewCommand(stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "proto",
		Short: "Protocol Buffers tools",
	}
	cmd.SetOut(stdout)
	cmd.AddCommand(&cobra.Command{
		Use:   "print-options-proto",
		Short: "Print foundrytools/options.proto",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write(foundrytoolspb.Bytes())
			return err
		},
	})
	cmd.AddCommand(newGenerateCommand(stdout))
	return cmd
}

func newGenerateCommand(stdout io.Writer) *cobra.Command {
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
			parsedFiles, err := ParseFiles(args, opts.importPath)
			if err != nil {
				return err
			}
			for _, parsed := range parsedFiles {
				if validationErrors := Validate(parsed.File, parsed.Filename); len(validationErrors) != 0 {
					return validationErrorList(validationErrors)
				}
				files, err := Generate(parsed.File, parsed.Filename, nil)
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

type validationErrorList []ValidationError

func (l validationErrorList) Error() string {
	return FormatValidationErrors(l)
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
