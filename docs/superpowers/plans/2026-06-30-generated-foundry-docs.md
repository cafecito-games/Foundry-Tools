# Generated Foundry Docs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add deterministic Godot/Foundry-style `##` documentation comments to the public API emitted by the Foundry Script protobuf generator.

**Architecture:** Add renderer-level documentation support in `internal/fsast`, then have `internal/fsgenerator` supply deterministic prose for generated messages, enums, and public methods. Keep `.proto` comment preservation out of scope; that follow-up is tracked in GitHub issue #2.

**Tech Stack:** Go 1.26, existing `fsast` renderer, existing `fsgenerator` tests/goldens, Foundry Script `##` doc comments.

---

## File Structure

Modify these files:

```text
internal/fsast/declarations.go
internal/fsast/render_test.go
internal/fsgenerator/generator.go
internal/fsgenerator/fields.go
internal/fsgenerator/serialize.go
internal/fsgenerator/deserialize.go
internal/fsgenerator/generator_test.go
examples/golden/cafecito/game/v1/Player.pb.fs
examples/golden/cafecito/game/v1/PlayerStatus.pb.fs
```

Responsibilities:

- `internal/fsast/declarations.go`: render `##` docs for documented declarations.
- `internal/fsast/render_test.go`: cover class, function, and documented raw declaration rendering.
- `internal/fsgenerator/*`: attach deterministic docs to generated public API declarations.
- `examples/golden/*`: reflect the generated source format after docs are added.

---

### Task 1: Add Renderer Support for Documentation Comments

**Files:**
- Modify: `internal/fsast/declarations.go`
- Modify: `internal/fsast/render_test.go`

- [ ] **Step 1: Write failing renderer tests**

Replace `TestFileRendering` in `internal/fsast/render_test.go` with this version:

```go
func TestFileRendering(t *testing.T) {
	file := File{
		Namespace: "cafecito.game.v1",
		Imports:   []string{"foundry.proto"},
		Declarations: []Node{
			Class{
				Doc:     []string{"Generated protobuf message binding for Player."},
				Final:   true,
				Name:    "Player",
				Extends: "RefCounted",
				Uses:    []string{"foundry.proto.Message[Player]"},
				Members: []Node{
					Var{Name: "_name", Type: fstypes.Named("String"), Value: `""`},
					Func{
						Doc:        []string{"Returns the name protobuf field."},
						Name:       "get_name",
						ReturnType: fstypes.Named("String"),
						Body:       []Node{Return{Value: "_name"}},
					},
					Doc{Lines: []string{"Generated protobuf enum binding for PlayerStatus."}, Node: Raw{Code: "enum_name PlayerStatus {}\n"}},
				},
			},
		},
	}

	require.Equal(t, "namespace cafecito.game.v1\nimport foundry.proto\n\n## Generated protobuf message binding for Player.\nfinal class_name Player extends RefCounted uses foundry.proto.Message[Player]\n\nvar _name: String = \"\"\n\n## Returns the name protobuf field.\nfunc get_name() -> String:\n\treturn _name\n\n## Generated protobuf enum binding for PlayerStatus.\nenum_name PlayerStatus {}\n", file.Render())
}
```

- [ ] **Step 2: Run renderer tests and verify they fail**

Run:

```bash
go test ./internal/fsast -run TestFileRendering -count=1
```

Expected failure:

```text
unknown field Doc in struct literal of type Class
unknown field Doc in struct literal of type Func
```

- [ ] **Step 3: Implement renderer support**

In `internal/fsast/declarations.go`, add doc fields and a documented wrapper:

```go
// Class represents a Foundry Script class declaration.
type Class struct {
	Doc     []string
	Final   bool
	Name    string
	Extends string
	Uses    []string
	Members []Node
}

// Var represents a typed variable declaration.
type Var struct {
	Name  string
	Type  fstypes.Type
	Value string
}

// Func represents a typed function declaration.
type Func struct {
	Doc        []string
	Static     bool
	Name       string
	Parameters []Parameter
	ReturnType fstypes.Type
	ReturnVoid bool
	Body       []Node
}

// Doc adds documentation comments above another node.
type Doc struct {
	Lines []string
	Node  Node
}
```

Add this helper near the render methods:

```go
func renderDoc(builder *strings.Builder, indent int, lines []string) {
	for _, line := range lines {
		builder.WriteString(indentation(indent))
		builder.WriteString("##")
		if line != "" {
			builder.WriteByte(' ')
			builder.WriteString(line)
		}
		builder.WriteByte('\n')
	}
}
```

Update `Class.RenderAt` so the first lines are:

```go
func (c Class) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, c.Doc)
	builder.WriteString(indentation(indent))
```

Update `Func.RenderAt` so the first lines are:

```go
func (fn Func) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, fn.Doc)
	builder.WriteString(indentation(indent))
```

Add the wrapper renderer:

```go
// RenderAt renders d at indent.
func (d Doc) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, d.Lines)
	if d.Node != nil {
		builder.WriteString(d.Node.RenderAt(indent))
	}
	return builder.String()
}
```

- [ ] **Step 4: Run renderer tests and verify they pass**

Run:

```bash
go test ./internal/fsast -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsast
```

- [ ] **Step 5: Commit renderer support**

Run:

```bash
git add internal/fsast/declarations.go internal/fsast/render_test.go
git commit -m "feat: render foundry doc comments"
```

---

### Task 2: Generate Deterministic Docs for Public API

**Files:**
- Modify: `internal/fsgenerator/generator.go`
- Modify: `internal/fsgenerator/fields.go`
- Modify: `internal/fsgenerator/serialize.go`
- Modify: `internal/fsgenerator/deserialize.go`
- Modify: `internal/fsgenerator/generator_test.go`

- [ ] **Step 1: Write failing generator tests**

Append these assertions to `TestGenerateMessageAndEnumSkeletons` in `internal/fsgenerator/generator_test.go` after `source :=` or direct map access setup. If the test does not already assign `source`, add:

```go
messageSource := files["cafecito/game/v1/Player.pb.fs"]
enumSource := files["cafecito/game/v1/PlayerStatus.pb.fs"]
require.Contains(t, messageSource, "## Generated protobuf message binding for Player.\nfinal class_name Player extends RefCounted")
require.Contains(t, enumSource, "## Generated protobuf enum binding for PlayerStatus.\nenum_name PlayerStatus")
require.NotContains(t, messageSource, "## var _name")
```

Append these assertions to `TestGenerateTypedAccessorsAndDecodeFactory`:

```go
require.Contains(t, source, "## Sets the name protobuf field.\nfunc set_name(value: String) -> void:")
require.Contains(t, source, "## Returns the name protobuf field.\nfunc get_name() -> String:")
require.Contains(t, source, "## Sets the level protobuf field.\nfunc set_level(value: int) -> void:")
require.Contains(t, source, "## Decodes protobuf wire data into a new Player message.\nstatic func from_bytes(data: PackedByteArray) -> DecodeResult[Player]:")
```

Append these assertions to `TestGenerateScalarSerialization`:

```go
require.Contains(t, source, "## Serializes this message to protobuf wire data.\nfunc to_bytes() -> PackedByteArray:")
require.Contains(t, source, "## Merges protobuf wire data into this message.\nfunc merge_from_bytes(data: PackedByteArray) -> foundry.proto.ProtobufError:")
```

- [ ] **Step 2: Run generator tests and verify they fail**

Run:

```bash
go test ./internal/fsgenerator -run 'TestGenerateMessageAndEnumSkeletons|TestGenerateTypedAccessorsAndDecodeFactory|TestGenerateScalarSerialization' -count=1
```

Expected failure:

```text
does not contain "## Generated protobuf message binding for Player."
```

- [ ] **Step 3: Add deterministic doc helper functions**

Add these helpers to `internal/fsgenerator/generator.go` below `validateWireFields`:

```go
func messageDoc(typeName string) []string {
	return []string{"Generated protobuf message binding for " + typeName + "."}
}

func enumDoc(typeName string) []string {
	return []string{"Generated protobuf enum binding for " + typeName + "."}
}

func setterDoc(fieldName string) []string {
	return []string{"Sets the " + fieldName + " protobuf field."}
}

func getterDoc(fieldName string) []string {
	return []string{"Returns the " + fieldName + " protobuf field."}
}

func fromBytesDoc(typeName string) []string {
	return []string{"Decodes protobuf wire data into a new " + typeName + " message."}
}

func toBytesDoc() []string {
	return []string{"Serializes this message to protobuf wire data."}
}

func mergeFromBytesDoc() []string {
	return []string{"Merges protobuf wire data into this message."}
}
```

- [ ] **Step 4: Document enums and message classes**

In `renderEnum`, replace:

```go
return fsast.File{
	Namespace:    namespace,
	Declarations: []fsast.Node{fsast.Raw{Code: builder.String()}},
}.Render()
```

with:

```go
return fsast.File{
	Namespace: namespace,
	Declarations: []fsast.Node{
		fsast.Doc{
			Lines: enumDoc(typeName),
			Node:  fsast.Raw{Code: builder.String()},
		},
	},
}.Render()
```

In `renderMessage`, add `Doc: messageDoc(typeName),` to the `fsast.Class` literal:

```go
fsast.Class{
	Doc:     messageDoc(typeName),
	Final:   true,
	Name:    typeName,
	Extends: "RefCounted",
	Members: members,
},
```

- [ ] **Step 5: Document generated accessors**

In `internal/fsgenerator/fields.go`, update `fieldMembers` so the returned functions include docs:

```go
return []fsast.Node{
	fsast.Func{
		Doc:        setterDoc(name),
		Name:       "set_" + name,
		Parameters: []fsast.Parameter{{Name: "value", Type: typ}},
		ReturnVoid: true,
		Body:       []fsast.Node{fsast.Assign{Target: backing, Value: "value"}},
	},
	fsast.Func{
		Doc:        getterDoc(name),
		Name:       "get_" + name,
		ReturnType: typ,
		Body:       []fsast.Node{fsast.Return{Value: backing}},
	},
}
```

Update `fromBytesFactory` so the `fsast.Func` literal starts with:

```go
return fsast.Func{
	Doc:        fromBytesDoc(className),
	Static:     true,
	Name:       "from_bytes",
```

- [ ] **Step 6: Document serialization methods**

In `internal/fsgenerator/serialize.go`, update `toBytesFunction` so the return literal starts with:

```go
return fsast.Func{
	Doc:        toBytesDoc(),
	Name:       "to_bytes",
	ReturnType: fstypes.Named("PackedByteArray"),
	Body:       body,
}
```

In `internal/fsgenerator/deserialize.go`, update `mergeFromBytesFunction` so the return literal starts with:

```go
return fsast.Func{
	Doc:        mergeFromBytesDoc(),
	Name:       "merge_from_bytes",
	Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
	ReturnType: fstypes.Named("foundry.proto.ProtobufError"),
	Body:       body,
}
```

- [ ] **Step 7: Run generator tests and verify they pass**

Run:

```bash
go test ./internal/fsgenerator -count=1
```

Expected:

```text
ok  	github.com/cafecito-games/foundry-tools/internal/fsgenerator
```

- [ ] **Step 8: Commit generator docs**

Run:

```bash
git add internal/fsgenerator/generator.go internal/fsgenerator/fields.go internal/fsgenerator/serialize.go internal/fsgenerator/deserialize.go internal/fsgenerator/generator_test.go
git commit -m "feat: document generated foundry api"
```

---

### Task 3: Refresh Goldens and Verify

**Files:**
- Modify: `examples/golden/cafecito/game/v1/Player.pb.fs`
- Modify: `examples/golden/cafecito/game/v1/PlayerStatus.pb.fs`

- [ ] **Step 1: Refresh generated golden files**

Run:

```bash
rm -rf /tmp/foundry-tools-golden
task build
bin/foundry-tools proto generate -I . -o /tmp/foundry-tools-golden examples/example.proto
cp /tmp/foundry-tools-golden/cafecito/game/v1/Player.pb.fs examples/golden/cafecito/game/v1/Player.pb.fs
cp /tmp/foundry-tools-golden/cafecito/game/v1/PlayerStatus.pb.fs examples/golden/cafecito/game/v1/PlayerStatus.pb.fs
```

Expected:

```text
generated Foundry Script for 1 proto file(s)
```

- [ ] **Step 2: Inspect golden diff**

Run:

```bash
git diff -- examples/golden/cafecito/game/v1/Player.pb.fs examples/golden/cafecito/game/v1/PlayerStatus.pb.fs
```

Expected: only `##` documentation comments are added above public generated declarations.

- [ ] **Step 3: Run full verification**

Run:

```bash
task ci
task integration
FOUNDRY_BIN="$HOME/.foundry/bin/foundry.macos.editor.dev.arm64" task foundry:test
rg -n '(^|[^_])func [A-Za-z0-9_]+\(.*Variant|-> Variant' examples/golden internal/runtime/data || true
```

Expected:

```text
task ci passes
task integration passes
task foundry:test passes
Variant scan has no output
```

- [ ] **Step 4: Commit goldens**

Run:

```bash
git add examples/golden/cafecito/game/v1/Player.pb.fs examples/golden/cafecito/game/v1/PlayerStatus.pb.fs
git commit -m "test: refresh documented foundry goldens"
```

---

### Task 4: Update Pull Request Branch

**Files:**
- No file edits.

- [ ] **Step 1: Confirm clean status**

Run:

```bash
git status -sb
```

Expected:

```text
## feature/foundry-tools-bootstrap...origin/feature/foundry-tools-bootstrap [ahead N]
```

- [ ] **Step 2: Push the PR branch**

Run:

```bash
git push origin feature/foundry-tools-bootstrap
```

Expected: branch updates on GitHub.

- [ ] **Step 3: Confirm PR head**

Run:

```bash
gh pr view 1 --json number,url,headRefName,headRefOid,state,isDraft
```

Expected: PR #1 remains open/draft and `headRefOid` matches the latest local commit.
