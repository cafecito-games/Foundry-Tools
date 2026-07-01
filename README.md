# Foundry Tools

Tooling for Foundry Engine projects.

This repository ships package management and protobuf code generation for
Foundry Script:

- `anvil`: direct CLI for Foundry project tooling.
- `protoc-gen-foundryscript`: protoc and Buf plugin.

Generated `.pb.fs` files use Foundry Script namespaces, traits, generics,
nullable types, and typed collections. Public generated APIs avoid `Variant`
unless a Foundry Engine API requires it.

## Install

Homebrew:

```bash
brew install --cask cafecito-games/tap/foundry-tools
```

Go:

```bash
go install github.com/cafecito-games/foundry-tools/cmd/anvil@latest
go install github.com/cafecito-games/foundry-tools/cmd/protoc-gen-foundryscript@latest
```

## Direct CLI

```bash
anvil proto generate -I proto -o foundry/generated proto/player.proto
```

## Package Manager

`anvil pkg` installs Foundry packages declared in `packages.toml` into a
project's `addons/` directory and writes `packages.lock` for reproducible
installs.

Project layout:

```text
game/
  project.foundry
  packages.toml
  packages.lock
  addons/
    my_package/
```

Create a starter manifest next to `project.foundry`:

```bash
anvil pkg init
```

Add and install a package:

```bash
anvil pkg add --name my_package \
  --source git \
  --url https://github.com/org/my_package.git \
  --version v1.0.0 \
  --source-path addons/my_package
```

Install, update, remove, and list packages:

```bash
anvil pkg install
anvil pkg update
anvil pkg update my_package
anvil pkg remove my_package
anvil pkg list
```

Supported sources:

- `git`: clone a Git repository at a tag, branch, or commit SHA.
- `github-release`: download one asset from a GitHub release.
- `archive`: download a direct HTTP(S) zip, `.tar.gz`, or `.tgz` archive.

Example `packages.toml`:

```toml
[packages.my_package]
source = "git"
url = "https://github.com/org/my_package.git"
version = "v1.0.0"
source_path = "addons/my_package"
install_as = "my_package"
exclude = ["editor_only"]
```

Commit both `packages.toml` and `packages.lock`. `anvil pkg install` honors
existing lock pins when the manifest entry has not changed; `anvil pkg update`
intentionally re-resolves pins.

## protoc

```bash
protoc \
  --plugin=protoc-gen-foundryscript="$(which protoc-gen-foundryscript)" \
  --foundryscript_out=foundry/generated \
  -I proto \
  proto/player.proto
```

## Buf

```yaml
version: v2
plugins:
  - local: protoc-gen-foundryscript
    out: foundry/generated
```

Run:

```bash
buf generate
```

## Foundry Options Proto

Print the custom options schema:

```bash
anvil proto print-options-proto > proto/foundrytools/options.proto
protoc-gen-foundryscript --print-options-proto > proto/foundrytools/options.proto
```

Supported file options:

```protobuf
option (foundrytools.namespace) = "cafecito.game.v1";
option (foundrytools.type_prefix) = "Game";
option (foundrytools.emit_runtime) = true;
```

## Development

```bash
task              # local CI without Foundry
task build        # build binaries into ./bin
task test         # Go tests with -race
task integration  # protoc and Buf integration tests
task lint         # golangci-lint
```

Foundry checks require a recent Foundry editor build:

```bash
FOUNDRY_BIN="$HOME/.foundry/bin/foundry.macos.editor.dev.arm64" task foundry:test
```
