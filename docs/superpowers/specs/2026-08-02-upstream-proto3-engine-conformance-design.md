# Upstream Proto3 Engine Conformance Design

**Issue:** [cafecito-games/Foundry-Tools#28](https://github.com/cafecito-games/Foundry-Tools/issues/28)

**Builds on:** [cafecito-games/Foundry-Tools#38](https://github.com/cafecito-games/Foundry-Tools/issues/38)

**Status:** Approved

## Summary

The pinned, unmodified upstream `test_messages_proto3.proto` already passes an
integration generation test. This change puts the same schema into the Foundry
engine project so the engine lints all 40 real generated bindings, then runs a
small round-trip fixture for constructs the hand-written engine schemas do not
already prove.

The runtime fixture stays deliberately narrow. Existing engine tests already
cover all scalar framings, explicit packed and unpacked fields, nested-message
and nested-enum oneofs, representative maps, and the principal well-known
types. Repeating those values across the exhaustive upstream message would add
volume without adding confidence.

## Decisions

### Generate the pinned schema unchanged

`tests/foundry/run.sh` adds the conformance fixture include root and names
`test_messages_proto3.proto` plus its seven vendored well-known-type imports in
the normal generation invocation. The upstream files remain byte-for-byte
unchanged. Generated files remain ephemeral under `tests/foundry/generated/`
and are removed by the existing cleanup trap.

The project-wide Foundry Script lint continues to target `res://`. Consequently
every generated conformance declaration is parsed and typechecked, including
the complete scalar and map matrices, packed and unpacked field sets, unusual
field identifiers, nested declarations, recursive references, all ten oneof
arms, empty messages, aliased enums, and well-known-type fields. A regression
in any of those declarations fails the engine gate even when the runtime
fixture does not populate the field.

### Keep runtime coverage in a separate entry point

Create `tests/foundry/conformance.fs` rather than expanding the existing
`main.fs`, which is already responsible for the hand-written wire, well-known,
collision, and JSON fixtures. The new script imports
`protobuf_test_messages.proto3`, builds one finite `TestAllTypesProto3` graph,
round-trips it through `to_bytes` and `from_bytes`, and exits nonzero when an
assertion fails.

Its assertions cover only claims that are new here:

- a direct `recursive_message` edge;
- a `NestedMessage.corecursive` edge back to the parent type;
- the negative `NestedEnum.NEG` value and a noncanonical alias spelling from
  `AliasedEnum` surviving by numeric value;
- the real ten-arm oneof retaining its default-valued
  `google.protobuf.NullValue` case rather than confusing the value with absence;
- one representative map key/value combination not exercised by the existing
  wire fixtures.

The graph is finite and shallow so serialization cannot recurse indefinitely.
No attempt is made to populate every scalar, repeated field, map, wrapper, or
well-known type.

### Apply the existing process-error rule to both runners

`run.sh` invokes `main.fs` and `conformance.fs` separately. Each invocation must
exit zero. Their combined captured output is scanned for `SCRIPT ERROR`, because
the engine may unwind only the failing function and still return success from a
caller. Logging is refactored only enough to apply that established rule to both
scripts; no general test-runner abstraction is introduced.

## Failure Behavior

The gate fails when generation fails, any generated file produces a lint error,
either runtime entry point exits nonzero, either entry point emits `SCRIPT ERROR`,
or a focused assertion reports a mismatch. Cleanup removes generated bindings,
the Foundry import cache, generated UIDs, and the temporary log on every exit.

## Testing Strategy

Strict TDD starts with the engine gate: add the focused conformance runner and
its invocation before generating the upstream binding in `run.sh`. The expected
red result is an unresolved `protobuf_test_messages.proto3` import or missing
`TestAllTypesProto3` binding. The minimal green change adds the untouched pinned
schema and WKT inputs to generation.

After the focused red-green cycle, verify:

1. `task foundry:test` for generation, all-binding lint, and both round trips;
2. the focused integration conformance generation test;
3. `task ci`;
4. `task integration`;
5. a final `task foundry:test` after all edits.

## Non-Goals

- Editing or replacing the pinned upstream schema or WKT files.
- Reimplementing scalar or `[packed = false]` support already on `main`.
- Populating every field in `TestAllTypesProto3`.
- Integrating the upstream stdin/stdout conformance runner.
- Adding special JSON semantics for `Any`, `Struct`, or other WKTs.
