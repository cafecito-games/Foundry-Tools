# Upstream Proto3 Engine Conformance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate and lint the pinned upstream proto3 conformance bindings in the real Foundry project, then engine-round-trip the upstream constructs not already covered by hand-written fixtures.

**Architecture:** Extend the existing ephemeral generation command so the untouched conformance schema and its seven pinned WKT dependencies produce the real project bindings consumed by the project-wide lint. Run a separate `conformance.fs` entry point after the existing `main.fs`; one shallow message graph checks recursion, enum edges, a default-valued WKT oneof, and one representative unsigned map without duplicating the scalar corpus.

**Tech Stack:** Go 1.26 CLI generation, Bash, Foundry Script, Foundry Engine 0.1.alpha19 or newer, Task.

**Spec:** `docs/superpowers/specs/2026-08-02-upstream-proto3-engine-conformance-design.md`. Issue #28; fixture provenance and refresh remain in `tests/integration/fixtures/conformance/README.md`.

---

## File Structure

| Path | Responsibility |
|---|---|
| `tests/foundry/conformance.fs` | New focused engine entry point for upstream-only runtime claims. |
| `tests/foundry/run.sh` | Generate the pinned schema into the ephemeral project and apply exit/log error detection to both runtime entry points. |

The pinned files under `tests/integration/fixtures/conformance/` are inputs only and must not change.

## Task 1: Introduce the focused failing engine gate

**Files:**
- Create: `tests/foundry/conformance.fs`
- Modify: `tests/foundry/run.sh`

- [ ] **Step 1: Create the focused runtime entry point**

Create `tests/foundry/conformance.fs` exactly as follows:

```foundryscript
import foundry.proto
import foundry.proto.wkt
import protobuf_test_messages.proto3

extends SceneTree

var failures: int = 0

func check(condition: bool, label: String) -> void:
	if not condition:
		printerr("FAIL: ", label)
		failures += 1

func _init() -> void:
	var direct: TestAllTypesProto3 = TestAllTypesProto3.new()
	direct.optional_string = "direct child"

	var mutual: TestAllTypesProto3 = TestAllTypesProto3.new()
	mutual.optional_string = "mutual child"
	var nested: TestAllTypesProto3.NestedMessage = TestAllTypesProto3.NestedMessage.new()
	nested.a = 7
	nested.corecursive = mutual

	var suite: TestAllTypesProto3 = TestAllTypesProto3.new()
	suite.recursive_message = direct
	suite.optional_nested_message = nested
	suite.optional_nested_enum = TestAllTypesProto3.NestedEnum.NEG
	suite.optional_aliased_enum = TestAllTypesProto3.AliasedEnum.MOO
	suite.map_uint64_uint64 = {-1: -1}
	suite.oneof_field = TestAllTypesProto3OneofFieldCase.OneofNullValue(NullValue.NULL_VALUE)

	var (decoded, decode_error) = TestAllTypesProto3.from_bytes(suite.to_bytes())
	check(decode_error == ProtobufError.OK, "upstream conformance message decodes")
	if not (decoded is TestAllTypesProto3):
		printerr("FAIL: upstream conformance message was null")
		quit(1)
		return

	check(decoded.recursive_message is TestAllTypesProto3,
		"a directly recursive message edge round trips")
	if decoded.recursive_message is TestAllTypesProto3:
		check(decoded.recursive_message.optional_string == "direct child",
			"the directly recursive child keeps its fields")

	check(decoded.optional_nested_message is TestAllTypesProto3.NestedMessage,
		"the nested message round trips")
	if decoded.optional_nested_message is TestAllTypesProto3.NestedMessage:
		check(decoded.optional_nested_message.a == 7, "the nested message keeps its fields")
		check(decoded.optional_nested_message.corecursive is TestAllTypesProto3,
			"a nested message can recurse to its parent type")
		if decoded.optional_nested_message.corecursive is TestAllTypesProto3:
			check(decoded.optional_nested_message.corecursive.optional_string == "mutual child",
				"the mutually recursive child keeps its fields")

	check(decoded.optional_nested_enum == TestAllTypesProto3.NestedEnum.NEG,
		"a negative enum value round trips")
	check(decoded.optional_aliased_enum == TestAllTypesProto3.AliasedEnum.MOO,
		"an aliased enum keeps its numeric value")
	check(decoded.map_uint64_uint64 == {-1: -1},
		"the widest uint64 map key and value round trip")

	match decoded.oneof_field:
		TestAllTypesProto3OneofFieldCase.OneofNullValue(var null_value):
			check(null_value == NullValue.NULL_VALUE,
				"a default-valued NullValue oneof remains present")
		_:
			printerr("FAIL: the NullValue oneof case did not round trip")
			failures += 1

	if failures > 0:
		printerr("conformance round trip failed with ", failures, " error(s)")
		quit(1)
		return
	print("conformance round trip ok")
	quit(0)
```

This graph terminates after one direct child and one nested-parent child. Do not point either child back to `suite`.

- [ ] **Step 2: Make `run.sh` execute both runtime entry points**

In `tests/foundry/run.sh`, replace the single `main.fs` execution block with a small helper. Leave generation unchanged in this red step:

```bash
run_fixture() {
  local script="$1"
  local label="$2"
  if ! "$FOUNDRY" --headless project run --project "$PROJECT" --script "$script" 2>&1 | tee -a "$RUN_LOG"; then
    echo "$label failed its round-trip checks"
    exit 1
  fi
}

run_fixture "$PROJECT/main.fs" "generated Foundry Script"
run_fixture "$PROJECT/conformance.fs" "upstream conformance binding"

if grep -q "SCRIPT ERROR" "$RUN_LOG"; then
  echo "generated Foundry Script emitted a SCRIPT ERROR during a round-trip run"
  exit 1
fi
```

Keep the existing explanation immediately above this block about why exit status alone is insufficient.

- [ ] **Step 3: Run the focused gate and record the expected red result**

Run:

```bash
task foundry:test
```

Expected: FAIL during project import/lint because `protobuf_test_messages.proto3` and `TestAllTypesProto3` are unresolved. This proves the engine gate depends on generating the real upstream binding. Record the diagnostic in the handoff; do not weaken lint or remove the import.

## Task 2: Generate the untouched upstream bindings and turn the gate green

**Files:**
- Modify: `tests/foundry/run.sh`
- Test: `tests/foundry/conformance.fs`

- [ ] **Step 1: Add the conformance include root and all pinned inputs**

In the first `anvil proto generate` invocation in `tests/foundry/run.sh`, add this include path:

```bash
  -I "$ROOT/tests/integration/fixtures/conformance" \
```

Then append these inputs after the existing Foundry fixture protos:

```bash
  "$ROOT/tests/integration/fixtures/conformance/test_messages_proto3.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/any.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/duration.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/empty.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/field_mask.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/struct.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/timestamp.proto" \
  "$ROOT/tests/integration/fixtures/conformance/google/protobuf/wrappers.proto"
```

Do not copy, filter, patch, or regenerate any source under `tests/integration/fixtures/conformance/`. The generator routes the vendored WKT inputs onto the runtime's single `foundry.proto.wkt` binding set, so `$OUT/google` must remain absent and the existing assertion enforces it.

- [ ] **Step 2: Run the engine gate and verify the green result**

Run:

```bash
task foundry:test
```

Expected: PASS. Foundry lint reports zero diagnostics over `res://`; output includes both `round trip ok` and `conformance round trip ok`; no `SCRIPT ERROR` appears.

- [ ] **Step 3: Run the focused integration generation test**

Run:

```bash
go test -tags=integration ./tests/integration -run '^TestConformanceSchemaGenerates$' -count=1
```

Expected: PASS. This independently keeps the unmodified-schema generator gate green.

- [ ] **Step 4: Confirm the pinned upstream inputs are untouched**

Run:

```bash
git diff --exit-code origin/main -- tests/integration/fixtures/conformance
```

Expected: no output and exit 0.

- [ ] **Step 5: Commit the red-green implementation**

```bash
git add tests/foundry/conformance.fs tests/foundry/run.sh
git commit -m "test: engine-verify upstream proto3 conformance bindings"
```

## Task 3: Full verification and self-review

**Files:**
- Review: `tests/foundry/conformance.fs`
- Review: `tests/foundry/run.sh`
- Verify only: all repository packages and integration fixtures

- [ ] **Step 1: Run local CI without Foundry**

Run:

```bash
task ci
```

Expected: PASS, including formatting, tidiness, lint, race-enabled unit tests, and builds. Avoid running another `golangci-lint` concurrently.

- [ ] **Step 2: Run all integration tests**

Run:

```bash
task integration
```

Expected: PASS.

- [ ] **Step 3: Re-run the Foundry gate from a clean generated project**

Run:

```bash
task foundry:test
```

Expected: PASS with zero lint diagnostics, `round trip ok`, and `conformance round trip ok`. The cleanup trap leaves no `tests/foundry/generated`, `tests/foundry/.foundry`, or `tests/foundry/*.uid` artifacts.

- [ ] **Step 4: Review the final branch against issue #28**

Run:

```bash
git diff --check origin/main...HEAD
git diff --stat origin/main...HEAD
git status --short
find tests/foundry -maxdepth 1 \( -name '*.uid' -o -name '.foundry' -o -name 'generated' \) -print
```

Expected: no whitespace errors; only the approved design, plan, focused runner, and harness changes appear; status is clean; the artifact search prints nothing. Confirm by inspection that runtime assertions are limited to the approved claims and the upstream fixture files have no diff.
