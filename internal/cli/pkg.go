package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager"
	"github.com/spf13/cobra"
)

var (
	packageInit    = packagemanager.Init
	packageAdd     = packagemanager.Add
	packageInstall = packagemanager.Install
	packageUpdate  = packagemanager.Update
	packageRemove  = packagemanager.Remove
	packageList    = packagemanager.List
)

type packageCLIOptions struct {
	JSON              bool
	Verbose           bool
	Quiet             bool
	MaxDownloadBytes  int64
	MaxExtractedBytes int64
}

func newPkgCommand(stdout io.Writer, opts *packageCLIOptions) *cobra.Command {
	if opts == nil {
		opts = &packageCLIOptions{}
	}
	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "Manage Foundry project packages",
	}
	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "emit machine-readable JSON output")
	cmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "enable verbose logging")
	cmd.PersistentFlags().BoolVarP(&opts.Quiet, "quiet", "q", false, "suppress non-error output")
	cmd.PersistentFlags().Var((*bytesizeValue)(&opts.MaxDownloadBytes), "max-download-size",
		"maximum compressed download size (e.g. 512MB, 1GiB); 0 uses the built-in default")
	cmd.PersistentFlags().Var((*bytesizeValue)(&opts.MaxExtractedBytes), "max-extract-size",
		"maximum total uncompressed archive size (e.g. 1GiB); 0 uses the built-in default")

	cmd.AddCommand(newPkgInitCommand(opts))
	cmd.AddCommand(newPkgAddCommand(opts))
	cmd.AddCommand(newPkgInstallCommand(opts))
	cmd.AddCommand(newPkgUpdateCommand(opts))
	cmd.AddCommand(newPkgRemoveCommand(opts))
	cmd.AddCommand(newPkgListCommand(opts))
	_ = stdout
	return cmd
}

func newPkgInitCommand(opts *packageCLIOptions) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a starter packages.toml in the current directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := packageInit(packagemanager.InitOptions{Dir: dir})
			if err != nil {
				return err
			}
			verbosef(cmd, opts, "manifest: %s\n", result.ManifestPath)
			return renderPackageOutput(cmd.OutOrStdout(), opts, result, func() {
				if opts.Quiet {
					return
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", result.ManifestPath)
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "directory to create packages.toml in (default: current directory)")
	return cmd
}

type pkgAddFlags struct {
	name, source, url, repo, version, asset, sourcePath, installAs, dir string
}

func newPkgAddCommand(opts *packageCLIOptions) *cobra.Command {
	flags := &pkgAddFlags{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a package to packages.toml and install it",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			spec := packagemanager.PackageSpec{
				Name:       flags.name,
				Source:     packagemanager.SourceType(flags.source),
				URL:        flags.url,
				Repo:       flags.repo,
				Version:    flags.version,
				Asset:      flags.asset,
				SourcePath: flags.sourcePath,
				InstallAs:  flags.installAs,
			}
			results, err := packageAdd(cmd.Context(), packagemanager.AddOptions{
				Options: packageManagerOptions(flags.dir, opts),
				Spec:    spec,
			})
			if err != nil {
				return err
			}
			return renderPackageOutput(cmd.OutOrStdout(), opts, results, func() {
				if opts.Quiet {
					return
				}
				for _, result := range results {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "added and installed %s\n", result.Name)
				}
			})
		},
	}
	cmd.Flags().StringVar(&flags.name, "name", "", "package name (table key under [packages])")
	cmd.Flags().StringVar(&flags.source, "source", "", "source type: git, github-release, archive")
	cmd.Flags().StringVar(&flags.url, "url", "", "clone or archive URL")
	cmd.Flags().StringVar(&flags.repo, "repo", "", "GitHub owner/repo (github-release)")
	cmd.Flags().StringVar(&flags.version, "version", "", "git ref or release tag")
	cmd.Flags().StringVar(&flags.asset, "asset", "", "release asset name/glob (github-release)")
	cmd.Flags().StringVar(&flags.sourcePath, "source-path", "", "subdirectory within the source to install")
	cmd.Flags().StringVar(&flags.installAs, "install-as", "", "install directory name (default: package name)")
	cmd.Flags().StringVar(&flags.dir, "dir", "", "start directory for project discovery")
	return cmd
}

func newPkgInstallCommand(opts *packageCLIOptions) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "install [package...]",
		Short: "Install packages declared in packages.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := packageInstall(cmd.Context(), packagemanager.InstallOptions{
				Options: packageManagerOptions(dir, opts),
				Names:   args,
			})
			if err != nil {
				return err
			}
			return renderPackageOutput(cmd.OutOrStdout(), opts, results, func() {
				if opts.Quiet {
					return
				}
				for _, result := range results {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "installed %s @ %s\n", result.Name, result.ResolvedVersion)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d package(s) installed\n", len(results))
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "start directory for project discovery")
	return cmd
}

func newPkgUpdateCommand(opts *packageCLIOptions) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "update [package...]",
		Short: "Re-resolve and reinstall packages, rewriting packages.lock",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := packageUpdate(cmd.Context(), packagemanager.UpdateOptions{
				Options: packageManagerOptions(dir, opts),
				Names:   args,
			})
			if err != nil {
				return err
			}
			return renderPackageOutput(cmd.OutOrStdout(), opts, results, func() {
				if opts.Quiet {
					return
				}
				for _, result := range results {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "updated %s @ %s\n", result.Name, result.ResolvedVersion)
				}
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "start directory for project discovery")
	return cmd
}

func newPkgRemoveCommand(opts *packageCLIOptions) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "remove <package>",
		Short: "Remove a package from packages.toml, packages.lock, and disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := packageRemove(packagemanager.RemoveOptions{
				Options: packageManagerOptions(dir, opts),
				Name:    name,
			}); err != nil {
				return err
			}
			if !opts.Quiet && !opts.JSON {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "start directory for project discovery")
	return cmd
}

func newPkgListCommand(opts *packageCLIOptions) *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List packages declared in packages.toml",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			listings, err := packageList(packagemanager.ListOptions{
				Options: packageManagerOptions(dir, opts),
			})
			if err != nil {
				return err
			}
			return renderPackageOutput(cmd.OutOrStdout(), opts, listings, func() {
				if opts.Quiet {
					return
				}
				for _, listing := range listings {
					mark := " "
					if listing.Installed {
						mark = "x"
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %-20s %-16s %s\n", mark, listing.Name, listing.Source, listing.Version)
				}
			})
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "start directory for project discovery")
	return cmd
}

func packageManagerOptions(dir string, opts *packageCLIOptions) packagemanager.Options {
	return packagemanager.Options{
		Dir:               dir,
		MaxDownloadBytes:  opts.MaxDownloadBytes,
		MaxExtractedBytes: opts.MaxExtractedBytes,
	}
}

func renderPackageOutput(w io.Writer, opts *packageCLIOptions, payload any, textFn func()) error {
	if !opts.JSON {
		textFn()
		return nil
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func verbosef(cmd *cobra.Command, opts *packageCLIOptions, format string, args ...any) {
	if opts.Verbose && !opts.Quiet && !opts.JSON {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), format, args...)
	}
}
