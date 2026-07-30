# Foundry Member Type Collision Design

**Issue:** [cafecito-games/Foundry-Tools#32](https://github.com/cafecito-games/Foundry-Tools/issues/32)

**Related:** [cafecito-games/Foundry-Tools#30](https://github.com/cafecito-games/Foundry-Tools/issues/30)

**Status:** Approved

## Summary

A valid protobuf field can have the same case-sensitive spelling as a Foundry
built-in type or exposed native class. Foundry rejects that spelling when it is
emitted as a member of a generated message:

```protobuf
message Example {
  string Node = 1;
  string String = 2;
}
```

```foundryscript
var Node: String = ""
var String: String = ""
```

Foundry Tools will automatically append one underscore to an actual generated
message member whose protobuf name matches a reserved engine type:

```foundryscript
var Node_: String = ""
var String_: String = ""
```

The mapping is deterministic and follows the generator's existing policy for
Foundry keywords and generated members. Every declaration and internal
reference uses the escaped spelling. Protobuf descriptors, field numbers, JSON
names, and binary encoding remain unchanged.

If escaping makes two declarations map to the same generated member, generation
fails with an actionable diagnostic. The generator does not search for another
suffix.

## Decision Context

The alternatives considered were:

1. **Automatically escape the generated member and reject secondary
   collisions.** This preserves a valid protobuf schema, follows the existing
   `FieldName` convention, and matches the broad policy used by mature protobuf
   generators.
2. **Reject every engine-name collision.** This preserves exact generated
   member spelling but requires users to rename an otherwise valid protobuf
   field.
3. **Add a field-level rename or prefix option.** This gives users control but
   introduces a new public option and migration surface for a case the
   generator can resolve deterministically.

The approved choice is option 1.

### Java generator precedent

The Java protobuf generator transforms field names to target-language
accessors. Ordinary fields such as `Node` and `String` become `getNode()` and
`getString()`. When a generated accessor would collide with an inherited Java
or protobuf method, Java decorates the generated name instead of rejecting the
schema. For example, it generates `getClass_()`, `getSerializedSize_()`,
`getParserForType_()`, and `getUnknownFields_()`.

The Java generator implements this with a target-specific forbidden-name set
and a trailing-underscore decoration:

- [Java generator name handling](https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/compiler/java/names.cc#L787-L864)
- [Java generated field naming](https://protobuf.dev/reference/java/java-generated/#fields)
- [Protobuf guidance on language keyword collisions](https://protobuf.dev/best-practices/dos-donts/#avoid-using-language-keywords-for-field-names)

The portable precedent is not that every generator must emit the same spelling.
It is that a target generator owns deterministic API-name adaptation when a
valid schema identifier conflicts with its target language or runtime.

## Goals

- Generate Foundry-valid members for protobuf fields that match Foundry
  built-in types or exposed native classes.
- Preserve valid protobuf schemas rather than requiring schema renames for a
  target-specific collision.
- Use one deterministic, documented mapping across all generation entry points.
- Apply the mapped name consistently throughout declarations, serialization,
  deserialization, setters, clearing, oneof handling, and generated
  documentation.
- Detect every duplicate generated member produced by escaping or by existing
  generated-name rules.
- Report all post-mapping member collisions found in the current protobuf file
  in stable order.
- Prove the result with direct CLI, protoc-plugin, and real Foundry lint/runtime
  coverage.

## Non-goals

- Change generated type-declaration collision handling. That is covered by
  issue #30 and `(foundrytools.type_prefix)`.
- Rename named enum values or oneof alternative cases. Foundry does not apply
  this member-shadowing rule to those declarations.
- Add a field-level rename, prefix, alias, or other schema option.
- Detect names contributed by project-specific global classes or autoloads.
- Detect inherited native properties, signals, or constants, such as members
  already defined by `RefCounted`. Those require a separate inventory and
  policy.
- Change protobuf descriptor names, field numbers, default JSON names, text
  format names, or binary wire encoding.
- Search for a free spelling by appending repeated underscores, numeric
  suffixes, or hashes.

## Reserved Name Source

Member escaping reuses `foundryEngineReservedTypes`, the generated table
introduced for issue #30. It contains:

- Foundry built-in types;
- exposed native classes; and
- `AsyncCallable`, the analyzer-specific built-in not currently present in the
  extension API built-in list.

The table remains tied to the Foundry release pinned in CI and is refreshed by
the existing engine-table workflow. There is no second member-specific list and
normal generation does not require a Foundry installation.

The same table has intentionally different effects by declaration kind:

- a generated type declaration that matches the table is rejected and can be
  resolved with `(foundrytools.type_prefix)`;
- a generated message member that matches the table is escaped with one
  underscore.

## Naming Contract

`FieldName` remains the single public mapping from a protobuf member spelling to
an emitted Foundry member spelling.

For a protobuf member name:

1. Start with the identifier exactly as represented in the protobuf schema.
   Do not normalize case or separators.
2. If the name is in `reservedFieldNames`, `generatedMemberNames`, or
   `foundryEngineReservedTypes`, append exactly one underscore.
3. Otherwise, return it unchanged.

Membership checks are case-sensitive. A name that appears in more than one
reserved category still receives only one underscore.

Examples:

| Protobuf name | Generated member | Reason |
| --- | --- | --- |
| `Node` | `Node_` | exposed native class |
| `String` | `String_` | built-in type |
| `AsyncCallable` | `AsyncCallable_` | analyzer built-in |
| `node` | `node` | no case-sensitive match |
| `var` | `var_` | Foundry keyword |
| `to_bytes` | `to_bytes_` | generated method |
| `Node_` | `Node_` | no direct reserved-name match |

`(foundrytools.type_prefix)` does not participate in this algorithm. It applies
only to generated type names and cannot rename fields.

## Constructs Covered

The engine-name check applies wherever a protobuf declaration produces an
actual member in a generated message class:

- singular scalar, enum, and message fields;
- explicit-presence optional fields;
- repeated fields;
- map fields;
- oneof group members; and
- the same constructs inside nested messages.

A oneof alternative does not produce an independent message member. Its name
becomes a tagged-union case, which Foundry permits to match an engine type.
Therefore:

```protobuf
message Example {
  oneof Node {
    string String = 1;
  }
}
```

produces a member named `Node_`, while the union alternative remains `String`.
The generated oneof case type continues to use its existing type-name
algorithm.

Named enum values likewise remain unchanged.

## Planning and Emission

Member naming is resolved during planning, never in individual emitters.

Each planned field retains:

- the raw protobuf name;
- the final emitted member name;
- its schema kind;
- its source position when available; and
- the reason for escaping when the final name differs.

Each planned oneof retains equivalent information for the group member. The raw
name continues to drive protobuf identity and generated local-variable
derivation; the final name drives the public Foundry member.

All generated Foundry references use the planned final name:

- member declarations and setter bodies;
- presence and default-value checks;
- scalar, enum, message, repeated, and map serialization;
- scalar, enum, message, repeated, and map deserialization;
- assignments and clearing;
- oneof group access and assignment; and
- unknown-enum retention behavior.

Emitters must not call the reserved-name table or rederive `FieldName`.

User-authored protobuf documentation is emitted unchanged. When the generator
creates fallback field documentation, it identifies the raw protobuf name, not
the escaped Foundry spelling. For example, the member `Node_` is documented as
the `Node` protobuf field.

## Secondary Collision Validation

Appending one underscore can make two distinct declarations claim the same
member:

```protobuf
message Example {
  string Node = 1;
  string Node_ = 2;
}
```

Both declarations map to `Node_`. Generation rejects the message rather than
choosing `Node__`, `Node_2`, or another fallback.

Normal `protoc` rejects this particular example because `Node` and `Node_` have
the same default JSON name. Foundry Tools still validates the generated member
space because:

- the direct source parser must fail safely and clearly;
- descriptor inputs may be constructed without passing through normal protoc
  validation; and
- other collisions can involve oneofs or generated companion members.

The collision collector inventories every emitted member claim in a message:

- regular and map fields;
- oneof group members;
- retained-unknown-enum companion members;
- the generated unknown-field buffer; and
- names reserved for generated methods and members.

Oneof alternatives do not claim message members.

Validation runs before emission and collects all post-mapping member collisions
in the current protobuf file. Diagnostics are sorted by:

1. fully scoped protobuf message name;
2. final generated member name;
3. declaration kind; and
4. raw protobuf declaration name.

The ordering does not depend on Go map iteration, input declaration order, or
generation entry point.

## Diagnostics

Each collision diagnostic includes:

- source filename;
- line and column for both declarations when available;
- fully scoped protobuf message and declaration names;
- both declaration kinds;
- the shared generated Foundry member spelling;
- the escape reason for either transformed declaration; and
- an instruction to rename one protobuf declaration.

Example:

```text
members.proto:6:3: field probe.members.v1.Example.Node and
members.proto:7:3: field probe.members.v1.Example.Node_ both generate
Foundry member "Node_"; "Node" is escaped because it conflicts with
native class "Node"; rename one protobuf field
```

Built-in reasons say `built-in type`; native-class reasons say `native class`.
Existing keyword and generated-member reasons use their corresponding
descriptions.

Descriptor-based generation uses `SourceCodeInfo` positions when present. If a
position is unavailable, the diagnostic retains the filename, fully scoped
protobuf identity, declaration kind, and generated spelling.

### Entry-point behavior

The direct CLI returns the aggregated diagnostic through its normal command
error. Multi-input generation remains failure-atomic: it writes neither
generated bindings nor runtime files if any input fails.

The protoc plugin returns the same text through
`CodeGeneratorResponse.error`. A failed response contains no generated binding
or runtime files.

Buf generation inherits the protoc-plugin behavior.

## Component Boundaries

### Member naming

`names.go` owns the raw-to-generated member mapping and escape classification.
It uses the checked-in engine table but has no parser, descriptor, emitter, or
Foundry runtime dependency.

### Member plans

`fieldPlan` and `oneofPlan` carry raw identity, final spelling, source metadata,
and escape metadata. They are the interface between schema resolution and
emission.

### Collision collection

Member collision validation consumes plans and generated-member reservations.
It does not modify names. It returns one stable error after the current file has
been planned.

### Emitters

Field, serialization, and deserialization emitters consume planned final names.
Their control flow and wire behavior otherwise remain unchanged.

This separation keeps naming testable without rendering source and prevents
declaration and reference spellings from drifting apart.

## Compatibility

The initial change only renames members in schemas whose generated output
already fails Foundry analysis, so it does not break a previously usable
Foundry binding.

The protobuf schema and wire contract do not change. Existing data remains
readable because serialization and deserialization continue to use field
numbers and protobuf wire types.

Refreshing the reserved-name table for a later Foundry release can add an
engine name. A previously safe generated member with that spelling will then
gain a trailing underscore when Foundry Tools adopts the new table. That is a
generated Foundry API change and must be called out in the engine-table update
or release notes.

Removing a name from a later engine table would otherwise remove an underscore
and also change the generated API. Engine-table refreshes therefore require
review of additions and removals, with generated member API changes documented.

## Verification

### Naming unit tests

Cover:

- representative built-ins (`String`, `Array`, `NodePath`);
- representative native classes (`Node`, `Timer`, `Resource`);
- `AsyncCallable`;
- case-sensitive non-collisions (`node`, `string`, `timer`);
- existing keyword and generated-member escaping;
- a name present in multiple reservation categories receiving one underscore;
- already suffixed names remaining unchanged; and
- escape classification text used by diagnostics.

### Generator tests

Prove consistent generated references for:

- singular, optional, repeated, and map fields;
- scalar, enum, and message values;
- nested messages;
- a oneof group whose name matches an engine type;
- a oneof alternative whose name matches an engine type and remains unchanged;
- setters and retained unknown enum values;
- serialization, deserialization, assignment, and clearing; and
- fallback documentation using the raw protobuf name.

### Collision tests

Cover:

- `Node` colliding with `Node_`;
- an escaped field colliding with a oneof group;
- interaction with existing keyword and generated-member escaping;
- interaction with generated unknown and retained-enum companion members;
- multiple collisions aggregated in stable order;
- source-parser positions;
- descriptor `SourceCodeInfo` positions; and
- positionless descriptor diagnostics.

### Integration tests

Direct CLI and protoc-plugin fixtures cover successful engine-member escaping
and secondary-collision failures. They assert identical naming and diagnostics
across entry points and assert failure atomicity.

### Foundry integration

The Foundry project fixture includes:

- regular fields named `Node` and `String`;
- a map or repeated field with an engine-reserved name;
- a oneof group with an engine-reserved name;
- a oneof alternative with an engine-reserved name that remains unchanged; and
- runtime assignments plus an encode/decode round trip through every escaped
  member.

`task foundry:test` must produce no lint diagnostics and must complete the
runtime round trip successfully with the CI-pinned Foundry binary.

The existing engine-table drift check remains authoritative; this issue does
not add another drift mechanism.

## Acceptance Criteria

- `Node`, `String`, and every other checked-in reserved engine type are escaped
  as message members with one trailing underscore.
- Non-reserved and case-variant field names remain byte-for-byte unchanged.
- Oneof group members are escaped; oneof alternatives and enum values are not.
- Every generated reference uses the planned escaped name.
- Secondary generated-member collisions fail before emission with deterministic,
  actionable diagnostics.
- Direct CLI, protoc plugin, and Buf generation implement the same policy.
- Protobuf schema identities and wire behavior are unchanged.
- Generated fallback documentation names the original protobuf field.
- Unit, integration, and Foundry lint/runtime coverage demonstrate the complete
  behavior.
