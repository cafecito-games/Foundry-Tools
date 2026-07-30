# Foundry Engine Type Collisions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reject protobuf type declarations that collide with Foundry built-ins or exposed native classes, while making `foundrytools.type_prefix` a consistent file-wide escape hatch.

**Architecture:** Generate a versioned Go lookup table from the pinned Foundry release's `extension_api.json`, then use a file-scoped `typeNamer` throughout declaration planning, reference resolution, and output naming. A pre-render collision collector inventories local declarations and referenced imported declarations, returning one deterministic diagnostic through both direct CLI and protoc plugin paths.

**Tech Stack:** Go 1.26, Protocol Buffers descriptors/extensions, Task, Bash, Foundry CLI, testify.

---

## File Structure

New files:

- `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main.go` — parse an extension API dump and emit deterministic Go source.
- `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main_test.go` — refresh-command unit tests.
- `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/testdata/extension_api.json` — minimal API fixture.
- `internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go` — checked-in generated built-in/native lookup tables.
- `internal/proto/internal/foundryscript/generator/names_test.go` — focused prefix/naming tests.
- `internal/proto/internal/foundryscript/generator/collisions.go` — collision metadata, collection, sorting, and diagnostics.
- `internal/proto/internal/foundryscript/generator/collisions_test.go` — collision and diagnostic unit tests.
- `scripts/ci/sync-foundry-engine-types.sh` — refresh/check wrapper around Foundry's API dump.
- `tests/integration/fixtures/collisions/types.proto` — direct-CLI collision fixture.
- `tests/integration/fixtures/collisions/prefixed.proto` — direct-CLI prefix success fixture.
- `tests/foundry/collision_dependency.proto` — prefixed imported dependency.
- `tests/foundry/collisions.proto` — prefixed native/built-in/nested/oneof fixture.

Modified files:

- `Taskfile.yml` — add `gen-engine-types`.
- `internal/proto/internal/foundryscript/generator/names.go` — literal prefix validation and file-scoped naming.
- `internal/proto/internal/foundryscript/generator/generator.go` — plan before render and apply local naming.
- `internal/proto/internal/foundryscript/generator/plan.go` — separate proto lookup scopes from emitted names and apply dependency naming.
- `internal/proto/internal/foundryscript/generator/generator_test.go` — prefixed declarations, references, paths, and imported dependencies.
- `internal/plugin/plugin_test.go` — descriptor-path collision error and prefix success.
- `tests/integration/direct_cli_test.go` — direct CLI error and prefix success.
- `tests/foundry/run.sh` — table drift check and extra schema generation.
- `tests/foundry/main.fs` — execute prefixed bindings.
- `scripts/ci/install-foundry.sh` — pin alpha.14.
- `.github/workflows/foundry.yml` — pin alpha.14.
- `README.md` — document the option contract and errors.
- `proto/foundrytools/options.proto` — document `type_prefix`.
- `internal/foundrytoolspb/options.proto` — regenerated embedded schema.
- `internal/foundrytoolspb/options.pb.go` — regenerated Go descriptor.
- `internal/foundrytoolspb/embed_test.go` — embedded option documentation assertion.

### Task 1: Build the deterministic engine-type table generator

**Files:**

- Create: `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main.go`
- Create: `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main_test.go`
- Create: `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/testdata/extension_api.json`

- [ ] **Step 1: Write the minimal API fixture**

```json
{
  "header": {
    "version_full_name": "Foundry v0.1.alpha14.gh"
  },
  "builtin_classes": [
    {"name": "String"},
    {"name": "Array"}
  ],
  "classes": [
    {"name": "Timer"},
    {"name": "Node"}
  ]
}
```

- [ ] **Step 2: Write failing tests for extraction and deterministic rendering**

```go
func TestLoadReservedTypes(t *testing.T) {
	api, err := loadAPI("testdata/extension_api.json")
	require.NoError(t, err)
	require.Equal(t, []string{"Array", "AsyncCallable", "String"}, api.Builtins)
	require.Equal(t, []string{"Node", "Timer"}, api.NativeClasses)
}

func TestRenderGoIsDeterministicAndRecordsVersion(t *testing.T) {
	source, err := renderGo(reservedTypes{
		Version:       "0.1.alpha14.gh.b9a5e66c2",
		Builtins:      []string{"String", "AsyncCallable"},
		NativeClasses: []string{"Timer", "Node"},
	})
	require.NoError(t, err)
	require.Contains(t, string(source), `foundryEngineTypeSourceVersion = "0.1.alpha14.gh.b9a5e66c2"`)
	require.Less(t, bytes.Index(source, []byte(`"Node"`)), bytes.Index(source, []byte(`"Timer"`)))
	require.Contains(t, string(source), `"AsyncCallable": {kind: engineTypeBuiltin}`)
}

func TestLoadReservedTypesRejectsDuplicatesAndMissingNames(t *testing.T) {
	_, err := decodeAPI(strings.NewReader(`{"builtin_classes":[{"name":"String"},{"name":"String"}]}`))
	require.ErrorContains(t, err, `duplicate built-in type "String"`)
	_, err = decodeAPI(strings.NewReader(`{"classes":[{"name":""}]}`))
	require.ErrorContains(t, err, "native class has an empty name")
}
```

- [ ] **Step 3: Run the command-package tests and verify they fail**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types -count=1
```

Expected: FAIL because `loadAPI`, `decodeAPI`, `renderGo`, and `reservedTypes` do not exist.

- [ ] **Step 4: Implement parsing, validation, rendering, and the CLI**

Use these core types and signatures:

```go
type namedAPIEntry struct {
	Name string `json:"name"`
}

type extensionAPI struct {
	BuiltinClasses []namedAPIEntry `json:"builtin_classes"`
	Classes        []namedAPIEntry `json:"classes"`
}

type reservedTypes struct {
	Version       string
	Builtins      []string
	NativeClasses []string
}

func loadAPI(path string) (reservedTypes, error)
func decodeAPI(reader io.Reader) (reservedTypes, error)
func renderGo(types reservedTypes) ([]byte, error)
```

The CLI accepts `--api`, `--version`, and `--output`, requires every value,
calls `loadAPI`, sets `Version`, renders through `go/format.Source`, and writes
mode `0644`. Emit:

```go
type engineTypeKind uint8

const (
	engineTypeBuiltin engineTypeKind = iota + 1
	engineTypeNativeClass
)

type engineTypeEntry struct {
	kind engineTypeKind
}

const foundryEngineTypeSourceVersion = "..."

var foundryEngineReservedTypes = map[string]engineTypeEntry{
	"Array": {kind: engineTypeBuiltin},
	"Node":  {kind: engineTypeNativeClass},
}
```

Validate duplicates from the API first, add `AsyncCallable` to built-ins only
when the API does not already contain it, sort both categories, and reject a
name that appears in both categories.

- [ ] **Step 5: Run tests and verify they pass**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```text
git add internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types
git commit -m "feat: generate Foundry engine type tables"
```

### Task 2: Add refresh/check automation and generate the alpha.14 table

**Files:**

- Create: `scripts/ci/sync-foundry-engine-types.sh`
- Create: `internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go`
- Modify: `Taskfile.yml`

- [ ] **Step 1: Add a failing Task contract test**

Add to `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main_test.go`:

```go
func TestRepositoryTaskDeclaresEngineTypeRefresh(t *testing.T) {
	root := findRepoRoot(t)
	taskfile, err := os.ReadFile(filepath.Join(root, "Taskfile.yml"))
	require.NoError(t, err)
	require.Contains(t, string(taskfile), "gen-engine-types:")
	require.Contains(t, string(taskfile), "sync-foundry-engine-types.sh write")
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "repository root not found")
		dir = parent
	}
}
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types -run TestRepositoryTaskDeclaresEngineTypeRefresh -count=1
```

Expected: FAIL because `gen-engine-types` is absent.

- [ ] **Step 3: Implement the synchronization script**

The script must:

```bash
#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-check}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FOUNDRY="${FOUNDRY_BIN:-$(command -v foundry || true)}"
TARGET="$ROOT/internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go"

test "$MODE" = "write" -o "$MODE" = "check" || {
  echo "usage: $0 [write|check]" >&2
  exit 2
}
test -n "$FOUNDRY" -a -x "$FOUNDRY" || {
  echo "Foundry binary not found on PATH. Install foundry or set FOUNDRY_BIN." >&2
  exit 1
}

SYNC_DIR="$(mktemp -d)"
trap 'rm -rf "$SYNC_DIR"' EXIT

(
  cd "$SYNC_DIR"
  "$FOUNDRY" --headless --no-header docs generate-api
)
VERSION="$("$FOUNDRY" --version | head -n 1)"
CANDIDATE="$SYNC_DIR/engine_reserved_types.gen.go"

go run "$ROOT/internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types" \
  --api "$SYNC_DIR/extension_api.json" \
  --version "$VERSION" \
  --output "$CANDIDATE"

if [ "$MODE" = "write" ]; then
  cp "$CANDIDATE" "$TARGET"
  exit 0
fi

if ! cmp -s "$CANDIDATE" "$TARGET"; then
  echo "Foundry engine type table is stale for $VERSION." >&2
  echo "Run: task gen-engine-types" >&2
  diff -u "$TARGET" "$CANDIDATE" || true
  exit 1
fi
```

Use a cleanup function with explicit temporary paths if repository policy
rejects the inline trap command.

- [ ] **Step 4: Add the Task target**

```yaml
  gen-engine-types:
    desc: Refresh the checked-in Foundry built-in and native type table.
    cmds:
      - FOUNDRY_BIN="{{.FOUNDRY_BIN}}" bash scripts/ci/sync-foundry-engine-types.sh write
```

- [ ] **Step 5: Generate the real table**

Run:

```text
FOUNDRY_BIN=/Users/christian/bin/foundry task gen-engine-types
```

Expected: `engine_reserved_types.gen.go` is created with version
`0.1.alpha14.gh.b9a5e66c2`, 1,050 native entries, all extension API built-ins,
and `AsyncCallable`.

- [ ] **Step 6: Run focused verification**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types -count=1
FOUNDRY_BIN=/Users/christian/bin/foundry bash scripts/ci/sync-foundry-engine-types.sh check
```

Expected: both commands exit 0.

- [ ] **Step 7: Commit**

```text
git add Taskfile.yml scripts/ci/sync-foundry-engine-types.sh internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main_test.go
git commit -m "build: track Foundry engine type names"
```

### Task 3: Introduce literal file-scoped type naming

**Files:**

- Create: `internal/proto/internal/foundryscript/generator/names_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/names.go`

- [ ] **Step 1: Write failing prefix and naming-order tests**

```go
func TestNewTypeNamerValidatesLiteralPrefix(t *testing.T) {
	file := &protoast.ProtoFile{
		Options: map[string]any{typePrefixOptionKey: "Game_"},
		OptionPositions: map[string]protoast.Position{
			typePrefixOptionKey: {Line: 4, Column: 1},
		},
	}
	namer, err := newTypeNamer(file, "types.proto")
	require.NoError(t, err)
	require.Equal(t, "Game_Node", namer.Name("node"))

	for _, value := range []any{"", "game-tools", "game tools", "game.tools", "2D", int64(3)} {
		file.Options[typePrefixOptionKey] = value
		_, err := newTypeNamer(file, "types.proto")
		require.Error(t, err)
		require.Contains(t, err.Error(), "types.proto:4:1")
		require.Contains(t, err.Error(), "(foundrytools.type_prefix)")
	}
}

func TestTypeNamerPrefixesBeforeEscaping(t *testing.T) {
	namer := typeNamer{prefix: "Game"}
	require.Equal(t, "GameClass", namer.Name("class"))
	require.Equal(t, "GameMessage", namer.Name("message"))
	require.Equal(t, "GameOuter.GameInner", namer.Reference("outer.inner"))
	require.Equal(t, "Class_", TypeName("class"))
	require.Equal(t, "Message_", TypeName("message"))
}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator -run 'Test(NewTypeNamer|TypeNamer)' -count=1
```

Expected: FAIL because `typeNamer` and `newTypeNamer` do not exist.

- [ ] **Step 3: Implement the naming object**

Refactor `TypeName` around an unescaped normalizer:

```go
type typeNamer struct {
	prefix string
}

func newTypeNamer(file *protoast.ProtoFile, sourceName string) (typeNamer, error) {
	if file == nil || file.Options == nil {
		return typeNamer{}, nil
	}
	raw, present := file.Options[typePrefixOptionKey]
	if !present {
		return typeNamer{}, nil
	}
	prefix, ok := raw.(string)
	if !ok || prefix == "" || !identifierPattern.MatchString(prefix) {
		return typeNamer{}, optionError(file, sourceName, typePrefixOptionKey,
			fmt.Sprintf("must be a non-empty identifier fragment, got %q", raw))
	}
	return typeNamer{prefix: prefix}, nil
}

func (n typeNamer) Name(name string) string {
	return escapeIdentifier(n.prefix + normalizeTypeName(name))
}

func (n typeNamer) Reference(protoType string) string {
	parts := strings.Split(strings.TrimPrefix(protoType, "."), ".")
	for i := range parts {
		parts[i] = n.Name(parts[i])
	}
	return strings.Join(parts, ".")
}

func TypeName(name string) string {
	return (typeNamer{}).Name(name)
}
```

`optionError` formats `source:line:column` when `OptionPositions` has a nonzero
position and `source: option ...` otherwise.

- [ ] **Step 4: Run generator tests**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator -count=1
```

Expected: PASS with existing no-prefix tests unchanged.

- [ ] **Step 5: Commit**

```text
git add internal/proto/internal/foundryscript/generator/names.go internal/proto/internal/foundryscript/generator/names_test.go
git commit -m "feat: define Foundry type prefix naming"
```

### Task 4: Thread declaring-file names through planning and resolution

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/generator.go`
- Modify: `internal/proto/internal/foundryscript/generator/plan.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`

- [ ] **Step 1: Write failing declaration/path tests**

Add a helper:

```go
func prefixedFile(prefix string, messages []*protoast.Message, enums []*protoast.Enum) *protoast.ProtoFile {
	file := namespacedFile(messages, enums)
	file.Options = map[string]any{typePrefixOptionKey: prefix}
	return file
}
```

Add assertions:

```go
func TestGeneratePrefixesDeclarationsReferencesAndPaths(t *testing.T) {
	files := generate(t, prefixedFile("Game", []*protoast.Message{{
		Name: "Outer",
		NestedMessages: []*protoast.Message{{Name: "Inner"}},
		Fields: []*protoast.Field{{Name: "inner", Number: 1, FieldType: "Inner"}},
		Oneofs: []*protoast.Oneof{{
			Name: "choice",
			Fields: []*protoast.Field{{Name: "text", Number: 2, FieldType: "string"}},
		}},
	}}, []*protoast.Enum{{Name: "State", Values: []*protoast.EnumValue{{Name: "STATE_UNSPECIFIED", Number: 0}}}}))

	require.Contains(t, files["cafecito/game/v1/GameOuter.pb.fs"], "class_name GameOuter")
	require.Contains(t, files["cafecito/game/v1/GameOuter.pb.fs"], "final class GameInner")
	require.Contains(t, files["cafecito/game/v1/GameOuter.pb.fs"], "var inner: GameInner")
	require.Contains(t, files["cafecito/game/v1/GameOuterChoiceCase.pb.fs"], "enum_name GameOuterChoiceCase")
	require.Contains(t, files["cafecito/game/v1/GameState.pb.fs"], "enum_name GameState")
}

func TestNestedOneofFlattensPrefixedOwnerSegments(t *testing.T) {
	files := generate(t, prefixedFile("Game", []*protoast.Message{{
		Name: "Outer",
		NestedMessages: []*protoast.Message{{
			Name: "Inner",
			Oneofs: []*protoast.Oneof{{
				Name: "choice",
				Fields: []*protoast.Field{{Name: "text", Number: 1, FieldType: "string"}},
			}},
		}},
	}}, nil))
	require.Contains(t,
		files["cafecito/game/v1/GameOuterGameInnerChoiceCase.pb.fs"],
		"enum_name GameOuterGameInnerChoiceCase")
}
```

- [ ] **Step 2: Write a failing imported-prefix test**

```go
func TestReferenceUsesDeclaringFilesPrefix(t *testing.T) {
	imported := prefixedFile("Inventory", []*protoast.Message{{Name: "Item"}}, nil)
	imported.Package = "cafecito.inventory.v1"
	local := namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{{
			Name: "held", Number: 1, FieldType: "Item", SourceFile: "inventory.proto",
		}},
	}}, nil)

	files, err := Generate(local, "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.NoError(t, err)
	require.Contains(t, files["cafecito/game/v1/Player.pb.fs"], "var held: InventoryItem?")
}

func TestReferencedDependencyReportsInvalidPrefix(t *testing.T) {
	imported := prefixedFile("bad-prefix", []*protoast.Message{{Name: "Item"}}, nil)
	imported.Package = "cafecito.inventory.v1"
	local := namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{{
			Name: "held", Number: 1, FieldType: "Item", SourceFile: "inventory.proto",
		}},
	}}, nil)
	_, err := Generate(local, "player.proto", []FileEntry{{File: imported, Filename: "inventory.proto"}})
	require.ErrorContains(t, err, "inventory.proto")
	require.ErrorContains(t, err, "(foundrytools.type_prefix)")
}
```

- [ ] **Step 3: Run focused tests and verify they fail**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator -run 'TestGeneratePrefixes|TestReferenceUsesDeclaring' -count=1
```

Expected: FAIL because generation still ignores `type_prefix`.

- [ ] **Step 4: Separate proto lookup paths from emitted references**

Change `typeInfo` to keep:

```go
type typeInfo struct {
	ProtoReference string
	Reference      string
	IsEnum         bool
	ZeroCase       string
	Namespace      string
	TopLevel       string
	Declaration    declarationInfo
}
```

Keep registry keys based on the existing no-prefix `TypeReference` so protobuf
lexical resolution is unchanged. Populate `Reference` and `TopLevel` with the
declaring file's `typeNamer`.

In `namedValuePlan`, resolve with the no-prefix proto reference but emit with
the declaring namer:

```go
protoReference := TypeReference(use.ProtoType)
emittedReference := r.namerFor(use.SourceFile).Reference(use.ProtoType)
info, found := r.resolve(use, scope, protoReference)
lexical, qualified := emittedReference, info.Reference
```

Inside a declaring class, `Inner` therefore becomes `GameInner`; a qualified
local reference `Outer.Inner` becomes `GameOuter.GameInner`; and the hoisted
oneof payload type continues to use `info.Reference`.

Change resolver construction to:

```go
func newResolver(file *protoast.ProtoFile, sourceName string, imports []FileEntry, localNamer typeNamer) *resolver
```

Store dependency namers and any invalid dependency-prefix errors by
`FileEntry.Filename`. Report an invalid imported prefix only when a field
actually references that dependency.

- [ ] **Step 5: Carry proto and generated scopes through message planning**

Change `planMessage` to receive both scopes:

```go
func planMessage(
	message *protoast.Message,
	protoParentScope string,
	generatedParentScope string,
	resolve *resolver,
) (messagePlan, error)
```

Build the lookup scope with empty-prefix `TypeName`, the emitted scope with
`resolve.localNamer.Name`, and store the final `Name` in `messagePlan`.
Render nested enums with the same local namer.

Build oneof names from the generated scope:

```go
func oneofTypeName(generatedScope string, oneof *protoast.Oneof) string {
	return strings.ReplaceAll(generatedScope, ".", "") + TypeName(oneof.Name) + "Case"
}
```

- [ ] **Step 6: Plan all declarations before rendering**

In `Generate`, validate the local prefix first, construct the resolver, build a
slice of top-level enum plans and message plans, and only then populate
`GeneratedFiles`. Use final names for every `outputPath` call.

Use:

```go
type enumPlan struct {
	Name string
	Enum *protoast.Enum
}
```

so rendering never recomputes a top-level enum name without its file namer.

- [ ] **Step 7: Run focused and full generator tests**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator -run 'TestGeneratePrefixes|TestReferenceUsesDeclaring' -count=1
go test ./internal/proto/internal/foundryscript/generator -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```text
git add internal/proto/internal/foundryscript/generator/generator.go internal/proto/internal/foundryscript/generator/plan.go internal/proto/internal/foundryscript/generator/generator_test.go
git commit -m "feat: apply protobuf type prefixes consistently"
```

### Task 5: Reject and aggregate engine-reserved declarations

**Files:**

- Create: `internal/proto/internal/foundryscript/generator/collisions.go`
- Create: `internal/proto/internal/foundryscript/generator/collisions_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator.go`
- Modify: `internal/proto/internal/foundryscript/generator/plan.go`

- [ ] **Step 1: Write failing current-file collision tests**

```go
func TestGenerateAggregatesEngineTypeCollisions(t *testing.T) {
	file := namespacedFile(
		[]*protoast.Message{{
			Position: protoast.Position{Line: 4, Column: 1},
			Name: "Node",
			NestedMessages: []*protoast.Message{{
				Position: protoast.Position{Line: 6, Column: 3},
				Name: "Timer",
			}},
		}},
		[]*protoast.Enum{{
			Position: protoast.Position{Line: 9, Column: 1},
			Name: "String",
			Values: []*protoast.EnumValue{{Name: "STRING_UNSPECIFIED", Number: 0}},
		}},
	)

	_, err := Generate(file, "types.proto", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), `types.proto:4:1: message cafecito.game.v1.Node generates Foundry type "Node", which conflicts with native class "Node"`)
	require.Contains(t, err.Error(), `types.proto:9:1: enum cafecito.game.v1.String generates Foundry type "String", which conflicts with built-in type "String"`)
	require.Contains(t, err.Error(), `types.proto:6:3: message cafecito.game.v1.Node.Timer generates Foundry type "Timer"`)
	require.Contains(t, err.Error(), `option (foundrytools.type_prefix) = "Game";`)
}
```

- [ ] **Step 2: Write failing prefix and non-exposed tests**

```go
func TestPrefixResolvesEngineCollisions(t *testing.T) {
	file := prefixedFile("Game", []*protoast.Message{{Name: "Node"}}, []*protoast.Enum{{
		Name: "String", Values: []*protoast.EnumValue{{Name: "STRING_UNSPECIFIED", Number: 0}},
	}})
	files, err := Generate(file, "types.proto", nil)
	require.NoError(t, err)
	require.Contains(t, files["cafecito/game/v1/GameNode.pb.fs"], "class_name GameNode")
}

func TestPrefixCanStillProduceReservedName(t *testing.T) {
	_, err := Generate(prefixedFile("Animation", []*protoast.Message{{Name: "Node"}}, nil), "types.proto", nil)
	require.ErrorContains(t, err, `current prefix "Animation" still produces reserved Foundry type names`)
}

func TestInternalNonExposedNameRemainsLegal(t *testing.T) {
	files, err := Generate(namespacedFile([]*protoast.Message{{Name: "FSNativeClass"}}, nil), "types.proto", nil)
	require.NoError(t, err)
	require.Contains(t, files["cafecito/game/v1/FSNativeClass.pb.fs"], "class_name FSNativeClass")
}
```

- [ ] **Step 3: Write a failing referenced-dependency deduplication test**

Create an imported `Node` used by two fields. Assert the error contains the
declaring `inventory.proto` collision exactly once and directs the prefix to
that file:

```go
require.Equal(t, 1, strings.Count(err.Error(), "inventory.proto: message cafecito.inventory.v1.Node"))
require.Contains(t, err.Error(), "set or change (foundrytools.type_prefix) in inventory.proto")
```

- [ ] **Step 4: Run focused tests and verify they fail**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator -run 'TestGenerateAggregates|TestPrefixResolves|TestPrefixCanStill|TestInternalNonExposed|TestReferencedDependency' -count=1
```

Expected: FAIL because reserved names are not validated.

- [ ] **Step 5: Implement collision metadata and formatting**

Use:

```go
type declarationInfo struct {
	SourceName    string
	Position      protoast.Position
	Kind          string
	ProtoName     string
	GeneratedName string
}

type typeCollision struct {
	Declaration declarationInfo
	EngineName  string
	EngineKind  engineTypeKind
}

type collisionCollector struct {
	byDeclaration map[string]typeCollision
}

func (c *collisionCollector) Add(info declarationInfo)
func (c *collisionCollector) Err(prefix string) error
```

`Add` looks up `info.GeneratedName` in `foundryEngineReservedTypes`. The
deduplication key includes source, declaration kind, and fully scoped proto
name. `Err` sorts by source then proto name and formats category-specific lines
plus one remediation block.

- [ ] **Step 6: Register declaration metadata**

While `typeRegistry.registerFile` walks top-level and nested declarations,
populate `declarationInfo` with package-qualified proto names, final generated
segment names, source file, and positions. Add all local declarations to the
collector. Add imported declarations only from `namedValuePlan` after the
resolver confirms the field uses that declaration.

When `planMessage` creates a oneof plan, create a declaration record with kind
`oneof enum`, the oneof position, scoped proto name, and flattened generated
name.

- [ ] **Step 7: Return collisions before rendering**

After all messages are planned, call `collector.Err(localNamer.prefix)`. Return
`nil, err` when non-nil; otherwise render the precomputed plans.

- [ ] **Step 8: Run generator tests**

Run:

```text
go test ./internal/proto/internal/foundryscript/generator -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```text
git add internal/proto/internal/foundryscript/generator/collisions.go internal/proto/internal/foundryscript/generator/collisions_test.go internal/proto/internal/foundryscript/generator/generator.go internal/proto/internal/foundryscript/generator/plan.go
git commit -m "fix: reject Foundry engine type collisions"
```

### Task 6: Cover direct CLI and protoc plugin behavior

**Files:**

- Create: `tests/integration/fixtures/collisions/types.proto`
- Create: `tests/integration/fixtures/collisions/prefixed.proto`
- Modify: `tests/integration/direct_cli_test.go`
- Modify: `internal/plugin/plugin_test.go`

- [ ] **Step 1: Add the direct CLI fixture**

```protobuf
syntax = "proto3";

package probe.collisions.v1;

message Node {
  string label = 1;
}
```

`tests/integration/fixtures/collisions/prefixed.proto`:

```protobuf
syntax = "proto3";

package probe.collisions.v1;

option (foundrytools.type_prefix) = "Game";

message Node {
  string label = 1;
}
```

- [ ] **Step 2: Add failing direct CLI tests**

```go
func TestDirectCLIReportsEngineTypeCollision(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	output := runFailure(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/collisions",
		"-o", outDir,
		"tests/integration/fixtures/collisions/types.proto")
	require.Contains(t, output, `native class "Node"`)
	require.Contains(t, output, `(foundrytools.type_prefix)`)
}

func TestDirectCLIAppliesTypePrefix(t *testing.T) {
	root := repoRoot(t)
	outDir := t.TempDir()
	run(t, root, "go", "run", "./cmd/anvil", "proto", "generate",
		"-I", "tests/integration/fixtures/collisions",
		"-o", outDir,
		"tests/integration/fixtures/collisions/prefixed.proto")
	data, err := os.ReadFile(filepath.Join(outDir, "probe/collisions/v1/GameNode.pb.fs"))
	require.NoError(t, err)
	require.Contains(t, string(data), "class_name GameNode")
}
```

Add this helper beside `run` in `tests/integration/helpers_test.go`:

```go
func runFailure(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "command unexpectedly succeeded:\n%s", out)
	return string(out)
}
```

- [ ] **Step 3: Add descriptor-path plugin tests**

Construct a `FileDescriptorProto` named `node.proto` with message `Node`.
Assert `Run` itself succeeds, the decoded response contains the collision in
`resp.GetError()`, and `resp.GetFile()` is empty.

For the success case:

```go
options := &descriptorpb.FileOptions{}
proto.SetExtension(options, foundrytoolspb.E_TypePrefix, "Game")
descriptor.Options = options
```

Assert the response has no error and includes
`probe/collisions/v1/GameNode.pb.fs` containing `class_name GameNode`.

- [ ] **Step 4: Run tests and verify behavior**

Run:

```text
go test ./internal/plugin -count=1
go test -tags=integration ./tests/integration -run 'TestDirectCLI' -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```text
git add internal/plugin/plugin_test.go tests/integration/direct_cli_test.go tests/integration/helpers_test.go tests/integration/fixtures/collisions
git commit -m "test: cover engine type collision entry points"
```

### Task 7: Verify generated bindings in Foundry and pin alpha.14

**Files:**

- Create: `tests/foundry/collision_dependency.proto`
- Create: `tests/foundry/collisions.proto`
- Modify: `tests/foundry/run.sh`
- Modify: `tests/foundry/main.fs`
- Modify: `scripts/ci/install-foundry.sh`
- Modify: `.github/workflows/foundry.yml`

- [ ] **Step 1: Add prefixed Foundry schemas**

`tests/foundry/collision_dependency.proto`:

```protobuf
syntax = "proto3";
package probe.dependency.v1;

option (foundrytools.type_prefix) = "Dependency";

message Timer {
  string label = 1;
}
```

`tests/foundry/collisions.proto`:

```protobuf
syntax = "proto3";
package probe.collisions.v1;

import "collision_dependency.proto";

option (foundrytools.type_prefix) = "Game";

enum String {
  STRING_UNSPECIFIED = 0;
  STRING_READY = 1;
}

message Node {
  message Timer {
    string label = 1;
  }

  Timer nested = 1;
  probe.dependency.v1.Timer imported = 2;
  String state = 3;

  oneof payload {
    string text = 4;
    int32 amount = 5;
  }
}
```

- [ ] **Step 2: Extend Foundry generation**

Add `tests/foundry` to the include paths and pass both new root files to
`anvil proto generate`. Before generation, run:

```bash
FOUNDRY_BIN="$FOUNDRY" bash "$ROOT/scripts/ci/sync-foundry-engine-types.sh" check
```

- [ ] **Step 3: Add runtime assertions**

In `tests/foundry/main.fs`, import both new namespaces and add:

```foundryscript
var collision: GameNode = GameNode.new()
var nested_timer: GameNode.GameTimer = GameNode.GameTimer.new()
nested_timer.label = "nested"
collision.nested = nested_timer

var dependency_timer: DependencyTimer = DependencyTimer.new()
dependency_timer.label = "imported"
collision.imported = dependency_timer
collision.state = GameString.STRING_READY
collision.payload = GameNodePayloadCase.Text("safe")

var (collision_decoded, collision_error) = GameNode.from_bytes(collision.to_bytes())
check(collision_error == ProtobufError.OK, "prefixed collision fixture decodes")
check(collision_decoded is GameNode, "prefixed collision fixture has the renamed type")
if collision_decoded is GameNode:
	check(collision_decoded.nested is GameNode.GameTimer, "prefixed nested type")
	check(collision_decoded.imported is DependencyTimer, "dependency prefix")
	check(collision_decoded.state == GameString.STRING_READY, "prefixed built-in collision")
```

Match the generated tagged-union case spelling exactly; the current case
normalization makes field `text` become `Text`.

- [ ] **Step 4: Update both Foundry pins**

Change the default and workflow environment from `v0.1.0-alpha.11` to
`v0.1.0-alpha.14`.

- [ ] **Step 5: Build and run Foundry verification**

Run:

```text
task build
FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test
```

Expected: the drift check passes, Foundry lint emits zero error diagnostics,
and the runner prints `round trip ok`.

- [ ] **Step 6: Commit**

```text
git add tests/foundry/collision_dependency.proto tests/foundry/collisions.proto tests/foundry/run.sh tests/foundry/main.fs scripts/ci/install-foundry.sh .github/workflows/foundry.yml
git commit -m "test: exercise prefixed engine type collisions"
```

### Task 8: Document the option and run full verification

**Files:**

- Modify: `README.md`
- Modify: `proto/foundrytools/options.proto`
- Modify: `internal/foundrytoolspb/options.proto`
- Modify: `internal/foundrytoolspb/options.pb.go`
- Modify: `internal/foundrytoolspb/embed_test.go`

- [ ] **Step 1: Add option-schema documentation**

Above `type_prefix`:

```protobuf
// Literal prefix applied to every generated type in this file.
// Use it to avoid Foundry built-in, native-class, or project-name collisions.
optional string type_prefix = 52001;
```

- [ ] **Step 2: Add the failing embedded-schema assertion**

```go
func TestEmbeddedOptionsProtoDocumentsTypePrefix(t *testing.T) {
	text := string(Bytes())
	require.Contains(t, text, "// Literal prefix applied to every generated type in this file.")
	require.Contains(t, text, "// Use it to avoid Foundry built-in, native-class, or project-name collisions.")
}
```

Run:

```text
go test ./internal/foundrytoolspb -run TestEmbeddedOptionsProtoDocumentsTypePrefix -count=1
```

Expected: FAIL until the embedded schema is regenerated.

- [ ] **Step 3: Expand the README custom-options section**

Document:

```protobuf
option (foundrytools.type_prefix) = "Game";
```

Include the mappings:

```text
Node                         -> GameNode
Outer.Inner                  -> GameOuter.GameInner
Player.payload oneof enum    -> GamePlayerPayloadCase
Node.pb.fs                   -> GameNode.pb.fs
```

State that the value is a literal non-empty identifier fragment, that it is
applied before keyword/runtime escaping, that it changes the public generated
API, and that project-specific names remain Foundry lint-time concerns.

- [ ] **Step 4: Regenerate option artifacts**

Run:

```text
task gen-options
```

Expected: both embedded `options.proto` and `options.pb.go` carry the new
comment.

- [ ] **Step 5: Run formatting and generated-output checks**

Run:

```text
task fmt
go test ./internal/proto -run TestGoldenExampleProto -count=1
git diff --check
```

Expected: all commands pass; `examples/golden` remains unchanged.

- [ ] **Step 6: Run complete verification**

Run:

```text
task ci
task integration
task build
FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test
```

Expected:

- `task ci`: zero formatting, tidy, lint, unit-test, or build failures.
- `task integration`: all direct CLI, protoc plugin, and Buf tests pass.
- `task foundry:test`: table drift check passes, Foundry lint has no errors,
  and runtime prints `round trip ok`.

- [ ] **Step 7: Commit**

```text
git add README.md proto/foundrytools/options.proto internal/foundrytoolspb/options.proto internal/foundrytoolspb/options.pb.go internal/foundrytoolspb/embed_test.go
git commit -m "docs: explain Foundry type prefixes"
```

- [ ] **Step 8: Verify branch state**

Run:

```text
git status --short --branch
git log --oneline --decorate -10
```

Expected: clean `fix/issue-30-engine-type-collisions` worktree with the approved
spec, safety commit, implementation plan, and focused implementation commits.
