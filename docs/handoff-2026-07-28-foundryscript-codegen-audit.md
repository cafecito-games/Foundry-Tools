# Handoff — Foundry Script codegen audit

**Date:** 2026-07-28
**Repo:** `cafecito-games/Foundry-Tools`
**Engine repo:** `cafecito-games/Foundry` (sibling checkout at `../Foundry`)

## What this is

An audit of the Foundry Script that `anvil proto generate` emits, against the current
Foundry Script language. Findings are filed as issues #10–#26. Nothing has been implemented
yet — this is a clean starting point with the analysis already done and verified.

## Why it was worth doing

The generator was written against an older Foundry Script and has drifted. Two classes of
problem:

1. **It emits code the current language rejects** — the enum syntax changed and every enum
   this tool produces is now a parse error.
2. **It ignores most of protobuf.** `oneof`, `map`, nested messages, nested enums,
   `repeated`, and `optional` are all parsed into the proto AST and then never read by the
   Foundry Script emitter. They produce no output *and no error*, so a user gets a clean
   build and a binding that silently drops data.

Neither was caught because `examples/golden/` is not regenerated or diffed by any test.

## Verified facts

Everything below was confirmed by execution, not inference. Re-verify if the toolchain moves.

**Enum syntax is the brace/indent change.** Confirmed against `foundry v0.1.alpha9`:

```
$ foundry script lint --no-header <brace-form>.fs
[error] Expected ":" after enum name.   (parse-error)
$ foundry script lint --no-header <indent-form>.fs
CLEAN
```

**Trait requirements must be `abstract func`, not bare `func`.** This is the whole cause of
the deferred `uses Message` conformance. Isolation run:

| Probe | Requirements | Namespaced | Result |
|---|---|---|---|
| E | `abstract func` | no | clean |
| F | bare `func` | no | `Could not resolve body of trait` |
| G | `abstract func` | yes | clean |

Namespaces, imports, generics, and dotted names are all irrelevant. With `abstract` added and
nothing else changed, all four `uses` spellings lint clean — including
`uses probe.rt.ProbeMessageGeneric[ProbeD]`, the dotted-generic form that
`tests/foundry/run.sh` explicitly guards against.

The stale comment at `generator.go:167-168` claiming Foundry "cannot resolve/apply imported
runtime trait bodies" is wrong and should be deleted with the fix.

**Named tuples are nominal, dual-access, and erasing.** Fields readable by name *and* index;
a `Vec2` is assignable to `(float, float)`. Source: the language's own fixture
`analyzer/features/tuple_named_construction.fs`.

**`self` and `Self` both work inside enum-hosted functions.** Source:
`runtime/features/enum_host_functions_nested.fs`.

## Language features that landed after the audit

**Read this before touching #16 or #23** — both were written under constraints that no longer
apply. From `../Foundry` git log, after the audit snapshot:

- **`tuple_name` (#1288)** — a file-level global tuple type, registered as a global qualified
  by `namespace`, referenced by name or `import`, exactly like `class_name`. Still takes no
  type parameters.
- **Tuple destructuring in `var`/`const` (#1292)** — `var (value, offset, error) = ...`.
  Statement-only, minimum arity 2, `_` discards, initializer mandatory, no type annotations
  per binding, no nested destructuring, not available in `for`.
- **`is` with case binds (#1300)** — `if message is Message.Move(x, y):`. Valid only as the
  condition of `if`/`elif`/`while`/`assert` (or an `and` operand within one); binds become
  locals of the guarded suite. Exactly one bind per payload field.
- **Tagged-union case patterns in `match` (#1304)**.

Consequences:

- **#23 is now a clear win, not a trade.** It was hedged because tuple element access would
  have regressed `read.error` to `read.2`. Destructuring removes that objection, and
  `tuple_name` means the carrier types can stay file-level globals matching the current
  runtime layout — `tuple_name FieldRead(value: int, offset: int, error: ProtobufError)` in
  `foundry/proto/field_read.fs`. The generic constraint still stands, so a generic
  `FieldRead[T]` needs either one named tuple per element type (small closed set) or the
  structural `(T, int, ProtobufError)`.
- **#16 gets better decode ergonomics** from case binds and case patterns.

Upstream `cafecito-games/Foundry#1313` (misleading GRAMMAR.md §4.5 + uninformative cascading
diagnostic) is **fixed** — §4.5 now states the `abstract` requirement correctly, and #1315
points cascading trait errors at the failing file. The rule itself did not change.

## Issues, in suggested order

**Land first — these are infrastructure for everything else:**

- **#12** — golden fixtures are never regenerated or diffed. Add a `Generate`-over-
  `example.proto` test with a `-update` flag, wired into CI. Do this first so every
  subsequent fix is verified rather than hand-maintained.
- **#26** — `tests/foundry/run.sh` calls removed CLI flags (`--path`, `--check-only`,
  `--import`) and its guard actively blocks the fix for #20. CI does not notice because
  `install-foundry.sh` pins `v0.1.0` while the engine is on `alpha9`. Adopt
  `foundry script lint --json` for structured diagnostics.
- **#11** — make `validateWireFields` reject the constructs the emitter cannot handle, so
  users get an error instead of silent data loss while real support lands. Each per-construct
  issue removes its own guard.

**Correctness:**

- **#10** — enum brace/comma syntax will not parse. Small, self-contained, unblocks the
  golden fixture. Also fix the hand-written runtime `protobuf_error.fs`.
- **#13** `repeated` · **#14** message-typed fields emitted as varints · **#15** `optional`
  presence. Independent of each other.

**Features:**

- **#16** oneof → tagged union · **#17** map → `Dictionary[K, V]` · **#18** nested types
- **#19** enum-hosted `to_wire`/`from_wire` — touches the same function as #10, so sequence
  it after
- **#20** `uses Message` — now unblocked; see verified facts
- **#21** redundant qualification · **#22** property accessors · **#23** tuple carriers ·
  **#24** hoist `skip_field` · **#25** naming

**Coupling to respect:** #22 and #15 change the same public surface and should land together.
#19 after #10. #17 shares machinery with #13 and probably wants to follow it.

## Open decisions

- **#18 needs a design call** — nested types as flattened `Outer.Inner.pb.fs` files (matches
  today's one-type-per-file layout) or as inner `class`/`enum` members (matches proto
  scoping). Affects how `fieldType` names every nested-typed reference, so settle it before
  implementing.
- **#23 generic carriers** — one named tuple per element type, or structural tuples with
  index access in the generic positions.

## Working with this

**Toolchain.** Use `which foundry` (currently `~/bin/foundry`, `v0.1.alpha9`) — newer than
the `~/.foundry/bin` copy and much newer than the CI pin. The CLI was rewritten; `--path`,
`--check-only`, and top-level `--import` are gone. `foundry --help` is authoritative.

Lint a file, with cross-file resolution:

```bash
foundry --headless project import --project <dir>     # build the script index first
foundry script lint --no-header --json --project <dir> <file.fs>
```

Empty `diagnostics` means clean. Without the import step every file reports spurious
unresolved-namespace errors.

**Probing.** Build throwaway projects in a scratch directory — a `project.foundry` plus the
`.fs` files is enough. When something fails, vary one dimension at a time until exactly one
flips the result; Foundry's cascading diagnostics historically named the consumer rather than
the cause, which is what produced the wrong conclusion in `generator.go:167`.

**The reviewer agent.** `~/.claude/agents/foundry-script-reviewer.md` — read-only Foundry
Script auditor that produced these findings. It carries a dated language snapshot, an idiom
rubric (I1–I13), codegen-aware mode, and instructions to verify via lint before reporting.

**Its snapshot is dated 2026-07-28 and is now stale** in the four ways listed above —
`tuple_name`, destructuring, `is` case binds, and case patterns are all absent from it, and
its §4.5 note about the grammar being misleading is now obsolete. Update it before the next
audit run, or it will under-recommend exactly the idioms that just got better.

## Repo state

- Branch `main`, clean except untracked `docs/foundry-lint-json-sarif-agent-prompt.md`
  (unrelated to this work).
- No code changes made. Issues #10–#26 filed; #20 rewritten after the `abstract` finding;
  Foundry#1313 filed upstream and since fixed.
