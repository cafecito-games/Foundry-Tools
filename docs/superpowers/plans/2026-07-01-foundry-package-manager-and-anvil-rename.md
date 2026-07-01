# Foundry Package Manager And Anvil Rename Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `anvil pkg ...` package-manager commands for Foundry projects and rename the main CLI binary from `foundry-tools` to `anvil`.

**Architecture:** The package manager lives behind `internal/packagemanager`, with implementation packages under `internal/packagemanager/internal`. The CLI layer only imports `internal/packagemanager` and remains responsible for Cobra command wiring and rendering. Package-manager behavior is adapted from `/Users/christian/CafecitoGames/godot-addon-manager`, using `project.foundry`, `packages.toml`, `packages.lock`, and the project `addons/` directory.

**Tech Stack:** Go 1.26, Cobra, BurntSushi TOML, testify, system `git`, net/http test servers, Taskfile.

---

## File Structure

- Create `internal/packagemanager/api.go`: public package-manager API, public types, error-code mapping.
- Create `internal/packagemanager/runner.go`: operation orchestration for add/install/update/remove/list/init.
- Create `internal/packagemanager/internal/manifest/*`: `packages.toml` parsing, saving, hashing, validation.
- Create `internal/packagemanager/internal/lockfile/*`: `packages.lock` parsing, saving, pin matching.
- Create `internal/packagemanager/internal/project/*`: `project.foundry` discovery and managed path derivation.
- Create `internal/packagemanager/internal/installer/*`: staging copy, source-path selection, exclusions, atomic replacement.
- Create `internal/packagemanager/internal/source/*`: git/archive/GitHub-release fetchers and size limits.
- Create `internal/packagemanager/internal/output/*`: typed package-manager errors and exit codes.
- Create `internal/cli/pkg.go`: `anvil pkg` command group and subcommands.
- Modify `internal/cli/root.go`, `internal/cli/root_test.go`, and `internal/cli/version.go`: rename command identity and add `pkg`.
- Rename `cmd/foundry-tools/main.go` to `cmd/anvil/main.go`.
- Modify `Taskfile.yml`, `README.md`, `tests/integration/direct_cli_test.go`, and `tests/foundry/run.sh` for `anvil`.
- Modify `go.mod` / `go.sum` through `go mod tidy` to add TOML support.

### Task 1: Binary Rename

**Files:**
- Modify: `internal/cli/root_test.go`
- Modify: `internal/cli/root.go`
- Move: `cmd/foundry-tools/main.go` -> `cmd/anvil/main.go`
- Modify: `Taskfile.yml`
- Modify: `README.md`
- Modify: `tests/integration/direct_cli_test.go`
- Modify: `tests/foundry/run.sh`

- [ ] **Step 1: Write failing rename expectations**

Change `internal/cli/root_test.go` so the version test expects:

```go
require.Contains(t, stdout.String(), "anvil dev")
```

Change `tests/integration/direct_cli_test.go` so direct CLI execution uses:

```go
run(t, root, "go", "run", "./cmd/anvil", "proto", "generate", "-I", "tests/integration/fixtures/basic", "-o", outDir, "tests/integration/fixtures/basic/player.proto")
```

- [ ] **Step 2: Run rename tests to verify failure**

Run:

```bash
go test ./internal/cli -run TestVersionCommandPrintsVersion -count=1
go test -tags=integration ./tests/integration -run TestDirectCLIGeneratesFoundryScript -count=1
```

Expected: first test fails because output still says `foundry-tools`; integration test fails because `./cmd/anvil` does not exist.

- [ ] **Step 3: Rename command implementation**

Update root command identity:

```go
cmd := &cobra.Command{
    Use:           "anvil",
    Short:         "Tooling for Foundry Engine projects",
    SilenceUsage:  true,
    SilenceErrors: true,
}
```

Update version output:

```go
_, err := fmt.Fprintf(stdout, "anvil %s\n", Version)
```

Move the command package:

```bash
mkdir -p cmd/anvil
git mv cmd/foundry-tools/main.go cmd/anvil/main.go
```

Update the main package comment to say it provides the `anvil` command.

- [ ] **Step 4: Update build/docs references**

Update `Taskfile.yml` to build and install `./cmd/anvil` as `bin/anvil` while keeping `protoc-gen-foundryscript`.

Update `README.md`, `tests/foundry/run.sh`, and direct CLI integration references from `foundry-tools` to `anvil`.

- [ ] **Step 5: Verify rename passes**

Run:

```bash
go test ./internal/cli -run TestVersionCommandPrintsVersion -count=1
go test -tags=integration ./tests/integration -run TestDirectCLIGeneratesFoundryScript -count=1
task build
```

Expected: tests pass and `bin/anvil` plus `bin/protoc-gen-foundryscript` exist.

- [ ] **Step 6: Commit**

```bash
git add internal/cli cmd Taskfile.yml README.md tests/integration/direct_cli_test.go tests/foundry/run.sh
git commit -m "feat: rename cli to anvil"
```

### Task 2: Package Manifest, Lockfile, Project, And Errors

**Files:**
- Create: `internal/packagemanager/internal/output/output.go`
- Create: `internal/packagemanager/internal/manifest/spec.go`
- Create: `internal/packagemanager/internal/manifest/manifest.go`
- Create: `internal/packagemanager/internal/manifest/validate.go`
- Create: `internal/packagemanager/internal/lockfile/lockfile.go`
- Create: `internal/packagemanager/internal/project/project.go`
- Test: `internal/packagemanager/internal/manifest/*_test.go`
- Test: `internal/packagemanager/internal/lockfile/*_test.go`
- Test: `internal/packagemanager/internal/project/*_test.go`

- [ ] **Step 1: Write failing package metadata tests**

Create tests that assert:

```go
loaded.Packages["dialogue"].Name == "dialogue"
loaded.Packages["dialogue"].Source == SourceGit
PackageSpec{Name: "foo"}.InstallName() == "foo"
PackageSpec{Name: "foo", InstallAs: "bar"}.InstallName() == "bar"
Hash() changes when version or exclude changes
Validate() rejects path traversal, bad checksums, unknown sources, missing required fields
LoadLock(missingPath) returns an empty lockfile
NeedsResolve(spec, lock) is false only when spec_hash matches
Discover(start) walks up to a regular project.foundry and returns packages/addons paths
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./internal/packagemanager/internal/manifest ./internal/packagemanager/internal/lockfile ./internal/packagemanager/internal/project -count=1
```

Expected: package paths do not exist or exported identifiers are undefined.

- [ ] **Step 3: Implement minimal packages**

Adapt `gpm` manifest, lock, project, and output code with these naming changes:

```go
type Manifest struct {
    Packages map[string]PackageSpec `toml:"packages"`
}

type PackageSpec struct {
    Name       string     `toml:"-"`
    Source     SourceType `toml:"source"`
    URL        string     `toml:"url,omitempty"`
    Repo       string     `toml:"repo,omitempty"`
    Version    string     `toml:"version,omitempty"`
    Asset      string     `toml:"asset,omitempty"`
    SourcePath string     `toml:"source_path,omitempty"`
    InstallAs  string     `toml:"install_as,omitempty"`
    Exclude    []string   `toml:"exclude,omitempty"`
    Checksum   string     `toml:"checksum,omitempty"`
}

type Project struct {
    Root         string
    ManifestPath string // <Root>/packages.toml
    LockPath     string // <Root>/packages.lock
    AddonsDir    string // <Root>/addons
}
```

Use `project.foundry` as the discovery marker and `packages.toml` / `packages.lock` as managed file names.

- [ ] **Step 4: Verify metadata tests pass**

Run:

```bash
go test ./internal/packagemanager/internal/manifest ./internal/packagemanager/internal/lockfile ./internal/packagemanager/internal/project -count=1
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/packagemanager/internal
git commit -m "feat: add foundry package metadata"
```

### Task 3: Source Fetchers And Installer

**Files:**
- Create: `internal/packagemanager/internal/source/source.go`
- Create: `internal/packagemanager/internal/source/git.go`
- Create: `internal/packagemanager/internal/source/archive.go`
- Create: `internal/packagemanager/internal/source/githubrelease.go`
- Create: `internal/packagemanager/internal/installer/installer.go`
- Test: `internal/packagemanager/internal/source/*_test.go`
- Test: `internal/packagemanager/internal/installer/*_test.go`

- [ ] **Step 1: Copy failing behavior tests from gpm with package names adapted**

Adapt `gpm` tests to assert:

```go
ArchiveFetcher downloads zip/tar.gz into a temp tree and records checksum
ArchiveFetcher rejects unsupported archive names, zip-slip entries, symlinks, and oversized payloads
GitHubReleaseFetcher selects one asset by exact name or glob and uses GITHUB_TOKEN/GH_TOKEN headers
GitFetcher resolves a local repository tag or commit to a commit SHA
Install selects explicit source_path, auto-detects exactly one fetched addons/<name> directory, rejects multiple addon directories, applies exclude, rejects symlinks, and replaces existing installs safely
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./internal/packagemanager/internal/source ./internal/packagemanager/internal/installer -count=1
```

Expected: packages or identifiers are missing.

- [ ] **Step 3: Implement fetchers and installer**

Adapt `gpm` source and installer code, changing imports to:

```go
github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/manifest
github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output
github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/source
```

Use temp prefixes such as `anvil-git-*`, `anvil-archive-*`, `anvil-download-*`, `.anvil-staging-*`, and `.anvil-backup` in new code.

- [ ] **Step 4: Verify source and installer tests pass**

Run:

```bash
go test ./internal/packagemanager/internal/source ./internal/packagemanager/internal/installer -count=1
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/packagemanager/internal/source internal/packagemanager/internal/installer
git commit -m "feat: add foundry package fetching and install"
```

### Task 4: Public Package Manager API

**Files:**
- Create: `internal/packagemanager/api.go`
- Create: `internal/packagemanager/runner.go`
- Test: `internal/packagemanager/api_test.go`
- Test: `internal/packagemanager/runner_test.go`

- [ ] **Step 1: Write failing API tests**

Create tests for:

```go
Init creates packages.toml and fails when it already exists
InstallPackages installs all packages and writes packages.lock
InstallPackages honors locked git commit SHAs when spec_hash matches
UpdatePackages re-resolves selected names
AddPackage persists then installs, and rolls back packages.toml on install failure
RemovePackage removes addons/<installName>, packages.toml entry, and packages.lock entry
ListPackages returns sorted package records with installed status
CodeFor maps usage/manifest/fetch/install errors to 2/3/4/5
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
go test ./internal/packagemanager -count=1
```

Expected: public package-manager API is missing.

- [ ] **Step 3: Implement API and runner**

Expose the public API:

```go
type Options struct {
    Dir               string
    MaxDownloadBytes  int64
    MaxExtractedBytes int64
}

type AddOptions struct {
    Options
    Spec PackageSpec
}

type InstallOptions struct {
    Options
    Names []string
}

type UpdateOptions struct {
    Options
    Names []string
}

type RemoveOptions struct {
    Options
    Name string
}

type PackageResult struct {
    Name            string `json:"name"`
    ResolvedVersion string `json:"resolved_version"`
    InstallPath     string `json:"install_path"`
}

type PackageListing struct {
    Name      string `json:"name"`
    Source    string `json:"source"`
    Version   string `json:"version"`
    Installed bool   `json:"installed"`
}
```

Implement:

```go
func Init(opts InitOptions) (*InitResult, error)
func Add(ctx context.Context, opts AddOptions) ([]PackageResult, error)
func Install(ctx context.Context, opts InstallOptions) ([]PackageResult, error)
func Update(ctx context.Context, opts UpdateOptions) ([]PackageResult, error)
func Remove(opts RemoveOptions) error
func List(opts ListOptions) ([]PackageListing, error)
func Discover(dir string) (*Project, error)
func CodeFor(err error) ExitCode
```

- [ ] **Step 4: Verify API tests pass**

Run:

```bash
go test ./internal/packagemanager -count=1
```

Expected: all API tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/packagemanager
git commit -m "feat: expose foundry package manager api"
```

### Task 5: CLI Package Commands

**Files:**
- Create: `internal/cli/pkg.go`
- Modify: `internal/cli/root.go`
- Test: `internal/cli/pkg_test.go`
- Test: `internal/cli/root_test.go`

- [ ] **Step 1: Write failing CLI tests**

Add tests that exercise `NewRootCommand` with temp Foundry projects:

```go
anvil pkg init --dir <dir> creates packages.toml
anvil pkg list --dir <project> prints sorted package rows
anvil pkg list --dir <project> --json emits JSON records
anvil pkg remove --dir <project> name deletes manifest, lock, and addons/name
anvil pkg install --dir <project> calls the package manager and prints installed rows
anvil pkg update --dir <project> name prints updated rows
anvil pkg add --dir <project> --name p --source archive --url https://example.com/p.zip validates and delegates add
root help includes pkg
```

- [ ] **Step 2: Run CLI tests to verify failure**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: `pkg` command is missing.

- [ ] **Step 3: Implement `newPkgCommand`**

Add `cmd.AddCommand(newPkgCommand(stdout))` to `NewRootCommand`.

Implement subcommands that build `packagemanager` option structs and render text/JSON. Use package-manager errors only through public `packagemanager.CodeFor`.

- [ ] **Step 4: Verify CLI tests pass**

Run:

```bash
go test ./internal/cli -count=1
```

Expected: all CLI tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/cli
git commit -m "feat: add anvil package commands"
```

### Task 6: Documentation, Tidy, And Full Verification

**Files:**
- Modify: `README.md`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Write README package-manager docs**

Add examples for:

```bash
anvil pkg init
anvil pkg add --name my_package --source git --url https://github.com/org/my_package.git --version v1.0.0 --source-path addons/my_package
anvil pkg install
anvil pkg update
anvil pkg remove my_package
anvil pkg list
```

Document `packages.toml`, `packages.lock`, `project.foundry`, and `addons/`.

- [ ] **Step 2: Run formatting and tidy**

Run:

```bash
task fmt
task tidy
```

Expected: Go files are formatted and BurntSushi TOML is present in `go.mod`.

- [ ] **Step 3: Run full verification**

Run:

```bash
task
task integration
```

Expected: local CI and integration tests pass.

- [ ] **Step 4: Commit**

```bash
git add README.md go.mod go.sum docs/superpowers/plans/2026-07-01-foundry-package-manager-and-anvil-rename.md
git commit -m "docs: document anvil package manager"
```

## Self-Review

- Spec coverage: all spec items map to tasks. AssetLib/TUI stay excluded. `protoc-gen-foundryscript` remains unchanged.
- Placeholder scan: no placeholder-only tasks; each task has concrete paths, expected failures, and verification commands.
- Type consistency: public names use `PackageSpec`, `Manifest.Packages`, `PackageResult`, `PackageListing`, and `packages.toml` / `packages.lock` consistently.
