# Any Conformance, Regeneration, and Documentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the Any epic with end-to-end compatibility evidence, deterministic regeneration, public documentation, and a complete verification matrix.

**Architecture:** Treat the implemented registry/wire/JSON stack as fixed and add black-box coverage at integration and Foundry-engine boundaries. Regenerate all committed artifacts from their sources, document only supported public behavior, and use independent reviews to catch design drift before closing the parent epic.

**Tech Stack:** Go integration tests, protoc/Buf, Foundry Engine `/Users/christian/bin/foundry`, Task, Markdown

---

**Issue:** [#103](https://github.com/cafecito-games/Foundry-Tools/issues/103)

**Depends on:** #98–#102
**Design:** [`docs/superpowers/specs/2026-08-02-any-type-url-registry-design.md`](../specs/2026-08-02-any-type-url-registry-design.md)

## Ownership and dependency boundary

This issue owns final black-box tests, artifact determinism, README/user documentation, and epic-level review. It should fix only integration defects found while exercising the already-approved behavior; architecture changes go back to the owning child issue.

### Task 1: Build the final acceptance matrix before changing docs

**Files:**

- Modify: `tests/integration/conformance_test.go`
- Modify: `tests/integration/protoc_plugin_test.go` if plugin invocation needs an Any-specific assertion
- Modify: `tests/foundry/main.fs`
- Modify: `tests/foundry/conformance.fs`

- [ ] Add or consolidate black-box cases for all of the following:

  - generated top-level and nested identity/factory methods;
  - registry idempotence, conflict, clear, malformed URL, and missing type;
  - invalid hand-written identity plus typed-container rejection of unrelated handles and instances;
  - pack without registration, exact bytes, foreign/bare URLs, dynamic unpack;
  - ordinary Any JSON encode/decode and URL preservation;
  - every special WKT form category and Empty’s ordinary exception;
  - nested Any encode/decode and `$.value` error rerooting;
  - null/empty object behavior, missing `@type`, non-JSON types, corrupt wire bytes, and transactional failures;
  - embedded unknown-field, type, range, oneof, and duplicate-where-representable errors with exact paths;
  - no regression in non-JSON generation or ordinary canonical JSON.

- [ ] Run the smallest affected Go or Foundry command after each new assertion. New acceptance assertions must fail if the feature PRs have not supplied their behavior and pass on the completed dependency stack.
- [ ] Avoid re-testing generator source strings where a black-box generated file or engine behavior proves the same fact more strongly.

### Task 2: Add upstream-compatible ProtoJSON fixtures

**Files:**

- Create or modify fixtures under: `tests/integration/fixtures/`
- Modify: `tests/integration/conformance_test.go`

- [ ] Add canonical JSON documents for an ordinary packed message, Timestamp, wrapper, Struct/Value/ListValue, Empty, and nested Any. Include canonical `type.googleapis.com`, a foreign prefix accepted on input, and malformed cases.
- [ ] Compare emitted structures and decoded wire bytes to protobuf’s documented Any JSON mapping. Do not require prefix normalization: this implementation intentionally preserves a valid supplied URL.
- [ ] If the Go protobuf JSON library is used as an oracle, restrict comparison to behavior shared by the approved design; register the same descriptors explicitly and document any accepted-prefix difference in the test.
- [ ] Run `task integration`. Expect PASS.
- [ ] Commit acceptance coverage: `git add tests && git commit -m "test: complete Any conformance coverage"`.

### Task 3: Regenerate from source and prove determinism

**Files:**

- Regenerate: `examples/golden/`
- Regenerate: `examples/golden-json/generated/`
- Regenerate: `examples/golden-wkt/generated/`
- Regenerate: `internal/runtime/data/foundry/proto/wkt/`

- [ ] Run `go test ./internal/proto -run TestGolden -update -count=1`.
- [ ] Run `task gen-wkt`.
- [ ] Stage the regenerated paths with `git add examples internal/runtime/data/foundry/proto/wkt`, run both commands a second time, then run `git diff --exit-code -- examples internal/runtime/data/foundry/proto/wkt`. There must be no unstaged second-run changes.
- [ ] Inspect `Any.pb.fs`; confirm it is generated, enum 11 remains but is unused, identity/wire/JSON methods are present, and no manual-only block exists.
- [ ] Run `go test ./internal/proto -run TestGolden -count=1` and `go test ./internal/runtime -run TestWellKnownBindingsAreUpToDate -count=1`.
- [ ] Commit only genuinely changed artifacts: `git add examples internal/runtime/data/foundry/proto/wkt && git commit -m "test: refresh final Any generated artifacts"`.

### Task 4: Document the supported API and constraints

**Files:**

- Modify: `README.md`
- Modify: other existing user documentation only where it already describes protobuf runtime or JSON behavior

- [ ] Replace the README statement “Any has no JSON form yet (#48)” with a concise example that:

  1. calls `AnyTypeRegistry.register(Player)` during startup;
  2. packs a `Player` without requiring registration;
  3. unpacks through the registry and checks `ProtobufError.OK`;
  4. notes that JSON requires a JSON-enabled generated binding.

- [ ] Document accepted canonical, foreign-prefix, and bare type URLs; preservation on JSON decode; the explicit/no-network registry model; `clear()` for isolated tests; and deterministic errors 15–19.
- [ ] Document ordinary inline Any JSON, the WKT `value` envelope, nested Any, and Empty’s inline-object exception.
- [ ] State that wire-only messages can pack/unpack but return `ANY_JSON_UNSUPPORTED` on Any JSON conversion.
- [ ] Keep `JSON_ANY_UNSUPPORTED = 11` documented as deprecated/retained for numbering, not as active behavior.
- [ ] Run `rg -n 'Any has no JSON form|JSON_ANY_UNSUPPORTED' README.md docs` and update stale current-behavior claims while preserving historical design documents.
- [ ] Commit: `git add README.md docs && git commit -m "docs: document complete Any support"`.

### Task 5: Run the complete release-grade verification matrix

- [ ] Run `task gen-wkt`, stage `internal/runtime/data/foundry/proto/wkt`, run it again, then run `git diff --exit-code -- internal/runtime/data/foundry/proto/wkt` to prove the second run is stable.
- [ ] Run `task fmt`.
- [ ] Run `task ci`.
- [ ] Run `task integration`.
- [ ] Run `FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test` and record Foundry version `0.1.dev.custom_build.c11e3a080` in the PR.
- [ ] Run `git diff --check origin/main...HEAD`.
- [ ] Confirm no production code contains networking, a type server, reflection/load hooks, prototype instances, `Callable` factories, engine changes, or a second WKT special-form list.
- [ ] Confirm no public runtime signature exposes `Variant` and the registry stores only `Dictionary[String, Type[Message]]`.

### Task 6: Obtain independent reviews and close the epic

- [ ] Request one independent review focused on API/design fidelity: exact public signatures, error numbering, URL semantics, registration model, JSON shapes, and exclusions.
- [ ] Request a second independent review focused on implementation/testing: dynamic trait narrowing, transactionality, path rerooting, generated-artifact provenance, and verification evidence.
- [ ] Address every actionable finding with a focused test first, then re-run the relevant focused and full commands.
- [ ] Open the final PR linking #103 and #48. Include all commands and results; call out generated WKT/golden changes and the exact Foundry binary/version.
- [ ] After required checks and reviews are green, merge #103. Verify #98–#103 are closed/merged, update #48’s checklist if needed, and close #48 with links to all child PRs and the final verification evidence.
