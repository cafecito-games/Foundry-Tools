# Foundry Tools Bootstrap Design

## Purpose

Bootstrap `github.com/cafecito-games/foundry-tools` as a Go 1.26 repository for
Foundry Engine development tools. The first toolset generates Foundry Script from
Protocol Buffers schemas.

The initial release ships two binaries:

- `foundry-tools`: umbrella CLI for direct tool usage and future commands.
- `protoc-gen-foundryscript`: standard protoc plugin for protoc and Buf.

The generator should use `gogdproto` as an implementation reference, but the
output model must be idiomatic Foundry Script rather than a direct GDScript port.
Foundry Script features to use intentionally include namespaces, traits,
generics, nullable types, typed collections, `enum_name`, and static typing.

Public generated APIs must not expose `Variant` unless a Foundry API forces that
type and no typed wrapper can reasonably hide it.

## Context

The current repository contains only the initial files: `README.md`, `LICENSE`,
and `.gitignore`.

The reference repository at `/Users/christian/CafecitoGames/gogdproto` already
contains a mature protobuf generator structure:

- Cobra-based direct CLI.
- `protoc-gen-gdscript` plugin protocol handling.
- Proto source parser, descriptor conversion, importer, AST, validator, and
  generator packages.
- Golden tests, integration tests, and Godot runtime fixture tests.
- `Taskfile.yml`, `.golangci.yml`, `.pre-commit-config.yaml`, GitHub Actions,
  GoReleaser, and Homebrew cask publishing.

The Foundry Script grammar at
`/Users/christian/CafecitoGames/godot/modules/foundry_script/GRAMMAR.md`
confirms support for the target language features:

- `namespace` and `import` at file scope.
- `class_name`, `trait_name`, and `enum_name`.
- `extends` and `uses`.
- Generic classes, traits, and functions with type parameter bounds.
- Nullable type suffixes.
- `Array[T]`, `Dictionary[K, V]`, `Callable`, `Signal`, `Coroutine[T]`, and
  `Type[T]`.
- Retroactive conformance via root-level `extend ... uses ...`.

The local Foundry binary is available at
`~/.foundry/bin/foundry.macos.editor.dev.arm64` and reports
`0.1.dev.custom_build.89031bfb6`. It supports `--headless`, `--script`,
`--check-only`, `--import`, and `--test`.

GitHub currently exposes no published latest Foundry release. The releases list
contains a draft `v0.1.0` with platform assets. CI/CD must support a configured
tag and token for private/draft release assets until public releases are
available.

## Goals

- Create a Go 1.26 monorepo foundation that can host multiple Foundry tools.
- Implement the Foundry Script protobuf generator as a first-class tool.
- Preserve the useful `gogdproto` architecture while replacing the GDScript
  output backend.
- Generate strongly typed Foundry Script with namespaces and minimal global name
  collisions.
- Support protoc, Buf, and direct CLI workflows.
- Include local development automation, pre-commit hooks, CI, Foundry
  integration checks, GoReleaser publishing, and Homebrew installation.
- Keep package boundaries small and testable.

## Non-Goals

- Do not generate GDScript compatibility output.
- Do not build a docs website in the bootstrap phase.
- Do not implement every future Foundry tool. The bootstrap only prepares the
  layout for future tools.
- Do not expose untyped public APIs for convenience when a typed Foundry Script
  shape exists.

## High-Level Architecture

The repository is a Go module with command packages under `cmd/` and focused
internal packages under `internal/`. Shared protobuf parsing and validation code
is independent from the Foundry Script renderer so future tools can reuse it.

The generator pipeline is:

1. Parse or receive protobuf descriptors.
2. Convert descriptors into a stable internal proto AST.
3. Validate supported proto3 features and Foundry-specific options.
4. Resolve all generated type names and namespaces across the request import
   closure.
5. Build a Foundry Script AST.
6. Render `.pb.fs` files plus the typed protobuf runtime support files.
7. Verify generated output through golden tests and Foundry headless checks.

Each stage owns one responsibility and communicates through explicit data
structures. CLI code wires stages together but does not contain generation
logic.

## Package Layout

```text
cmd/foundry-tools/
cmd/protoc-gen-foundryscript/

internal/cli/
internal/plugin/
internal/protoast/
internal/protodesc/
internal/protoparse/
internal/protovalidate/
internal/foundrytoolspb/
internal/fsast/
internal/fstypes/
internal/fsgenerator/
internal/runtime/
internal/foundrytest/

proto/foundrytools/options.proto
examples/
tests/integration/
tests/foundry/
scripts/ci/
.github/workflows/
```

Responsibilities:

- `cmd/foundry-tools`: program entrypoint for the umbrella CLI.
- `cmd/protoc-gen-foundryscript`: program entrypoint for protoc plugin IO.
- `internal/cli`: Cobra commands, version flag, direct generation command, and
  `--print-options-proto`.
- `internal/plugin`: CodeGeneratorRequest/Response protocol handling and plugin
  parameter parsing.
- `internal/protoast`: internal protobuf schema model copied and adapted from
  `gogdproto/internal/ast`.
- `internal/protodesc`: descriptor-to-AST conversion copied and adapted from
  `gogdproto/internal/descriptors`.
- `internal/protoparse`: source parser and filesystem importer copied and
  adapted from `gogdproto/internal/parser`, `lexer`, and `importer`.
- `internal/protovalidate`: proto3 and Foundry-option validation.
- `internal/foundrytoolspb`: generated Go package for
  `proto/foundrytools/options.proto`, plus embedded option schema bytes.
- `internal/fsast`: renderer-neutral Foundry Script AST nodes.
- `internal/fstypes`: type model and type rendering for Foundry Script.
- `internal/fsgenerator`: protobuf-to-Foundry-Script generation.
- `internal/runtime`: embedded typed `.fs` runtime support library.
- `internal/foundrytest`: helpers to locate/download Foundry and run headless
  syntax/runtime checks.
- `scripts/ci`: shell scripts for Foundry binary installation and release asset
  normalization.

## Commands

`foundry-tools` starts with these commands:

```text
foundry-tools version
foundry-tools proto generate [flags] <proto files...>
foundry-tools proto print-options-proto
```

The direct generation command accepts:

- `-I`, `--proto_path`: repeatable import roots.
- `-o`, `--out`: output directory.
- `--emit-runtime`: `auto`, `always`, or `never`; default `auto`.
- `--namespace`: optional override for files without a proto package.
- `--strict-public-api`: default true; rejects public generated `Variant`.

`protoc-gen-foundryscript` accepts the standard plugin request on stdin and
writes a `CodeGeneratorResponse` to stdout. It also supports
`--print-options-proto` for users who need to vendor the options schema.

## Foundry Protobuf Options

Create `proto/foundrytools/options.proto` using proto2 custom extensions:

```protobuf
syntax = "proto2";

package foundrytools;

import "google/protobuf/descriptor.proto";

option go_package = "github.com/cafecito-games/foundry-tools/internal/foundrytoolspb";

extend google.protobuf.FileOptions {
  optional string namespace = 52000;
  optional string type_prefix = 52001;
  optional bool emit_runtime = 52002;
}
```

Rules:

- `namespace` overrides the namespace derived from `package`.
- `type_prefix` is available as a collision escape hatch, not the default naming
  strategy.
- `emit_runtime` forces runtime emission for a file when direct CLI or plugin
  callers need explicit behavior.
- Extension field numbers start at `52000` to avoid conflicting with
  `gdproto.class_prefix` at `51000`.

## Generated File Model

Generated files use `.pb.fs`.

Default mapping:

- Proto package `cafecito.game.v1` becomes `namespace cafecito.game.v1`.
- Imports become Foundry Script `import` declarations for generated namespaces.
- Top-level proto messages become `final class_name`.
- Top-level proto enums use `enum_name` when emitted as standalone enum files.
- Nested messages and enums remain nested where Foundry Script can express them
  clearly. If a nested type needs a standalone file for cyclic dependency or
  import reasons, the resolver emits a deterministic sibling file and keeps the
  source type path in comments for diagnostics.

Example shape:

```gdscript
namespace cafecito.game.v1
import foundry.proto

final class_name Player extends RefCounted uses foundry.proto.Message[Player]

var _name: String = ""
var _level: int = 0
var _position: Position? = null
var _tags: Array[String] = []
var _scores: Dictionary[String, int] = {}

func set_name(value: String) -> void:
	_name = value

func get_name() -> String:
	return _name

static func from_bytes(data: PackedByteArray) -> foundry.proto.DecodeResult[Player]:
	var message: Player = Player.new()
	var err: foundry.proto.ProtobufError = message.merge_from_bytes(data)
	return foundry.proto.DecodeResult[Player].from(message, err)

func to_bytes() -> PackedByteArray:
	pass

func merge_from_bytes(data: PackedByteArray) -> foundry.proto.ProtobufError:
	pass
```

The implementation should render real bodies, not `pass`; the snippet above
shows the public type shape.

## Type Mapping

Scalar mapping:

```text
double   -> float
float    -> float
int32    -> int
int64    -> int
uint32   -> int
uint64   -> int
sint32   -> int
sint64   -> int
fixed32  -> int
fixed64  -> int
sfixed32 -> int
sfixed64 -> int
bool     -> bool
string   -> String
bytes    -> PackedByteArray
enum     -> Namespace.EnumType or wrapper enum type
message  -> MessageType
```

Field shape:

- Singular scalar: non-nullable with proto3 zero default.
- Singular message: nullable `MessageType?` by default.
- Optional scalar: generated presence flag plus typed getter/setter; getter
  remains scalar, `has_field() -> bool` exposes presence.
- Optional message: nullable `MessageType?` plus `has_field() -> bool`.
- Repeated scalar/message: `Array[T]`.
- Map: `Dictionary[K, V]`.
- Oneof: generated enum for the active case plus typed `set_`, `get_`, and
  `has_` accessors per field.

Public API methods return typed values. Public decode helpers return
`DecodeResult[T]` instead of untyped dictionaries.

## Runtime Library

The generator emits or installs a typed runtime namespace, `foundry.proto`.

Runtime types:

- `trait_name Message[T]`
  - `func to_bytes() -> PackedByteArray`
  - `func merge_from_bytes(data: PackedByteArray) -> ProtobufError`
- `trait_name Codec[T]`
  - typed encode/decode operations for reusable codecs.
- `class_name DecodeResult[T]`
  - `var value: T?`
  - `var error: ProtobufError`
  - `func is_ok() -> bool`
- `class_name FieldRead[T]`
  - `var value: T`
  - `var offset: int`
  - `var error: ProtobufError`
- `enum_name ProtobufError`
  - `OK`
  - `VARINT_NOT_FOUND`
  - `VARINT_TOO_LONG`
  - `WIRE_TYPE_MISMATCH`
  - `LENGTH_DELIMITED_SIZE_NOT_FOUND`
  - `LENGTH_DELIMITED_SIZE_MISMATCH`
  - `UNKNOWN_REQUIRED_FEATURE`

Runtime helpers use typed `FieldRead[T]` instead of
`Dictionary[String, int]`-style return payloads.

`Variant` is allowed only in private adapter code when Foundry Engine APIs
force it. Any such usage must be isolated in runtime internals and covered by a
public API signature scan.

## Name Resolution

The resolver indexes every proto type by fully qualified proto name and maps it
to a generated Foundry namespace plus type path.

Resolution rules:

- Proto package controls default namespace.
- `foundrytools.namespace` overrides file namespace.
- Same-file references use the shortest unambiguous name.
- Cross-file references use imported namespace qualification.
- Collisions in one namespace are errors unless `type_prefix` disambiguates.
- Generated names must be valid Foundry Script identifiers.
- Reserved Foundry Script keywords are escaped deterministically, for example by
  suffixing `_`.

## Error Handling

CLI and direct parser errors return process exit code `1` with actionable file,
line, and column details.

Plugin errors are written to `CodeGeneratorResponse.Error` and the process exits
cleanly, matching protoc plugin expectations.

Validation catches:

- non-proto3 syntax.
- unsupported groups.
- invalid Foundry namespace/type options.
- generated name collisions.
- unsupported map key types.
- public `Variant` signature violations.
- unresolved cross-file type references.

Generation errors include the source proto filename and type path.

## Testing Strategy

Unit tests:

- Proto AST conversion from descriptors.
- Source parser/importer behavior.
- Validation errors.
- Foundry Script AST rendering.
- Type mapping and nullable/collection rendering.
- Name resolution and collision diagnostics.
- Runtime helper rendering.
- Public API signature scanner for `Variant`.

Golden tests:

- Scalars.
- Repeated fields.
- Maps.
- Oneofs.
- Optional fields.
- Nested messages/enums.
- Cross-file imports.
- Namespace overrides.
- Type prefix collision escape.
- Runtime output.

Integration tests:

- Direct CLI generation from `.proto`.
- protoc plugin generation.
- Buf generation.
- `--print-options-proto`.
- binary wire compatibility against Go protobuf messages.

Foundry tests:

- Generate fixture `.pb.fs` files into a minimal Foundry project.
- Run Foundry with `--headless --check-only --script` for syntax/type checks.
- Run a small script that serializes and deserializes generated messages.
- Run strict public API scan before Foundry runtime tests.

Local Foundry lookup order:

1. `FOUNDRY_BIN`.
2. `~/.foundry/bin/foundry.macos.editor.dev.arm64`.
3. CI-installed binary under `.cache/foundry`.

## Development Automation

Use `Taskfile.yml` with these tasks:

```text
task build
task install
task test
task test:cover
task integration
task foundry:test
task lint
task fmt
task fmt:check
task gen-options
task tidy
task tidy:check
task ci
```

`task ci` runs:

1. `fmt:check`
2. `tidy:check`
3. `lint`
4. `test`
5. `build`

Foundry integration remains a separate task because it downloads or runs a large
engine binary.

## Pre-Commit

Copy the `gogdproto` pre-commit structure and adapt excludes:

- `trailing-whitespace`
- `end-of-file-fixer`
- `check-yaml`
- `check-added-large-files`
- `check-merge-conflict`
- local `task fmt:check`
- local `task lint`
- local `task test`

Generated `.pb.fs` golden fixtures may need whitespace/EOF exclusions if stable
rendering requires exact fixture formatting.

## CI/CD

### Pull Request CI

`.github/workflows/ci.yml`:

- checkout.
- set up Go `1.26.x`.
- install Task.
- install golangci-lint.
- install protoc.
- install Buf.
- run `task ci`.
- run `task integration`.
- run `task test:cover`.
- upload coverage artifact.

### Foundry Integration CI

`.github/workflows/foundry.yml`:

- checkout.
- set up Go `1.26.x`.
- install protoc and Buf.
- build generator binaries.
- install Foundry.
- run `task foundry:test`.

Foundry install behavior:

- Inputs/env:
  - `FOUNDRY_RELEASE_TAG`, default `v0.1.0` once assets are public.
  - `FOUNDRY_RELEASE_TOKEN`, optional token for draft/private assets.
  - `FOUNDRY_ASSET_PATTERN`, default platform-specific zip pattern.
- Public releases are downloaded through GitHub release URLs.
- Draft/private releases are downloaded through `gh api` using
  `FOUNDRY_RELEASE_TOKEN`.
- Assets are unpacked into `.cache/foundry`.
- The binary path is exported as `FOUNDRY_BIN`.

### Release

`.github/workflows/release.yml`:

- tag-triggered and manual semantic bump release.
- require manual releases from `main`.
- compute next strict semver tag for manual runs.
- run GoReleaser.
- publish archives and checksums.
- update Homebrew cask in `cafecito-games/homebrew-tap`.

GoReleaser builds:

- `foundry-tools`
- `protoc-gen-foundryscript`

Targets:

- darwin amd64
- darwin arm64
- linux amd64
- linux arm64

Homebrew cask:

- name: `foundry-tools`
- binaries:
  - `foundry-tools`
  - `protoc-gen-foundryscript`
- strip macOS quarantine attributes in post-install hook, matching the
  `gogdproto` release pattern.

## Documentation

Initial README sections:

- Install through Homebrew.
- Install through Go.
- Direct CLI quickstart.
- protoc quickstart.
- Buf quickstart.
- Foundry options proto installation.
- Development commands.
- CI and Foundry binary notes.

Deeper docs can be added later after the generator stabilizes.

## Migration From `gogdproto`

Copy and adapt:

- Go module tooling patterns.
- CLI/version structure.
- protoc plugin protocol handling.
- proto parser, importer, descriptor converter, AST, and validator.
- golden and integration test patterns.
- `.golangci.yml`.
- `.pre-commit-config.yaml`.
- `Taskfile.yml`.
- GoReleaser and Homebrew cask structure.

Rewrite:

- GDScript AST package into Foundry Script AST.
- Generator package.
- Runtime support file.
- Naming model from class-prefix-first to namespace-first.
- Public decode helpers from dictionary-shaped results to generic typed result
  classes.

## Acceptance Criteria

- `go test -race -count=1 ./...` passes.
- `task ci` passes locally without Foundry.
- `task integration` passes with protoc and Buf.
- `task foundry:test` passes when `FOUNDRY_BIN` points at a recent Foundry
  editor build.
- Generated golden `.pb.fs` files parse under Foundry `--check-only`.
- Generated public APIs have no `Variant` in signatures.
- GoReleaser snapshot builds both binaries.
- Homebrew cask links both binaries.

## Deferred Work

- Documentation website.
- Binary package managers beyond Homebrew.
- Foundry editor plugin integration.
- Performance benchmarking against large proto schemas.
- Proto2 support.
- gRPC/service generation.
