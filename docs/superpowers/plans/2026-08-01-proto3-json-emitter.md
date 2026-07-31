# proto3 canonical JSON — the emitter

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generated messages gain proto3 canonical JSON in and out, behind the `json` option that already exists but currently does nothing.

**Architecture:** A `JsonNode` tagged union in the runtime models a JSON document over six closed cases and owns the only `Variant` boundary in the system. Two new emitters beside `serialize.go` and `deserialize.go` walk the same field model `plan.go` already builds, producing per-message `to_json_node`/`merge_from_json_node` plus their string conveniences. A seven-entry table special-cases the well-known types, three of whose helpers already shipped in the foundations epic.

**Tech Stack:** Go 1.26 with testify, Foundry Script for the runtime. Engine checks run through `tests/foundry/run.sh`; the full gate is `task test && task integration && task foundry:test`.

**Spec:** `docs/superpowers/specs/2026-07-31-proto3-canonical-json-design.md`. Issue #44. Follows the foundations plan, `docs/superpowers/plans/2026-07-31-proto3-json-foundations.md`.

---

## Scope

Steps 3 onward of the roadmap in #44. The foundations epic (#50) landed the option
plumbing and four runtime helpers; nothing consumes the option yet. This plan makes
it do something.

Out of scope, and ordered after: `Any` pack/unpack and the type-URL registry (#48);
`Struct` <-> `Variant` and `Timestamp`/`Duration` <-> float-seconds conversions (#43),
which are further entries in the same emitter table but conversions rather than a
JSON form, and landable in either order.

## What the foundations epic already gives you

Read these on `main` before starting; they are the precedent to match.

| Already merged | Where |
|---|---|
| `Options{JSON: bool}` threaded through `Generate` | `generator/options.go` |
| `--foundryscript_opt=json` and `anvil proto generate --json` | `internal/plugin/parameter.go`, `internal/proto/command.go` |
| `ProtobufError` cases 7–11 | `internal/runtime/data/foundry/proto/protobuf_error.fs` |
| `JsonBase64`, `JsonTimestamp`, `JsonDuration`, `JsonFieldMask` | `internal/runtime/data/foundry/proto/json_*.fs` |
| `task foundry:test` rebuilds `bin/anvil` first | `Taskfile.yml` (#65) |

## File structure

| File | Responsibility |
|---|---|
| `internal/runtime/data/foundry/proto/json_node.fs` | New. The `JsonNode` union and its two `Variant` conversions. |
| `internal/runtime/runtime_test.go` | Modify. Narrow the `Variant` assertion to exempt `json_node.fs` only. |
| `internal/proto/internal/foundryscript/generator/names.go` | Modify. `JsonNode` in `runtimeTypeNames`. |
| `internal/proto/internal/foundryscript/generator/json_serialize.go` | New. Emits `to_json_node` and `to_json_string`. |
| `internal/proto/internal/foundryscript/generator/json_deserialize.go` | New. Emits `merge_from_json_node` and `from_json_string`. |
| `internal/proto/internal/foundryscript/generator/json_wellknown.go` | New. The seven-entry well-known form table, shared by both emitters. |
| `internal/proto/internal/foundryscript/generator/plan.go` | Modify, if the field model lacks a JSON name. See task 2. |
| `internal/proto/wellknown/gen/gen.go` | Already forces `Options{JSON: true}`; regenerating is the task, not a code change. |
| `internal/runtime/data/foundry/proto/wkt/*.pb.fs` | Regenerate. The checked-in bindings gain JSON. |
| `examples/golden-json/` | New. A JSON-enabled golden corpus. |
| `tests/foundry/main.fs` | Modify. Engine-run round-trip assertions. |
| `README.md` | Modify. The documented limitations. |

## Ownership and ordering

`json_node.fs` gates everything: neither emitter can be written before it exists.
After it, serialization and deserialization are independent and can run in parallel —
they touch disjoint files. Both must land before the golden corpus, because the
corpus is their output.

Three files are serialization points where more than one task writes. Anything
touching them runs sequentially, not in parallel:

- `tests/foundry/main.fs`
- `internal/proto/internal/foundryscript/generator/names.go`
- `internal/runtime/data/foundry/proto/wkt/*.pb.fs`

## Assumptions

Foundry Script's payload-carrying `enum_name`, `Dictionary[K, V]`, `Array[T]`, and
recursion through them all work. This is not speculative: `ValueKindCase.pb.fs`,
`Struct.pb.fs:14`, and `ListValue.pb.fs:9` already prove every one of them in
checked-in runtime output. If `JsonNode`'s *direct* self-recursion
(`List(values: Array[JsonNode])`) turns out to differ from `Value`'s mutual recursion
through a class, the engine lint reports it by name and task 1 stops there rather
than working around it.

---

## Task 1 — `JsonNode` and its `Variant` boundary

Nothing else in this plan can start until this lands.

**Files:** `internal/runtime/data/foundry/proto/json_node.fs` (new),
`internal/proto/internal/foundryscript/generator/names.go`,
`internal/runtime/runtime_test.go`, `tests/foundry/main.fs`.

- [ ] Define the union with exactly six cases: `Null`, `Bool(value: bool)`,
      `Number(value: float)`, `Text(value: String)`, `List(values: Array[JsonNode])`,
      `Object(fields: Dictionary[String, JsonNode])`.
- [ ] `static func to_variant(_pb_node: JsonNode) -> Variant`, total over all six.
- [ ] `static func from_variant(_pb_value: Variant) -> (JsonNode?, ProtobufError)`,
      returning `JSON_TYPE_MISMATCH` for anything outside the six shapes.
- [ ] Add `JsonNode` to `runtimeTypeNames` in `names.go`, or `task test` fails on
      `TestRuntimeTypeNamesCoverEveryExportedRuntimeType`.
- [ ] Narrow the assertion in `runtime_test.go:21` so it exempts
      `foundry/proto/json_node.fs` **by file name** and still fails on a `Variant`
      anywhere else in the runtime. Do not delete the check, and do not widen it
      beyond that one file.
- [ ] Round-trip assertions in `tests/foundry/main.fs`: every case survives
      `to_variant` then `from_variant`; a nested `Object` containing a `List`
      containing an `Object` survives; `from_variant` on an unsupported dynamic
      value returns `JSON_TYPE_MISMATCH`.

**Acceptance:** `task build && task foundry:test && task test` pass. The runtime
`Variant` assertion still fails if you temporarily add `-> Variant` to `wire.fs` —
verify that by doing it and reverting, the way #65 was verified.

**Verify:** `task build && task test && task foundry:test`

## Task 2 — JSON field names in the field model

**Files:** `generator/plan.go`, `generator/names.go`, tests beside them.

- [ ] Establish first whether the field model already carries a JSON name. Field
      options parse into a generic map in `internal/proto/internal/parser/messages.go`,
      so `json_name` may already be reachable without a model change. Report which.
- [ ] `[json_name = "..."]` wins when present; otherwise derive camelCase per the
      specification.
- [ ] Both spellings — the JSON name and the original proto name — are accepted on
      input. Only the JSON name is emitted on output.

**Acceptance:** unit tests over the derivation, including a field whose `json_name`
disagrees with the derived form.

**Verify:** `task test`

## Task 3 — `json_serialize.go`

Depends on task 1. Parallel with task 4.

**Files:** `generator/json_serialize.go` (new), `generator/json_wellknown.go` (new),
`generator/generator_test.go`.

- [ ] Emit `to_json_node() -> JsonNode` and `to_json_string() -> String`, gated on
      `Options.JSON`, beside the existing `to_bytes`.
- [ ] Scalars per the spec table: 32-bit as number, 64-bit as string, `float`/`double`
      as number or `"NaN"`/`"Infinity"`/`"-Infinity"`, `bytes` through `JsonBase64`.
- [ ] Output presence: proto3 zero values omitted; `optional`, message, and oneof
      members emitted only when present; a present-but-null wrapper writes `Null`.
- [ ] Enums as case names. `repeated` to `List`, `map` to `Object` with keys
      stringified per spec.
- [ ] The well-known table in `json_wellknown.go`, keyed on import path, calling the
      already-merged `JsonTimestamp`, `JsonDuration`, `JsonFieldMask` helpers.
      `Any` returns `JSON_ANY_UNSUPPORTED`.

**Acceptance:** generator unit tests assert emitted text for each field kind. No
golden file outside the new JSON corpus changes.

**Verify:** `task test`

## Task 4 — `json_deserialize.go`

Depends on task 1. Parallel with task 3. Shares `json_wellknown.go` with task 3 — if
both are in flight, whichever lands second rebases onto the table rather than
rewriting it.

**Files:** `generator/json_deserialize.go` (new), `generator/generator_test.go`.

- [ ] Emit `merge_from_json_node(_pb_node: JsonNode) -> ProtobufError` and
      `static func from_json_string(_pb_text: String) -> (X?, ProtobufError)`.
- [ ] `from_json_string` is `JSON.parse_string`, then `JsonNode.from_variant`, then
      `merge_from_json_node`. A malformed document is `JSON_PARSE_FAILED`.
- [ ] Accept both name spellings from task 2. A member matching no field is
      `JSON_UNKNOWN_FIELD`.
- [ ] 64-bit integers accepted from number or string; a 32-bit field given an
      out-of-domain value is `JSON_VALUE_OUT_OF_RANGE`.
- [ ] Enums read from a case name or a number; an unrecognized number takes the
      default rather than erroring.
- [ ] Errors are returned, never thrown, matching the wire path.

**Acceptance:** generator unit tests per field kind, plus one test per error case
proving the specific `ProtobufError` comes back.

**Verify:** `task test`

## Task 5 — regenerate the well-known bindings

Depends on tasks 3 and 4. Touches `wkt/*.pb.fs`, a serialization point.

- [ ] Regenerate. `internal/proto/wellknown/gen/gen.go` already passes
      `Options{JSON: true}`, so this should be a regeneration, not a code change. If
      it is not, say so rather than editing generated files by hand.
- [ ] The drift test in `internal/runtime/runtime_test.go` keeps the checked-in output
      honest — it must pass without being relaxed.
- [ ] `Value`'s JSON form maps onto `JsonNode` directly; confirm the two agree rather
      than each having its own notion of a JSON value.

**Verify:** `task build && task test && task foundry:test`

## Task 6 — the JSON golden corpus

Depends on tasks 3, 4, 5.

- [ ] Add `examples/golden-json/` generated with the option on.
- [ ] Leave the existing `examples/golden/` corpus JSON-free, so every current test
      keeps covering the option's off-path. This is the point of a separate corpus —
      do not regenerate the existing one.
- [ ] Round-trip assertions in `tests/foundry/main.fs`: a message with every field
      kind survives `to_json_string` then `from_json_string`.

**Verify:** `task test && task integration && task foundry:test`

## Task 7 — documented limitations

Depends on nothing; can land any time after task 4. Last for reporting reasons.

- [ ] README, beside the existing open-enum note: a JSON round trip is lossy where a
      wire round trip is not — no unknown-member preservation, an unrecognized enum
      number becomes the default, a bare 64-bit JSON number loses precision past 2^53.
- [ ] `Any` has no JSON form yet; link #48.
- [ ] Note the `JsonNode` surface and why it is a union rather than a `Variant`, so
      the next reader does not "simplify" it back.

**Verify:** `task ci`

---

## Final gate

The last task to land runs the full suite and reports all three results:

```
task test && task integration && task foundry:test
```

`task integration` exercises protoc, plugin, and Buf flows that the foundations epic
never touched. It passed at the end of #58, so a failure here is this plan's doing
until proven otherwise — but establish that before fixing, and do not widen scope to
repair a pre-existing failure on `main`.

## Known traps

- **`names.go`.** Every new runtime type needs a `runtimeTypeNames` entry or
  `task test` fails. This cost the foundations epic a cycle.
- **The runtime is embedded in `bin/anvil`.** `task foundry:test` now depends on
  `task build` (#65), but run `task build` explicitly anyway rather than trusting it.
- **`@warning_ignore_start` is file-scoped.** A function-level `@warning_ignore` does
  not reach body statements; the file-level form above `namespace` is what works.
- **Validate input shape explicitly.** A Codex round on #55 flagged unbounded input
  acceptance; #56 and #58 responded by bounding input before scanning. Do the same.
- **`json_timestamp.fs` suppresses `INTEGER_DIVISION` file-wide.** If you add
  arithmetic there, the suppression will hide a non-integral divide.
