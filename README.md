# Foundry Tools

Tooling for Foundry Engine projects.

The first tools in this repository generate strongly typed Foundry Script from
Protocol Buffers schemas:

- `foundry-tools`: direct CLI and future umbrella command.
- `protoc-gen-foundryscript`: protoc and Buf plugin.

## Development

```bash
task          # local CI without Foundry
task build    # build binaries into ./bin
task test     # Go tests with -race
task lint     # golangci-lint
```

Foundry integration tests require a recent Foundry editor binary:

```bash
FOUNDRY_BIN="$HOME/.foundry/bin/foundry.macos.editor.dev.arm64" task foundry:test
```
