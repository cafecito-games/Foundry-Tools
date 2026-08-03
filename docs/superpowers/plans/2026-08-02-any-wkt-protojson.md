# WKT and Nested-Any ProtoJSON Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete canonical Any ProtoJSON for custom-form well-known types and recursively nested Any values.

**Architecture:** Derive the special-Any set from the generator’s existing `wellKnownJSONForms` table and expose only the generated type’s form category to runtime code. Special payloads live under `value`; nested Any is naturally recursive because its own JSON node becomes that value. Empty remains ordinary and therefore uses inline object semantics.

**Tech Stack:** Go generator metadata, Foundry Script JSON builtins, protobuf WKTs, Task, Foundry Engine `/Users/christian/bin/foundry`

---

**Issue:** [#102](https://github.com/cafecito-games/Foundry-Tools/issues/102)

**Depends on:** #101
**Design:** [`docs/superpowers/specs/2026-08-02-any-type-url-registry-design.md`](../specs/2026-08-02-any-type-url-registry-design.md)

## Ownership and dependency boundary

This issue owns the special `value` envelope, special-form metadata, nested Any recursion, and their path behavior. It must not introduce a second handwritten list of WKT names.

### Task 1: Derive one Any form category from the existing WKT table

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/json_wellknown.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/names.go`

- [ ] Add a failing table-driven test that generates every entry in `wellKnownJSONForms` and asserts its message identity method reports whether its Any payload is ordinary or value-wrapped. Required value-wrapped forms are Any, Timestamp, Duration, FieldMask, every wrapper, Struct, Value, and ListValue. `Empty` and `wellKnownJSONNone` are ordinary.
- [ ] Add one generator-owned identity method or constant with a reserved `_pb_` name, for example:

```foundryscript
static func _pb_any_uses_value() -> bool:
	return true
```

- [ ] Compute its boolean solely from `wellKnownJSONFormFor`; do not compare protobuf names in runtime Foundry Script and do not create another Go map or slice of special names.
- [ ] Run the focused test. Expect failure, then implement the derived metadata and expect PASS.

### Task 2: Serialize special WKTs under `value`

**Files:**

- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `internal/runtime/runtime_test.go`

- [ ] Extend `_any_to_json` after dynamic serialization. If the resolved handle reports special Any form, return exactly two members in this order: `@type`, then `value` containing the embedded type’s complete `to_json()` node.
- [ ] Cover representative shapes in source/engine tests:

```json
{"@type":"type.googleapis.com/google.protobuf.Timestamp","value":"2006-01-02T15:04:05Z"}
{"@type":"type.googleapis.com/google.protobuf.StringValue","value":"hello"}
{"@type":"type.googleapis.com/google.protobuf.Struct","value":{"enabled":true}}
```

- [ ] Confirm `google.protobuf.Empty` follows the ordinary path and emits only `{"@type":"...Empty"}` with no `value`.
- [ ] Confirm the runtime does not inspect the type-name string to decide the form.

### Task 3: Decode special WKTs and reroot errors

**Files:**

- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `internal/runtime/data/foundry/proto/json_result.fs` only if its existing `nested` helper cannot express the required path
- Test: `internal/runtime/runtime_test.go`

- [ ] For a special form, require a `value` member and call the concrete static `from_json` with that node directly. A missing value must fail at `$.value`.
- [ ] Reroot all embedded decode failures below `value`: a bad Timestamp, wrapper, Struct, or nested Any payload must report `$.value` plus its nested path.
- [ ] Reject extra top-level members other than `@type` and `value` for special forms using the repository’s unknown-field category and the offending root path.
- [ ] Preserve transactional behavior: only return packed bytes and the supplied URL after the embedded result succeeds.
- [ ] Add focused engine tests before implementation, then run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test` until these cases pass.

### Task 4: Implement nested Any recursion

**Files:**

- Modify: `tests/foundry/main.fs`
- Regenerate: `internal/runtime/data/foundry/proto/wkt/Any.pb.fs`

- [ ] Register `Any` and an ordinary JSON-enabled payload type. Pack the ordinary payload into an inner Any, then pack that Any into an outer Any.
- [ ] Assert the outer JSON is shaped as:

```json
{
  "@type": "type.googleapis.com/google.protobuf.Any",
  "value": {
    "@type": "type.googleapis.com/cafecito.json.v1.Reference",
    "label": "nested"
  }
}
```

- [ ] Decode it, unpack the outer value to `Any`, unpack the inner value to the concrete message, and assert exact dynamic types, data, URLs, and bytes.
- [ ] Add a malformed inner `@type` and require the error path to begin `$.value["@type"]`. Add an invalid inner `level` field and require `$.value.level`.
- [ ] Verify a nested failure returns no partial outer Any.
- [ ] Run `task gen-wkt`, stage `internal/runtime/data/foundry/proto/wkt`, run `task gen-wkt` again, and require `git diff --exit-code -- internal/runtime/data/foundry/proto/wkt`; then run the full Foundry test. Expect PASS.

### Task 5: Cover every special form without duplicating implementation tables

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`
- Modify: `tests/foundry/main.fs`

- [ ] Add a Go table assertion covering Any, Timestamp, Duration, FieldMask, all nine wrappers, Struct, Value, ListValue, and Empty’s ordinary exception.
- [ ] Add Foundry round trips for at least Timestamp, Duration, FieldMask, one numeric wrapper, StringValue, Struct, Value, ListValue, Empty, and nested Any. The Go table pins complete dispatch coverage; the engine matrix pins each distinct JSON node shape.
- [ ] Assert foreign URL prefixes survive decode for one scalar WKT and nested Any.
- [ ] Commit: `git add internal tests/foundry/main.fs && git commit -m "feat: support WKT payloads in Any ProtoJSON"`.

### Task 6: Verify the issue boundary

- [ ] Run `task fmt`.
- [ ] Run `task gen-wkt`, stage `internal/runtime/data/foundry/proto/wkt`, run it again, then run `git diff --exit-code -- internal/runtime/data/foundry/proto/wkt` to prove the second run is stable.
- [ ] Run `task ci`.
- [ ] Run `task integration`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`.
- [ ] Run `git diff --check origin/main...HEAD`.
- [ ] Search for special WKT name comparisons outside `wellKnownJSONForms`; the result must be empty.
- [ ] Open a PR linking #102 and depending on #101. Do not close #48.
