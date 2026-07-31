# proto3 canonical JSON

Resolves the question posed in #44: whether `protoc-gen-foundryscript` supports the
proto3 canonical JSON mapping, and if so, how far.

## Decision

Yes, with `google.protobuf.Any` deferred.

Generated messages gain JSON serialization and deserialization behind an opt-in
plugin option. The mapping follows the canonical proto3 JSON specification,
including the well-known types' special forms. `Any` is the one exception: its
JSON form requires resolving a type URL to a generated binding, which needs a
runtime type registry that does not exist yet, so it returns an error until that
lands separately.

Four subsidiary decisions, taken here so they are not answered piecemeal:

| Question | Decision |
|---|---|
| Where the logic lives | Per-message emitted methods, not a runtime codec over descriptors |
| Emission | Opt-in via `--foundryscript_opt=json` |
| 64-bit int arriving as a JSON number | Accepted, with documented precision loss past 2^53 |
| JSON member matching no field | Rejected with an error |

## Why these, and not the alternatives

**Per-message methods over a runtime codec.** The alternative was emitting a field
descriptor plus `Variant`-typed dynamic accessors on every message and letting one
hand-written runtime class walk them. That produces less generated code, keeps the
JSON rules in a single reviewable place, and is the same substrate that applying a
`FieldMask` to a message would need. It was rejected because it is real architecture
bought to serve a feature — mask application — that is speculative today, and because
the dynamic accessor pair would put `Variant` on every message for a project that
never asks for JSON. Per-message methods mirror how `serialize.go` and
`deserialize.go` already work, keep the golden-file diffs legible, and do not
foreclose the descriptor approach: if masks later turn out to matter, descriptors can
be added for that purpose alone without the JSON path having been the wrong shape.

**Opt-in.** JSON roughly doubles the size of a generated file. There is no plugin
option plumbing in `internal/proto/api.go` today, so this is new — but it is small,
and it keeps the cost on the projects that asked for it. The well-known type
generation forces the option on.

**Accepting a bare 64-bit number.** The specification permits a 64-bit integer to
arrive as a JSON number, and `JSON.parse_string` hands numbers back as floats, so
values past 2^53 lose precision. Rejecting bare numbers would be safe but
specification-noncompliant, and would fail against conformant peers that emit them.
Accepting is the compliant behavior; the limitation is documented rather than hidden.

**Rejecting unknown members.** What the specification says a parser should do by
default. JSON has no unknown-field preservation, so there is no `_pb_unknown_fields`
equivalent to round-trip an unrecognized member back out — a lenient parser would
silently drop it either way. Failing loudly at least catches a misspelled name.

## What changed since #39 and #43

Two facts in the repository make this landable now that were not true when #39 and
#43 were written.

**The `extend` blocker does not apply.** #43 was blocked because it attached
well-known type semantics to generated bindings via retroactive conformance, and
cafecito-games/Foundry#1376 and #1377 make that unworkable. The well-known bindings
are now checked-in generator *output* (`internal/runtime/data/foundry/proto/wkt/*.pb.fs`,
produced by `internal/proto/wellknown/gen`). The emitter can therefore special-case
the seven well-known files directly and the bindings stay regenerable. No `extend`,
no engine fix required. The same table serves the conversions #43 still wants, so
that issue is unblocked as a side effect rather than superseded.

**No JSON lexer is needed.** The engine exposes `JSON` and `Marshalls` as native
classes (`engine_reserved_types.gen.go`) with the same surface Godot gives them.
Serialization builds a `Variant` tree and hands it to `JSON.stringify`;
deserialization runs `JSON.parse_string` and walks the resulting `Variant`. `bytes` to
and from base64 falls out of `Marshalls`. Godot's parser returning every number as a
float is what forces the 64-bit precision decision above.

## Generated surface

A message gains four members when the option is on:

```
func to_json_variant() -> Variant
func to_json_string() -> String
func merge_from_json_variant(_pb_value: Variant) -> ProtobufError
static func from_json_string(_pb_text: String) -> (X?, ProtobufError)
```

`to_json_string` is `JSON.stringify` over the variant tree; `from_json_string` is
`JSON.parse_string` followed by `merge_from_json_variant`. The static constructor
returns the same `(X?, ProtobufError)` tuple as `from_bytes`, and errors are returned
rather than thrown, matching the wire path exactly.

These entry points expose `Variant`, which the README otherwise tells generated
public APIs to avoid. That is deliberate: a JSON document is a dynamic value, and a
caller who wants to embed a message in a larger `Variant` tree should not have to
round-trip through a string to do it.

## Mapping

Emitted per field, branching on kind the way the existing emitters do.

**Names.** `[json_name = "..."]` when the field carries it, otherwise the
specification's camelCase derivation from the proto name. Field options already parse
into a generic map (`internal/proto/internal/parser/messages.go`), so honoring
`json_name` costs nothing extra. The parser accepts either spelling on input.

**Output presence.** proto3 zero values are omitted. An `optional` field, a message
field, and a oneof member are emitted only when present. A wrapper that is present
but null writes an explicit `null`.

**Scalars.**

| Proto | JSON out | JSON in |
|---|---|---|
| `int32`, `uint32`, `fixed32`, `sfixed32`, `sint32` | number | number or string |
| `int64`, `uint64`, `fixed64`, `sfixed64`, `sint64` | string | number or string |
| `float`, `double` | number, or `"NaN"` / `"Infinity"` / `"-Infinity"` | number or those three strings |
| `bool` | boolean | boolean |
| `string` | string | string |
| `bytes` | base64 string | base64 string |

**Enums.** Written as the case name, read from either a case name or a number. An
unrecognized number takes the enum's default: `from_wire` returns null for it, and
unlike the wire path there is no field-position byte buffer to retain it in. This is
the same trade the README already documents for a `repeated` or map-valued enum.

**Composites.** `repeated` becomes an array. `map` becomes an object, with integer
and boolean keys stringified per the specification. Nested messages recurse.

**Well-known types**, a seven-entry table in the emitter keyed on import path:

| Type | JSON form |
|---|---|
| `Timestamp` | RFC-3339 string, always UTC `Z` on output; offsets accepted on input |
| `Duration` | Seconds with up to nine fractional digits and an `s` suffix |
| `FieldMask` | One string of comma-joined camelCase paths |
| `Struct`, `Value`, `ListValue` | Plain JSON |
| Wrappers | The bare scalar |
| `Empty` | `{}` |
| `Any` | Returns `JSON_ANY_UNSUPPORTED` until the type registry lands |

RFC-3339 formatting and parsing, the `FieldMask` path conversion, and base64 go in
the runtime as shared helpers under `foundry/proto/`, not inlined per message — they
are type-shaped, not field-shaped, and inlining would duplicate them once per
referencing message.

## Errors

`ProtobufError` gains five cases. Existing numbering is untouched.

```
JSON_PARSE_FAILED = 7
JSON_TYPE_MISMATCH = 8
JSON_UNKNOWN_FIELD = 9
JSON_VALUE_OUT_OF_RANGE = 10
JSON_ANY_UNSUPPORTED = 11
```

`JSON_PARSE_FAILED` covers a malformed document. `JSON_TYPE_MISMATCH` covers a
well-formed value of the wrong shape for its field. `JSON_VALUE_OUT_OF_RANGE` covers
a number outside a field's domain, including a 32-bit field given a value that does
not fit.

## Generator changes

- `internal/proto/api.go` grows plugin option parsing for `--foundryscript_opt=json`.
  Nothing parses plugin options today.
- New `json_serialize.go` and `json_deserialize.go` beside the existing emitters,
  driven off the same field model `plan.go` already builds.
- New runtime sources under `internal/runtime/data/foundry/proto/` for RFC-3339,
  `FieldMask` paths, and base64.
- `internal/proto/wellknown/gen` forces the option on, so the checked-in
  `wkt/*.pb.fs` regenerate carrying JSON. The drift test in
  `internal/runtime/runtime_test.go` keeps the checked-in output honest.
- Golden tests gain a JSON-enabled variant. The existing golden corpus stays
  JSON-free, so every existing test covers the option's off-path.

## Documented limitations

These go in the README beside the existing open-enum note.

A JSON round trip is lossy where a wire round trip is not. JSON has no unknown-field
preservation, so a member the schema does not recognize is an error on the way in and
has nothing to re-emit on the way out. An unrecognized enum number becomes the
default rather than being retained. A 64-bit integer that arrives as a bare JSON
number loses precision past 2^53, because the engine's JSON parser returns floats.

`Any` has no JSON form yet.

## Out of scope

**`Any` pack, unpack, and JSON.** Needs `Message.type_name()` on the trait plus a
runtime type-URL registry. Filed as #48, ordered after this.

**`Struct` <-> `Variant` and `Timestamp`/`Duration` <-> float seconds.** #43. Further
entries in the same well-known-type emitter table, but conversions rather than a JSON
form, and landable in either order.

**Applying a `FieldMask` to a message**, and mask union and intersection. Not a JSON
problem — it needs runtime reflection over generated messages: field lookup by name,
nested traversal, clearing a field by path. The generator emits none of that. This
design produces the camelCase path conversion a mask needs for its own
serialization and nothing more.
