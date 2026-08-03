# Ordinary-Message Any ProtoJSON Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Encode and decode `google.protobuf.Any` containing an ordinary JSON-enabled generated message using canonical inline ProtoJSON.

**Architecture:** The generated Any binding remains the public JSON surface, while `AnyTypeRegistry` performs dynamic lookup and a private, checked `Variant` narrowing between the independent `Message` and `JsonSerializable` traits. Ordinary payload fields are merged beside `@type`; decoding removes only `@type`, delegates the remaining root object to the registered concrete type, and packs the resulting message without rewriting the supplied URL.

**Tech Stack:** Go generator AST, Foundry Script JSON builtins, explicit runtime registry, Task, Foundry Engine `/Users/christian/bin/foundry`

---

**Issue:** [#101](https://github.com/cafecito-games/Foundry-Tools/issues/101)

**Depends on:** #98–#100
**Design:** [`docs/superpowers/specs/2026-08-02-any-type-url-registry-design.md`](../specs/2026-08-02-any-type-url-registry-design.md)

## Ownership and dependency boundary

This issue owns ordinary-message Any JSON, dynamic JSON-trait narrowing, root-path error behavior, and removal of the old unsupported Any JSON result. It deliberately does not add the special `value` envelope for WKTs; issue #102 owns that branch.

### Task 1: Replace the unsupported serializer contract

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/json_serialize.go`
- Modify: `internal/proto/internal/foundryscript/generator/json_wellknown.go`

- [ ] Replace `TestWellKnownAnyReportsThatItHasNoJSONForm` with `TestWellKnownAnySerializesThroughTheRegistry`. Require generated Any `to_json()` to delegate through the registry and require the old `push_error("JSON_ANY_UNSUPPORTED` text to be absent.
- [ ] Require the failure channel to remain the repository convention: on malformed URL, unregistered type, corrupt payload, or non-JSON binding, the helper pushes an error containing the new category and `to_json()` returns `JsonNode.Null`.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run TestWellKnownAnySerializesThroughTheRegistry -count=1`. Expect failure on the legacy unsupported body.
- [ ] Replace the `wellKnownJSONAny` serializer arm with a small generated call such as `return AnyTypeRegistry._any_to_json(type_url, value)`. Remove `jsonAnyUnsupportedMessage` only after both encode and decode no longer use it.

### Task 2: Implement checked dynamic serialization

**Files:**

- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `internal/runtime/runtime_test.go`

- [ ] Add source assertions that no public method, return type, field, or stored value mentions `Variant`; the only permitted seam is a private checked local inside JSON helpers.
- [ ] Implement `_any_to_json(type_url: String, bytes: PackedByteArray) -> JsonNode` in this deterministic order:

  1. Resolve the type URL.
  2. Assign the resolved type handle to a local `Variant` only long enough to check `is Type[JsonSerializable]`.
  3. If the capability check fails, push `ANY_JSON_UNSUPPORTED` and return `JsonNode.Null` without inspecting payload bytes.
  4. Construct and merge wire bytes; on failure push the exact `ProtobufError` name and return `JsonNode.Null`.
  5. Assign the concrete message to a second local `Variant`, check `is JsonSerializable`, and call `to_json()` through a typed `JsonSerializable` local.
  6. Require `JsonNode.Object`; copy its `Dictionary[String, JsonNode]`, insert `@type` first, then copy embedded fields without mutating the embedded document.

- [ ] Ordinary serialization output must be structurally equivalent to:

```json
{"@type":"type.googleapis.com/cafecito.json.v1.Reference","label":"primary","weight":"7"}
```

- [ ] If an ordinary embedded message produces a non-object node, treat it as `ANY_JSON_UNSUPPORTED`; the special WKT branch is not added until #102.
- [ ] Run `go test ./internal/runtime -count=1`. Expect PASS.

### Task 3: Replace the unsupported decoder contract

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/json_deserialize.go`

- [ ] Replace `TestWellKnownAnyReportsThatItCannotBeDecoded` with `TestWellKnownAnyDecodesThroughTheRegistry`. Require the generated merge body to delegate to the runtime helper and assign `type_url` and `value` only after it succeeds.
- [ ] Pin these cases in generator/runtime source assertions:

  - `JsonNode.Null` succeeds as an empty Any.
  - `{}` succeeds as an empty Any.
  - A non-object fails at `$` with `JSON_TYPE_MISMATCH`.
  - Any nonempty object missing `@type` fails at `$["@type"]`.
  - A non-string, empty, malformed, or unregistered `@type` fails at `$["@type"]`.

- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run TestWellKnownAnyDecodesThroughTheRegistry -count=1`. Expect failure.
- [ ] Make the Any merge arm delegate to a registry helper returning decoded `type_url`, packed bytes, and `JsonDecodeError?`; assign the two fields only after the error is null. This preserves the pre-call Any on failure.

### Task 4: Implement ordinary dynamic decoding and path rules

**Files:**

- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `internal/runtime/runtime_test.go`

- [ ] Implement the private decode helper so it validates `@type`, resolves a `Type[Message]`, and uses a private checked local `Variant` to narrow the type handle to `Type[JsonSerializable]`. A direct `Type[Message]` → `Type[JsonSerializable]` cast is forbidden because the engine rejects it statically.
- [ ] Copy all members except `@type` into a fresh `Dictionary[String, JsonNode]`, then call the concrete static `from_json(JsonNode.object_of(payload))` through the narrowed handle.
- [ ] Preserve ordinary embedded paths at root: an invalid `level` field must remain `$.level`, not gain an Any wrapper segment.
- [ ] On successful decode, convert the dynamic result back to `Message`, call `Any.pack`, then overwrite only the packed result’s `type_url` with the exact caller-supplied URL. Return its exact bytes.
- [ ] On non-JSON registered types, return `ANY_JSON_UNSUPPORTED` at `$["@type"]`.
- [ ] Ensure every failure returns no partial fields and leaves the receiver unchanged.

### Task 5: Regenerate Any and exercise ordinary JSON in Foundry

**Files:**

- Regenerate: `internal/runtime/data/foundry/proto/wkt/Any.pb.fs`
- Modify: `examples/golden-json/json_suite.proto` only if an existing JSON-enabled fixture cannot serve as the packed payload
- Regenerate if schema changes: `examples/golden-json/generated/`
- Modify: `tests/foundry/main.fs`

- [ ] Run `task gen-wkt`, stage `internal/runtime/data/foundry/proto/wkt`, run `task gen-wkt` again, and require `git diff --exit-code -- internal/runtime/data/foundry/proto/wkt`; then run `go test ./internal/runtime -run TestWellKnownBindingsAreUpToDate -count=1`.
- [ ] Add `check_any_ordinary_json()`; register a JSON-enabled generated type and cover encode/decode, canonical and foreign URL preservation, inline field layout, empty/null input, missing/nonstring/empty/malformed/unregistered `@type`, non-JSON registered type, embedded unknown-field/type/range/oneof failures at their root paths, duplicate keys where the `JsonNode` input can represent them, corrupt wire payload during encode, and transactional failure.
- [ ] Give the wire-only case corrupt bytes too and assert `ANY_JSON_UNSUPPORTED` wins, pinning capability-before-wire-decode ordering.
- [ ] Explicitly assert the emitted object contains no `value` member for an ordinary message.
- [ ] Run `go test ./internal/proto -run TestGolden -update -count=1` only if the example schema changed; then run without `-update`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`. Expect `round trip ok`.
- [ ] Commit: `git add internal examples/golden-json tests/foundry/main.fs && git commit -m "feat: support ordinary Any ProtoJSON"`.

### Task 6: Verify the issue boundary

- [ ] Run `task fmt`.
- [ ] Run `task ci`.
- [ ] Run `task integration`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`.
- [ ] Run `git diff --check origin/main...HEAD`.
- [ ] Confirm `rg -n 'JSON_ANY_UNSUPPORTED' internal/proto/internal/foundryscript/generator internal/runtime/data/foundry/proto/wkt/Any.pb.fs` finds no active Any JSON path, while enum value 11 remains.
- [ ] Confirm the implementation has no special-WKT name list or nested-Any `value` branch yet.
- [ ] Open a PR linking #101 and depending on #98–#100. Do not close #48.
