# Well-Known Schema Compatibility Design

**Issue:** [cafecito-games/Foundry-Tools#47](https://github.com/cafecito-games/Foundry-Tools/issues/47)

**Builds on:** [cafecito-games/Foundry-Tools#45](https://github.com/cafecito-games/Foundry-Tools/pull/45)

**Status:** Approved design checkpoint; awaiting written-spec approval

## Summary

Foundry Tools identifies the seven supported `google/protobuf/*.proto` files by
their canonical import paths and substitutes the runtime's generated bindings.
That is the correct identity model, but today substitution never checks whether
a caller-supplied file still has the wire shape of the canonical schema.

Add one AST-based compatibility validator shared by the direct CLI and protoc
plugin paths. It treats the runtime schema as a required subset: canonical
types, fields, enum numbers, cardinalities, maps, and oneof relationships must
remain compatible, while additive declarations and source-only changes remain
quiet. An actual incompatibility is a generation error before any runtime
substitution or output.

## Goals

- Reject a supported well-known import path whose supplied schema cannot be
  safely represented by the runtime binding.
- Apply exactly the same structural policy to hand-parsed sources and protoc
  descriptors.
- Keep normal protoc requests and compatible copies from other protobuf
  releases silent.
- Identify the canonical path, type, field, and mismatch in deterministic
  diagnostics.
- Derive the expected shape from the same embedded sources that generate the
  checked-in runtime bindings.

## Non-Goals

- Supporting protobuf editions in the hand-written parser. The compatibility
  model ignores syntax and edition metadata after parsing, but this issue does
  not expand the parser's accepted grammar.
- Warning about compatible extensions or presentation-only differences.
- Validating unsupported `google/protobuf` files beyond the existing rejection
  and referenced-dependency behavior.
- Exposing additive fields or types through runtime bindings. Additions are
  wire-compatible, but code that tries to use a type or member absent from the
  runtime can still fail at the normal generated-API or Foundry lint boundary.
- Comparing source text, comments, language options, or upstream release
  versions.

## Where the Candidate Schema Exists

### Direct CLI and hand parser

`ParseFiles` resolves each command-line path to the import path protoc would
assign it, then parses the root. `ResolveExternalWithFiles` parses and retains
the root's direct imports and transitive public re-exports. `WellKnownFS` gives
a caller-supplied source precedence over the embedded fallback, so the retained
AST is the exact candidate that affected name resolution.

Those ASTs still exist when `anvil proto generate` returns from `ParseFiles`.
The command has not yet skipped a well-known root, and the generator has not yet
rewritten an imported well-known namespace to `foundry.proto.wkt`. The command
therefore validates:

1. every parsed root, using `ParsedFile.ImportPath`; and
2. every retained imported AST, using `ImportedFile.Filename`.

The parser deliberately does not traverse a private import below a dependency
unless that dependency is itself a generation root. Such a schema is not part
of the AST set used to generate the current roots and cannot influence their
emitted references. The rule is to validate every supported well-known AST the
front end actually receives, not to add a second import traversal.

### Protoc plugin and descriptors

A `CodeGeneratorRequest` carries descriptors for every file to generate and
everything they import. `FromCodeGeneratorRequest` converts the full
`proto_file` list before `plugin.Run` skips any well-known `file_to_generate`
entry or asks the generator to resolve an imported type.

The plugin validates every converted file whose descriptor name is one of the
seven supported paths. It does not restrict validation to `file_to_generate`:
protoc normally supplies well-known files as dependency descriptors, which is
the most common substitution path and the main path this check protects.
Descriptors for `descriptor.proto` and other unsupported files remain outside
this compatibility pass.

## Architecture

### Shared compatibility package

Create `internal/proto/internal/wktcompat`. It owns all canonical parsing,
normalization, comparison, mismatch ordering, and formatting rules.

The package depends on:

- the common protobuf AST;
- the hand parser, only to parse the embedded canonical sources; and
- `internal/proto/wellknown`, which owns the supported path table and embedded
  source bytes.

It does not sit underneath the hand parser. Both front ends have already
produced ASTs before they call it, so this dependency direction has no cycle:

```text
direct command ─┐
                ├─ internal/proto facade ─ wktcompat ─ parser ─ wellknown
plugin ─────────┘
```

The `internal/proto` facade exposes a `SchemaFile` carrying an import path and
`*ProtoFile`, plus `CheckWellKnownCompatibility([]SchemaFile) error`. Both front
ends adapt their existing containers to that batch operation. They do not
implement endpoint-specific compatibility rules.

### Canonical model and cache

On first use, the compatibility package reads each path from
`wellknown.Files()`, parses `wellknown.Source(path)`, and normalizes the result.
Use a one-time lazy cache so a process parses the seven tiny canonical sources
once. A canonical parse failure is an internal error, not a candidate mismatch.

Candidate normalization remains linear in the declarations of each supplied
well-known file. Protoc provides at most one descriptor per path. The direct
parser may retain the same path for more than one root; validating those ASTs is
cheap and avoids incorrectly assuming two roots resolved the same physical
copy. No persistent or content-addressed cache is needed.

### Normalized schema shape

Normalization removes syntax that does not affect substitution and produces a
single comparison model for parser and descriptor ASTs:

- package name;
- fully qualified message and enum identities, including nested types;
- messages indexed by full name;
- fields indexed by number, including a normalized type and cardinality;
- oneof partitions represented by their canonical field-number sets rather
  than group names;
- maps represented as maps, with normalized key and value types; and
- enums represented by their numeric value set.

Type normalization recognizes scalar spellings exactly. Message and enum
references resolve to fully qualified names using `FullTypePath` when the
descriptor converter supplied it and protobuf lexical scope resolution when the
hand parser left a local reference short. The normalized type also records
whether the target is a message or enum.

Ambiguous duplicate canonical type or field-number definitions are themselves
incompatible; normalization must not silently choose the last declaration.

## Compatibility Policy

The runtime's canonical structure is a required subset of the candidate.

### File and declarations

- The package must be exactly `google.protobuf`.
- Every canonical fully qualified message must exist as a message.
- Every canonical fully qualified enum must exist as an enum.
- Renaming or removing a canonical type therefore fails.
- Additional messages and enums are compatible and ignored.
- Declaration order is ignored.

### Message fields

Canonical fields are matched by field number, never by field name.

For every canonical field number, the candidate must preserve:

- the exact scalar type, or the exact fully qualified message or enum identity
  and kind;
- cardinality: ordinary singular, proto3 `optional`, or repeated;
- map semantics, including exact key and value types; and
- oneof semantics among canonical fields.

Oneof semantics compare partitions rather than spellings. If canonical fields
`#1` through `#6` share one oneof, their candidate counterparts must still
share one group. Renaming that group is compatible. Splitting the fields,
moving one out, or moving a canonical non-oneof field into a oneof fails. A new
field may join a canonical oneof because the runtime treats its unknown wire
record like any other forward-compatible addition.

The following are compatible:

- renaming a field while retaining its number and shape;
- adding a field at a new number;
- adding declarations used only by the newer schema; and
- reordering fields.

The following fail:

- removing a canonical field;
- moving it to another number;
- changing its scalar or referenced type;
- changing message to enum or enum to message;
- changing singular, optional, or repeated cardinality;
- changing a map to a repeated entry message or ordinary field;
- changing a map key or value type; or
- changing the canonical oneof partition.

This policy uses exact protobuf types, not merely wire categories. For example,
`int64`, `uint64`, and `sint64` are different even though all use varint records,
because the runtime API and value interpretation differ.

### Enums

Every numeric value present in a canonical enum must remain representable in
the candidate enum. Enum value names and aliases are ignored; the wire carries
the number. Additional numeric values and aliases are compatible.

### Deliberately ignored input

The comparison ignores:

- field, oneof, and enum-value spellings;
- comments and documentation;
- imports;
- file, message, field, enum, and enum-value options;
- source positions;
- reservations;
- effective JSON names;
- packed-option spelling;
- syntax or edition metadata after the source has parsed; and
- any additive declarations described above.

These differences cannot make the canonical runtime codec misinterpret its
known fields. Current local evidence reinforces the asymmetry: protoc 33.4's
installed `timestamp.proto` differs from the repository pin in comments and
documented ranges while retaining the same structural model.

## Failure Behavior and Diagnostics

An incompatible candidate is a hard generation error. A warning is not used:
`CodeGeneratorResponse` has an error field but no warning field, and a
direct-CLI-only warning would make the two front ends behave differently.
Because compatible extensions are ignored, routine protoc descriptors and
newer source copies remain silent.

The batch validator collects mismatches before either front end substitutes or
writes output. Candidates are processed by canonical path, canonical types by
canonical declaration order, and fields by number, producing stable output.
Missing containers suppress dependent field diagnostics so one removal does not
create a cascade.

Examples:

```text
google/protobuf/timestamp.proto: google.protobuf.Timestamp.seconds (#1): expected singular int64; found repeated string
google/protobuf/struct.proto: missing canonical message google.protobuf.Value
google/protobuf/struct.proto: google.protobuf.Struct.fields (#1): expected map<string, google.protobuf.Value>; found repeated google.protobuf.Value
```

Diagnostics always name the canonical import path and identity and deliberately
omit source positions. Protoc descriptors do not reliably carry positions, so
including them only on the direct path would make equivalent inputs render
different errors.

For the direct command, the error occurs before `writeFiles`, preserving its
atomic output behavior. For the plugin, the error is returned in
`CodeGeneratorResponse.Error` before response files are appended.

## Source and Runtime Synchronization

The embedded sources remain the sole canonical definition:

1. the compatibility cache parses them into expected shapes;
2. `wellknowngen.Generate` parses them to build runtime bindings; and
3. `TestWellKnownBindingsAreUpToDate` regenerates and compares every checked-in
   runtime WKT binding.

No handwritten shape table or generated fingerprint is introduced. Add an
exact source-equality test between the embedded WKT files and the copies in
`tests/integration/fixtures/conformance/google/protobuf`. That enforces the
shared pin documented by both READMEs; an intentional upstream refresh updates
both trees, regenerates runtime bindings, and reviews the resulting changes.

## Testing Strategy

Implementation follows strict TDD.

### Shared validator unit tests

Start with candidate ASTs parsed from source and table-test:

- all seven embedded canonical schemas pass;
- comments, options, imports, source order, and source positions do not matter;
- field, oneof, and enum-value renames pass;
- additive fields, messages, enums, and enum numbers pass;
- package, missing/renamed type, and message/enum kind mismatches fail;
- missing or moved fields fail;
- scalar, message, enum, cardinality, and proto3-optional changes fail;
- oneof removal, insertion, merge, and split changes fail;
- map/non-map, key-type, value-type, and value-kind changes fail;
- missing canonical enum numbers fail; and
- multiple mismatches render in the specified deterministic order.

Include ASTs produced through the descriptor converter so short hand-parser
references and fully qualified descriptor references normalize identically.

### Direct command tests

Prove failures before output for:

- an incompatible well-known file passed as a direct root;
- an incompatible caller-supplied well-known direct import; and
- an incompatible well-known file reached through a public re-export.

Keep green cases for the embedded fallback, a caller copy with presentation
changes, and a compatible additive extension. A private import below an
ungenerated dependency remains outside the retained AST set and outside this
new traversal.

### Plugin tests

Using `CodeGeneratorRequest` fixtures, prove:

- an incompatible WKT dependency fails even when only its importer is in
  `file_to_generate`;
- an incompatible WKT direct output fails before the normal skip;
- a routine compatible WKT dependency remains quiet; and
- presentation changes and additions remain quiet.

### Integration and synchronization tests

- Run the direct CLI over the existing unmodified conformance fixture, which
  supplies all seven canonical WKT roots and imports.
- Run the real protoc plugin against a schema importing an installed WKT so
  protoc's normal dependency-descriptor path is exercised.
- Add a direct-CLI incompatible-copy failure case with a diagnostic assertion.
- Assert the embedded and conformance-fixture WKT source trees are byte-equal.
- Run `task ci`, `task integration`, and `task foundry:test`; this change does
  not intentionally alter generated `.pb.fs` output.

## Alternatives Considered

### Canonical descriptor fingerprints

Embedding normalized `FileDescriptorProto` fingerprints makes the plugin path
straightforward, but the hand parser then needs an AST-to-descriptor adapter or
a second normalizer. It also adds a generated canonical artifact that can drift
from the sources and runtime bindings.

### Text comparison or per-front-end validators

Text or hash comparison rejects harmless comments, options, formatting, and
release metadata. Separate AST and descriptor comparators avoid an adapter but
create two policy implementations that can disagree on oneofs, maps, and
additive compatibility. Both choices recreate the problem this design is meant
to solve.
