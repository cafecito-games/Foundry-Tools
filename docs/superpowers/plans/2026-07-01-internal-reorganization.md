# Internal Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize `internal/` so domain packages own their Cobra command APIs and protobuf/Foundry Script internals are hidden behind an `internal/proto` facade.

**Architecture:** `cmd/anvil` becomes the composition point for root, version, proto, and package-manager commands. `internal/packagemanager` exposes `NewCommand` plus the existing package-manager API, while `internal/proto` exposes `NewCommand` plus source and descriptor generation helpers backed by private parser, validator, descriptor, AST, and Foundry Script generator packages.

**Tech Stack:** Go 1.26, Cobra, google.golang.org/protobuf plugin protocol, existing Task workflows.

---

## File Structure

Create or modify these command facade files:

- `internal/anvil/root.go`: root command shell and `Execute`.
- `internal/anvil/root_test.go`: root command registration and error-rendering tests.
- `internal/version/command.go`: `version` Cobra command.
- `internal/proto/api.go`: protobuf facade helpers used by CLI and plugin.
- `internal/proto/command.go`: `proto` Cobra command.
- `internal/proto/command_test.go`: proto command tests and generated file permission tests.
- `internal/packagemanager/command.go`: `pkg` Cobra command moved from `internal/cli/pkg.go`.
- `internal/packagemanager/command_test.go`: package-manager command tests moved from `internal/cli/pkg_test.go`.
- `cmd/anvil/main.go`: compose root, version, proto, and package-manager commands.
- `internal/plugin/plugin.go`: delegate descriptor conversion and generation through `internal/proto`.

Move these implementation packages:

- `internal/protoast` -> `internal/proto/internal/ast`
- `internal/protoparse` -> `internal/proto/internal/parser`
- `internal/protodesc` -> `internal/proto/internal/desc`
- `internal/protovalidate` -> `internal/proto/internal/validate`
- `internal/fsast` -> `internal/proto/internal/foundryscript/ast`
- `internal/fstypes` -> `internal/proto/internal/foundryscript/types`
- `internal/fsgenerator` -> `internal/proto/internal/foundryscript/generator`

Delete after migration:

- `internal/cli/`

## Task 1: Add Command Ownership Tests

**Files:**
- Create: `internal/anvil/root_test.go`
- Create: `internal/proto/command_test.go`
- Create: `internal/packagemanager/command_test.go`
- Remove after migration: `internal/cli/root_test.go`
- Remove after migration: `internal/cli/pkg_test.go`

- [ ] **Step 1: Write failing tests for the new command package APIs**

Create `internal/anvil/root_test.go` with tests that call:

```go
root := anvil.NewRootCommand(&stdout, &stderr)
root.AddCommand(version.NewCommand(&stdout))
root.AddCommand(proto.NewCommand(&stdout))
root.AddCommand(packagemanager.NewCommand(&stdout, &stderr))
```

The tests should assert:

- `version` prints `anvil dev`.
- `proto print-options-proto` prints `package foundrytools;`.
- `proto generate` without input returns `at least one .proto file is required`.
- `pkg list --dir <temp>` maps to exit code `3` with text errors.
- `pkg --json list --dir <temp>` maps to exit code `3` with JSON errors.

Create `internal/proto/command_test.go` with tests that assert:

- `NewCommand` wires `print-options-proto`.
- `generate` requires at least one `.proto` file.
- Generated directories are forced to `0755` and generated files to `0644`.

Create `internal/packagemanager/command_test.go` by moving the existing package command tests and changing command construction from `cli.NewRootCommand` to `packagemanager.NewCommand`.

- [ ] **Step 2: Run the new tests to verify they fail before production code**

Run:

```bash
go test ./internal/anvil ./internal/proto ./internal/packagemanager
```

Expected: FAIL because `internal/anvil`, `internal/proto`, `internal/version`, and `packagemanager.NewCommand` do not exist yet.

## Task 2: Move Package-Manager Command Ownership

**Files:**
- Create: `internal/packagemanager/command.go`
- Modify: `internal/packagemanager/command_test.go`
- Delete during Task 7: `internal/cli/pkg.go`

- [ ] **Step 1: Move package command implementation**

Move the command code from `internal/cli/pkg.go` into `internal/packagemanager/command.go`.

Required API:

```go
func NewCommand(stdout, stderr io.Writer) *cobra.Command
```

The implementation should keep the existing subcommands and output behavior for:

- `pkg init`
- `pkg add`
- `pkg install`
- `pkg update`
- `pkg remove`
- `pkg list`

- [ ] **Step 2: Run package-manager command tests**

Run:

```bash
go test ./internal/packagemanager
```

Expected: package-manager tests pass once the command API is implemented.

## Task 3: Move Protobuf Internals Behind `internal/proto`

**Files:**
- Move: `internal/protoast/*` -> `internal/proto/internal/ast/`
- Move: `internal/protoparse/*` -> `internal/proto/internal/parser/`
- Move: `internal/protodesc/*` -> `internal/proto/internal/desc/`
- Move: `internal/protovalidate/*` -> `internal/proto/internal/validate/`
- Move: `internal/fsast/*` -> `internal/proto/internal/foundryscript/ast/`
- Move: `internal/fstypes/*` -> `internal/proto/internal/foundryscript/types/`
- Move: `internal/fsgenerator/*` -> `internal/proto/internal/foundryscript/generator/`
- Create: `internal/proto/api.go`

- [ ] **Step 1: Move directories with `git mv`**

Run:

```bash
mkdir -p internal/proto/internal/foundryscript
git mv internal/protoast internal/proto/internal/ast
git mv internal/protoparse internal/proto/internal/parser
git mv internal/protodesc internal/proto/internal/desc
git mv internal/protovalidate internal/proto/internal/validate
git mv internal/fsast internal/proto/internal/foundryscript/ast
git mv internal/fstypes internal/proto/internal/foundryscript/types
git mv internal/fsgenerator internal/proto/internal/foundryscript/generator
```

- [ ] **Step 2: Update package declarations and imports**

Use package names:

```text
protoast
protoparse
protodesc
protovalidate
fsast
fstypes
fsgenerator
```

Keep names stable to reduce code churn, but update import paths to:

```text
github.com/cafecito-games/foundry-tools/internal/proto/internal/ast
github.com/cafecito-games/foundry-tools/internal/proto/internal/parser
github.com/cafecito-games/foundry-tools/internal/proto/internal/desc
github.com/cafecito-games/foundry-tools/internal/proto/internal/validate
github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast
github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types
github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/generator
```

- [ ] **Step 3: Create the `internal/proto` facade**

Create `internal/proto/api.go` with helpers equivalent to:

```go
package proto

import (
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/cafecito-games/foundry-tools/internal/proto/internal/desc"
	"github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/generator"
	"github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
	"github.com/cafecito-games/foundry-tools/internal/proto/internal/validate"
)

type ParsedFile = parser.ParsedFile
type FileEntry = generator.FileEntry
type ValidationError = validate.ValidationError

func ParseFiles(filenames, importRoots []string) ([]ParsedFile, error) {
	return parser.ParseFiles(filenames, importRoots)
}

func Validate(file *ast.ProtoFile, filename string) []ValidationError {
	return validate.Validate(file, filename)
}

func Generate(file *ast.ProtoFile, sourceName string, imports []FileEntry) (map[string]string, error) {
	return generator.Generate(file, sourceName, imports)
}

func FromCodeGeneratorRequest(req *pluginpb.CodeGeneratorRequest) ([]*ast.ProtoFile, error) {
	return desc.FromCodeGeneratorRequest(req)
}
```

- [ ] **Step 4: Run moved package tests**

Run:

```bash
go test ./internal/proto/...
```

Expected: all moved protobuf and Foundry Script tests pass.

## Task 4: Add Proto Command Facade

**Files:**
- Create: `internal/proto/command.go`
- Modify: `internal/proto/command_test.go`

- [ ] **Step 1: Move proto command code**

Move the proto command implementation from `internal/cli/root.go` into `internal/proto/command.go`.

Required API:

```go
func NewCommand(stdout io.Writer) *cobra.Command
```

The command should keep:

- `proto print-options-proto`
- `proto generate [flags] <proto files...>`
- `--out, -o`
- `--proto_path, -I`
- Generated file sorting and permissions.

- [ ] **Step 2: Run proto command tests**

Run:

```bash
go test ./internal/proto
```

Expected: proto command facade tests pass.

## Task 5: Add Root And Version Command Facades

**Files:**
- Create: `internal/anvil/root.go`
- Create: `internal/version/command.go`
- Modify: `cmd/anvil/main.go`
- Delete: `internal/cli/root.go`
- Delete: `internal/cli/version.go`
- Delete: `internal/cli/doc.go`

- [ ] **Step 1: Implement root execution package**

Create `internal/anvil/root.go` with:

```go
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command
func Execute(cmd *cobra.Command, stdout, stderr io.Writer) int
```

`Execute` should preserve current package-manager error behavior:

- JSON errors when the package-manager command has `--json`.
- Text errors as `anvil: <error>`.
- Exit code from `packagemanager.CodeFor(err)`.

- [ ] **Step 2: Implement version command package**

Create `internal/version/command.go` with:

```go
const Version = "dev"

func NewCommand(stdout io.Writer) *cobra.Command
```

The command should print:

```text
anvil dev
```

- [ ] **Step 3: Make `cmd/anvil/main.go` compose commands**

Update `cmd/anvil/main.go` so it imports `internal/anvil`, `internal/packagemanager`, `internal/proto`, and `internal/version`, adds the three domain commands to the root, and exits through `anvil.Execute`.

- [ ] **Step 4: Run root command tests**

Run:

```bash
go test ./internal/anvil
```

Expected: root registration and error-rendering tests pass.

## Task 6: Update Plugin To Use The Proto Facade

**Files:**
- Modify: `internal/plugin/plugin.go`
- Modify: `internal/plugin/plugin_test.go`
- Modify: `cmd/protoc-gen-foundryscript/main.go` only if imports need cleanup.

- [ ] **Step 1: Replace private imports**

Change `internal/plugin/plugin.go` to import `internal/proto` instead of descriptor, validator, generator, and AST internals.

The plugin flow should remain:

```text
read request -> proto.FromCodeGeneratorRequest -> proto.Validate -> proto.Generate -> append response files -> append runtime files
```

- [ ] **Step 2: Run plugin tests**

Run:

```bash
go test ./internal/plugin
```

Expected: plugin tests pass.

## Task 7: Remove Old CLI Package And Clean Imports

**Files:**
- Delete: `internal/cli/`
- Modify: any remaining imports of `internal/cli`, old proto internal paths, old Foundry Script paths.

- [ ] **Step 1: Confirm no old imports remain**

Run:

```bash
rg 'internal/(cli|protoast|protoparse|protodesc|protovalidate|fsast|fstypes|fsgenerator)' .
```

Expected: no Go imports of old internal package paths remain. Historical docs may still mention old names only if they describe prior architecture.

- [ ] **Step 2: Run all Go package tests**

Run:

```bash
go test ./...
```

Expected: all packages pass.

## Task 8: Format, Build, And Integration Verification

**Files:**
- Modify: generated import formatting only where `goimports` changes files.

- [ ] **Step 1: Format code**

Run:

```bash
task fmt
```

Expected: Go files are formatted and imports are organized.

- [ ] **Step 2: Run unit tests**

Run:

```bash
task test
```

Expected: all unit tests pass with `-race`.

- [ ] **Step 3: Build binaries**

Run:

```bash
task build
```

Expected: `bin/anvil` and `bin/protoc-gen-foundryscript` build successfully.

- [ ] **Step 4: Run integration tests**

Run:

```bash
task integration
```

Expected: protoc, plugin, and Buf integration tests pass.

- [ ] **Step 5: Run full CI if tooling is available**

Run:

```bash
task ci
```

Expected: local CI passes. If `golangci-lint` is not installed, record that blocker and report the verified subset.

## Self-Review

- Spec coverage: command ownership, package-manager boundary, protobuf facade, plugin boundary, runtime/schema top-level decisions, tests, and non-goals are covered by Tasks 1 through 8.
- Placeholder scan: no placeholders or deferred implementation steps remain.
- Type consistency: public command APIs consistently use `NewCommand`, root execution uses `anvil.Execute`, protobuf facade uses existing parser/validator/generator types through aliases.
