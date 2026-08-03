# Explicit Any Type Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a deterministic, explicit runtime registry from fully-qualified protobuf names to generated `Type[Message]` handles.

**Architecture:** A runtime-wide `AnyTypeRegistry` owns one typed dictionary and validates both registered names and incoming type URLs. Registration is idempotent for the same handle, conflicts are errors, lookup never performs networking or reflection, and tests clear global state between scenarios.

**Tech Stack:** Foundry Script, Go runtime-source tests, Task, Foundry Engine `/Users/christian/bin/foundry`

---

**Issue:** [#99](https://github.com/cafecito-games/Foundry-Tools/issues/99)

**Depends on:** #98
**Design:** [`docs/superpowers/specs/2026-08-02-any-type-url-registry-design.md`](../specs/2026-08-02-any-type-url-registry-design.md)

## Ownership and dependency boundary

This issue owns `AnyTypeRegistry`, errors 15–19, runtime export collision protection, and registry-focused engine tests. It does not add methods to `google.protobuf.Any`; issue #100 consumes the private resolver.

### Task 1: Pin the new error numbers

**Files:**

- Modify: `internal/runtime/runtime_test.go`
- Modify: `internal/runtime/data/foundry/proto/protobuf_error.fs`

- [ ] Add `TestProtobufErrorCarriesTheAnyRegistryCases` first. Assert 11 remains `JSON_ANY_UNSUPPORTED`, 12–14 remain unchanged, and the new exact assignments are:

```foundryscript
ANY_TYPE_NAME_INVALID = 15
ANY_REGISTRY_CONFLICT = 16
ANY_TYPE_URL_INVALID = 17
ANY_TYPE_NOT_REGISTERED = 18
ANY_JSON_UNSUPPORTED = 19
```

- [ ] Run `go test ./internal/runtime -run TestProtobufErrorCarriesTheAnyRegistryCases -count=1`. Expect failure.
- [ ] Append the five enum cases without renumbering existing values. Mark `JSON_ANY_UNSUPPORTED = 11` as retained/deprecated in its comment; do not remove it.
- [ ] Re-run the focused test. Expect PASS.

### Task 2: Specify registry storage and public surface

**Files:**

- Create: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `internal/runtime/runtime_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/runtime_names_test.go`

- [ ] Add a runtime-source test requiring:

```foundryscript
class_name AnyTypeRegistry extends RefCounted
static var _types: Dictionary[String, Type[Message]] = {}
static func register(message_type: Type[Message]) -> ProtobufError:
static func clear() -> void:
```

- [ ] Require public source to contain no prototype instance, `Callable`, network hook, dynamic loader, or `Variant` signature.
- [ ] Run `go test ./internal/runtime -run 'TestFilesReturnsRuntimeSources|AnyTypeRegistry' -count=1`. Expect failure because the file does not exist.
- [ ] Create the class with the exact typed dictionary and public methods. `clear()` must replace it with `{}` so tests and callers can isolate registry state.
- [ ] Keep all lookup, URL parsing, JSON narrowing, and form-dispatch helpers private (`_` prefix).

### Task 3: Implement deterministic name registration

**Files:**

- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `tests/foundry/main.fs`

- [ ] Add `check_any_registry()` to the Foundry harness and call `AnyTypeRegistry.clear()` at its start and end.
- [ ] Before implementation, assert: `Player` registers successfully; registering `Player` twice is `OK`; an unregistered lookup fails; and clearing removes the entry.
- [ ] Add two test-only hand-written `Message` classes in `tests/foundry/main.fs`. `InvalidIdentityMessage.protobuf_type_name()` returns `"bad name"`; `ConflictingPlayer.protobuf_type_name()` returns `"cafecito.game.v1.Player"` but is a different handle. Implement the post-#98 trait methods and trivial wire methods on both, without adding either class to production runtime code.
- [ ] Assert the invalid handle returns `ANY_TYPE_NAME_INVALID` and leaves the registry unchanged. After registering real `Player`, assert `ConflictingPlayer` returns `ANY_REGISTRY_CONFLICT`, lookup still returns real `Player`, and same-handle re-registration is still `OK`.
- [ ] Implement `register` as:

```foundryscript
static func register(message_type: Type[Message]) -> ProtobufError:
	var name: String = message_type.protobuf_type_name()
	if not _is_valid_type_name(name):
		return ProtobufError.ANY_TYPE_NAME_INVALID
	if _types.has(name):
		if _types[name] == message_type:
			return ProtobufError.OK
		return ProtobufError.ANY_REGISTRY_CONFLICT
	_types[name] = message_type
	return ProtobufError.OK
```

- [ ] Do not accept a caller-supplied string key: the generated type handle is the source of identity.

### Task 4: Parse type URLs and resolve handles

**Files:**

- Modify: `internal/runtime/data/foundry/proto/any_type_registry.fs`
- Modify: `tests/foundry/main.fs`

- [ ] Add failing cases for canonical `type.googleapis.com/cafecito.game.v1.Player`, a foreign prefix, and bare `cafecito.game.v1.Player`; all resolve to `Player` after registration.
- [ ] Add failures for empty input, `/`, a trailing slash, empty segments, leading/trailing dots, digit-leading segments, whitespace, and punctuation. They must return `ANY_TYPE_URL_INVALID`, not `ANY_TYPE_NOT_REGISTERED`.
- [ ] Add a valid but unregistered name and require `ANY_TYPE_NOT_REGISTERED`.
- [ ] Implement `_type_name_from_url(type_url: String) -> (String, ProtobufError)` by selecting the substring after the final slash, or the entire string when no slash exists. Validate a dotted protobuf full name: every segment is nonempty; first byte is ASCII letter or underscore; remaining bytes are ASCII letter, digit, or underscore.
- [ ] Implement `_resolve(type_url: String) -> (Type[Message]?, ProtobufError)` with an explicitly typed null failure value. Return the exact registered handle and never normalize or rewrite the caller’s URL.
- [ ] Add engine assertions that the typed dictionary rejects a non-`Message` type handle and a `Player` instance if either is forced through a local dynamic value. Keep these assertions confined to tests; production storage remains statically typed.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`. Expect PASS.

### Task 5: Protect the new runtime export and dynamic seam

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/names.go`
- Modify: `internal/proto/internal/foundryscript/generator/runtime_names_test.go`
- Modify: `internal/runtime/runtime_test.go`

- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run TestRuntimeTypeNamesCoverEveryExportedRuntimeType -count=1`. Expect failure naming `AnyTypeRegistry`.
- [ ] Add `AnyTypeRegistry` to `runtimeTypeNames`; do not weaken the coverage test.
- [ ] Keep the blanket public `Variant` ban. If later JSON narrowing requires a private local `Variant`, update the test only in #101 to inspect public signatures rather than exempting the entire registry file now.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run TestRuntimeTypeNamesCoverEveryExportedRuntimeType -count=1` and `go test ./internal/runtime -count=1`. Expect PASS.
- [ ] Commit: `git add internal tests/foundry/main.fs && git commit -m "feat: add explicit Any type registry"`.

### Task 6: Verify the issue boundary

- [ ] Run `task fmt`.
- [ ] Run `task ci`.
- [ ] Run `task integration`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`.
- [ ] Run `git diff --check origin/main...HEAD`.
- [ ] Confirm the registry stores exactly `Dictionary[String, Type[Message]]`, has no fallback discovery, and no `Any.pack`, `Any.unpack`, or Any JSON implementation landed.
- [ ] Open a PR linking #99 and depending on #98. Do not close the epic.
