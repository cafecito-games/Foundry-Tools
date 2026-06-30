# Foundry Tools Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the Go 1.26 Foundry Tools repository and ship the first Foundry Script protobuf generator through CLI, protoc plugin, tests, CI/CD, and Homebrew release automation.

**Architecture:** The repository uses small Go packages for protobuf parsing/conversion, validation, Foundry Script AST rendering, generation, runtime embedding, CLI wiring, and CI helpers. Existing `gogdproto` packages provide the proto parser/importer/descriptors/tooling baseline, while the Foundry Script backend is new and namespace-first. The generator pipeline is descriptor/source input -> internal proto AST -> validation -> name resolution -> Foundry Script AST/source -> runtime files -> CLI/plugin response.

**Tech Stack:** Go 1.26, Cobra, google.golang.org/protobuf, protoc plugin protocol, Buf/protoc integration, Task, golangci-lint v2, pre-commit, Foundry headless checks, GitHub Actions, GoReleaser, Homebrew casks.

---

## File Structure

Create these files and directories:

```text
go.mod
go.sum
Taskfile.yml
.editorconfig
.golangci.yml
.pre-commit-config.yaml
.goreleaser.yaml

cmd/foundry-tools/main.go
cmd/protoc-gen-foundryscript/main.go

internal/cli/doc.go
internal/cli/root.go
internal/cli/root_test.go
internal/cli/version.go

internal/plugin/doc.go
internal/plugin/plugin.go
internal/plugin/plugin_test.go

internal/protoast/doc.go
internal/protoast/ast.go
internal/protoast/ast_test.go

internal/protodesc/doc.go
internal/protodesc/converter.go
internal/protodesc/converter_test.go

internal/protoparse/doc.go
internal/protoparse/importer.go
internal/protoparse/parser.go
internal/protoparse/parser_test.go

internal/protovalidate/doc.go
internal/protovalidate/error.go
internal/protovalidate/validator.go
internal/protovalidate/validator_test.go

internal/foundrytoolspb/doc.go
internal/foundrytoolspb/embed.go
internal/foundrytoolspb/embed_test.go
internal/foundrytoolspb/options.proto
internal/foundrytoolspb/options.pb.go

internal/fstypes/doc.go
internal/fstypes/types.go
internal/fstypes/types_test.go

internal/fsast/doc.go
internal/fsast/node.go
internal/fsast/declarations.go
internal/fsast/expressions.go
internal/fsast/statements.go
internal/fsast/render_test.go

internal/runtime/doc.go
internal/runtime/runtime.go
internal/runtime/runtime_test.go
internal/runtime/data/foundry/proto/message.fs
internal/runtime/data/foundry/proto/codec.fs
internal/runtime/data/foundry/proto/decode_result.fs
internal/runtime/data/foundry/proto/field_read.fs
internal/runtime/data/foundry/proto/protobuf_error.fs
internal/runtime/data/foundry/proto/wire.fs

internal/fsgenerator/doc.go
internal/fsgenerator/generator.go
internal/fsgenerator/names.go
internal/fsgenerator/types.go
internal/fsgenerator/files.go
internal/fsgenerator/fields.go
internal/fsgenerator/serialize.go
internal/fsgenerator/deserialize.go
internal/fsgenerator/public_api.go
internal/fsgenerator/generator_test.go
internal/fsgenerator/public_api_test.go

internal/foundrytest/doc.go
internal/foundrytest/foundry.go
internal/foundrytest/foundry_test.go

proto/foundrytools/options.proto

examples/example.proto
examples/golden/cafecito/game/v1/Player.pb.fs
examples/golden/cafecito/game/v1/Position.pb.fs
examples/golden/cafecito/game/v1/PlayerStatus.pb.fs

tests/integration/helpers_test.go
tests/integration/direct_cli_test.go
tests/integration/protoc_plugin_test.go
tests/integration/buf_test.go
tests/integration/fixtures/basic/player.proto
tests/integration/fixtures/basic/buf.yaml
tests/integration/fixtures/basic/buf.gen.yaml

tests/foundry/project.foundry
tests/foundry/main.fs
tests/foundry/run.sh

scripts/ci/install-foundry.sh

.github/workflows/ci.yml
.github/workflows/foundry.yml
.github/workflows/release.yml
```

Copy and adapt these existing reference areas from `/Users/christian/CafecitoGames/gogdproto`:

```text
internal/ast -> internal/protoast
internal/descriptors -> internal/protodesc
internal/importer, internal/lexer, internal/parser -> internal/protoparse
internal/validator -> internal/protovalidate
internal/cli structure -> internal/cli
cmd/gdproto/main.go -> cmd/foundry-tools/main.go
cmd/protoc-gen-gdscript/main.go -> cmd/protoc-gen-foundryscript/main.go
Taskfile.yml
.editorconfig
.golangci.yml
.pre-commit-config.yaml
.goreleaser.yaml
.github/workflows/pr.yml
.github/workflows/godot.yml
.github/workflows/release.yml
```

Rename package imports from `github.com/cafecito-games/gdproto` to `github.com/cafecito-games/foundry-tools`, and rename GDScript/Godot terms to Foundry Script/Foundry where those terms describe generated output or integration tests.

---

### Task 1: Repository Tooling Scaffold

**Files:**
- Create: `go.mod`
- Create: `Taskfile.yml`
- Create: `.editorconfig`
- Create: `.golangci.yml`
- Create: `.pre-commit-config.yaml`
- Modify: `.gitignore`
- Modify: `README.md`

- [ ] **Step 1: Write the expected module and tooling files**

Create `go.mod`:

```go
module github.com/cafecito-games/foundry-tools

go 1.26

require (
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
```

Create `Taskfile.yml`:

```yaml
version: '3'

vars:
  PKG: ./...
  BIN_DIR: bin
  FOUNDRY_BIN:
    sh: printf '%s' "${FOUNDRY_BIN:-$HOME/.foundry/bin/foundry.macos.editor.dev.arm64}"

tasks:
  default:
    desc: Run the full local CI pipeline.
    cmds:
      - task: ci

  build:
    desc: Build all binaries into ./bin.
    cmds:
      - mkdir -p {{.BIN_DIR}}
      - go build -o {{.BIN_DIR}}/foundry-tools ./cmd/foundry-tools
      - go build -o {{.BIN_DIR}}/protoc-gen-foundryscript ./cmd/protoc-gen-foundryscript

  install:
    desc: Install Foundry Tools binaries into $GOPATH/bin.
    cmds:
      - go install ./cmd/foundry-tools
      - go install ./cmd/protoc-gen-foundryscript

  test:
    desc: Run unit tests.
    cmds:
      - go test -race -count=1 {{.PKG}}

  test:cover:
    desc: Run tests with coverage; output coverage.out.
    cmds:
      - go test -race -count=1 -coverprofile=coverage.out -covermode=atomic {{.PKG}}
      - go tool cover -func=coverage.out | tail -1

  integration:
    desc: Run protoc, plugin, and Buf integration tests.
    cmds:
      - go test -tags=integration ./tests/integration/... -v -count=1

  foundry:test:
    desc: Run generated Foundry Script checks with a Foundry editor binary.
    cmds:
      - test -x "{{.FOUNDRY_BIN}}" || (echo "Foundry binary not found or not executable: {{.FOUNDRY_BIN}}" && exit 1)
      - FOUNDRY_BIN="{{.FOUNDRY_BIN}}" bash tests/foundry/run.sh

  lint:
    desc: Run golangci-lint.
    cmds:
      - golangci-lint run

  fmt:
    desc: Format Go source.
    cmds:
      - go fmt {{.PKG}}
      - go run golang.org/x/tools/cmd/goimports@latest -w .

  fmt:check:
    desc: Verify all Go files are formatted.
    cmds:
      - |
        unformatted=$(gofmt -l . | grep -v '^vendor/' || true)
        if [ -n "$unformatted" ]; then
          echo "Unformatted files:"
          echo "$unformatted"
          exit 1
        fi
    silent: true

  gen-options:
    desc: Generate Go stubs for foundrytools/options.proto and refresh the embedded copy.
    cmds:
      - protoc -I proto --go_out=. --go_opt=module=github.com/cafecito-games/foundry-tools proto/foundrytools/options.proto
      - cp proto/foundrytools/options.proto internal/foundrytoolspb/options.proto

  tidy:
    desc: Run go mod tidy.
    cmds:
      - go mod tidy

  tidy:check:
    desc: Verify go.mod/go.sum are tidy.
    cmds:
      - go mod tidy
      - |
        if ! git diff --quiet -- go.mod go.sum; then
          echo "go.mod/go.sum are not tidy. Run 'task tidy' and commit the result."
          git --no-pager diff -- go.mod go.sum
          exit 1
        fi
    silent: true

  ci:
    desc: Full local CI pipeline without Foundry.
    cmds:
      - task: fmt:check
      - task: tidy:check
      - task: lint
      - task: test
      - task: build
```

Copy `.editorconfig`, `.golangci.yml`, and `.pre-commit-config.yaml` from `gogdproto`, then apply these substitutions:

```bash
cp /Users/christian/CafecitoGames/gogdproto/.editorconfig .
cp /Users/christian/CafecitoGames/gogdproto/.golangci.yml .
cp /Users/christian/CafecitoGames/gogdproto/.pre-commit-config.yaml .
perl -0pi -e 's/gdproto/foundry-tools/g; s/protoc-gen-gdscript/protoc-gen-foundryscript/g; s/GDScript/Foundry Script/g' .pre-commit-config.yaml
```

Update `.gitignore` to include generated binaries and Foundry cache:

```gitignore
bin/
.cache/
dist/
coverage.out
```

Replace `README.md` with:

```markdown
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
```

- [ ] **Step 2: Run tidy and verify the initial scaffold fails only because code packages do not exist**

Run:

```bash
go mod tidy
task fmt:check
```

Expected:

```text
task fmt:check
```

passes because no Go source exists yet. `go mod tidy` creates `go.sum`.

- [ ] **Step 3: Commit**

Run:

```bash
git add go.mod go.sum Taskfile.yml .editorconfig .golangci.yml .pre-commit-config.yaml .gitignore README.md
git commit -m "chore: scaffold go tooling"
```

---

### Task 2: CLI Entrypoints and Version Command

**Files:**
- Create: `cmd/foundry-tools/main.go`
- Create: `cmd/protoc-gen-foundryscript/main.go`
- Create: `internal/cli/doc.go`
- Create: `internal/cli/version.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/root_test.go`

- [ ] **Step 1: Write failing CLI tests**

Create `internal/cli/root_test.go`:

```go
package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommandPrintsVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "foundry-tools dev")
	require.Empty(t, stderr.String())
}

func TestProtoPrintOptionsCommandIsWired(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "print-options-proto"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "options proto support is not wired")
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestVersionCommandPrintsVersion|TestProtoPrintOptionsCommandIsWired' -count=1
```

Expected:

```text
FAIL
undefined: NewRootCommand
```

- [ ] **Step 3: Implement CLI package**

Create `internal/cli/doc.go`:

```go
// Package cli defines the Foundry Tools command-line interface.
package cli
```

Create `internal/cli/version.go`:

```go
package cli

// Version is set by release builds through ldflags.
var Version = "dev"
```

Create `internal/cli/root.go`:

```go
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// NewRootCommand returns the root foundry-tools command.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "foundry-tools",
		Short:         "Tooling for Foundry Engine projects",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.AddCommand(newVersionCommand(stdout))
	cmd.AddCommand(newProtoCommand(stdout))
	return cmd
}

func newVersionCommand(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(stdout, "foundry-tools %s\n", Version)
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
			_ = stdout
			return fmt.Errorf("options proto support is not wired")
		},
	})
	return cmd
}
```

Create `cmd/foundry-tools/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/cafecito-games/foundry-tools/internal/cli"
)

func main() {
	cmd := cli.NewRootCommand(os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Create `cmd/protoc-gen-foundryscript/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "protoc-gen-foundryscript plugin support is not wired")
	os.Exit(1)
}
```

- [ ] **Step 4: Run tests and build**

Run:

```bash
go test ./internal/cli -count=1
task build
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/cli
```

and `bin/foundry-tools`, `bin/protoc-gen-foundryscript` exist.

- [ ] **Step 5: Commit**

Run:

```bash
git add cmd internal/cli
git commit -m "feat: add foundry-tools cli skeleton"
```

---

### Task 3: Foundry Tools Protobuf Options Package

**Files:**
- Create: `proto/foundrytools/options.proto`
- Create: `internal/foundrytoolspb/doc.go`
- Create: `internal/foundrytoolspb/embed.go`
- Create: `internal/foundrytoolspb/embed_test.go`
- Create: `internal/foundrytoolspb/options.proto`
- Create: `internal/foundrytoolspb/options.pb.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Write failing embed test**

Create `internal/foundrytoolspb/embed_test.go`:

```go
package foundrytoolspb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytesReturnsOptionsProto(t *testing.T) {
	text := string(Bytes())

	require.Contains(t, text, `package foundrytools;`)
	require.Contains(t, text, `optional string namespace = 52000;`)
	require.Contains(t, text, `optional string type_prefix = 52001;`)
	require.Contains(t, text, `optional bool emit_runtime = 52002;`)
	require.True(t, strings.HasSuffix(text, "\n"))
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/foundrytoolspb -run TestBytesReturnsOptionsProto -count=1
```

Expected:

```text
FAIL
no Go files in internal/foundrytoolspb
```

- [ ] **Step 3: Create options proto and embed package**

Create `proto/foundrytools/options.proto`:

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

Create `internal/foundrytoolspb/doc.go`:

```go
// Package foundrytoolspb contains generated Go bindings and embedded schema
// bytes for foundrytools/options.proto.
package foundrytoolspb
```

Create `internal/foundrytoolspb/embed.go`:

```go
package foundrytoolspb

import _ "embed"

//go:embed options.proto
var optionsProto []byte

// Bytes returns the embedded foundrytools/options.proto schema.
func Bytes() []byte {
	out := make([]byte, len(optionsProto))
	copy(out, optionsProto)
	return out
}
```

Run:

```bash
task gen-options
```

Expected:

```text
internal/foundrytoolspb/options.pb.go
internal/foundrytoolspb/options.proto
```

exist and compile.

- [ ] **Step 4: Wire CLI print-options-proto**

Modify `internal/cli/root.go` imports to include:

```go
	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
```

Replace the `print-options-proto` command `RunE` body with:

```go
		RunE: func(_ *cobra.Command, _ []string) error {
			_, err := stdout.Write(foundrytoolspb.Bytes())
			return err
		},
```

Update `internal/cli/root_test.go` `TestProtoPrintOptionsCommandIsWired`:

```go
func TestProtoPrintOptionsCommandIsWired(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "print-options-proto"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `package foundrytools;`)
	require.Empty(t, stderr.String())
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/foundrytoolspb ./internal/cli -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/foundrytoolspb
ok  	github.com/cafecito-games/foundry-tools/internal/cli
```

- [ ] **Step 6: Commit**

Run:

```bash
git add proto/foundrytools internal/foundrytoolspb internal/cli/root.go internal/cli/root_test.go
git commit -m "feat: add foundry protobuf options schema"
```

---

### Task 4: Proto AST, Descriptor Conversion, Parser, Importer, and Validator Baseline

**Files:**
- Create/modify: `internal/protoast/*`
- Create/modify: `internal/protodesc/*`
- Create/modify: `internal/protoparse/*`
- Create/modify: `internal/protovalidate/*`

- [ ] **Step 1: Copy reference packages**

Run:

```bash
mkdir -p internal/protoast internal/protodesc internal/protoparse internal/protovalidate
cp /Users/christian/CafecitoGames/gogdproto/internal/ast/*.go internal/protoast/
cp /Users/christian/CafecitoGames/gogdproto/internal/descriptors/*.go internal/protodesc/
cp /Users/christian/CafecitoGames/gogdproto/internal/lexer/*.go internal/protoparse/
cp /Users/christian/CafecitoGames/gogdproto/internal/parser/*.go internal/protoparse/
cp /Users/christian/CafecitoGames/gogdproto/internal/importer/*.go internal/protoparse/
cp /Users/christian/CafecitoGames/gogdproto/internal/validator/*.go internal/protovalidate/
```

- [ ] **Step 2: Rename packages and imports**

Run:

```bash
perl -pi -e 's/package ast/package protoast/g' internal/protoast/*.go
perl -pi -e 's/package descriptors/package protodesc/g' internal/protodesc/*.go
perl -pi -e 's/package (lexer|parser|importer)/package protoparse/g' internal/protoparse/*.go
perl -pi -e 's/package validator/package protovalidate/g' internal/protovalidate/*.go
perl -pi -e 's#github.com/cafecito-games/gdproto/internal/ast#github.com/cafecito-games/foundry-tools/internal/protoast#g' internal/protodesc/*.go internal/protoparse/*.go internal/protovalidate/*.go
perl -pi -e 's#github.com/cafecito-games/gdproto/internal/gdprotopb#github.com/cafecito-games/foundry-tools/internal/foundrytoolspb#g' internal/protodesc/*.go internal/protoparse/*.go
perl -pi -e 's#github.com/cafecito-games/gdproto#github.com/cafecito-games/foundry-tools#g' internal/protoast/*.go internal/protodesc/*.go internal/protoparse/*.go internal/protovalidate/*.go
perl -pi -e 's/gdproto/foundrytools/g' internal/protodesc/*.go internal/protoparse/*.go internal/protovalidate/*.go
perl -pi -e 's/GDScript/Foundry Script/g; s/Godot/Foundry/g' internal/protodesc/*.go internal/protoparse/*.go internal/protovalidate/*.go
```

Manual package import cleanup after this command must ensure every file imports these local package paths:

```go
github.com/cafecito-games/foundry-tools/internal/protoast
github.com/cafecito-games/foundry-tools/internal/foundrytoolspb
```

and no file imports `github.com/cafecito-games/gdproto`.

- [ ] **Step 3: Update custom option keys**

In copied code, replace the old option key and field access:

```go
const namespaceOptionKey = "(foundrytools.namespace)"
const typePrefixOptionKey = "(foundrytools.type_prefix)"
const emitRuntimeOptionKey = "(foundrytools.emit_runtime)"
```

Descriptor conversion must read:

```go
if proto.HasExtension(fdOpts, foundrytoolspb.E_Namespace) {
	if v, ok := proto.GetExtension(fdOpts, foundrytoolspb.E_Namespace).(string); ok {
		file.Options["(foundrytools.namespace)"] = v
	}
}
if proto.HasExtension(fdOpts, foundrytoolspb.E_TypePrefix) {
	if v, ok := proto.GetExtension(fdOpts, foundrytoolspb.E_TypePrefix).(string); ok {
		file.Options["(foundrytools.type_prefix)"] = v
	}
}
if proto.HasExtension(fdOpts, foundrytoolspb.E_EmitRuntime) {
	if v, ok := proto.GetExtension(fdOpts, foundrytoolspb.E_EmitRuntime).(bool); ok {
		file.Options["(foundrytools.emit_runtime)"] = v
	}
}
```

- [ ] **Step 4: Run focused tests**

Run:

```bash
go test ./internal/protoast ./internal/protodesc ./internal/protoparse ./internal/protovalidate -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/protoast
ok  	github.com/cafecito-games/foundry-tools/internal/protodesc
ok  	github.com/cafecito-games/foundry-tools/internal/protoparse
ok  	github.com/cafecito-games/foundry-tools/internal/protovalidate
```

- [ ] **Step 5: Run import hygiene check**

Run:

```bash
rg "github.com/cafecito-games/gdproto|gdprotopb|gdproto\\.class_prefix" internal/protoast internal/protodesc internal/protoparse internal/protovalidate
```

Expected: no matches.

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/protoast internal/protodesc internal/protoparse internal/protovalidate
git commit -m "feat: port protobuf parsing and validation baseline"
```

---

### Task 5: Foundry Script Type Renderer

**Files:**
- Create: `internal/fstypes/doc.go`
- Create: `internal/fstypes/types.go`
- Create: `internal/fstypes/types_test.go`

- [ ] **Step 1: Write failing type renderer tests**

Create `internal/fstypes/types_test.go`:

```go
package fstypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeRendering(t *testing.T) {
	tests := []struct {
		name string
		typ  Type
		want string
	}{
		{name: "simple", typ: Named("Player"), want: "Player"},
		{name: "namespace", typ: Named("cafecito.game.v1.Player"), want: "cafecito.game.v1.Player"},
		{name: "nullable", typ: Nullable(Named("Player")), want: "Player?"},
		{name: "array", typ: Array(Named("String")), want: "Array[String]"},
		{name: "dictionary", typ: Dictionary(Named("String"), Named("int")), want: "Dictionary[String, int]"},
		{name: "generic", typ: Generic("foundry.proto.DecodeResult", Named("Player")), want: "foundry.proto.DecodeResult[Player]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.typ.Render())
		})
	}
}

func TestNoVariantPublicTypeByDefault(t *testing.T) {
	require.False(t, Named("String").IsVariant())
	require.True(t, Named("Variant").IsVariant())
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/fstypes -count=1
```

Expected:

```text
FAIL
undefined: Type
```

- [ ] **Step 3: Implement type renderer**

Create `internal/fstypes/doc.go`:

```go
// Package fstypes models Foundry Script type annotations.
package fstypes
```

Create `internal/fstypes/types.go`:

```go
package fstypes

import "strings"

// Type is a renderable Foundry Script type annotation.
type Type struct {
	name     string
	args     []Type
	nullable bool
}

// Named returns a non-generic named type.
func Named(name string) Type {
	return Type{name: name}
}

// Generic returns name[args...].
func Generic(name string, args ...Type) Type {
	return Type{name: name, args: args}
}

// Nullable returns typ with a nullable suffix.
func Nullable(typ Type) Type {
	typ.nullable = true
	return typ
}

// Array returns Array[element].
func Array(element Type) Type {
	return Generic("Array", element)
}

// Dictionary returns Dictionary[key, value].
func Dictionary(key, value Type) Type {
	return Generic("Dictionary", key, value)
}

// Render returns the Foundry Script syntax for the type.
func (t Type) Render() string {
	var b strings.Builder
	b.WriteString(t.name)
	if len(t.args) > 0 {
		b.WriteByte('[')
		for i, arg := range t.args {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(arg.Render())
		}
		b.WriteByte(']')
	}
	if t.nullable {
		b.WriteByte('?')
	}
	return b.String()
}

// IsVariant reports whether this type's outermost public spelling is Variant.
func (t Type) IsVariant() bool {
	return t.name == "Variant"
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/fstypes -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fstypes
```

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/fstypes
git commit -m "feat: add foundry script type renderer"
```

---

### Task 6: Foundry Script AST Renderer

**Files:**
- Create: `internal/fsast/doc.go`
- Create: `internal/fsast/node.go`
- Create: `internal/fsast/declarations.go`
- Create: `internal/fsast/expressions.go`
- Create: `internal/fsast/statements.go`
- Create: `internal/fsast/render_test.go`

- [ ] **Step 1: Write failing renderer test**

Create `internal/fsast/render_test.go`:

```go
package fsast

import (
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/stretchr/testify/require"
)

func TestFileRendering(t *testing.T) {
	file := File{
		Namespace: "cafecito.game.v1",
		Imports:   []string{"foundry.proto"},
		Declarations: []Node{
			Class{
				Final:   true,
				Name:    "Player",
				Extends: "RefCounted",
				Uses:    []string{"foundry.proto.Message[Player]"},
				Members: []Node{
					Var{Name: "_name", Type: fstypes.Named("String"), Value: `""`},
					Func{
						Name:       "get_name",
						ReturnType: fstypes.Named("String"),
						Body:       []Node{Return{Value: "_name"}},
					},
				},
			},
		},
	}

	require.Equal(t, "namespace cafecito.game.v1\nimport foundry.proto\n\nfinal class_name Player extends RefCounted uses foundry.proto.Message[Player]\n\nvar _name: String = \"\"\n\nfunc get_name() -> String:\n\treturn _name\n", file.Render())
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/fsast -count=1
```

Expected:

```text
FAIL
undefined: File
```

- [ ] **Step 3: Implement renderer primitives**

Create `internal/fsast/doc.go`:

```go
// Package fsast renders a small Foundry Script AST used by code generation.
package fsast
```

Create `internal/fsast/node.go`:

```go
package fsast

import (
	"strings"
)

// Node renders Foundry Script source at an indentation level.
type Node interface {
	RenderAt(indent int) string
}

// File is one Foundry Script source file.
type File struct {
	Namespace    string
	Imports      []string
	Declarations []Node
}

// Render renders the file with a trailing newline.
func (f File) Render() string {
	var parts []string
	if f.Namespace != "" {
		parts = append(parts, "namespace "+f.Namespace)
	}
	for _, imp := range f.Imports {
		parts = append(parts, "import "+imp)
	}
	if len(f.Declarations) > 0 {
		if len(parts) > 0 {
			parts = append(parts, "")
		}
		for _, decl := range f.Declarations {
			parts = append(parts, strings.TrimRight(decl.RenderAt(0), "\n"))
		}
	}
	return strings.Join(parts, "\n") + "\n"
}

func indent(level int) string {
	return strings.Repeat("\t", level)
}
```

Create `internal/fsast/declarations.go`:

```go
package fsast

import (
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/fstypes"
)

// Class renders a class_name declaration.
type Class struct {
	Final   bool
	Name    string
	Extends string
	Uses    []string
	Members []Node
}

// RenderAt renders a class.
func (c Class) RenderAt(level int) string {
	var head strings.Builder
	head.WriteString(indent(level))
	if c.Final {
		head.WriteString("final ")
	}
	head.WriteString("class_name ")
	head.WriteString(c.Name)
	if c.Extends != "" {
		head.WriteString(" extends ")
		head.WriteString(c.Extends)
	}
	if len(c.Uses) > 0 {
		head.WriteString(" uses ")
		head.WriteString(strings.Join(c.Uses, ", "))
	}
	lines := []string{head.String()}
	for _, member := range c.Members {
		lines = append(lines, "")
		lines = append(lines, strings.TrimRight(member.RenderAt(level), "\n"))
	}
	return strings.Join(lines, "\n")
}

// Var renders a variable declaration.
type Var struct {
	Name  string
	Type  fstypes.Type
	Value string
}

// RenderAt renders a variable declaration.
func (v Var) RenderAt(level int) string {
	line := indent(level) + "var " + v.Name + ": " + v.Type.Render()
	if v.Value != "" {
		line += " = " + v.Value
	}
	return line
}

// Func renders a function declaration.
type Func struct {
	Static     bool
	Name       string
	Parameters []Parameter
	ReturnType fstypes.Type
	ReturnVoid bool
	Body       []Node
}

// Parameter is a typed function parameter.
type Parameter struct {
	Name string
	Type fstypes.Type
}

// RenderAt renders a function declaration.
func (f Func) RenderAt(level int) string {
	var b strings.Builder
	b.WriteString(indent(level))
	if f.Static {
		b.WriteString("static ")
	}
	b.WriteString("func ")
	b.WriteString(f.Name)
	b.WriteByte('(')
	for i, p := range f.Parameters {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Name)
		b.WriteString(": ")
		b.WriteString(p.Type.Render())
	}
	b.WriteByte(')')
	if f.ReturnVoid {
		b.WriteString(" -> void")
	} else if f.ReturnType.Render() != "" {
		b.WriteString(" -> ")
		b.WriteString(f.ReturnType.Render())
	}
	b.WriteString(":\n")
	if len(f.Body) == 0 {
		b.WriteString(indent(level + 1))
		b.WriteString("return\n")
		return b.String()
	}
	for i, stmt := range f.Body {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(stmt.RenderAt(level + 1))
	}
	return b.String()
}
```

Create `internal/fsast/statements.go`:

```go
package fsast

// Return renders a return statement.
type Return struct {
	Value string
}

// RenderAt renders a return statement.
func (r Return) RenderAt(level int) string {
	if r.Value == "" {
		return indent(level) + "return"
	}
	return indent(level) + "return " + r.Value
}

// Assign renders an assignment statement.
type Assign struct {
	Target string
	Value  string
}

// RenderAt renders an assignment statement.
func (a Assign) RenderAt(level int) string {
	return indent(level) + a.Target + " = " + a.Value
}

// Expr renders a raw expression statement.
type Expr struct {
	Code string
}

// RenderAt renders an expression statement.
func (e Expr) RenderAt(level int) string {
	return indent(level) + e.Code
}
```

Create `internal/fsast/expressions.go`:

```go
package fsast

// QuoteString returns a Foundry Script string literal for simple generated
// ASCII content.
func QuoteString(value string) string {
	out := `"`
	for _, r := range value {
		switch r {
		case '\\':
			out += `\\`
		case '"':
			out += `\"`
		case '\n':
			out += `\n`
		default:
			out += string(r)
		}
	}
	out += `"`
	return out
}
```

- [ ] **Step 4: Run renderer tests**

Run:

```bash
go test ./internal/fsast -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsast
```

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/fsast
git commit -m "feat: add foundry script ast renderer"
```

---

### Task 7: Typed Runtime Library Embedding

**Files:**
- Create: `internal/runtime/doc.go`
- Create: `internal/runtime/runtime.go`
- Create: `internal/runtime/runtime_test.go`
- Create: `internal/runtime/data/foundry/proto/message.fs`
- Create: `internal/runtime/data/foundry/proto/codec.fs`
- Create: `internal/runtime/data/foundry/proto/decode_result.fs`
- Create: `internal/runtime/data/foundry/proto/field_read.fs`
- Create: `internal/runtime/data/foundry/proto/protobuf_error.fs`
- Create: `internal/runtime/data/foundry/proto/wire.fs`

- [ ] **Step 1: Write failing runtime tests**

Create `internal/runtime/runtime_test.go`:

```go
package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilesReturnsRuntimeSources(t *testing.T) {
	files := Files()

	require.Contains(t, files, "foundry/proto/message.fs")
	require.Contains(t, files, "foundry/proto/decode_result.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	require.Contains(t, files["foundry/proto/message.fs"], "trait_name Message[T]")
	require.Contains(t, files["foundry/proto/decode_result.fs"], "class_name DecodeResult[T]")
	require.NotContains(t, PublicSource(files), "Variant")
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/runtime -count=1
```

Expected:

```text
FAIL
undefined: Files
```

- [ ] **Step 3: Add runtime source files**

Create `internal/runtime/data/foundry/proto/message.fs`:

```gdscript
namespace foundry.proto

trait_name Message[T]

func to_bytes() -> PackedByteArray

func merge_from_bytes(data: PackedByteArray) -> ProtobufError
```

Create `internal/runtime/data/foundry/proto/codec.fs`:

```gdscript
namespace foundry.proto

trait_name Codec[T]

func encode(value: T) -> PackedByteArray

func decode(data: PackedByteArray, offset: int) -> FieldRead[T]
```

Create `internal/runtime/data/foundry/proto/decode_result.fs`:

```gdscript
namespace foundry.proto

class_name DecodeResult[T] extends RefCounted

var value: T?
var error: ProtobufError = ProtobufError.OK

static func from(value_: T?, error_: ProtobufError) -> DecodeResult[T]:
	var result: DecodeResult[T] = DecodeResult[T].new()
	result.value = value_
	result.error = error_
	return result

func is_ok() -> bool:
	return error == ProtobufError.OK
```

Create `internal/runtime/data/foundry/proto/field_read.fs`:

```gdscript
namespace foundry.proto

class_name FieldRead[T] extends RefCounted

var value: T
var offset: int = 0
var error: ProtobufError = ProtobufError.OK

static func from(value_: T, offset_: int, error_: ProtobufError) -> FieldRead[T]:
	var result: FieldRead[T] = FieldRead[T].new()
	result.value = value_
	result.offset = offset_
	result.error = error_
	return result
```

Create `internal/runtime/data/foundry/proto/protobuf_error.fs`:

```gdscript
namespace foundry.proto

enum_name ProtobufError {
	OK = 0,
	VARINT_NOT_FOUND = 1,
	VARINT_TOO_LONG = 2,
	WIRE_TYPE_MISMATCH = 3,
	LENGTH_DELIMITED_SIZE_NOT_FOUND = 4,
	LENGTH_DELIMITED_SIZE_MISMATCH = 5,
	UNKNOWN_REQUIRED_FEATURE = 6,
}
```

Create `internal/runtime/data/foundry/proto/wire.fs`:

```gdscript
namespace foundry.proto

class_name Wire extends RefCounted

const WIRE_VARINT: int = 0
const WIRE_64BIT: int = 1
const WIRE_LENGTH_DELIMITED: int = 2
const WIRE_32BIT: int = 5

static func make_tag(field_number: int, wire_type: int) -> int:
	return (field_number << 3) | wire_type

static func get_wire_type(tag: int) -> int:
	return tag & 0x7

static func get_field_number(tag: int) -> int:
	return tag >> 3

static func encode_varint(value: int) -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	var unsigned_value: int = value
	while unsigned_value > 0x7F or unsigned_value < 0:
		result.append((unsigned_value & 0x7F) | 0x80)
		unsigned_value = (unsigned_value >> 7) & 0x01FFFFFFFFFFFFFF
	result.append(unsigned_value & 0x7F)
	return result

static func decode_varint(data: PackedByteArray, offset: int) -> FieldRead[int]:
	var result_value: int = 0
	var shift: int = 0
	var cursor: int = offset
	while cursor < data.size():
		var byte: int = data[cursor]
		result_value |= (byte & 0x7F) << shift
		cursor += 1
		if (byte & 0x80) == 0:
			return FieldRead[int].from(result_value, cursor, ProtobufError.OK)
		shift += 7
		if shift > 63:
			return FieldRead[int].from(0, cursor, ProtobufError.VARINT_TOO_LONG)
	return FieldRead[int].from(0, cursor, ProtobufError.VARINT_NOT_FOUND)

static func encode_string(value: String) -> PackedByteArray:
	return value.to_utf8_buffer()

static func decode_string(data: PackedByteArray, offset: int, length: int) -> FieldRead[String]:
	if offset + length > data.size():
		return FieldRead[String].from("", offset, ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH)
	var slice: PackedByteArray = data.slice(offset, offset + length)
	return FieldRead[String].from(slice.get_string_from_utf8(), offset + length, ProtobufError.OK)
```

- [ ] **Step 4: Implement embedding**

Create `internal/runtime/doc.go`:

```go
// Package runtime embeds the typed Foundry Script protobuf runtime.
package runtime
```

Create `internal/runtime/runtime.go`:

```go
package runtime

import (
	"embed"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed data/**/*.fs
var runtimeFS embed.FS

// Files returns runtime source files keyed by generated output path.
func Files() map[string]string {
	out := map[string]string{}
	err := fs.WalkDir(runtimeFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".fs" {
			return nil
		}
		data, readErr := runtimeFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		out[strings.TrimPrefix(path, "data/")] = string(data)
		return nil
	})
	if err != nil {
		panic(err)
	}
	return out
}

// PublicSource concatenates runtime sources in stable order for public API scans.
func PublicSource(files map[string]string) string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(files[key])
		b.WriteByte('\n')
	}
	return b.String()
}
```

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/runtime -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/runtime
```

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/runtime
git commit -m "feat: embed typed foundry protobuf runtime"
```

---

### Task 8: Generator Names, Type Mapping, and Public API Scanner

**Files:**
- Create: `internal/fsgenerator/doc.go`
- Create: `internal/fsgenerator/names.go`
- Create: `internal/fsgenerator/types.go`
- Create: `internal/fsgenerator/public_api.go`
- Create: `internal/fsgenerator/public_api_test.go`
- Create: `internal/fsgenerator/generator_test.go`

- [ ] **Step 1: Write failing generator foundation tests**

Create `internal/fsgenerator/generator_test.go`:

```go
package fsgenerator

import (
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
	"github.com/stretchr/testify/require"
)

func TestNamespaceFromPackageAndOption(t *testing.T) {
	require.Equal(t, "cafecito.game.v1", NamespaceFor(&protoast.ProtoFile{Package: "cafecito.game.v1"}))
	require.Equal(t, "custom.ns", NamespaceFor(&protoast.ProtoFile{
		Package: "ignored",
		Options: map[string]any{"(foundrytools.namespace)": "custom.ns"},
	}))
}

func TestScalarTypeMapping(t *testing.T) {
	require.Equal(t, "int", ScalarType("int32").Render())
	require.Equal(t, "float", ScalarType("double").Render())
	require.Equal(t, "String", ScalarType("string").Render())
	require.Equal(t, "PackedByteArray", ScalarType("bytes").Render())
}
```

Create `internal/fsgenerator/public_api_test.go`:

```go
package fsgenerator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRejectsPublicVariantSignatures(t *testing.T) {
	source := "func get_value() -> Variant:\n\treturn value\n"
	err := CheckPublicAPI(source)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public Variant")
}

func TestAllowsPrivateVariantSignatures(t *testing.T) {
	source := "func _decode_dynamic(value: Variant) -> int:\n\treturn 0\n"
	require.NoError(t, CheckPublicAPI(source))
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test ./internal/fsgenerator -count=1
```

Expected:

```text
FAIL
undefined: NamespaceFor
```

- [ ] **Step 3: Implement generator foundation**

Create `internal/fsgenerator/doc.go`:

```go
// Package fsgenerator generates Foundry Script protobuf bindings.
package fsgenerator
```

Create `internal/fsgenerator/names.go`:

```go
package fsgenerator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

const namespaceOptionKey = "(foundrytools.namespace)"
const typePrefixOptionKey = "(foundrytools.type_prefix)"

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// NamespaceFor returns the Foundry Script namespace for a proto file.
func NamespaceFor(file *protoast.ProtoFile) string {
	if file != nil && file.Options != nil {
		if raw, ok := file.Options[namespaceOptionKey]; ok {
			if value, isString := raw.(string); isString && value != "" {
				return value
			}
		}
	}
	if file == nil {
		return ""
	}
	return file.Package
}

// ValidateNamespace validates a dotted Foundry Script namespace.
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}
	for _, part := range strings.Split(namespace, ".") {
		if !identifierPattern.MatchString(part) {
			return fmt.Errorf("invalid namespace segment %q in %q", part, namespace)
		}
	}
	return nil
}

// TypeName converts a proto identifier to a Foundry Script type identifier.
func TypeName(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	nextUpper := true
	for _, r := range name {
		if r == '_' || r == '-' || r == '.' {
			nextUpper = true
			continue
		}
		if nextUpper {
			b.WriteRune(unicode.ToUpper(r))
			nextUpper = false
			continue
		}
		b.WriteRune(r)
	}
	return escapeIdentifier(b.String())
}

func escapeIdentifier(name string) string {
	switch name {
	case "class", "class_name", "enum", "enum_name", "extends", "func", "import", "namespace", "trait", "trait_name", "uses", "var":
		return name + "_"
	default:
		return name
	}
}
```

Create `internal/fsgenerator/types.go`:

```go
package fsgenerator

import "github.com/cafecito-games/foundry-tools/internal/fstypes"

// ScalarType maps a protobuf scalar name to a Foundry Script type.
func ScalarType(protoType string) fstypes.Type {
	switch protoType {
	case "double", "float":
		return fstypes.Named("float")
	case "int32", "int64", "uint32", "uint64", "sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64":
		return fstypes.Named("int")
	case "bool":
		return fstypes.Named("bool")
	case "string":
		return fstypes.Named("String")
	case "bytes":
		return fstypes.Named("PackedByteArray")
	default:
		return fstypes.Named(TypeName(protoType))
	}
}
```

Create `internal/fsgenerator/public_api.go`:

```go
package fsgenerator

import (
	"bufio"
	"fmt"
	"strings"
)

// CheckPublicAPI rejects public generated function signatures that expose Variant.
func CheckPublicAPI(source string) error {
	scanner := bufio.NewScanner(strings.NewReader(source))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "func _") || strings.HasPrefix(line, "static func _") {
			continue
		}
		if (strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "static func ")) && strings.Contains(line, "Variant") {
			return fmt.Errorf("public Variant in generated signature at line %d: %s", lineNumber, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/fsgenerator -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsgenerator
```

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/fsgenerator
git commit -m "feat: add foundry generator foundations"
```

---

### Task 9: Generate Enums and Message Skeletons

**Files:**
- Create: `internal/fsgenerator/files.go`
- Create: `internal/fsgenerator/generator.go`
- Modify: `internal/fsgenerator/generator_test.go`
- Create: `examples/example.proto`
- Create: `examples/golden/cafecito/game/v1/Player.pb.fs`
- Create: `examples/golden/cafecito/game/v1/PlayerStatus.pb.fs`

- [ ] **Step 1: Add failing golden test**

Append to `internal/fsgenerator/generator_test.go`:

```go
func TestGenerateMessageAndEnumSkeletons(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{{
				FieldType: "string",
				Name:      "name",
				Number:    1,
			}},
		}},
		Enums: []*protoast.Enum{{
			Name: "PlayerStatus",
			Values: []*protoast.EnumValue{
				{Name: "PLAYER_STATUS_UNSPECIFIED", Number: 0},
				{Name: "PLAYER_STATUS_ONLINE", Number: 1},
			},
		}},
	}

	files, err := Generate(file, "examples/example.proto", nil)
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "final class_name Player extends RefCounted uses foundry.proto.Message[Player]")
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var _name: String = \"\"")
	require.Contains(t, files["cafecito/game/v1/PlayerStatus.pb.fs"], "enum_name PlayerStatus")
	require.NoError(t, CheckPublicAPI(files["cafecito/game/v1/Player.pb.fs"]))
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/fsgenerator -run TestGenerateMessageAndEnumSkeletons -count=1
```

Expected:

```text
FAIL
undefined: Generate
```

- [ ] **Step 3: Implement file generation**

Create `internal/fsgenerator/files.go`:

```go
package fsgenerator

import (
	"path"
	"strings"
)

// GeneratedFiles maps output filenames to source content.
type GeneratedFiles map[string]string

func namespacePath(namespace string) string {
	if namespace == "" {
		return ""
	}
	return strings.ReplaceAll(namespace, ".", "/")
}

func outputPath(namespace, typeName string) string {
	if namespace == "" {
		return typeName + ".pb.fs"
	}
	return path.Join(namespacePath(namespace), typeName+".pb.fs")
}
```

Create `internal/fsgenerator/generator.go`:

```go
package fsgenerator

import (
	"fmt"

	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

// FileEntry pairs a proto AST with its source filename for cross-file resolution.
type FileEntry struct {
	File     *protoast.ProtoFile
	Filename string
}

// Generate creates Foundry Script source files for one proto file.
func Generate(file *protoast.ProtoFile, sourceName string, _ []FileEntry) (GeneratedFiles, error) {
	namespace := NamespaceFor(file)
	if err := ValidateNamespace(namespace); err != nil {
		return nil, err
	}

	out := GeneratedFiles{}
	for _, enum := range file.Enums {
		name := TypeName(enum.Name)
		source := renderEnumFile(namespace, name, enum)
		out[outputPath(namespace, name)] = source
	}
	for _, message := range file.Messages {
		name := TypeName(message.Name)
		source, err := renderMessageFile(namespace, name, message)
		if err != nil {
			return nil, fmt.Errorf("generate message %s: %w", message.Name, err)
		}
		out[outputPath(namespace, name)] = source
	}
	return out, nil
}

func renderEnumFile(namespace, name string, enum *protoast.Enum) string {
	lines := "enum_name " + name + " {\n"
	for _, value := range enum.Values {
		lines += "\t" + value.Name + " = " + fmt.Sprintf("%d", value.Number) + ",\n"
	}
	lines += "}\n"
	return fsast.File{
		Namespace:    namespace,
		Declarations: []fsast.Node{fsast.Raw{Code: lines}},
	}.Render()
}

func renderMessageFile(namespace, name string, message *protoast.Message) (string, error) {
	class := fsast.Class{
		Final:   true,
		Name:    name,
		Extends: "RefCounted",
		Uses:    []string{"foundry.proto.Message[" + name + "]"},
	}
	for _, field := range message.Fields {
		class.Members = append(class.Members, fsast.Var{
			Name:  "_" + field.Name,
			Type:  fieldType(field),
			Value: fieldDefault(field),
		})
	}
	class.Members = append(class.Members,
		fsast.Func{
			Name:       "to_bytes",
			ReturnType: fstypes.Named("PackedByteArray"),
			Body: []fsast.Node{
				fsast.Raw{Code: "\tvar result: PackedByteArray = PackedByteArray()"},
				fsast.Return{Value: "result"},
			},
		},
		fsast.Func{
			Name:       "merge_from_bytes",
			Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
			ReturnType: fstypes.Named("foundry.proto.ProtobufError"),
			Body: []fsast.Node{
				fsast.Raw{Code: "\tvar _unused_size: int = data.size()"},
				fsast.Return{Value: "foundry.proto.ProtobufError.OK"},
			},
		},
	)
	source := fsast.File{
		Namespace:    namespace,
		Imports:      []string{"foundry.proto"},
		Declarations: []fsast.Node{class},
	}.Render()
	return source, CheckPublicAPI(source)
}

func fieldType(field *protoast.Field) fstypes.Type {
	return ScalarType(field.FieldType)
}

func fieldDefault(field *protoast.Field) string {
	switch field.FieldType {
	case "string":
		return `""`
	case "bytes":
		return "PackedByteArray()"
	case "bool":
		return "false"
	case "double", "float":
		return "0.0"
	default:
		return "0"
	}
}
```

Modify `internal/fsast/statements.go` to add `Raw`:

```go
// Raw renders preformatted Foundry Script code.
type Raw struct {
	Code string
}

// RenderAt renders raw code.
func (r Raw) RenderAt(level int) string {
	return r.Code
}
```

- [ ] **Step 4: Run generator tests**

Run:

```bash
go test ./internal/fsast ./internal/fsgenerator -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsast
ok  	github.com/cafecito-games/foundry-tools/internal/fsgenerator
```

- [ ] **Step 5: Add example fixture and goldens**

Create `examples/example.proto`:

```protobuf
syntax = "proto3";

package cafecito.game.v1;

message Player {
  string name = 1;
}

enum PlayerStatus {
  PLAYER_STATUS_UNSPECIFIED = 0;
  PLAYER_STATUS_ONLINE = 1;
}
```

Generate actual golden files by running the direct generator after Task 12. For this task, create the expected files from the current unit output:

`examples/golden/cafecito/game/v1/Player.pb.fs`:

```gdscript
namespace cafecito.game.v1
import foundry.proto

final class_name Player extends RefCounted uses foundry.proto.Message[Player]

var _name: String = ""

func to_bytes() -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	return result

func merge_from_bytes(data: PackedByteArray) -> foundry.proto.ProtobufError:
	var _unused_size: int = data.size()
	return foundry.proto.ProtobufError.OK
```

`examples/golden/cafecito/game/v1/PlayerStatus.pb.fs`:

```gdscript
namespace cafecito.game.v1

enum_name PlayerStatus {
	PLAYER_STATUS_UNSPECIFIED = 0,
	PLAYER_STATUS_ONLINE = 1,
}
```

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/fsast internal/fsgenerator examples
git commit -m "feat: generate foundry script message skeletons"
```

---

### Task 10: Field Accessors and Public Decode Shape

**Files:**
- Create: `internal/fsgenerator/fields.go`
- Modify: `internal/fsgenerator/generator.go`
- Modify: `internal/fsgenerator/generator_test.go`

- [ ] **Step 1: Write failing accessor test**

Append to `internal/fsgenerator/generator_test.go`:

```go
func TestGenerateTypedAccessorsAndDecodeFactory(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "name", Number: 1},
				{FieldType: "int32", Name: "level", Number: 2},
			},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "func set_name(value: String) -> void:")
	require.Contains(t, source, "func get_name() -> String:")
	require.Contains(t, source, "func set_level(value: int) -> void:")
	require.Contains(t, source, "static func from_bytes(data: PackedByteArray) -> foundry.proto.DecodeResult[Player]:")
	require.NotContains(t, source, "Variant")
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/fsgenerator -run TestGenerateTypedAccessorsAndDecodeFactory -count=1
```

Expected:

```text
FAIL
does not contain "func set_name"
```

- [ ] **Step 3: Implement field accessor generation**

Create `internal/fsgenerator/fields.go`:

```go
package fsgenerator

import (
	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func fieldMembers(field *protoast.Field) []fsast.Node {
	typ := fieldType(field)
	name := field.Name
	backing := "_" + name
	return []fsast.Node{
		fsast.Func{
			Name:       "set_" + name,
			Parameters: []fsast.Parameter{{Name: "value", Type: typ}},
			ReturnVoid: true,
			Body:       []fsast.Node{fsast.Assign{Target: backing, Value: "value"}},
		},
		fsast.Func{
			Name:       "get_" + name,
			ReturnType: typ,
			Body:       []fsast.Node{fsast.Return{Value: backing}},
		},
	}
}

func fromBytesFactory(className string) fsast.Func {
	return fsast.Func{
		Static:     true,
		Name:       "from_bytes",
		Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Generic("foundry.proto.DecodeResult", fstypes.Named(className)),
		Body: []fsast.Node{
			fsast.Raw{Code: "\tvar message: " + className + " = " + className + ".new()"},
			fsast.Raw{Code: "\tvar err: foundry.proto.ProtobufError = message.merge_from_bytes(data)"},
			fsast.Return{Value: "foundry.proto.DecodeResult[" + className + "].from(message, err)"},
		},
	}
}
```

Modify `renderMessageFile` in `internal/fsgenerator/generator.go`:

```go
	for _, field := range message.Fields {
		class.Members = append(class.Members, fsast.Var{
			Name:  "_" + field.Name,
			Type:  fieldType(field),
			Value: fieldDefault(field),
		})
		class.Members = append(class.Members, fieldMembers(field)...)
	}
	class.Members = append(class.Members, fromBytesFactory(name))
```

Ensure `fromBytesFactory` is appended before `to_bytes`.

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/fsgenerator -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsgenerator
```

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/fsgenerator
git commit -m "feat: generate typed field accessors"
```

---

### Task 11: Scalar Serialization and Deserialization

**Files:**
- Create: `internal/fsgenerator/serialize.go`
- Create: `internal/fsgenerator/deserialize.go`
- Modify: `internal/fsgenerator/generator.go`
- Modify: `internal/fsgenerator/generator_test.go`

- [ ] **Step 1: Write failing scalar serialization test**

Append to `internal/fsgenerator/generator_test.go`:

```go
func TestGenerateScalarSerialization(t *testing.T) {
	file := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.game.v1",
		Messages: []*protoast.Message{{
			Name: "Player",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "name", Number: 1},
				{FieldType: "int32", Name: "level", Number: 2},
			},
		}},
	}

	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)
	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "foundry.proto.Wire.encode_varint(10)")
	require.Contains(t, source, "foundry.proto.Wire.encode_string(_name)")
	require.Contains(t, source, "foundry.proto.Wire.encode_varint(16)")
	require.Contains(t, source, "match field_number:")
	require.Contains(t, source, "_name = string_read.value")
	require.Contains(t, source, "_level = value_read.value")
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/fsgenerator -run TestGenerateScalarSerialization -count=1
```

Expected:

```text
FAIL
does not contain "foundry.proto.Wire.encode_varint(10)"
```

- [ ] **Step 3: Implement scalar serialization generation**

Create `internal/fsgenerator/serialize.go`:

```go
package fsgenerator

import (
	"fmt"

	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func toBytesFunction(fields []*protoast.Field) fsast.Func {
	body := []fsast.Node{fsast.Raw{Code: "\tvar result: PackedByteArray = PackedByteArray()"}}
	for _, field := range fields {
		body = append(body, serializeField(field)...)
	}
	body = append(body, fsast.Return{Value: "result"})
	return fsast.Func{
		Name:       "to_bytes",
		ReturnType: fstypes.Named("PackedByteArray"),
		Body:       body,
	}
}

func serializeField(field *protoast.Field) []fsast.Node {
	tag := field.Number << 3
	wireType := 0
	if field.FieldType == "string" || field.FieldType == "bytes" {
		wireType = 2
	}
	tag |= wireType
	backing := "_" + field.Name
	switch field.FieldType {
	case "string":
		return []fsast.Node{
			fsast.Raw{Code: fmt.Sprintf("\tif %s != \"\":", backing)},
			fsast.Raw{Code: fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)},
			fsast.Raw{Code: fmt.Sprintf("\t\tvar %s_data: PackedByteArray = foundry.proto.Wire.encode_string(%s)", field.Name, backing)},
			fsast.Raw{Code: fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s_data.size()))", field.Name)},
			fsast.Raw{Code: fmt.Sprintf("\t\tresult.append_array(%s_data)", field.Name)},
		}
	default:
		return []fsast.Node{
			fsast.Raw{Code: fmt.Sprintf("\tif %s != 0:", backing)},
			fsast.Raw{Code: fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%d))", tag)},
			fsast.Raw{Code: fmt.Sprintf("\t\tresult.append_array(foundry.proto.Wire.encode_varint(%s))", backing)},
		}
	}
}
```

- [ ] **Step 4: Implement scalar deserialization generation**

Create `internal/fsgenerator/deserialize.go`:

```go
package fsgenerator

import (
	"fmt"

	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func mergeFromBytesFunction(fields []*protoast.Field) fsast.Func {
	body := []fsast.Node{
		fsast.Raw{Code: "\tvar offset: int = 0"},
		fsast.Raw{Code: "\twhile offset < data.size():"},
		fsast.Raw{Code: "\t\tvar tag_read: foundry.proto.FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"},
		fsast.Raw{Code: "\t\tif tag_read.error != foundry.proto.ProtobufError.OK:"},
		fsast.Raw{Code: "\t\t\treturn tag_read.error"},
		fsast.Raw{Code: "\t\toffset = tag_read.offset"},
		fsast.Raw{Code: "\t\tvar field_number: int = foundry.proto.Wire.get_field_number(tag_read.value)"},
		fsast.Raw{Code: "\t\tmatch field_number:"},
	}
	for _, field := range fields {
		body = append(body, deserializeField(field)...)
	}
	body = append(body,
		fsast.Raw{Code: "\t\t\t_:"},
		fsast.Raw{Code: "\t\t\t\treturn foundry.proto.ProtobufError.UNKNOWN_REQUIRED_FEATURE"},
		fsast.Return{Value: "foundry.proto.ProtobufError.OK"},
	)
	return fsast.Func{
		Name:       "merge_from_bytes",
		Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Named("foundry.proto.ProtobufError"),
		Body:       body,
	}
}

func deserializeField(field *protoast.Field) []fsast.Node {
	switch field.FieldType {
	case "string":
		return []fsast.Node{
			fsast.Raw{Code: fmt.Sprintf("\t\t\t%d:", field.Number)},
			fsast.Raw{Code: "\t\t\t\tvar length_read: foundry.proto.FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"},
			fsast.Raw{Code: "\t\t\t\tif length_read.error != foundry.proto.ProtobufError.OK:"},
			fsast.Raw{Code: "\t\t\t\t\treturn length_read.error"},
			fsast.Raw{Code: "\t\t\t\toffset = length_read.offset"},
			fsast.Raw{Code: "\t\t\t\tvar string_read: foundry.proto.FieldRead[String] = foundry.proto.Wire.decode_string(data, offset, length_read.value)"},
			fsast.Raw{Code: "\t\t\t\tif string_read.error != foundry.proto.ProtobufError.OK:"},
			fsast.Raw{Code: "\t\t\t\t\treturn string_read.error"},
			fsast.Raw{Code: fmt.Sprintf("\t\t\t\t_%s = string_read.value", field.Name)},
			fsast.Raw{Code: "\t\t\t\toffset = string_read.offset"},
		}
	default:
		return []fsast.Node{
			fsast.Raw{Code: fmt.Sprintf("\t\t\t%d:", field.Number)},
			fsast.Raw{Code: "\t\t\t\tvar value_read: foundry.proto.FieldRead[int] = foundry.proto.Wire.decode_varint(data, offset)"},
			fsast.Raw{Code: "\t\t\t\tif value_read.error != foundry.proto.ProtobufError.OK:"},
			fsast.Raw{Code: "\t\t\t\t\treturn value_read.error"},
			fsast.Raw{Code: fmt.Sprintf("\t\t\t\t_%s = value_read.value", field.Name)},
			fsast.Raw{Code: "\t\t\t\toffset = value_read.offset"},
		}
	}
}
```

Modify `renderMessageFile` in `internal/fsgenerator/generator.go` so it appends:

```go
class.Members = append(class.Members, fromBytesFactory(name))
class.Members = append(class.Members, toBytesFunction(message.Fields))
class.Members = append(class.Members, mergeFromBytesFunction(message.Fields))
```

and remove the earlier stub `to_bytes` and `merge_from_bytes` functions.

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/fsgenerator -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsgenerator
```

- [ ] **Step 6: Commit**

Run:

```bash
git add internal/fsgenerator
git commit -m "feat: generate scalar protobuf wire code"
```

---

### Task 12: Direct CLI Generation

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/root_test.go`
- Modify: `cmd/foundry-tools/main.go`

- [ ] **Step 1: Write failing direct CLI test**

Append to `internal/cli/root_test.go`:

```go
func TestProtoGenerateRequiresInputs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"proto", "generate", "-o", t.TempDir()})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one .proto file is required")
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/cli -run TestProtoGenerateRequiresInputs -count=1
```

Expected:

```text
FAIL
unknown command "generate"
```

- [ ] **Step 3: Add generate command shell**

Modify `internal/cli/root.go` by adding a `protoGenerateOptions` type and command:

```go
type protoGenerateOptions struct {
	outDir      string
	importPath []string
}

func newProtoGenerateCommand(stdout io.Writer) *cobra.Command {
	opts := &protoGenerateOptions{}
	cmd := &cobra.Command{
		Use:   "generate [flags] <proto files...>",
		Short: "Generate Foundry Script from Protocol Buffers",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("at least one .proto file is required")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			_ = stdout
			_ = opts
			_ = args
			return fmt.Errorf("direct generation is not wired")
		},
	}
	cmd.Flags().StringVarP(&opts.outDir, "out", "o", ".", "output directory")
	cmd.Flags().StringArrayVarP(&opts.importPath, "proto_path", "I", nil, "protobuf import root")
	return cmd
}
```

Add it inside `newProtoCommand`:

```go
cmd.AddCommand(newProtoGenerateCommand(stdout))
```

- [ ] **Step 4: Run focused CLI tests**

Run:

```bash
go test ./internal/cli -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/cli
```

- [ ] **Step 5: Wire parser/importer/generator**

Replace the `RunE` body in `newProtoGenerateCommand` with:

```go
		RunE: func(_ *cobra.Command, args []string) error {
			files, err := protoparse.ParseFiles(args, opts.importPath)
			if err != nil {
				return err
			}
			for _, parsed := range files {
				if errs := protovalidate.Validate(parsed.File, parsed.Filename); len(errs) > 0 {
					return errs[0]
				}
				generated, genErr := fsgenerator.Generate(parsed.File, parsed.Filename, nil)
				if genErr != nil {
					return genErr
				}
				for name, content := range generated {
					target := filepath.Join(opts.outDir, name)
					if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
						return err
					}
					if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
						return err
					}
				}
			}
			for name, content := range runtime.Files() {
				target := filepath.Join(opts.outDir, name)
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(stdout, "generated Foundry Script for %d proto file(s)\n", len(files))
			return err
		},
```

Add imports:

```go
	"os"
	"path/filepath"

	"github.com/cafecito-games/foundry-tools/internal/fsgenerator"
	"github.com/cafecito-games/foundry-tools/internal/protoparse"
	"github.com/cafecito-games/foundry-tools/internal/protovalidate"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
```

Create `internal/protoparse/files.go` with this stable direct-CLI parsing seam:

```go
package protoparse

import (
	"os"
	"path/filepath"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

// ParsedFile is a parsed protobuf file and the filename used for diagnostics.
type ParsedFile struct {
	Filename string
	File     *protoast.ProtoFile
}

// ParseFiles parses root proto files using importRoots.
func ParseFiles(filenames []string, importRoots []string) ([]ParsedFile, error) {
	out := make([]ParsedFile, 0, len(filenames))
	for _, filename := range filenames {
		data, err := os.ReadFile(filename) //nolint:gosec // CLI input path is explicitly user-provided.
		if err != nil {
			return nil, err
		}
		tokens, err := Tokenize(string(data), filename)
		if err != nil {
			return nil, err
		}
		file, err := Parse(tokens, filename)
		if err != nil {
			return nil, err
		}
		importFS := &OSFS{
			BaseDir:      filepath.Dir(filename),
			IncludePaths: importRoots,
		}
		if _, err := ResolveExternalWithFiles(file, filename, importFS); err != nil {
			return nil, err
		}
		out = append(out, ParsedFile{Filename: filename, File: file})
	}
	return out, nil
}
```

The direct CLI generates only the root files passed on the command line. Imported files are parsed for type resolution through `ResolveExternalWithFiles`; they are not appended to `out`.

Add `internal/protoparse/files_test.go`:

```go
package protoparse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFilesParsesRootProto(t *testing.T) {
	dir := t.TempDir()
	protoPath := filepath.Join(dir, "player.proto")
	require.NoError(t, os.WriteFile(protoPath, []byte(`syntax = "proto3";
package cafecito.game.v1;
message Player {
  string name = 1;
}
`), 0o644))

	files, err := ParseFiles([]string{protoPath}, []string{dir})
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, protoPath, files[0].Filename)
	require.Equal(t, "cafecito.game.v1", files[0].File.Package)
	require.Len(t, files[0].File.Messages, 1)
}
```

Run:

```bash
go test ./internal/protoparse -run TestParseFilesParsesRootProto -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/protoparse
```

If the copied package already contains a conflicting `ParsedFile` or `ParseFiles` name, delete the conflicting copied helper and keep the exact `files.go` API above.

- [ ] **Step 6: Run direct generation smoke**

Run:

```bash
go run ./cmd/foundry-tools proto generate -I . -o /tmp/foundry-tools-out examples/example.proto
find /tmp/foundry-tools-out -type f | sort
```

Expected output includes:

```text
/tmp/foundry-tools-out/cafecito/game/v1/Player.pb.fs
/tmp/foundry-tools-out/cafecito/game/v1/PlayerStatus.pb.fs
/tmp/foundry-tools-out/foundry/proto/message.fs
/tmp/foundry-tools-out/foundry/proto/wire.fs
```

- [ ] **Step 7: Commit**

Run:

```bash
git add internal/cli internal/protoparse
git commit -m "feat: wire direct protobuf generation cli"
```

---

### Task 13: protoc Plugin Protocol

**Files:**
- Create: `internal/plugin/doc.go`
- Create: `internal/plugin/plugin.go`
- Create: `internal/plugin/plugin_test.go`
- Modify: `cmd/protoc-gen-foundryscript/main.go`

- [ ] **Step 1: Write failing plugin test**

Create `internal/plugin/plugin_test.go`:

```go
package plugin

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/stretchr/testify/require"
)

func TestRunGeneratesRequestedFile(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"player.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("player.proto"),
			Syntax:  proto.String("proto3"),
			Package: proto.String("cafecito.game.v1"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("Player"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   proto.String("name"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}},
			}},
		}},
	}
	data, err := proto.Marshal(req)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, Run(bytes.NewReader(data), &out))

	resp := &pluginpb.CodeGeneratorResponse{}
	require.NoError(t, proto.Unmarshal(out.Bytes(), resp))
	require.Empty(t, resp.GetError())
	require.NotEmpty(t, resp.GetFile())
	require.Equal(t, "cafecito/game/v1/Player.pb.fs", resp.GetFile()[0].GetName())
}
```

- [ ] **Step 2: Run test and verify it fails**

Run:

```bash
go test ./internal/plugin -run TestRunGeneratesRequestedFile -count=1
```

Expected:

```text
FAIL
undefined: Run
```

- [ ] **Step 3: Implement plugin protocol**

Create `internal/plugin/doc.go`:

```go
// Package plugin implements the protoc CodeGeneratorRequest protocol.
package plugin
```

Create `internal/plugin/plugin.go`:

```go
package plugin

import (
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/foundry-tools/internal/fsgenerator"
	"github.com/cafecito-games/foundry-tools/internal/protodesc"
	"github.com/cafecito-games/foundry-tools/internal/protovalidate"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

// Run executes one protoc plugin request.
func Run(in io.Reader, out io.Writer) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		return writeError(out, fmt.Sprintf("unmarshal request: %v", err))
	}

	response := &pluginpb.CodeGeneratorResponse{}
	features := uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
	response.SupportedFeatures = &features

	files, err := protodesc.FromCodeGeneratorRequest(req)
	if err != nil {
		return writeError(out, err.Error())
	}
	indexByName := map[string]int{}
	for i, descriptor := range req.GetProtoFile() {
		indexByName[descriptor.GetName()] = i
	}

	for _, name := range req.GetFileToGenerate() {
		index, ok := indexByName[name]
		if !ok {
			continue
		}
		file := files[index]
		if errs := protovalidate.Validate(file, name); len(errs) > 0 {
			return writeError(out, errs[0].Error())
		}
		generated, genErr := fsgenerator.Generate(file, name, nil)
		if genErr != nil {
			return writeError(out, genErr.Error())
		}
		for filename, content := range generated {
			response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
				Name:    proto.String(filename),
				Content: proto.String(content),
			})
		}
	}
	if len(response.File) > 0 {
		for filename, content := range runtime.Files() {
			response.File = append(response.File, &pluginpb.CodeGeneratorResponse_File{
				Name:    proto.String(filename),
				Content: proto.String(content),
			})
		}
	}
	return writeResponse(out, response)
}

func writeError(out io.Writer, message string) error {
	return writeResponse(out, &pluginpb.CodeGeneratorResponse{Error: proto.String(message)})
}

func writeResponse(out io.Writer, response *pluginpb.CodeGeneratorResponse) error {
	data, err := proto.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	_, err = out.Write(data)
	return err
}
```

Modify `cmd/protoc-gen-foundryscript/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
	"github.com/cafecito-games/foundry-tools/internal/plugin"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--print-options-proto" {
			if _, err := os.Stdout.Write(foundrytoolspb.Bytes()); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}
	if err := plugin.Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Run plugin tests**

Run:

```bash
go test ./internal/plugin -count=1
go run ./cmd/protoc-gen-foundryscript --print-options-proto | sed -n '1,20p'
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/plugin
syntax = "proto2";
```

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/plugin cmd/protoc-gen-foundryscript/main.go
git commit -m "feat: add protoc plugin protocol"
```

---

### Task 14: Integration Tests for Direct CLI, protoc, and Buf

**Files:**
- Create: `tests/integration/helpers_test.go`
- Create: `tests/integration/direct_cli_test.go`
- Create: `tests/integration/protoc_plugin_test.go`
- Create: `tests/integration/buf_test.go`
- Create: `tests/integration/fixtures/basic/player.proto`
- Create: `tests/integration/fixtures/basic/buf.yaml`
- Create: `tests/integration/fixtures/basic/buf.gen.yaml`

- [ ] **Step 1: Create integration fixture**

Create `tests/integration/fixtures/basic/player.proto`:

```protobuf
syntax = "proto3";

package cafecito.game.v1;

message Player {
  string name = 1;
  int32 level = 2;
}
```

Create `tests/integration/fixtures/basic/buf.yaml`:

```yaml
version: v2
modules:
  - path: .
```

Create `tests/integration/fixtures/basic/buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: ../../../../bin/protoc-gen-foundryscript
    out: out
```

- [ ] **Step 2: Write integration test helpers**

Create `tests/integration/helpers_test.go`:

```go
//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "repo root not found")
		dir = parent
	}
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	return string(out)
}
```

Create `tests/integration/direct_cli_test.go`:

```go
//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectCLIGeneratesFoundryScript(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "run", "./cmd/foundry-tools", "proto", "generate", "-I", "tests/integration/fixtures/basic", "-o", outDir, "tests/integration/fixtures/basic/player.proto")

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito/game/v1/Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "namespace cafecito.game.v1")
	require.Contains(t, string(data), "class_name Player")
}
```

Create `tests/integration/protoc_plugin_test.go`:

```go
//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocPluginGeneratesFoundryScript(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()

	run(t, root, "go", "build", "-o", "bin/protoc-gen-foundryscript", "./cmd/protoc-gen-foundryscript")
	run(t, root, "protoc",
		"--plugin=protoc-gen-foundryscript="+filepath.Join(root, "bin/protoc-gen-foundryscript"),
		"--foundryscript_out="+outDir,
		"-I", "tests/integration/fixtures/basic",
		"tests/integration/fixtures/basic/player.proto",
	)

	data, err := os.ReadFile(filepath.Join(outDir, "cafecito/game/v1/Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "func to_bytes() -> PackedByteArray:")
}
```

Create `tests/integration/buf_test.go`:

```go
//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBufGeneratesFoundryScript(t *testing.T) {
	root := repoRoot(t)
	fixture := filepath.Join(root, "tests/integration/fixtures/basic")
	run(t, root, "go", "build", "-o", "bin/protoc-gen-foundryscript", "./cmd/protoc-gen-foundryscript")
	run(t, fixture, "buf", "generate")

	data, err := os.ReadFile(filepath.Join(fixture, "out/cafecito/game/v1/Player.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "class_name Player")
	require.NoError(t, os.RemoveAll(filepath.Join(fixture, "out")))
}
```

- [ ] **Step 3: Run integration tests**

Run:

```bash
task build
go test -tags=integration ./tests/integration/... -v -count=1
```

Expected:

```text
PASS
ok  	github.com/cafecito-games/foundry-tools/tests/integration
```

- [ ] **Step 4: Commit**

Run:

```bash
git add tests/integration
git commit -m "test: add protobuf generation integration tests"
```

---

### Task 15: Foundry Headless Fixture

**Files:**
- Create: `internal/foundrytest/doc.go`
- Create: `internal/foundrytest/foundry.go`
- Create: `internal/foundrytest/foundry_test.go`
- Create: `tests/foundry/project.foundry`
- Create: `tests/foundry/main.fs`
- Create: `tests/foundry/run.sh`

- [ ] **Step 1: Write Foundry binary lookup tests**

Create `internal/foundrytest/foundry_test.go`:

```go
package foundrytest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultBinaryCandidates(t *testing.T) {
	candidates := BinaryCandidates("")
	require.Contains(t, candidates[0], ".foundry/bin/foundry.macos.editor.dev.arm64")
}

func TestEnvBinaryCandidateWins(t *testing.T) {
	candidates := BinaryCandidates("/tmp/foundry")
	require.Equal(t, "/tmp/foundry", candidates[0])
}
```

- [ ] **Step 2: Implement Foundry helper**

Create `internal/foundrytest/doc.go`:

```go
// Package foundrytest contains helpers for locating Foundry test binaries.
package foundrytest
```

Create `internal/foundrytest/foundry.go`:

```go
package foundrytest

import (
	"os"
	"path/filepath"
)

// BinaryCandidates returns possible Foundry editor binary paths in priority order.
func BinaryCandidates(envValue string) []string {
	var out []string
	if envValue != "" {
		out = append(out, envValue)
	}
	home, err := os.UserHomeDir()
	if err == nil {
		out = append(out, filepath.Join(home, ".foundry/bin/foundry.macos.editor.dev.arm64"))
	}
	out = append(out, filepath.Join(".cache", "foundry", "foundry"))
	return out
}
```

- [ ] **Step 3: Create Foundry fixture**

Create `tests/foundry/project.foundry`:

```ini
; Engine configuration file.

config_version=5

[application]

config/name="Foundry Tools Generated Protobuf Fixture"
run/main_scene=""
```

Create `tests/foundry/main.fs`:

```gdscript
extends SceneTree

func _init() -> void:
	var player: cafecito.game.v1.Player = cafecito.game.v1.Player.new()
	player.set_name("Ava")
	player.set_level(7)
	var data: PackedByteArray = player.to_bytes()
	var decoded: foundry.proto.DecodeResult[cafecito.game.v1.Player] = cafecito.game.v1.Player.from_bytes(data)
	if not decoded.is_ok():
		printerr("decode failed")
		quit(1)
		return
	if decoded.value == null:
		printerr("decoded value missing")
		quit(1)
		return
	if decoded.value.get_name() != "Ava":
		printerr("decoded name mismatch")
		quit(1)
		return
	quit(0)
```

Create `tests/foundry/run.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/tests/foundry/generated"
FOUNDRY="${FOUNDRY_BIN:-$HOME/.foundry/bin/foundry.macos.editor.dev.arm64}"

rm -rf "$OUT"
mkdir -p "$OUT"

"$ROOT/bin/foundry-tools" proto generate \
  -I "$ROOT/tests/integration/fixtures/basic" \
  -o "$OUT" \
  "$ROOT/tests/integration/fixtures/basic/player.proto"

if rg -n '(^|[^_])func [A-Za-z0-9_]+\(.*Variant|-> Variant' "$OUT"; then
  echo "public Variant signature found in generated Foundry Script"
  exit 1
fi

"$FOUNDRY" --headless --check-only --script "$ROOT/tests/foundry/main.fs" --path "$ROOT/tests/foundry"
```

Run:

```bash
chmod +x tests/foundry/run.sh
```

- [ ] **Step 4: Run helper tests and Foundry check**

Run:

```bash
go test ./internal/foundrytest -count=1
task build
FOUNDRY_BIN="$HOME/.foundry/bin/foundry.macos.editor.dev.arm64" task foundry:test
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/foundrytest
```

and Foundry exits with status `0`.

- [ ] **Step 5: Commit**

Run:

```bash
git add internal/foundrytest tests/foundry
git commit -m "test: add foundry headless fixture"
```

---

### Task 16: CI Workflows and Foundry Installer

**Files:**
- Create: `scripts/ci/install-foundry.sh`
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/foundry.yml`

- [ ] **Step 1: Create Foundry installer script**

Create `scripts/ci/install-foundry.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

TAG="${FOUNDRY_RELEASE_TAG:-v0.1.0}"
ASSET_PATTERN="${FOUNDRY_ASSET_PATTERN:-linux.x86_64.zip}"
CACHE_DIR="${FOUNDRY_CACHE_DIR:-.cache/foundry}"
REPO="${FOUNDRY_REPO:-cafecito-games/Foundry}"

mkdir -p "$CACHE_DIR"

asset_json="$(gh api "repos/$REPO/releases/tags/$TAG" 2>/dev/null || true)"
if [ -z "$asset_json" ]; then
  echo "Unable to read Foundry release $TAG from $REPO. Set FOUNDRY_RELEASE_TOKEN for draft/private assets."
  exit 1
fi

asset_id="$(printf '%s' "$asset_json" | jq -r --arg pattern "$ASSET_PATTERN" '.assets[] | select(.name | contains($pattern)) | .id' | head -n1)"
asset_name="$(printf '%s' "$asset_json" | jq -r --arg pattern "$ASSET_PATTERN" '.assets[] | select(.name | contains($pattern)) | .name' | head -n1)"

if [ -z "$asset_id" ] || [ "$asset_id" = "null" ]; then
  echo "No Foundry asset matching $ASSET_PATTERN on $TAG"
  exit 1
fi

archive="$CACHE_DIR/$asset_name"
gh api "repos/$REPO/releases/assets/$asset_id" \
  -H "Accept: application/octet-stream" > "$archive"

unzip -o "$archive" -d "$CACHE_DIR"
foundry_bin="$(find "$CACHE_DIR" -type f -perm -111 -name 'foundry*' | head -n1)"
if [ -z "$foundry_bin" ]; then
  echo "No executable Foundry binary found after extracting $asset_name"
  exit 1
fi

echo "FOUNDRY_BIN=$foundry_bin" >> "$GITHUB_ENV"
```

Run:

```bash
chmod +x scripts/ci/install-foundry.sh
```

- [ ] **Step 2: Add CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  ci:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go 1.26
        uses: actions/setup-go@v5
        with:
          go-version: '1.26.x'
          cache: true

      - name: Install Task
        uses: arduino/setup-task@v2
        with:
          version: 3.x
          repo-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Install golangci-lint
        uses: golangci/golangci-lint-action@v7
        with:
          version: v2.11

      - name: Install protoc
        run: |
          PROTOC_VERSION=27.2
          PROTOC_ZIP=protoc-${PROTOC_VERSION}-linux-x86_64.zip
          curl -fsSL -o /tmp/${PROTOC_ZIP} \
            https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/${PROTOC_ZIP}
          sudo unzip -o /tmp/${PROTOC_ZIP} -d /usr/local
          protoc --version

      - name: Install buf
        uses: bufbuild/buf-setup-action@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}

      - name: Run CI
        run: task ci

      - name: Run integration tests
        run: task integration

      - name: Run coverage
        run: task test:cover

      - name: Upload coverage
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out
          if-no-files-found: error
```

- [ ] **Step 3: Add Foundry workflow**

Create `.github/workflows/foundry.yml`:

```yaml
name: Foundry Integration

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  foundry:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Go 1.26
        uses: actions/setup-go@v5
        with:
          go-version: '1.26.x'
          cache: true

      - name: Install Task
        uses: arduino/setup-task@v2
        with:
          version: 3.x
          repo-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Install protoc
        run: sudo apt-get update && sudo apt-get install -y protobuf-compiler jq unzip

      - name: Install buf
        uses: bufbuild/buf-setup-action@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}

      - name: Build tools
        run: task build

      - name: Install Foundry
        env:
          GH_TOKEN: ${{ secrets.FOUNDRY_RELEASE_TOKEN || secrets.GITHUB_TOKEN }}
          FOUNDRY_RELEASE_TAG: v0.1.0
          FOUNDRY_ASSET_PATTERN: linux.x86_64.zip
        run: bash scripts/ci/install-foundry.sh

      - name: Run Foundry tests
        run: task foundry:test
```

- [ ] **Step 4: Validate workflow YAML**

Run:

```bash
pre-commit run check-yaml --files .github/workflows/ci.yml .github/workflows/foundry.yml
```

Expected:

```text
check yaml...............................................................Passed
```

- [ ] **Step 5: Commit**

Run:

```bash
git add scripts/ci/install-foundry.sh .github/workflows/ci.yml .github/workflows/foundry.yml
git commit -m "ci: add go and foundry workflows"
```

---

### Task 17: Release Automation and Homebrew Cask

**Files:**
- Create: `.goreleaser.yaml`
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Add GoReleaser configuration**

Create `.goreleaser.yaml`:

```yaml
version: 2

project_name: foundry-tools

before:
  hooks:
    - go mod tidy

builds:
  - id: foundry-tools
    main: ./cmd/foundry-tools
    binary: foundry-tools
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X github.com/cafecito-games/foundry-tools/internal/cli.Version={{ .Version }}

  - id: protoc-gen-foundryscript
    main: ./cmd/protoc-gen-foundryscript
    binary: protoc-gen-foundryscript
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64

archives:
  - id: foundry-tools
    ids:
      - foundry-tools
      - protoc-gen-foundryscript
    formats: [tar.gz]
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"

homebrew_casks:
  - name: foundry-tools
    binaries:
      - foundry-tools
      - protoc-gen-foundryscript
    repository:
      owner: cafecito-games
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://github.com/cafecito-games/foundry-tools"
    description: "Foundry Engine development tools"
    hooks:
      post:
        install: |
          if system_command("/usr/bin/xattr", args: ["-h"]).exit_status == 0
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/foundry-tools"]
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/protoc-gen-foundryscript"]
          end
```

- [ ] **Step 2: Add release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - "v*"
  workflow_dispatch:
    inputs:
      bump:
        description: "Which semantic version part to bump"
        required: true
        type: choice
        default: patch
        options:
          - patch
          - minor
          - major

permissions:
  contents: write
  pull-requests: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Ensure manual releases run from main
        if: github.event_name == 'workflow_dispatch'
        run: |
          set -euo pipefail
          if [ "${GITHUB_REF_NAME}" != "main" ]; then
            echo "::error::Manual releases must be run from main, not ${GITHUB_REF_NAME}"
            exit 1
          fi

      - name: Compute release version
        id: release-version
        if: github.event_name == 'workflow_dispatch'
        run: |
          set -euo pipefail
          latest=$(git tag --list 'v*' --sort=-v:refname | head -n1)
          if [ -z "$latest" ]; then
            latest="v0.0.0"
          fi
          if ! [[ "$latest" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
            echo "::error::Latest tag '$latest' is not a strict vMAJOR.MINOR.PATCH version"
            exit 1
          fi
          major=${BASH_REMATCH[1]}
          minor=${BASH_REMATCH[2]}
          patch=${BASH_REMATCH[3]}
          case "${{ inputs.bump }}" in
            major) major=$((major + 1)); minor=0; patch=0 ;;
            minor) minor=$((minor + 1)); patch=0 ;;
            patch) patch=$((patch + 1)) ;;
          esac
          new_tag="v${major}.${minor}.${patch}"
          echo "tag=$new_tag" >> "$GITHUB_OUTPUT"

      - name: Tag release from main
        if: github.event_name == 'workflow_dispatch'
        env:
          NEW_TAG: ${{ steps.release-version.outputs.tag }}
        run: |
          set -euo pipefail
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git tag -a "$NEW_TAG" -m "Release $NEW_TAG"
          git push origin "$NEW_TAG"
          git checkout "$NEW_TAG"

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
          cache: true

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

- [ ] **Step 3: Validate release config**

Run:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Expected:

```text
  • configuration is valid
```

and `dist/` contains archives for both binaries.

- [ ] **Step 4: Commit**

Run:

```bash
git add .goreleaser.yaml .github/workflows/release.yml
git commit -m "ci: add release and homebrew automation"
```

---

### Task 18: README and Full Verification

**Files:**
- Modify: `README.md`
- Modify: `examples/golden/cafecito/game/v1/Player.pb.fs`
- Modify: `examples/golden/cafecito/game/v1/PlayerStatus.pb.fs`

- [ ] **Step 1: Refresh example goldens from the direct CLI**

Run:

```bash
rm -rf /tmp/foundry-tools-golden
task build
bin/foundry-tools proto generate -I . -o /tmp/foundry-tools-golden examples/example.proto
cp /tmp/foundry-tools-golden/cafecito/game/v1/Player.pb.fs examples/golden/cafecito/game/v1/Player.pb.fs
cp /tmp/foundry-tools-golden/cafecito/game/v1/PlayerStatus.pb.fs examples/golden/cafecito/game/v1/PlayerStatus.pb.fs
```

- [ ] **Step 2: Replace README with usage docs**

Update `README.md`:

```markdown
# Foundry Tools

Tooling for Foundry Engine projects.

This repository currently ships protobuf code generation for Foundry Script:

- `foundry-tools`: direct CLI and umbrella command for future tools.
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
go install github.com/cafecito-games/foundry-tools/cmd/foundry-tools@latest
go install github.com/cafecito-games/foundry-tools/cmd/protoc-gen-foundryscript@latest
```

## Direct CLI

```bash
foundry-tools proto generate -I proto -o foundry/generated proto/player.proto
```

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
foundry-tools proto print-options-proto > proto/foundrytools/options.proto
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
```

- [ ] **Step 3: Run full verification**

Run:

```bash
task ci
task integration
FOUNDRY_BIN="$HOME/.foundry/bin/foundry.macos.editor.dev.arm64" task foundry:test
goreleaser release --snapshot --clean
```

Expected:

```text
task ci passes
task integration passes
task foundry:test passes
goreleaser snapshot completes
```

- [ ] **Step 4: Inspect public Variant scan**

Run:

```bash
rg -n '(^|[^_])func [A-Za-z0-9_]+\(.*Variant|-> Variant' examples/golden internal/runtime/data || true
```

Expected: no output.

- [ ] **Step 5: Commit**

Run:

```bash
git add README.md examples/golden
git commit -m "docs: add foundry tools usage"
```

---

## Self-Review Checklist

- Spec coverage:
  - Go 1.26 bootstrap: Task 1.
  - CLI and plugin binaries: Tasks 2, 12, 13.
  - Foundry options proto: Task 3.
  - `gogdproto` parser/descriptor/validator reuse: Task 4.
  - Foundry Script namespaces/types/AST/generator: Tasks 5 through 11.
  - Typed runtime: Task 7.
  - No public `Variant`: Tasks 7, 8, 15, 18.
  - Integration tests: Task 14.
  - Foundry tests: Task 15.
  - CI/CD and Homebrew: Tasks 16 and 17.
  - README: Task 18.
- Marker scan: no unresolved marker terms remain.
- Type consistency:
  - `foundrytoolspb` is the generated options package.
  - `protoast` is the package name for `internal/protoast`.
  - `fsgenerator.Generate` returns `GeneratedFiles`.
  - Runtime namespace is `foundry.proto`.
  - Generated decode helper returns `foundry.proto.DecodeResult[T]`.
