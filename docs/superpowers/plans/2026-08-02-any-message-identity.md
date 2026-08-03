# Any Message Identity and Construction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every generated protobuf message expose its exact protobuf identity and a trait-dispatched constructor through `foundry.proto.Message`.

**Architecture:** Carry the fully-qualified protobuf name already computed by recursive message planning into `messagePlan`, then emit three small methods on every top-level and nested class. Keep construction on the `Message` trait because Foundry cannot call `.new()` through `Type[Message]`; the generated static factory preserves the dynamic concrete type.

**Tech Stack:** Go 1.26, Foundry Script, `testify/require`, Task, Foundry Engine `/Users/christian/bin/foundry`

---

**Issue:** [#98](https://github.com/cafecito-games/Foundry-Tools/issues/98)

**Epic:** [#48](https://github.com/cafecito-games/Foundry-Tools/issues/48)
**Design:** [`docs/superpowers/specs/2026-08-02-any-type-url-registry-design.md`](../specs/2026-08-02-any-type-url-registry-design.md)

## Ownership and dependency boundary

This issue owns the universal message-trait contract, generator planning/emission, generated-name collision handling, and all resulting checked-in generated files. It does not add `AnyTypeRegistry` or Any pack/unpack/JSON behavior. Issue #99 depends on this contract.

### Task 1: Specify the runtime trait contract

**Files:**

- Modify: `internal/runtime/runtime_test.go`
- Modify: `internal/runtime/data/foundry/proto/message.fs`

- [ ] Extend `TestTraitRequirementsAreAbstract` first. Require these exact declarations:

```go
require.Contains(t, source, "abstract static func create_message() -> Self")
require.Contains(t, source, "abstract static func protobuf_type_name() -> String")
require.Contains(t, source, "abstract func type_name() -> String")
```

- [ ] Run `go test ./internal/runtime -run TestTraitRequirementsAreAbstract -count=1`. Expect failure because `message.fs` lacks all three requirements.
- [ ] Add the declarations to `message.fs`, ahead of the wire methods:

```foundryscript
abstract static func create_message() -> Self
abstract static func protobuf_type_name() -> String
abstract func type_name() -> String
```

- [ ] Re-run the focused test. Expect PASS.
- [ ] Commit this contract checkpoint: `git add internal/runtime && git commit -m "feat: define generated message identity contract"`.

### Task 2: Carry exact protobuf identity through planning

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/plan.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`

- [ ] Add a failing generator test named `TestMessagesExposeExactProtobufIdentityAndConstruction`. Generate a file in package `cafecito.game.v1` containing `Player` and `Player.Badge`; assert the top-level source contains:

```foundryscript
static func create_message() -> Player:
	return Player.new()

static func protobuf_type_name() -> String:
	return "cafecito.game.v1.Player"

func type_name() -> String:
	return Player.protobuf_type_name()
```

- [ ] Assert the nested class independently contains `"cafecito.game.v1.Player.Badge"`, returns `Badge`, and delegates to `Badge.protobuf_type_name()`.
- [ ] Add a case with `(foundrytools.namespace)` and `(foundrytools.type_prefix)` and assert protobuf identity still uses the proto package/name, never the Foundry namespace or emitted class prefix.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run TestMessagesExposeExactProtobufIdentityAndConstruction -count=1`. Expect failure.
- [ ] Add `ProtoName string` to `messagePlan`. Populate it from the existing `protoOwnerIdentity` passed to `planMessage`; recursive nested planning must receive and retain the parent-qualified proto identity.
- [ ] Re-run the focused test. It must still fail only because emission is absent.

### Task 3: Emit the factory and identity methods

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/generator.go`
- Modify or add: `internal/proto/internal/foundryscript/generator/message_methods.go`
- Test: `internal/proto/internal/foundryscript/generator/generator_test.go`

- [ ] Add AST helpers that emit the exact signatures and bodies below using `fsast.Func`, `fstypes.Named`, `fsast.Return`, and the planned concrete name:

```foundryscript
static func create_message() -> Player:
	return Player.new()

static func protobuf_type_name() -> String:
	return "cafecito.game.v1.Player"

func type_name() -> String:
	return Player.protobuf_type_name()
```

- [ ] Append these methods in `messageClass` immediately after `_pb_unknown_fields` and before `from_bytes`. Do not change `jsonUses`: wire-only classes remain `uses Message`; JSON classes remain `uses Message, JsonSerializable`.
- [ ] Run the focused generator test and expect PASS.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -count=1`. Expect existing collision/golden-adjacent assertions to expose reserved-name work still required.

### Task 4: Reserve the new generated method names

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/names.go`
- Modify: `internal/proto/internal/foundryscript/generator/names_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/member_collisions_test.go`

- [ ] Extend `TestGeneratedMethodNamesAreFreshAndReserved` first so the ordered inventory is:

```go
[]string{
	"create_message", "protobuf_type_name", "type_name",
	"from_bytes", "to_bytes", "merge_from_bytes", "to_json", "from_json",
}
```

- [ ] Add collision fixtures for schema fields named `create_message`, `protobuf_type_name`, and `type_name`; require each emitted member to gain one trailing underscore and its documentation to explain the generated-member collision.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator -run 'TestGeneratedMethodNamesAreFreshAndReserved|GeneratedMember' -count=1`. Expect failure.
- [ ] Define constants for all three names and prepend them in `generatedMethodNames()`. Let the existing `generatedMemberNames` derivation reserve them uniformly.
- [ ] Re-run the focused tests, then the full generator package. Expect PASS.
- [ ] Commit generator behavior: `git add internal/proto/internal/foundryscript/generator && git commit -m "feat: generate protobuf message identity"`.

### Task 5: Regenerate every checked-in binding

**Files:**

- Modify generated files under: `examples/golden/`
- Modify generated files under: `examples/golden-json/generated/`
- Modify generated files under: `examples/golden-wkt/generated/`
- Modify generated files under: `internal/runtime/data/foundry/proto/wkt/`

- [ ] Run `go test ./internal/proto -run TestGolden -update -count=1`.
- [ ] Run `task gen-wkt`, then stage the regenerated golden and WKT paths with `git add examples internal/runtime/data/foundry/proto/wkt`.
- [ ] Run `go test ./internal/proto -run TestGolden -update -count=1` and `task gen-wkt` again, then run `git diff --exit-code -- examples internal/runtime/data/foundry/proto/wkt`. Expect no unstaged second-run changes.
- [ ] Inspect representative wire, JSON, nested, and WKT outputs. Confirm exact proto names, concrete factories, unchanged `uses` lists, and no hand edits in `internal/runtime/data/foundry/proto/wkt/*.pb.fs`.
- [ ] Run `go test ./internal/proto -run TestGolden -count=1` and `go test ./internal/runtime -run TestWellKnownBindingsAreUpToDate -count=1`. Expect PASS.
- [ ] Commit regeneration: `git add examples internal/runtime/data/foundry/proto/wkt && git commit -m "test: regenerate message identity fixtures"`.

### Task 6: Prove trait dispatch in the engine

**Files:**

- Modify: `tests/foundry/main.fs`

- [ ] Add `check_message_identity()` and invoke it from `_init()` before the Any-specific checks introduced by later issues. Store `Player` in `Type[Message]`, call the static trait factory, and assert the dynamic result is `Player`, the static identity is `cafecito.game.v1.Player`, and the instance identity matches it. Repeat for `Player.Badge`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`. Expect PASS with `round trip ok`.
- [ ] Commit the engine proof: `git add tests/foundry/main.fs && git commit -m "test: prove dynamic message construction in Foundry"`.

### Task 7: Verify the issue boundary

- [ ] Run `task fmt`.
- [ ] Run `task ci`.
- [ ] Run `task integration`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test`.
- [ ] Run `git diff --check origin/main...HEAD`.
- [ ] Confirm `rg -n 'AnyTypeRegistry|static func pack|func unpack' internal` finds no implementation introduced by this issue.
- [ ] Open a PR that links #98 and explains the universal generated-output churn. Do not close #48.
