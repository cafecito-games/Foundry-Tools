# Internal Reorganization Design

## Goal

Reorganize `internal/` so each major tool domain has a clear facade package,
with private implementation packages hidden below it. The package manager is the
model: `internal/packagemanager` exposes the API and Cobra command surface, while
`internal/packagemanager/internal/...` holds implementation details.

This is a structural refactor only. Behavior, command names, generated output,
exit-code mapping, and package-manager file formats stay unchanged.

## Current Problems

The package manager already has a clear boundary:

```text
internal/packagemanager/
  api.go
  runner.go
  internal/
    installer/
    lockfile/
    manifest/
    output/
    project/
    source/
```

The protobuf and Foundry Script generation code is still flat:

```text
internal/protoast
internal/protoparse
internal/protodesc
internal/protovalidate
internal/fsast
internal/fstypes
internal/fsgenerator
```

`internal/cli` also acts as a central registry and knows too much about domain
implementation. It wires package-manager commands, protobuf commands, direct
file generation, runtime file writing, package-manager error rendering, and the
root command. That makes `main.go` small, but puts the wrong ownership in
`internal/cli`.

## Target Shape

The command binary should compose domain commands. The domain packages should own
their Cobra command APIs.

```text
internal/
  anvil/
  foundrytest/
  foundrytoolspb/
  packagemanager/
    internal/
      installer/
      lockfile/
      manifest/
      output/
      project/
      source/
  plugin/
  proto/
    internal/
      ast/
      desc/
      foundryscript/
        ast/
        generator/
        types/
      parser/
      validate/
  runtime/
```

`internal/cli` should be removed or reduced into `internal/anvil`. The
`internal/anvil` package owns only root-command composition helpers and
process-level execution behavior such as package-manager exit-code rendering.
It does not parse protobuf files, generate Foundry Script, or implement package
operations.

## Command Ownership

`cmd/anvil/main.go` becomes the composition point. It should create the root
command, register domain commands, and execute:

```go
func main() {
	stdout := os.Stdout
	stderr := os.Stderr

	root := anvil.NewRootCommand(stdout, stderr)
	root.AddCommand(version.NewCommand(stdout))
	root.AddCommand(proto.NewCommand(stdout))
	root.AddCommand(packagemanager.NewCommand(stdout, stderr))

	os.Exit(anvil.Execute(root, stdout, stderr))
}
```

Exact function signatures may differ slightly if tests show a cleaner shape,
but command ownership must remain the same:

- `internal/anvil`: root command shell, shared execution, top-level error
  rendering.
- `internal/packagemanager`: public package-manager API and `pkg` Cobra command.
- `internal/proto`: public protobuf generation API and `proto` Cobra command.
- `internal/plugin`: protoc plugin adapter for `protoc-gen-foundryscript`.
- `cmd/anvil`: command composition only.
- `cmd/protoc-gen-foundryscript`: plugin process adapter only.

## Protobuf Domain Boundary

`internal/proto` is the stable internal facade for source-based and
descriptor-based protobuf generation. It should expose the operations needed by
the CLI and plugin without forcing either package to import parser, validator,
descriptor, AST, or Foundry Script generator internals.

The package moves are:

```text
internal/protoast      -> internal/proto/internal/ast
internal/protoparse    -> internal/proto/internal/parser
internal/protodesc     -> internal/proto/internal/desc
internal/protovalidate -> internal/proto/internal/validate
internal/fsast         -> internal/proto/internal/foundryscript/ast
internal/fstypes       -> internal/proto/internal/foundryscript/types
internal/fsgenerator   -> internal/proto/internal/foundryscript/generator
```

The intended internal flow is:

```text
source .proto files or protoc descriptors
  -> proto/internal/parser or proto/internal/desc
  -> proto/internal/validate
  -> proto/internal/foundryscript/generator
  -> generated .pb.fs files
```

The `internal/proto` facade should provide enough API for both command paths:

- Direct CLI generation from `.proto` files and import roots.
- Descriptor-based generation from `CodeGeneratorRequest`.
- Validation error formatting in the same form users see today.
- Access to the embedded options proto through the `proto` command.

The facade can expose small primitives such as `ParseFiles`, `Validate`,
`Generate`, and `FromCodeGeneratorRequest`, or higher-level helpers such as
`GenerateFiles` and `GenerateFromCodeGeneratorRequest`. The implementation plan
should choose the lowest-churn API that keeps `internal/anvil`,
`internal/plugin`, and command code out of private packages.

## Package Manager Boundary

`internal/packagemanager` already follows the desired implementation structure.
The reorganization should add command ownership to the package rather than
changing its file formats or install behavior.

The current `internal/cli/pkg.go` command code should move into
`internal/packagemanager`, adjusted to expose `NewCommand(...) *cobra.Command`.
The command can continue calling package-level variables for tests if that is
the lowest-risk way to preserve existing command tests.

The existing public API types and functions should remain available:

- `Init`, `Add`, `Install`, `Update`, `Remove`, `List`
- `Discover`, `LoadManifest`, `LoadLock`
- `PackageSpec`, `Manifest`, `Lockfile`, `LockEntry`, `Project`
- `Options`, operation option types, result types, exit-code helpers

## Runtime And Schema Packages

`internal/runtime` remains top-level. It owns embedded Foundry Script runtime
files that are appended to generated output. The protobuf facade can call it,
but the runtime package does not need to become a private package unless future
work makes it exclusive to one domain.

`internal/foundrytoolspb` also remains top-level because it is generated schema
support and is consumed by both descriptor conversion and command surfaces. The
`gen-options` task may need its copy path updated only if this package moves;
this design keeps it in place, so the task should remain unchanged.

## Plugin Boundary

`internal/plugin` stays as the protoc plugin adapter. It reads a
`CodeGeneratorRequest`, delegates conversion, validation, and Foundry Script
generation to `internal/proto`, appends runtime files through that facade or the
existing runtime package, and writes a `CodeGeneratorResponse`.

It should not import these private packages after the reorganization:

```text
internal/proto/internal/ast
internal/proto/internal/desc
internal/proto/internal/validate
internal/proto/internal/foundryscript/generator
```

## Testing Strategy

This refactor should preserve the existing behavioral tests while moving them
next to their new packages:

- Package-manager command tests move with the package-manager command.
- Proto command tests move with the proto command or remain at the root-command
  level only when they test registration.
- Parser, validator, descriptor, Foundry Script AST, type, generator, runtime,
  plugin, and package-manager implementation tests move with their packages.
- Integration tests under `tests/integration` should continue to pass unchanged.

Verification commands:

```bash
task fmt
task test
task build
task integration
```

`task ci` is the final local verification target if lint tooling is available.

## Non-Goals

- No command renames.
- No output format changes.
- No package-manager manifest or lockfile changes.
- No generated Foundry Script changes.
- No changes to public protobuf schemas under `proto/foundrytools/`.
- No Foundry runtime behavior changes.
