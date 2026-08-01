# proto3 canonical JSON — the emitter

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generated messages gain proto3 canonical JSON in and out, behind the `json` option that already exists but currently does nothing.

**Architecture:** The engine's builtin `JsonNode` tagged union models a JSON document over seven closed cases, and its `JsonSerializable` trait is the generated surface. Two new emitters beside `serialize.go` and `deserialize.go` walk the same field model `plan.go` already builds, producing per-message `to_json`/`from_json`. A seven-entry table special-cases the well-known types, three of whose helpers already shipped in the foundations epic.

> **Amended 2026-08-01 for engine `0.1.alpha19`.** This plan was written against
> `alpha14`, where a runtime-defined `JsonNode` and a runtime-owned `Variant` boundary
> were necessary. The engine now ships `JsonNode`, `JsonSerializable`, `JsonResult`,
> and `JsonDecodeError` as builtins, and both JSON boundaries are typed. Task 1 is
> rewritten and shrunk accordingly; tasks 3, 4, and 7 inherit the new surface and error
> type. The reasoning is on #71 and in the amended spec.

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
| `internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go` | Regenerate. Stale at `alpha14`, which fails `task foundry:test` outright. |
| `internal/proto/internal/foundryscript/generator/names.go` | Modify. Reserve the four JSON builtins, which `extension_api.json` does not carry. |
| `internal/proto/internal/foundryscript/generator/json_serialize.go` | New. Emits `to_json` and the `uses JsonSerializable` conformance. |
| `internal/proto/internal/foundryscript/generator/json_deserialize.go` | New. Emits `from_json` and `_pb_merge_from_json`. |
| `internal/proto/internal/foundryscript/generator/json_wellknown.go` | New. The seven-entry well-known form table, shared by both emitters. |
| `internal/proto/internal/foundryscript/generator/plan.go` | Modify, if the field model lacks a JSON name. See task 2. |
| `internal/proto/wellknown/gen/gen.go` | Already forces `Options{JSON: true}`; regenerating is the task, not a code change. |
| `internal/runtime/data/foundry/proto/wkt/*.pb.fs` | Regenerate. The checked-in bindings gain JSON. |
| `examples/golden-json/` | New. A JSON-enabled golden corpus. |
| `tests/foundry/main.fs` | Modify. Engine-run round-trip assertions. |
| `README.md` | Modify. The documented limitations. |

## Ownership and ordering

Task 1 gates everything: it lands the engine type table both emitters typecheck
against, and pins the engine behavior they assume. After it, serialization and
deserialization are independent and can run in parallel — they touch disjoint files.
Both must land before the golden corpus, because the corpus is their output.

Three files are serialization points where more than one task writes. Anything
touching them runs sequentially, not in parallel:

- `tests/foundry/main.fs`
- `internal/proto/internal/foundryscript/generator/names.go`
- `internal/runtime/data/foundry/proto/wkt/*.pb.fs`

## Assumptions

All of these were run against `0.1.alpha19.gh.7a86a1464` while re-planning #71, not
inferred. Re-verify if the engine moves.

- Direct self-recursion in a tagged union constructs and matches at run time
  (`List(values: Array[Self])` and `Object(fields: Dictionary[String, Self])`). This
  is what blocked the plan on `alpha14`.
- A `final class ... extends RefCounted uses Message, JsonSerializable` conforms to
  both traits, inside a namespace, and `JSON.stringify` reaches its `to_json`.
- `JSON.stringify` sorts keys unless passed `sort_keys = false`.
- `JSON.stringify` mangles non-finite floats (`NaN` to `null`, infinities to
  `±1e99999`), so the emitter produces their string forms itself.
- `JSON.parse_to_node` returns a `Float`, not an `Int`, for an integer literal too
  large for a double to hold exactly.

---

## Task 1 — the engine JSON types

Nothing else in this plan can start until this lands. Originally "define a runtime
`JsonNode` union and its `Variant` boundary"; the engine now owns both, so what is
left is the type table, the reserved names, and pinning the behavior tasks 3 and 4
depend on.

**Files:** `generator/engine_reserved_types.gen.go`, `generator/names.go`,
`tests/foundry/main.fs`.

- [ ] Regenerate the engine type table with `task gen-engine-types`. It is stale at
      `alpha14`, and `sync-foundry-engine-types.sh check` runs at the top of
      `tests/foundry/run.sh`, so `task foundry:test` fails on `main` until this lands.
      The diff should be the version constant alone — if it is not, say what else moved.
- [ ] Reserve `JsonNode`, `JsonResult`, `JsonDecodeError`, and `JsonSerializable` in
      `names.go`. They are builtin *script* classes, absent from `extension_api.json`,
      so the regenerated table does not contain them and a proto message of the same
      name would silently shadow the builtin. Add a test that a message named
      `JsonNode` is escaped.
- [ ] Assertions in `tests/foundry/main.fs` pinning what the emitters assume: a
      nested `Object` containing an `Array` containing an `Object` constructs and
      round-trips; `Int` and `Float` stay distinct through `JSON.stringify`
      (`1` versus `1.0`); a 64-bit integer survives as a `Str` and does not as a bare
      number; `JSON.parse_to_node` on a malformed document reports through
      `JsonResult` rather than raising.
- [ ] Do **not** touch `internal/runtime/runtime_test.go` or `tests/foundry/run.sh`.
      No `Variant` enters the runtime or the generated surface, so both gates stay as
      they are — the carve-outs the earlier design needed are unnecessary.

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

- [ ] Emit `func to_json() -> JsonNode` and add `JsonSerializable` to the class's
      `uses` list, both gated on `Options.JSON`, beside the existing `to_bytes`.
      Conformance is what teaches `JSON.stringify` to lower the message, so a message
      without it has no route to JSON text; there is no emitted `to_json_string`.
- [ ] Scalars per the spec table: 32-bit to `Int`, 64-bit to `Str`, `float`/`double`
      to `Float` when finite and to `Str("NaN")`/`Str("Infinity")`/`Str("-Infinity")`
      when not — never hand a non-finite float to `Float`, since the encoder turns it
      into `null` or `1e99999`. `bytes` through `JsonBase64`.
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

- [ ] Emit `static func from_json(_pb_node: JsonNode) -> JsonResult[X]` and a private
      `func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?`. `from_json`
      is construct-then-merge; decoding `repeated`, `map`, and `oneof` members is
      merge-shaped, so a second implementation would duplicate it. There is no emitted
      `from_json_string` — the caller passes `JSON.parse_to_node(text).value`, and a
      malformed document already comes back as a `JsonResult` failure.
- [ ] Failures are `JsonDecodeError`, not `ProtobufError`. Set `path` to the JSONPath
      of the offending value and use `JsonResult.nested(error, key)` to re-root a
      nested failure, so a field decoder reports from the document root. Lead the
      `message` with the matching `ProtobufError` case name — `JSON_TYPE_MISMATCH`,
      `JSON_UNKNOWN_FIELD`, `JSON_VALUE_OUT_OF_RANGE` — so categories stay greppable.
- [ ] Accept both name spellings from task 2. A member matching no field fails.
- [ ] Accept per the spec table, not per the case a value "should" be in: a 64-bit
      field takes `Str` (exact), `Int`, and `Float` (lossy, and the only case a large
      bare number arrives in); a 32-bit field takes `Int`, `Str`, and an integral
      `Float`, and fails out-of-domain values; a `float`/`double` field takes `Float`,
      `Int`, and the three non-finite strings.
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
- [ ] `Value`'s JSON form maps onto the engine's `JsonNode` directly; confirm the two
      agree rather than each having its own notion of a JSON value. `ValueKindCase`
      has no `Int` case, so a `Value` holding a whole number decides between
      `JsonNode.Int` and `JsonNode.Float` — pick `Float`, since `Value.number_value`
      is a `double`, and assert it.

**Verify:** `task build && task test && task foundry:test`

## Task 6 — the JSON golden corpus

Depends on tasks 3, 4, 5.

- [ ] Add `examples/golden-json/` generated with the option on.
- [ ] Leave the existing `examples/golden/` corpus JSON-free, so every current test
      keeps covering the option's off-path. This is the point of a separate corpus —
      do not regenerate the existing one.
- [ ] Round-trip assertions in `tests/foundry/main.fs`: a message with every field
      kind survives `JSON.stringify(msg, "", false)` then
      `X.from_json(JSON.parse_to_node(text).value)`.

**Verify:** `task test && task integration && task foundry:test`

## Task 7 — documented limitations

Depends on nothing; can land any time after task 4. Last for reporting reasons.

- [ ] README, beside the existing open-enum note: a JSON round trip is lossy where a
      wire round trip is not — no unknown-member preservation, an unrecognized enum
      number becomes the default, and a bare 64-bit JSON number loses precision past
      2^53, arriving as a `JsonNode.Float` rather than an `Int`. Our own output emits
      64-bit integers as strings, so it round-trips exactly.
- [ ] `Any` has no JSON form yet; link #48.
- [ ] Note that the JSON surface is the engine's `JsonSerializable` trait, that
      failures come back as `JsonDecodeError` with a JSONPath while the wire path keeps
      `ProtobufError`, and that `ProtobufError`'s five JSON cases are vestigial —
      retained for numbering, used only as message prefixes.

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
- **The engine type table gates `task foundry:test`.** `sync-foundry-engine-types.sh check`
  runs before anything else in `run.sh`, so a table generated against a different engine
  build fails the whole gate with an unrelated-looking message.
- **`JSON.stringify` sorts keys by default.** Pass `sort_keys = false` to keep field
  declaration order, or golden files will disagree with the emitted field ordering.
- **`JSON.stringify` on a bare `JsonNode` gives the raw tagged array.** The marshaller
  only fires for objects that conform to `JsonSerializable`; there is no `JsonNode` to
  text API. Text always goes through a conforming message.
- **The runtime is embedded in `bin/anvil`.** `task foundry:test` now depends on
  `task build` (#65), but run `task build` explicitly anyway rather than trusting it.
- **`@warning_ignore_start` is file-scoped.** A function-level `@warning_ignore` does
  not reach body statements; the file-level form above `namespace` is what works.
- **Validate input shape explicitly.** A Codex round on #55 flagged unbounded input
  acceptance; #56 and #58 responded by bounding input before scanning. Do the same.
- **`json_timestamp.fs` suppresses `INTEGER_DIVISION` file-wide.** If you add
  arithmetic there, the suppression will hide a non-integral divide.
