# Repository Guidelines

## Project Structure & Module Organization

This is a Go module for Foundry Engine tooling. CLI entry points live in `cmd/anvil` and `cmd/protoc-gen-foundryscript`. Core packages are under `internal/`, including protobuf parsing/validation, Foundry Script AST and generation, plugin support, and embedded runtime files in `internal/runtime/data/`. Public protobuf schemas live in `proto/foundrytools/`. Integration fixtures are in `tests/integration/fixtures/`, Foundry project checks are in `tests/foundry/`, and generated examples/goldens are in `examples/golden/`. CI helper scripts live in `scripts/ci/`.

## Build, Test, and Development Commands

Use Task for normal workflows:

- `task` or `task ci`: run local CI without Foundry (`fmt:check`, `tidy:check`, lint, unit tests, build).
- `task build`: build `anvil` and `protoc-gen-foundryscript` into `./bin`.
- `task test`: run all Go unit tests with `-race`.
- `task integration`: run protoc, plugin, and Buf integration tests.
- `task test:cover`: write `coverage.out` and print total coverage.
- `task lint`: run `golangci-lint`.
- `task fmt`: run `go fmt` and `goimports`.
- `task gen-options`: regenerate Go stubs and embedded copies for `proto/foundrytools/options.proto`.

Foundry checks require `FOUNDRY_BIN`, for example:

```bash
FOUNDRY_BIN="$HOME/.foundry/bin/foundry.macos.editor.dev.arm64" task foundry:test
```

## Coding Style & Naming Conventions

Target Go `1.26`. Format Go code with `gofmt`/`goimports`; Go files use tabs, while YAML/JSON use two spaces and other files default to four spaces per `.editorconfig`. Keep package names short, lowercase, and domain-specific (`protoparse`, `fsgenerator`, `protovalidate`). Test files should use Go’s `*_test.go` naming. Generated Foundry Script files use `.pb.fs`.

## Testing Guidelines

Place unit tests next to the package they cover. Use integration tests under `tests/integration` with the `integration` build tag when testing external tool flows such as protoc and Buf. Update `examples/golden/` when generator output intentionally changes, and run the relevant unit and integration tasks before opening a PR.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit prefixes such as `feat:`, `fix:`, `test:`, and `ci:`. Keep subjects imperative and scoped to one change. PRs should describe behavior changes, list commands run, link related issues when applicable, and call out regenerated protobuf, golden, runtime, or release artifacts. Include Foundry test notes when a change affects generated `.fs` output or runtime behavior.
