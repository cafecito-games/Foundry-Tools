# Inherited Engine Member Collision Escaping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Escape generated message members that collide with exact-case symbols inherited from the generated `RefCounted` base class.

**Architecture:** Extend the existing `extension_api.json` metadata generator to traverse only the generated base ancestry and emit a categorized, nearest-owner member map beside the type map. Feed that map into the existing member-name planner so fields and oneof groups reuse the one-trailing-underscore policy and secondary-collision diagnostics.

**Tech Stack:** Go 1.26, Foundry `extension_api.json`, protobuf generator AST tests, Foundry Script lint and runtime fixtures, Task.

---

### Task 1: Pin the metadata and analyzer regressions

**Files:**
- Modify: `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/testdata/extension_api.json`
- Modify: `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/names_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/member_collisions_test.go`
- Modify: `tests/foundry/collisions.proto`
- Modify: `tests/foundry/main.fs`
- Modify: `tests/foundry/run.sh`

- [x] **Step 1: Extend the compact API fixture with `RefCounted → Object`**

Add methods on both classes plus Object property, signal, constant, enum, and enum values. Include an overridden method so the test can prove nearest-owner metadata.

- [x] **Step 2: Add failing metadata-generator tests**

Assert `loadAPI` returns sorted entries such as:

```go
InheritedMembers: []inheritedMember{
    {Name: "CONNECT_DEFERRED", Kind: inheritedMemberEnumValue, Owner: "Object"},
    {Name: "ConnectFlags", Kind: inheritedMemberEnum, Owner: "Object"},
    {Name: "reference", Kind: inheritedMemberMethod, Owner: "RefCounted"},
},
```

Add focused decoder tests for a missing `RefCounted`, a missing ancestor, and an inheritance cycle. Extend the deterministic renderer assertion to require `foundryEngineReservedMembers` entries containing the symbol kind and nearest owner.

- [x] **Step 3: Add failing fsgenerator naming tests**

Table-test representative inherited spellings (`reference`, `get_class`, `script_changed`, `NOTIFICATION_PREDELETE`, `ConnectFlags`, `CONNECT_DEFERRED`) and assert each receives one underscore. Assert `Reference` remains unchanged and `planOneofAlternativeName("reference")` plus `EnumValueName("reference")` remain unchanged.

- [x] **Step 4: Add a failing secondary-collision diagnostic test**

Generate a message containing `reference` and `reference_`; assert the existing aggregate reports both claims and describes the first as escaping inherited method `RefCounted.reference`.

- [x] **Step 5: Add the Foundry fixture and warning gate**

Add representative reachable-base field names to `GameNode`, assign and round-trip their underscored spellings in `main.fs`, and change the project lint threshold from `--fail-on=error` to `--fail-on=warning`.

- [x] **Step 6: Run the focused tests and Foundry fixture to capture RED**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types \
  ./internal/proto/internal/foundryscript/generator -count=1
task foundry:test
```

Expected: Go compilation/test failures because inherited metadata and escaping do not exist, and Foundry lint fails with `SHADOWED_VARIABLE_BASE_CLASS` for `reference`/other inherited spellings.

### Task 2: Generate reachable inherited-member metadata

**Files:**
- Modify: `internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main.go`
- Regenerate: `internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go`

- [x] **Step 1: Decode the class symbol surface**

Add JSON structures for class inheritance, methods, properties, signals, constants, enums, and enum values. Add categorized `inheritedMember` records to `reservedTypes`.

- [x] **Step 2: Traverse the generated base ancestry**

Implement a helper starting from `RefCounted`, visiting the most-derived class first, retaining the first owner of duplicate spellings, and rejecting missing classes or cycles. Collect categories in deterministic order, then sort the final entries by name.

- [x] **Step 3: Render the member metadata**

Emit `engineMemberKind`, `engineMemberEntry`, and `foundryEngineReservedMembers`, including category and owner on each entry. Preserve the existing version and type map output.

- [x] **Step 4: Run generator-command tests GREEN**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types -count=1
```

Expected: all command tests pass.

- [x] **Step 5: Regenerate from the installed Foundry API**

Run:

```bash
task gen-engine-types
```

Expected: `engine_reserved_types.gen.go` refreshes at the installed Foundry version and includes only symbols reachable through `RefCounted → Object`.

### Task 3: Apply inherited metadata to member naming

**Files:**
- Modify: `internal/proto/internal/foundryscript/generator/names.go`

- [x] **Step 1: Add an inherited-engine escape reason**

Extend `memberEscapeKind` and `memberEscape.description()` so diagnostics identify the generated entry's category, owner, and raw spelling.

- [x] **Step 2: Reserve inherited spellings in `planMemberName`**

After existing non-engine and engine-type handling, consult `foundryEngineReservedMembers` and return the same `name + "_"` mapping. Do not change `planOneofAlternativeName` or `planEnumValueName`.

- [x] **Step 3: Run focused fsgenerator tests GREEN**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator -count=1
```

Expected: naming, generation, and secondary-collision tests pass.

### Task 4: Verify generated Foundry Script and the repository

**Files:**
- Verify: all modified and generated files above

- [x] **Step 1: Build and run the Foundry regression GREEN**

Run:

```bash
task foundry:test
```

Expected: warning-level lint is clean and the inherited-collision values round-trip.

- [x] **Step 2: Run formatting and generated sync checks**

Run:

```bash
task fmt
FOUNDRY_BIN="$(command -v foundry)" bash scripts/ci/sync-foundry-engine-types.sh check
git diff --check
```

Expected: no formatting or metadata drift.

- [x] **Step 3: Run full requested verification sequentially**

Run:

```bash
task ci
task integration
task foundry:test
```

Expected: every command exits zero. If golangci-lint reports its process lock, retry `task ci` sequentially after the other lint process exits.

- [x] **Step 4: Self-review issue scope and repository state**

Confirm the map contains only `RefCounted` and `Object` owners, exact-case non-collisions remain unchanged, enum-value naming remains unchanged, generated source is synchronized, and no `.foundry`, `.uid`, probe, or unrelated files remain.

- [x] **Step 5: Commit the implementation**

Run:

```bash
git add docs/superpowers/plans/2026-08-02-inherited-engine-member-collisions.md \
  internal/proto/internal/foundryscript/generator \
  tests/foundry/collisions.proto tests/foundry/main.fs tests/foundry/run.sh
git commit -m "fix: escape inherited engine member collisions"
```
