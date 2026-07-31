# Well-Known Type Semantics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move every `google/protobuf/*.proto` well-known type into the runtime under `foundry.proto.wkt`, and give `Struct`, `Timestamp`/`Duration`, and `Any` the semantics a schema author expects.

**Architecture:** The seventeen well-known types are produced by the existing generator into checked-in runtime data, not hand-written — a `task gen-wkt` target regenerates them and a test asserts the checked-in output matches. Semantics attach to those generated classes through Foundry Script's retroactive conformance (`extend Target uses Trait`), so the generated files stay pristine and regenerable. The plugin stops emitting output for well-known files and rewrites references onto `foundry.proto.wkt`.

**Tech Stack:** Go 1.x (generator, `internal/proto`, `internal/plugin`, `internal/runtime`), Foundry Script (runtime `.fs` under `internal/runtime/data/`), Taskfile, testify.

---

## Status: Reduced Scope After Task 0

**Task 0 rejected the mechanism.** The spike found two gaps in the pinned engine:

- **cafecito-games/Foundry#1376** — `extend`-supplied `static` witnesses are not found by `Type.method()`. This kills every static constructor the design needs: `Timestamp.now()`, `Any.pack()`, `Struct.from_dictionary()`, `Value.from_variant()`, `ListValue.from_array()`.
- **cafecito-games/Foundry#1377** — a conformance is invisible through a namespace `import`; consumers need an explicit `preload()`. A direct witness call passes the analyzer without it and crashes at runtime. Generated code is wired by namespace import, so this is load-bearing.

Instance witnesses and tuple returns do work. But an API whose constructors are all static cannot be built on instance methods alone.

**In scope now** — the structural half, which depends on none of this:

| Plan task | Status |
|---|---|
| Task 0 | Done — mechanism rejected, nothing committed |
| Task 2 | **In scope** — vendor and generate the bindings |
| Task 3 | **In scope** — reference routing |
| Task 7 | **In scope, reduced** — document the move, not the conversions |

**Deferred to [#43](https://github.com/cafecito-games/Foundry-Tools/issues/43):** Tasks 1, 4, 5, and 6 — `Message.type_name()`, `Struct` ↔ `Variant`, `Timestamp`/`Duration`, and `Any`. Their task text below is retained as-is, since the design is settled and only the delivery mechanism is open.

`type_name()` (Task 1) goes with them because `Any.pack` is its only consumer. It is a breaking change to the `Message` trait that regenerates every golden file, so it should land with the work that uses it rather than sitting unused. **Task 2 therefore generates bindings without `type_name()`**; they regenerate via `task gen-wkt` when #43 lands.

The sections below are the plan as written before Task 0 ran, retained unchanged.

---

## Deviation From The Spec

The approved spec (`docs/superpowers/specs/2026-07-30-well-known-type-semantics-design.md`) says the well-known types are **hand-written** `.fs`, and flags the resulting drift risk: "they must be indistinguishable in shape from generator output … where this design and the generator's conventions disagree, the generator's conventions win."

Exploration found a better mechanism that satisfies the same intent with less risk:

1. **The generator can produce them.** All seven upstream files generate cleanly now that #36 (fixed-width and zig-zag scalars) has landed. Generating removes the drift risk entirely rather than managing it.
2. **Foundry Script has retroactive conformance** (`GRAMMAR.md` §4.8): `extend Target uses Trait:` supplies witness methods for a type the script does not own. The engine implements it — `fs_analyzer_conformance.cpp` and `fs_conformance_registry.cpp`. Witnesses may be `static`, so `Timestamp.now()` works alongside `timestamp.to_unix_time()`.

This preserves every call-site signature the spec specifies. It is a change of mechanism, not of design.

**It is gated on Task 0.** `extend` is verified present in the engine source, but not yet verified working against the pinned binary this repo tests with — particularly static witnesses and witnesses on a `RefCounted` subclass. If Task 0 fails, fall back to the spec's hand-written approach and adjust Tasks 2, 4, 5, and 6 accordingly.

---

## File Structure

**Created:**

| Path | Responsibility |
|---|---|
| `internal/proto/wellknown/wellknown.go` | The single source of truth for which `google/protobuf` files are well-known, the `foundry.proto.wkt` namespace constant, and the unsupported-file diagnostic. No dependencies on the generator, so both entry paths can use it. |
| `internal/proto/wellknown/wellknown_test.go` | Membership, path-normalization, and diagnostic tests. |
| `internal/proto/wellknown/proto/google/protobuf/*.proto` | The seven vendored upstream files (`any`, `duration`, `empty`, `field_mask`, `struct`, `timestamp`, `wrappers`), embedded so generation needs no external protoc include path. |
| `internal/runtime/data/foundry/proto/wkt/*.pb.fs` | Generated bindings for the seventeen well-known types. Checked in, regenerated by `task gen-wkt`. |
| `internal/runtime/data/foundry/proto/wkt/struct_variant.fs` | `extend Struct/Value/ListValue` — `Variant` conversion. |
| `internal/runtime/data/foundry/proto/wkt/time_conversion.fs` | `extend Timestamp/Duration` — `float` seconds conversion. |
| `internal/runtime/data/foundry/proto/wkt/any_packing.fs` | `extend Any` — pack, typed unpack, type test. |
| `examples/golden-wkt/` | Golden example importing `timestamp.proto`, `struct.proto`, and `any.proto`. |
| `tests/foundry/wkt_test.fs` | Engine-level round-trip and conversion checks. |

**Modified:**

| Path | Change |
|---|---|
| `internal/runtime/data/foundry/proto/message.fs` | Add `abstract func type_name() -> String`. |
| `internal/runtime/data/foundry/proto/protobuf_error.fs` | Add three error cases. |
| `internal/proto/internal/foundryscript/generator/generator.go` | Emit `type_name()` on every message class. |
| `internal/proto/internal/foundryscript/generator/plan.go:351` | Route well-known dependency namespaces to `foundry.proto.wkt`. |
| `internal/proto/command.go:57-68` | Skip generating well-known files. |
| `internal/plugin/plugin.go` | Same skip on the protoc/Buf path. |
| `Taskfile.yml` | Add `gen-wkt`. |
| `README.md` | Document the well-known type mapping. |
| `examples/golden/**` | Regenerated for `type_name()`. |

---

## Task 0: Verify retroactive conformance works against the pinned engine

**Goal:** Prove `extend` supports what Tasks 4–6 need before building on it, or fail fast to the hand-written fallback.

**Files:**
- Create: `tests/foundry/conformance_spike.fs` (deleted at the end of this task)

**Acceptance Criteria:**
- [ ] An `extend` on a `final class … extends RefCounted uses Message` class loads and runs
- [ ] An instance witness is callable as `instance.method()`
- [ ] A `static` witness is callable as `Type.method()`
- [ ] A witness returning a tuple `(T?, ProtobufError)` works
- [ ] Result recorded in the task comments either way

**Verify:** `task foundry:test` → exits 0 with the spike's assertions passing

**Steps:**

- [ ] **Step 1: Write the spike**

Create `tests/foundry/conformance_spike.fs`:

```foundryscript
namespace foundry.proto.spike
import foundry.proto

final class_name SpikeTarget extends RefCounted uses Message

var value: int = 0

func type_name() -> String:
	return "spike.SpikeTarget"

func to_bytes() -> PackedByteArray:
	return PackedByteArray()

func merge_from_bytes(_pb_data: PackedByteArray) -> ProtobufError:
	return ProtobufError.OK

trait_name Doubling

abstract func doubled() -> int

abstract static func of_value(value: int) -> SpikeTarget

abstract func checked() -> (SpikeTarget?, ProtobufError)

extend SpikeTarget uses Doubling:
	func doubled() -> int:
		return value * 2

	static func of_value(value: int) -> SpikeTarget:
		var target: SpikeTarget = SpikeTarget.new()
		target.value = value
		return target

	func checked() -> (SpikeTarget?, ProtobufError):
		return (self, ProtobufError.OK)
```

- [ ] **Step 2: Exercise it from the harness**

Add to `tests/foundry/main.fs`, following the assertion style already in that file:

```foundryscript
var spike: SpikeTarget = SpikeTarget.of_value(21)
assert(spike.doubled() == 42, "instance witness")
var checked: (SpikeTarget?, ProtobufError) = spike.checked()
assert(checked[1] == ProtobufError.OK, "tuple-returning witness")
```

- [ ] **Step 3: Run against the engine**

Run: `task foundry:test`
Expected: exits 0. If it fails on `extend` parsing, static witness dispatch, or tuple returns, record which and stop — the fallback is the spec's hand-written approach.

- [ ] **Step 4: Record the outcome and remove the spike**

```bash
rm tests/foundry/conformance_spike.fs
git checkout tests/foundry/main.fs
```

Record in the task comment: which of the four criteria passed. Do not commit the spike.

---

## Task 1: Add `type_name()` to the `Message` trait

**Goal:** Every generated message reports its fully qualified protobuf name, which `Any.pack` needs.

**Files:**
- Modify: `internal/runtime/data/foundry/proto/message.fs`
- Modify: `internal/proto/internal/foundryscript/generator/generator.go` (`messageClass`, ~line 332)
- Test: `internal/proto/internal/foundryscript/generator/generator_test.go`
- Modify: `examples/golden/**` (regenerated)

**Acceptance Criteria:**
- [ ] `Message` declares `abstract func type_name() -> String`
- [ ] Every generated message emits `type_name()` returning its fully qualified proto name
- [ ] A nested message returns `pkg.Outer.Inner`, not `pkg.Inner`
- [ ] A message in a file with no `package` returns the bare name
- [ ] Golden files regenerated and matching

**Verify:** `task test` → PASS, and `task foundry:test` → exits 0

**Steps:**

- [ ] **Step 1: Write the failing test**

In `generator_test.go`:

```go
func TestGeneratedMessageEmitsTypeName(t *testing.T) {
	file := parseProto(t, `
syntax = "proto3";
package cafecito.game.v1;
message Player {
  string name = 1;
  message Inner { string label = 1; }
}
`)
	files, err := Generate(file, "player.proto", nil)
	require.NoError(t, err)

	source := files["cafecito/game/v1/Player.pb.fs"]
	require.Contains(t, source, "func type_name() -> String:\n\treturn \"cafecito.game.v1.Player\"")
	require.Contains(t, source, "return \"cafecito.game.v1.Player.Inner\"")
}
```

Use whatever helper the existing tests in this file use to build a `*protoast.ProtoFile`; match their style rather than introducing `parseProto` if an equivalent already exists.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/proto/internal/foundryscript/generator/ -run TestGeneratedMessageEmitsTypeName -v`
Expected: FAIL — the emitted source has no `type_name`.

- [ ] **Step 3: Add the trait requirement**

`internal/runtime/data/foundry/proto/message.fs` becomes:

```foundryscript
namespace foundry.proto

trait_name Message

abstract func to_bytes() -> PackedByteArray

abstract func merge_from_bytes(_data: PackedByteArray) -> ProtobufError

abstract func type_name() -> String
```

- [ ] **Step 4: Emit it from the generator**

`messagePlan` already carries the qualified proto name (`qualifiedProtoName(file.Package, message.Name)` is passed into `planMessage` in `generator.go:54`). Confirm it is retained on the plan as a field; if it is not, add one — `QualifiedName string` — and populate it in `planMessage` for both the top-level and nested cases, so `Outer.Inner` is built from the parent's qualified name.

Add a doc helper next to the others in `generator.go`:

```go
func typeNameDoc() []string {
	return []string{"Returns this message's fully qualified protobuf type name."}
}
```

And in `messageClass`, in the trailing `members = append(...)` block:

```go
members = append(members,
	unknownFieldsMemberDeclaration(),
	typeNameFunction(plan.QualifiedName),
	fromBytesFactory(plan.Name),
	toBytesFunction(plan.Fields, plan.Oneofs),
	mergeFromBytesFunction(plan.Fields),
)
```

With:

```go
func typeNameFunction(qualifiedName string) fsast.Node {
	return fsast.Func{
		Doc:        typeNameDoc(),
		Name:       "type_name",
		ReturnType: fstypes.Named("String"),
		Body:       []fsast.Node{line(0, "return "+strconv.Quote(qualifiedName))},
	}
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/proto/internal/foundryscript/generator/ -run TestGeneratedMessageEmitsTypeName -v`
Expected: PASS

- [ ] **Step 6: Regenerate goldens and run the full suite**

Run: `task test`
Expected: the golden test fails with a diff showing `type_name()` added.

Regenerate using whatever mechanism `internal/proto/golden_test.go` documents (check for an `-update` flag or a Taskfile target before hand-editing), then:

Run: `task test && task foundry:test`
Expected: PASS, exit 0

- [ ] **Step 7: Commit**

```bash
git add internal/runtime/data/foundry/proto/message.fs \
        internal/proto/internal/foundryscript/generator/ \
        examples/golden/
git commit -m "feat: report a message's qualified protobuf name via Message.type_name()"
```

---

## Task 2: Vendor the well-known protos and generate their bindings

**Goal:** The seventeen well-known types exist in the runtime as generated, checked-in, regenerable `.fs`.

**Files:**
- Create: `internal/proto/wellknown/wellknown.go`
- Create: `internal/proto/wellknown/wellknown_test.go`
- Create: `internal/proto/wellknown/proto/google/protobuf/{any,duration,empty,field_mask,struct,timestamp,wrappers}.proto`
- Create: `internal/runtime/data/foundry/proto/wkt/*.pb.fs` (generated)
- Modify: `Taskfile.yml`

**Acceptance Criteria:**
- [ ] The seven upstream files are vendored verbatim, with their Google copyright headers intact
- [ ] `task gen-wkt` regenerates the bindings into `internal/runtime/data/foundry/proto/wkt/`
- [ ] All generated files declare `namespace foundry.proto.wkt`
- [ ] A test fails if the checked-in output drifts from what the generator produces
- [ ] `Struct`, `Value`, and `ListValue` resolve despite their mutual recursion
- [ ] Every generated file loads in the engine

**Verify:** `go test ./internal/proto/wellknown/... ./internal/runtime/... -v` → PASS, `task foundry:test` → exits 0

**Steps:**

- [ ] **Step 1: Vendor the upstream protos**

Fetch the seven files from the protobuf release matching what the repo targets. Do not hand-transcribe them — the point is fidelity to upstream.

```bash
mkdir -p internal/proto/wellknown/proto/google/protobuf
for f in any duration empty field_mask struct timestamp wrappers; do
  curl -fsSL "https://raw.githubusercontent.com/protocolbuffers/protobuf/v27.0/src/google/protobuf/$f.proto" \
    -o "internal/proto/wellknown/proto/google/protobuf/$f.proto"
done
```

Verify each retains its `// Protocol Buffers - Google's data interchange format` header and its `package google.protobuf;` line. Record the upstream tag used in a `README.md` beside them so a future refresh is reproducible.

- [ ] **Step 2: Write the membership test**

`internal/proto/wellknown/wellknown_test.go`:

```go
func TestIsWellKnown(t *testing.T) {
	require.True(t, IsWellKnown("google/protobuf/timestamp.proto"))
	require.True(t, IsWellKnown("google/protobuf/struct.proto"))
	require.False(t, IsWellKnown("cafecito/game/v1/player.proto"))
}

func TestUnsupportedGoogleFileIsRejected(t *testing.T) {
	err := Check("google/protobuf/descriptor.proto")
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptor.proto")
	require.Contains(t, err.Error(), "not supported")
	require.NoError(t, Check("google/protobuf/timestamp.proto"))
	require.NoError(t, Check("cafecito/game/v1/player.proto"))
}

func TestEmbeddedProtosArePresent(t *testing.T) {
	for _, name := range Files() {
		source, err := Source(name)
		require.NoError(t, err)
		require.Contains(t, string(source), "package google.protobuf;")
	}
	require.Len(t, Files(), 7)
}
```

- [ ] **Step 3: Run to verify it fails**

Run: `go test ./internal/proto/wellknown/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 4: Write the package**

`internal/proto/wellknown/wellknown.go`:

```go
// Package wellknown identifies the google/protobuf well-known types, which are
// shipped as runtime source rather than generated per project. Generating them
// per project would give every project its own incompatible Timestamp.
package wellknown

import (
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed proto/google/protobuf/*.proto
var protoFS embed.FS

// Namespace is the Foundry Script namespace the well-known bindings live in.
const Namespace = "foundry.proto.wkt"

const protoPrefix = "google/protobuf/"

// supported is the set of well-known files the runtime ships bindings for.
var supported = map[string]bool{
	protoPrefix + "any.proto":        true,
	protoPrefix + "duration.proto":   true,
	protoPrefix + "empty.proto":      true,
	protoPrefix + "field_mask.proto": true,
	protoPrefix + "struct.proto":     true,
	protoPrefix + "timestamp.proto":  true,
	protoPrefix + "wrappers.proto":   true,
}

// IsWellKnown reports whether filename names a well-known file the runtime
// ships bindings for.
func IsWellKnown(filename string) bool {
	return supported[normalize(filename)]
}

// Check rejects a google/protobuf file the runtime does not ship. Falling back
// to generic generation would silently produce a second, incompatible copy of a
// type the runtime already defines, so an unshipped file is an error rather
// than a quiet divergence.
func Check(filename string) error {
	name := normalize(filename)
	if !strings.HasPrefix(name, protoPrefix) || supported[name] {
		return nil
	}
	return fmt.Errorf(
		"%s is not supported: foundry-tools ships Foundry Script for %s, and generating another google/protobuf file would produce types the runtime does not recognize",
		name, strings.Join(Files(), ", "),
	)
}

// Files lists the supported well-known files in stable order.
func Files() []string {
	names := make([]string, 0, len(supported))
	for name := range supported {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Source returns the vendored text of a supported well-known file.
func Source(filename string) ([]byte, error) {
	name := normalize(filename)
	if !supported[name] {
		return nil, fmt.Errorf("%s is not a vendored well-known file", name)
	}
	return protoFS.ReadFile(path.Join("proto", name))
}

func normalize(filename string) string {
	return path.Clean(strings.ReplaceAll(filename, "\\", "/"))
}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/proto/wellknown/ -v`
Expected: PASS

- [ ] **Step 6: Add the generation target**

The generator takes the namespace from the file's `package` unless a `namespace` option overrides it (`NamespaceFor`, `names.go:29`). The vendored files declare `package google.protobuf`, so generation must override the namespace to `foundry.proto.wkt`.

Write `internal/proto/wellknown/cmd/gen-wkt/main.go` — a small program in the same spirit as `cmd/gen-engine-reserved-types` — that for each file in `Files()`: parses the vendored source, sets the namespace option to `wellknown.Namespace`, runs `Generate`, and writes the result under `internal/runtime/data/foundry/proto/wkt/`. Follow how `internal/proto/command.go:52-68` drives `ParseFiles`, `Validate`, `Generate`, and `ImportsOf`.

Add to `Taskfile.yml` beside `gen-engine-types`:

```yaml
  gen-wkt:
    desc: Regenerate the checked-in well-known type bindings.
    cmds:
      - go run ./internal/proto/wellknown/cmd/gen-wkt
```

- [ ] **Step 7: Generate and inspect**

Run: `task gen-wkt`
Expected: `internal/runtime/data/foundry/proto/wkt/` populated.

Confirm by hand before trusting it:
- every file starts `namespace foundry.proto.wkt`
- `Value.pb.fs` exists alongside a `ValueKindCase` union file, matching the oneof convention
- `Struct`, `Value`, and `ListValue` reference each other without a namespace import cycle
- each of the nine wrappers is present

If mutual recursion or the `google.protobuf` package name produces a generator error, that is a real finding — stop and report rather than special-casing around it.

- [ ] **Step 8: Add the drift guard**

In `internal/runtime/runtime_test.go`:

```go
func TestWellKnownBindingsAreUpToDate(t *testing.T) {
	generated, err := wellknowngen.Generate()
	require.NoError(t, err)

	embedded := Files()
	for name, want := range generated {
		got, ok := embedded[name]
		require.True(t, ok, "%s is generated but not checked in; run `task gen-wkt`", name)
		require.Equal(t, want, got, "%s is stale; run `task gen-wkt`", name)
	}
}
```

Structure `gen-wkt` so the generation logic is an importable function returning `map[string]string`, with `main` only writing files. That is what makes this test possible.

- [ ] **Step 9: Verify in the engine**

Run: `task test && task foundry:test`
Expected: PASS, exit 0. Every `wkt/*.pb.fs` is picked up by `runtime.Files()` automatically — its embed pattern is `data/*/**/*.fs`. Confirm the new nesting depth still matches that pattern; if it does not, widen it and note the change.

- [ ] **Step 10: Commit**

```bash
git add internal/proto/wellknown/ internal/runtime/ Taskfile.yml
git commit -m "feat: ship generated well-known type bindings in the runtime"
```

---

## Task 3: Route well-known references to the runtime and stop generating them

**Goal:** A schema importing `google/protobuf/timestamp.proto` references `foundry.proto.wkt.Timestamp` and produces no `Timestamp` of its own.

**Files:**
- Modify: `internal/proto/internal/foundryscript/generator/plan.go:351`
- Modify: `internal/proto/command.go:57-68`
- Modify: `internal/plugin/plugin.go`
- Create: `examples/golden-wkt/` and its fixtures
- Test: `internal/proto/golden_test.go`, `internal/plugin/plugin_test.go`

**Acceptance Criteria:**
- [ ] No output is generated for a well-known file, whether requested directly or pulled in as a dependency
- [ ] A field typed `google.protobuf.Timestamp` emits `Timestamp?` plus `import foundry.proto.wkt`
- [ ] The same holds for a map value, a `repeated` element, and a `oneof` member
- [ ] `google/protobuf/descriptor.proto` fails with the `wellknown.Check` diagnostic
- [ ] Both the `anvil` and protoc/Buf paths behave identically

**Verify:** `task test` → PASS

**Steps:**

- [ ] **Step 1: Write the failing tests**

In `internal/proto/golden_test.go` (or a new `wellknown_integration_test.go` in the same package):

```go
func TestWellKnownReferenceResolvesToRuntimeNamespace(t *testing.T) {
	files := generateFixture(t, "examples/golden-wkt/event.proto")

	source := files["cafecito/game/v1/Event.pb.fs"]
	require.Contains(t, source, "import foundry.proto.wkt")
	require.Contains(t, source, "var occurred_at: Timestamp? = null")

	for name := range files {
		require.NotContains(t, name, "google/protobuf/",
			"well-known types must come from the runtime, not per-project generation")
	}
}

func TestUnsupportedWellKnownFileIsRejected(t *testing.T) {
	_, err := ParseFiles([]string{"google/protobuf/descriptor.proto"}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
}
```

Create the fixture `examples/golden-wkt/event.proto`, covering every position the reference can appear in:

```protobuf
syntax = "proto3";

package cafecito.game.v1;

import "google/protobuf/any.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";

message Event {
  google.protobuf.Timestamp occurred_at = 1;
  google.protobuf.Struct attributes = 2;
  repeated google.protobuf.Any attachments = 3;
  map<string, google.protobuf.Timestamp> checkpoints = 4;
  oneof detail {
    google.protobuf.Any payload = 5;
    string note = 6;
  }
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/proto/ -run 'WellKnown' -v`
Expected: FAIL — a `google/protobuf/Timestamp.pb.fs` is generated and the import is `google.protobuf`.

- [ ] **Step 3: Route the namespace**

In `plan.go`, inside `newResolver`'s import loop, replace line 351:

```go
		namespace := NamespaceFor(imports[i].File)
		// The well-known types ship as runtime source, so a reference to one
		// resolves to the runtime namespace rather than to a per-project
		// binding generated from google.protobuf.
		if wellknown.IsWellKnown(imports[i].Filename) {
			namespace = wellknown.Namespace
		}
```

Everything downstream — the emitted `import`, the qualified reference, collision detection — already keys off this namespace, so no other resolver change is needed. Verify that assumption holds for the map, repeated, and oneof cases rather than assuming it.

- [ ] **Step 4: Skip generation on the anvil path**

In `command.go`'s `RunE`, inside the `for _, parsed := range parsedFiles` loop, before validation:

```go
			if wellknown.IsWellKnown(parsed.Filename) {
				continue
			}
			if err := wellknown.Check(parsed.Filename); err != nil {
				return err
			}
```

Apply `wellknown.Check` in `ParseFiles` as well, so an unsupported file is rejected when it arrives as a dependency and not only when named directly.

- [ ] **Step 5: Skip generation on the plugin path**

Apply the same two guards in `internal/plugin/plugin.go` where it iterates the request's files to generate. The protoc/Buf path receives well-known files as dependencies routinely, so the skip matters more here than on the anvil path.

- [ ] **Step 6: Run to verify they pass**

Run: `go test ./internal/proto/ ./internal/plugin/ -v`
Expected: PASS

- [ ] **Step 7: Add the golden fixture and verify in the engine**

Wire `examples/golden-wkt/` into the golden test the same way `examples/golden/` is wired, and regenerate. Then:

Run: `task test && task foundry:test`
Expected: PASS, exit 0

- [ ] **Step 8: Commit**

```bash
git add internal/proto/ internal/plugin/ examples/golden-wkt/
git commit -m "feat: resolve well-known type references to the runtime namespace"
```

---

## Task 4: `Struct` / `Value` / `ListValue` ↔ `Variant`

**Goal:** A schema author converts between `Struct` and a Foundry `Dictionary` without hand-assembling `Value` cases.

**Files:**
- Modify: `internal/runtime/data/foundry/proto/protobuf_error.fs`
- Create: `internal/runtime/data/foundry/proto/wkt/struct_variant.fs`
- Test: `tests/foundry/wkt_test.fs`

**Acceptance Criteria:**
- [ ] `Value` → `Variant` is total across all six cases
- [ ] `Variant` → `Value` accepts `bool`, `int`, `float`, `String`, `Array`, `Dictionary`, and `null`
- [ ] `int` narrows to `number_value`; a round trip returns `float`
- [ ] A `Dictionary` with a non-`String` key returns `STRUCT_KEY_NOT_STRING`
- [ ] An unsupported Variant type (`Vector2`) returns `STRUCT_VALUE_UNREPRESENTABLE`
- [ ] A failure anywhere in a nested tree aborts the whole conversion; no partial tree is returned
- [ ] Nested `Dictionary` and `Array` convert recursively

**Verify:** `task foundry:test` → exits 0

**Steps:**

- [ ] **Step 1: Add the error cases**

`protobuf_error.fs` — append only, so existing numbers are preserved:

```foundryscript
namespace foundry.proto

enum_name ProtobufError:
	OK = 0
	VARINT_NOT_FOUND = 1
	VARINT_TOO_LONG = 2
	WIRE_TYPE_MISMATCH = 3
	LENGTH_DELIMITED_SIZE_NOT_FOUND = 4
	LENGTH_DELIMITED_SIZE_MISMATCH = 5
	UNKNOWN_REQUIRED_FEATURE = 6
	STRUCT_KEY_NOT_STRING = 7
	STRUCT_VALUE_UNREPRESENTABLE = 8
	ANY_TYPE_MISMATCH = 9
```

- [ ] **Step 2: Write the failing engine test**

Create `tests/foundry/wkt_test.fs`, following the assertion style in `tests/foundry/main.fs`:

```foundryscript
namespace foundry.proto.tests
import foundry.proto
import foundry.proto.wkt

static func test_struct_round_trip() -> void:
	var source: Dictionary = {
		"name": "player",
		"level": 5,
		"ratio": 0.5,
		"active": true,
		"missing": null,
		"tags": ["a", "b"],
		"nested": {"inner": 1.5},
	}
	var converted: (Struct?, ProtobufError) = Struct.from_dictionary(source)
	assert(converted[1] == ProtobufError.OK, "conversion succeeds")
	var round_tripped: Dictionary = converted[0].to_dictionary()
	assert(round_tripped["name"] == "player", "string survives")
	assert(round_tripped["level"] == 5.0, "int narrows to float")
	assert(round_tripped["active"] == true, "bool survives")
	assert(round_tripped["missing"] == null, "null survives")
	assert(round_tripped["tags"].size() == 2, "array survives")
	assert(round_tripped["nested"]["inner"] == 1.5, "nested struct survives")

static func test_struct_rejects_non_string_key() -> void:
	var converted: (Struct?, ProtobufError) = Struct.from_dictionary({1: "a"})
	assert(converted[1] == ProtobufError.STRUCT_KEY_NOT_STRING, "non-string key rejected")
	assert(converted[0] == null, "no partial tree returned")

static func test_struct_rejects_unrepresentable_value() -> void:
	var converted: (Struct?, ProtobufError) = Struct.from_dictionary({"at": Vector2(1, 2)})
	assert(converted[1] == ProtobufError.STRUCT_VALUE_UNREPRESENTABLE, "Vector2 rejected")
	assert(converted[0] == null, "no partial tree returned")

static func test_struct_rejects_nested_unrepresentable_value() -> void:
	var converted: (Struct?, ProtobufError) = Struct.from_dictionary({"a": {"b": [Vector2(1, 2)]}})
	assert(converted[1] == ProtobufError.STRUCT_VALUE_UNREPRESENTABLE, "nested failure aborts")
	assert(converted[0] == null, "no partial tree returned")
```

Register these in `tests/foundry/run.sh` the way the existing checks are registered.

- [ ] **Step 3: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — `from_dictionary` is not defined.

- [ ] **Step 4: Write the conformance**

Create `internal/runtime/data/foundry/proto/wkt/struct_variant.fs`. Read the generated `Value.pb.fs` and its union file first — the exact case names and the union member spelling come from what the generator emitted in Task 2, and this code must match them.

The names below assume the generator emitted the `Value` oneof as a union named `ValueKindCase` with cases `NULL_VALUE`, `NUMBER_VALUE`, `STRING_VALUE`, `BOOL_VALUE`, `STRUCT_VALUE`, `LIST_VALUE`, assigned to a member named `kind`. **Read the generated `Value.pb.fs` and its union file from Task 2 and correct these spellings before running anything** — the union case convention comes from #16 and the member-escaping rules from #32/#34.

```foundryscript
namespace foundry.proto.wkt
import foundry.proto

## Conversion between protobuf's dynamic JSON value type and a Foundry value.
trait_name ValueConvertible

abstract func to_variant() -> Variant

abstract static func from_variant(source: Variant) -> (Value?, ProtobufError)

extend Value uses ValueConvertible:
	## Returns this value as the closest Foundry representation. Total: every
	## protobuf Value case has one.
	func to_variant() -> Variant:
		match kind:
			ValueKindCase.NULL_VALUE(_):
				return null
			ValueKindCase.NUMBER_VALUE(number):
				return number
			ValueKindCase.STRING_VALUE(text):
				return text
			ValueKindCase.BOOL_VALUE(flag):
				return flag
			ValueKindCase.STRUCT_VALUE(nested):
				return nested.to_dictionary()
			ValueKindCase.LIST_VALUE(items):
				return items.to_array()
			_:
				return null

	## Converts a Foundry value into a protobuf Value.
	##
	## int narrows to number_value, because protobuf's Value is a double. The
	## narrowing is lossy past 2^53 and an int round-trips back as a float.
	## A non-string Dictionary key or a Variant type with no JSON equivalent is
	## an error rather than a coercion: silently dropping a Vector2 would
	## surface as a missing field far from its cause.
	static func from_variant(source: Variant) -> (Value?, ProtobufError):
		var result: Value = Value.new()
		if source == null:
			result.kind = ValueKindCase.NULL_VALUE(NullValue.NULL_VALUE)
			return (result, ProtobufError.OK)
		if source is bool:
			result.kind = ValueKindCase.BOOL_VALUE(source as bool)
			return (result, ProtobufError.OK)
		if source is int:
			result.kind = ValueKindCase.NUMBER_VALUE(float(source as int))
			return (result, ProtobufError.OK)
		if source is float:
			result.kind = ValueKindCase.NUMBER_VALUE(source as float)
			return (result, ProtobufError.OK)
		if source is String:
			result.kind = ValueKindCase.STRING_VALUE(source as String)
			return (result, ProtobufError.OK)
		if source is Dictionary:
			var nested: (Struct?, ProtobufError) = Struct.from_dictionary(source as Dictionary)
			if nested[1] != ProtobufError.OK:
				return (null, nested[1])
			result.kind = ValueKindCase.STRUCT_VALUE(nested[0])
			return (result, ProtobufError.OK)
		if source is Array:
			var items: (ListValue?, ProtobufError) = ListValue.from_array(source as Array)
			if items[1] != ProtobufError.OK:
				return (null, items[1])
			result.kind = ValueKindCase.LIST_VALUE(items[0])
			return (result, ProtobufError.OK)
		return (null, ProtobufError.STRUCT_VALUE_UNREPRESENTABLE)

## Conversion between a protobuf Struct and a Foundry Dictionary.
trait_name StructConvertible

abstract func to_dictionary() -> Dictionary

abstract static func from_dictionary(source: Dictionary) -> (Struct?, ProtobufError)

extend Struct uses StructConvertible:
	func to_dictionary() -> Dictionary:
		var result: Dictionary = {}
		for key in fields:
			result[key] = fields[key].to_variant()
		return result

	static func from_dictionary(source: Dictionary) -> (Struct?, ProtobufError):
		var result: Struct = Struct.new()
		for key in source:
			if not (key is String):
				return (null, ProtobufError.STRUCT_KEY_NOT_STRING)
			var converted: (Value?, ProtobufError) = Value.from_variant(source[key])
			if converted[1] != ProtobufError.OK:
				return (null, converted[1])
			result.fields[key as String] = converted[0]
		return (result, ProtobufError.OK)

## Conversion between a protobuf ListValue and a Foundry Array.
trait_name ListValueConvertible

abstract func to_array() -> Array

abstract static func from_array(source: Array) -> (ListValue?, ProtobufError)

extend ListValue uses ListValueConvertible:
	func to_array() -> Array:
		var result: Array = []
		for item in values:
			result.append(item.to_variant())
		return result

	static func from_array(source: Array) -> (ListValue?, ProtobufError):
		var result: ListValue = ListValue.new()
		for item in source:
			var converted: (Value?, ProtobufError) = Value.from_variant(item)
			if converted[1] != ProtobufError.OK:
				return (null, converted[1])
			result.values.append(converted[0])
		return (result, ProtobufError.OK)
```

Three traits rather than one, because witness method names must be unique per target and these are three distinct targets.

Errors propagate rather than accumulate: the first failure returns `(null, error)` and unwinds, so no partial tree escapes.

- [ ] **Step 5: Run to verify it passes**

Run: `task foundry:test`
Expected: exits 0

- [ ] **Step 6: Commit**

```bash
git add internal/runtime/data/foundry/proto/ tests/foundry/
git commit -m "feat: convert Struct, Value, and ListValue to and from Foundry values"
```

---

## Task 5: `Timestamp` / `Duration` ↔ `float` seconds

**Goal:** A schema author converts a `Timestamp` to the unix-seconds float the engine's `Time` API speaks.

**Files:**
- Create: `internal/runtime/data/foundry/proto/wkt/time_conversion.fs`
- Test: `tests/foundry/wkt_test.fs`

**Acceptance Criteria:**
- [ ] `Timestamp.from_unix_time(float)` populates `seconds` and `nanos`
- [ ] `Timestamp.now()` returns a timestamp near `Time.get_unix_time_from_system()`
- [ ] `timestamp.to_unix_time()` returns the float
- [ ] `Duration.from_seconds` / `to_seconds` round-trip, including negative values
- [ ] A negative `Duration` keeps `seconds` and `nanos` the same sign, per the protobuf spec
- [ ] Sub-second values survive to the documented ~238 ns resolution

**Verify:** `task foundry:test` → exits 0

**Steps:**

- [ ] **Step 1: Write the failing engine test**

Append to `tests/foundry/wkt_test.fs`:

```foundryscript
static func test_timestamp_round_trip() -> void:
	var timestamp: Timestamp = Timestamp.from_unix_time(1700000000.5)
	assert(timestamp.seconds == 1700000000, "seconds extracted")
	assert(timestamp.nanos > 499000000 and timestamp.nanos < 501000000, "nanos extracted")
	assert(absf(timestamp.to_unix_time() - 1700000000.5) < 0.000001, "round trip")

static func test_timestamp_now() -> void:
	var before: float = Time.get_unix_time_from_system()
	var timestamp: Timestamp = Timestamp.now()
	assert(absf(timestamp.to_unix_time() - before) < 5.0, "now is near system time")

static func test_duration_round_trip() -> void:
	var duration: Duration = Duration.from_seconds(1.25)
	assert(duration.seconds == 1, "seconds extracted")
	assert(absf(duration.to_seconds() - 1.25) < 0.000001, "round trip")

static func test_negative_duration_signs_agree() -> void:
	var duration: Duration = Duration.from_seconds(-1.25)
	assert(duration.seconds <= 0 and duration.nanos <= 0, "signs agree per spec")
	assert(absf(duration.to_seconds() + 1.25) < 0.000001, "round trip")
```

- [ ] **Step 2: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — `from_unix_time` is not defined.

- [ ] **Step 3: Write the conformance**

Create `internal/runtime/data/foundry/proto/wkt/time_conversion.fs`:

```foundryscript
namespace foundry.proto.wkt
import foundry.proto

## Conversion between a protobuf Timestamp and unix seconds.
##
## seconds and nanos remain the source of truth. These helpers are a
## convenience: Foundry has no dedicated time type, and a float near 1.7e9
## resolves to roughly 238 ns, so converting out loses precision. Converting in
## does not -- seconds plus nanos is strictly more precise than the float.
trait_name UnixTimeConvertible

abstract func to_unix_time() -> float

abstract static func from_unix_time(seconds: float) -> Timestamp

abstract static func now() -> Timestamp

extend Timestamp uses UnixTimeConvertible:
	func to_unix_time() -> float:
		return float(seconds) + float(nanos) / 1000000000.0

	static func from_unix_time(unix_seconds: float) -> Timestamp:
		var timestamp: Timestamp = Timestamp.new()
		var whole: int = int(floorf(unix_seconds))
		timestamp.seconds = whole
		timestamp.nanos = int(roundf((unix_seconds - float(whole)) * 1000000000.0))
		# Rounding can carry into the next second.
		if timestamp.nanos >= 1000000000:
			timestamp.nanos -= 1000000000
			timestamp.seconds += 1
		return timestamp

	static func now() -> Timestamp:
		return Timestamp.from_unix_time(Time.get_unix_time_from_system())

## Conversion between a protobuf Duration and seconds.
trait_name SecondsConvertible

abstract func to_seconds() -> float

abstract static func from_seconds(seconds: float) -> Duration

extend Duration uses SecondsConvertible:
	func to_seconds() -> float:
		return float(seconds) + float(nanos) / 1000000000.0

	## The protobuf spec requires seconds and nanos to carry the same sign, so
	## this truncates toward zero rather than flooring.
	static func from_seconds(total_seconds: float) -> Duration:
		var duration: Duration = Duration.new()
		var whole: int = int(total_seconds)
		duration.seconds = whole
		duration.nanos = int(roundf((total_seconds - float(whole)) * 1000000000.0))
		if duration.nanos >= 1000000000:
			duration.nanos -= 1000000000
			duration.seconds += 1
		elif duration.nanos <= -1000000000:
			duration.nanos += 1000000000
			duration.seconds -= 1
		return duration
```

Confirm `Time`, `floorf`, `roundf`, and `absf` are reachable from runtime script scope; if `Time` needs an import in this context, add it.

- [ ] **Step 4: Run to verify it passes**

Run: `task foundry:test`
Expected: exits 0

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/data/foundry/proto/wkt/time_conversion.fs tests/foundry/
git commit -m "feat: convert Timestamp and Duration to and from seconds"
```

---

## Task 6: `Any` pack and typed unpack

**Goal:** A schema author packs a message into `Any` and unpacks it into a named type.

**Files:**
- Create: `internal/runtime/data/foundry/proto/wkt/any_packing.fs`
- Test: `tests/foundry/wkt_test.fs`

**Acceptance Criteria:**
- [ ] `Any.pack(message)` writes `type.googleapis.com/<type_name>` and the encoded bytes
- [ ] `any.unpack_into(target)` decodes on a matching type and returns `OK`
- [ ] A mismatched type returns `ANY_TYPE_MISMATCH` and leaves the target untouched
- [ ] A `type_url` with a different prefix still unpacks — only the segment after the last `/` is compared
- [ ] `any.is_type(message)` answers without decoding
- [ ] No dynamic unpack path exists

**Verify:** `task foundry:test` → exits 0

**Steps:**

- [ ] **Step 1: Write the failing engine test**

Append to `tests/foundry/wkt_test.fs`, using a message the harness already generates:

```foundryscript
static func test_any_round_trip() -> void:
	var slot: Slot = Slot.new()
	slot.label = "sword"
	slot.quantity = 3

	var packed: Any = Any.pack(slot)
	assert(packed.type_url == "type.googleapis.com/cafecito.game.v1.Slot", "type_url written")

	var decoded: Slot = Slot.new()
	assert(packed.unpack_into(decoded) == ProtobufError.OK, "unpack succeeds")
	assert(decoded.label == "sword", "payload survives")
	assert(decoded.quantity == 3, "payload survives")

static func test_any_rejects_mismatched_type() -> void:
	var slot: Slot = Slot.new()
	slot.label = "sword"
	var packed: Any = Any.pack(slot)

	var wrong: Player = Player.new()
	wrong.name = "untouched"
	assert(packed.unpack_into(wrong) == ProtobufError.ANY_TYPE_MISMATCH, "mismatch rejected")
	assert(wrong.name == "untouched", "target untouched on mismatch")
	assert(packed.is_type(wrong) == false, "is_type agrees")

static func test_any_accepts_foreign_prefix() -> void:
	var slot: Slot = Slot.new()
	slot.label = "sword"
	var packed: Any = Any.pack(slot)
	# The prefix is not part of the identity; a peer may use any host.
	packed.type_url = "example.com/some/path/cafecito.game.v1.Slot"

	var decoded: Slot = Slot.new()
	assert(packed.unpack_into(decoded) == ProtobufError.OK, "foreign prefix unpacks")
	assert(decoded.label == "sword", "payload survives")
```

- [ ] **Step 2: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — `Any.pack` is not defined.

- [ ] **Step 3: Write the conformance**

Create `internal/runtime/data/foundry/proto/wkt/any_packing.fs`:

```foundryscript
namespace foundry.proto.wkt
import foundry.proto

## Packing a message into a protobuf Any and back out into a named type.
##
## Unpacking requires the caller to name the target type. There is no registry
## mapping a type URL to a binding, so an Any holding an unnamed type stays as
## opaque bytes.
trait_name AnyPacking

abstract static func pack(message: Message) -> Any

abstract func unpack_into(target: Message) -> ProtobufError

abstract func is_type(target: Message) -> bool

extend Any uses AnyPacking:
	static func pack(message: Message) -> Any:
		var packed: Any = Any.new()
		packed.type_url = "type.googleapis.com/" + message.type_name()
		packed.value = message.to_bytes()
		return packed

	func unpack_into(target: Message) -> ProtobufError:
		if not is_type(target):
			return ProtobufError.ANY_TYPE_MISMATCH
		return target.merge_from_bytes(value)

	func is_type(target: Message) -> bool:
		return _type_name_of(type_url) == target.type_name()

## The protobuf spec defines the meaningful portion of a type URL as the
## substring after the last slash; the prefix is arbitrary. Comparing the whole
## string would fail against any peer that used a different host, and that
## failure would only appear in cross-implementation traffic.
static func _type_name_of(url: String) -> String:
	var separator: int = url.rfind("/")
	if separator < 0:
		return url
	return url.substr(separator + 1)
```

Check whether a bare `static func` at file scope is legal here, or whether `_type_name_of` must be a member; if the latter, move it onto the conformance body as a `static` witness of its own trait, since witness names must not collide.

- [ ] **Step 4: Run to verify it passes**

Run: `task foundry:test`
Expected: exits 0

- [ ] **Step 5: Commit**

```bash
git add internal/runtime/data/foundry/proto/wkt/any_packing.fs tests/foundry/
git commit -m "feat: pack and unpack a message through a protobuf Any"
```

---

## Task 7: Document the mapping and file the JSON follow-up

**Goal:** The README reflects the new behavior, and the deferred JSON question is captured.

**Files:**
- Modify: `README.md`

**Acceptance Criteria:**
- [ ] The README's mapping table gains a well-known types row
- [ ] The conversion helpers and their lossy direction are documented
- [ ] The non-goals are stated so they are not re-litigated
- [ ] A JSON follow-up issue exists

**Verify:** `task ci` → PASS

**Steps:**

- [ ] **Step 1: Update the README**

Add to the mapping table, after the imported-type row:

```markdown
| `google.protobuf.*` | `foundry.proto.wkt.X` from the runtime, never generated |
```

And a section after the enum discussion:

```markdown
## Well-known types

The `google/protobuf` well-known types ship as Foundry Script in the runtime
under `foundry.proto.wkt` rather than being generated per project, so every
project shares one `Timestamp` and a library can hand one to its consumer.
Importing `google/protobuf/timestamp.proto` produces no output of its own and
emits `import foundry.proto.wkt`.

Three of them carry semantics beyond a wire round-trip:

- `Struct`, `Value`, and `ListValue` convert to and from Foundry values with
  `from_dictionary` / `to_dictionary`, `from_variant` / `to_variant`, and
  `from_array` / `to_array`. The mapping is the one protobuf JSON defines.
  Converting out always succeeds. Converting in narrows `int` to a float, as
  every JSON implementation does, and rejects a non-string `Dictionary` key or
  a Variant type with no JSON equivalent rather than coercing it.
- `Timestamp` and `Duration` convert to and from `float` seconds, plus
  `Timestamp.now()`. `seconds` and `nanos` stay the source of truth: Foundry
  has no dedicated time type, and a float near 1.7e9 resolves to roughly
  238 ns, so converting out loses precision. Converting in does not.
- `Any` packs and unpacks against a type the caller names —
  `Any.pack(message)` and `any.unpack_into(target)`. There is no registry
  mapping a type URL to a binding, so an `Any` holding an unnamed type stays as
  opaque bytes.

`Empty`, `FieldMask`, and the scalar wrappers are ordinary messages. The
wrappers are not mapped onto nullable scalars — proto3 `optional` already gives
a scalar explicit presence, and the message form is needed anyway wherever a
wrapper appears as a map value, a repeated element, or a oneof member.

There is no JSON support, so `FieldMask` path conversion and RFC-3339
`Timestamp` strings are not available.
```

- [ ] **Step 2: File the follow-up issue**

```bash
gh issue create --repo cafecito-games/Foundry-Tools \
  --title "Decide whether the generator supports proto3 canonical JSON" \
  --label enhancement \
  --body "$(cat <<'BODY'
The well-known types are defined largely by their JSON representation, and we
have no JSON support at all. #39 deliberately deferred this rather than building
the individual conversions piecemeal.

Everything below hangs off this one question:

- `FieldMask` camelCase path conversion (`foo_bar` <-> `fooBar`), which only
  matters when a mask is serialized to JSON.
- `Timestamp` and `Duration` as RFC-3339 strings.
- `Any`'s JSON form, with `@type` alongside the payload's fields.
- `Struct` as plain JSON, which is most of what makes it worth having.

#39 does not foreclose it. The `Value` <-> `Variant` mapping it specifies *is*
the protobuf JSON value mapping, so `Struct` JSON support would be close to
free, and `Timestamp`'s seconds/nanos conversion is the harder half of RFC-3339
already done.

A further prerequisite, distinct from JSON: applying a `FieldMask` to a message
-- and mask union/intersection -- needs runtime reflection over generated
messages (field names, nested traversal, clearing by path). The generator emits
none of that today. Worth scoping separately if masks turn out to matter.

Follows from #39.
BODY
)"
```

- [ ] **Step 3: Run the full suite**

Run: `task ci && task foundry:test`
Expected: PASS, exit 0

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: document well-known type mapping and conversions"
```

---

## Notes For The Implementer

**The generated files are not yours to edit.** Anything under `internal/runtime/data/foundry/proto/wkt/*.pb.fs` comes from `task gen-wkt`. If one needs to change, change the generator or the vendored proto and regenerate. The drift guard in Task 2 will catch a hand-edit.

**Match the generated spelling exactly.** Tasks 4–6 reference generated members (`seconds`, `nanos`, `type_url`, `value`, the `Value` oneof union cases). Read the actual generated output from Task 2 before writing them — the member-escaping rules from #32 and #34 mean a name is not always the obvious one. `value` in particular is both a `Value` oneof member and an `Any` field, and may have been escaped.

**Two entry paths, one behavior.** Every change in Task 3 must land on both `internal/proto/command.go` (anvil) and `internal/plugin/plugin.go` (protoc/Buf). A test that only covers one will pass while half the feature is missing.

**Task 0 gates Tasks 4–6.** If retroactive conformance does not work as specified, those three tasks change shape — the semantics move into hand-written classes per the original spec, and Task 2's "generate them" strategy has to be reconsidered along with it. Do not start Task 4 before Task 0 has reported.
