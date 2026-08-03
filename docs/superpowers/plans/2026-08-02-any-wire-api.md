# Any Wire API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement canonical `Any.pack`, identity checks, and registry-driven unpacking while preserving exact wire bytes and deterministic errors.

**Architecture:** Generate Any-only native methods from the existing well-known declaration discriminator. Packing uses the message’s own type identity and wire encoder without registry access. Type checks compare the final URL segment. Unpacking asks `AnyTypeRegistry` for a concrete handle, constructs through `Message.create_message()`, and merges bytes transactionally.

**Tech Stack:** Go generator AST, Foundry Script runtime bindings, protobuf wire codec, Task, Foundry Engine `/Users/christian/bin/foundry`

---

**Issue:** [#100](https://github.com/cafecito-games/Foundry-Tools/issues/100)

**Depends on:** #98 and #99
**Design:** [`docs/superpowers/specs/2026-08-02-any-type-url-registry-design.md`](../specs/2026-08-02-any-type-url-registry-design.md)

## Ownership and dependency boundary

This issue owns the Any native wire surface and its generated WKT output. It does not implement Any ProtoJSON; issues #101 and #102 own that behavior.

### Task 1: Specify the Any-only generated API

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/wellknown_semantics_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/wellknown_semantics.go`

- [ ] Replace the existing assertions that Any lacks pack/unpack with a failing test named `TestWellKnownAnyEmitsPackTypeCheckAndUnpack`. Require exactly:

```foundryscript
static func pack(message: Message) -> Any:
func is_type(message_type: Type[Message]) -> bool:
func unpack() -> (Message?, ProtobufError):
```

- [ ] Assert an ordinary schema message named `Any` still receives none of these methods. Selection must remain keyed by declaration file plus message name through `wellKnownJSONFormFor`.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run TestWellKnownAnyEmitsPackTypeCheckAndUnpack -count=1`. Expect failure.
- [ ] Add `case wellKnownJSONAny: return anyNativeMembers(plan)` in `wellKnownNativeMembers`.

### Task 2: Implement pack and type matching

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/wellknown_semantics.go`
- Test: `internal/proto/internal/foundryscript/generator/wellknown_semantics_test.go`

- [ ] Strengthen the failing source assertions to require pack’s exact behavior:

```foundryscript
var _pb_result: Any = Any.new()
_pb_result.type_url = "type.googleapis.com/" + message.type_name()
_pb_result.value = message.to_bytes()
return _pb_result
```

- [ ] Require `is_type` to parse the current URL with the registry’s private URL-name helper or an equivalent shared private helper, return `false` on malformed URLs, and compare the final name to `message_type.protobuf_type_name()`.
- [ ] Implement both AST methods. `pack` must not call `register` or `_resolve`, and must copy `to_bytes()` exactly.
- [ ] Run the focused generator test. Expect pack/is_type assertions to pass while unpack remains incomplete.

### Task 3: Implement transactional unpack

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/wellknown_semantics.go`
- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Test: `internal/proto/internal/foundryscript/generator/wellknown_semantics_test.go`

- [ ] Require generated unpack to call `AnyTypeRegistry._resolve(type_url)`, return `(null, error)` on URL or registry failure, create only after successful resolution, and return no message if `merge_from_bytes(value)` fails.
- [ ] If generated WKT code cannot call a private registry helper across runtime files, expose the narrowest internal callable spelling supported by Foundry and document it as runtime-internal; do not make it part of the README public API.
- [ ] Emit the equivalent of:

```foundryscript
func unpack() -> (Message?, ProtobufError):
	var (_pb_message_type, _pb_error) = AnyTypeRegistry._resolve(type_url)
	var _pb_failed: Message? = null
	if _pb_error != ProtobufError.OK or _pb_message_type == null:
		return (_pb_failed, _pb_error)
	var _pb_message: Message = _pb_message_type.create_message()
	_pb_error = _pb_message.merge_from_bytes(value)
	if _pb_error != ProtobufError.OK:
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)
```

- [ ] Run the generator test and full generator package. Expect PASS.
- [ ] Commit generator semantics: `git add internal/proto/internal/foundryscript/generator internal/runtime/data/foundry/proto/any_type_registry.fs && git commit -m "feat: generate Any wire helpers"`.

### Task 4: Regenerate and pin the WKT binding

**Files:**

- Regenerate: `internal/runtime/data/foundry/proto/wkt/Any.pb.fs`
- Test: `internal/runtime/runtime_test.go`

- [ ] Run `task gen-wkt`, stage `internal/runtime/data/foundry/proto/wkt`, run `task gen-wkt` again, then run `git diff --exit-code -- internal/runtime/data/foundry/proto/wkt`. Expect no unstaged second-run changes.
- [ ] Add runtime-source assertions for the three methods and confirm `Any.pb.fs` was generator-produced. Never hand-edit it.
- [ ] Run `go test ./internal/runtime -run 'TestWellKnownBindingsAreUpToDate|Any' -count=1`. Expect PASS.
- [ ] Commit: `git add internal/runtime && git commit -m "test: regenerate Any wire helpers"`.

### Task 5: Exercise URL and wire behavior in Foundry

**Files:**

- Modify: `tests/foundry/main.fs`

- [ ] Add `check_any_wire_api()` and invoke it from `_init()`. Clear/register `Player`, then cover:

  - `Any.pack(player)` emits `type.googleapis.com/cafecito.game.v1.Player` and byte-for-byte `player.to_bytes()`.
  - Packing works before registration.
  - `is_type(Player)` accepts canonical, foreign-prefix, and bare URLs and rejects another registered type and malformed URLs.
  - `unpack()` preserves the concrete dynamic type and field values.
  - Empty, trailing-slash, and malformed URLs return `ANY_TYPE_URL_INVALID` with null message.
  - Valid unregistered names return `ANY_TYPE_NOT_REGISTERED` with null message.
  - A failed lookup leaves the original opaque `type_url` and `value` bytes unchanged.
  - Corrupt payload bytes return the wire decoder’s exact error with null message.
  - A failed unpack does not mutate the Any value or leak a partial message.

- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`. Expect `round trip ok`.
- [ ] Commit: `git add tests/foundry/main.fs && git commit -m "test: cover Any pack and unpack"`.

### Task 6: Verify the issue boundary

- [ ] Run `task fmt`.
- [ ] Run `task ci`.
- [ ] Run `task integration`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`.
- [ ] Run `git diff --check origin/main...HEAD`.
- [ ] Confirm `JSON_ANY_UNSUPPORTED` behavior is still present and no Any JSON implementation landed.
- [ ] Open a PR linking #100 and depending on #98/#99. Do not close #48.
