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
arrive as a JSON number, and the engine's parser hands such numbers back as floats, so
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
`bytes` to and from base64 falls out of `Marshalls`. The engine's parser producing
every number as a double is what forces the 64-bit precision decision above.

## Generated surface

**Amended for engine `0.1.alpha19`.** This section originally specified a runtime
`JsonNode` union and a `Variant` boundary owned by the runtime. The engine has since
grown both the union and the boundary natively, so the runtime defines neither. The
superseded design is recorded at the end of this section, because its reasoning is
what the engine's own type now embodies and a future reader should not re-derive it.

A message gains two members and one trait conformance when the option is on:

```
final class_name X extends RefCounted uses Message, JsonSerializable

func to_json() -> JsonNode
static func from_json(_pb_node: JsonNode) -> JsonResult[X]
```

Both come from the engine's builtin `JsonSerializable` trait. Conforming is not
cosmetic: it is what teaches `JSON.stringify` to lower a message at all, so a
non-conforming message has no route to JSON text.

Text conversion needs no emitted method in either direction.
`JSON.stringify(msg, "", false)` produces the document — the third argument turns off
key sorting, which is on by default, so members come out in field declaration order.
`JSON.parse_to_node(text)` produces a `JsonResult[JsonNode]` to hand to `from_json`.

A private `func _pb_merge_from_json(_pb_node: JsonNode) -> JsonDecodeError?` carries
the decoding, and `from_json` is construct-then-merge. Decoding `repeated`, `map`,
and `oneof` members is merge-shaped, so writing `from_json` as a second
implementation would duplicate it.

### The engine's `JsonNode`

A JSON document is a dynamic value, but it is dynamic over a *closed* set of shapes,
so the engine models it as a tagged union rather than a `Variant`. It is a global
class in `modules/foundry_script/builtin/json_node.fs`, with seven cases whose
ordinals are a wire contract with the native encoder (`FSJsonObjectMarshaller::Tag`)
and must not be reordered:

```
enum_name JsonNode:            # Null=0 Bool=1 Int=2 Float=3 Str=4 Array=5 Object=6
	Null
	Bool(value: bool)
	Int(value: int)
	Float(value: float)
	Str(value: String)
	Array(items: Array[JsonNode])
	Object(entries: Dictionary[String, JsonNode])

	static func array_of(items: Array[JsonNode]) -> JsonNode
	static func object_of(entries: Dictionary[String, JsonNode]) -> JsonNode
	static func of(value: Variant) -> JsonNode
```

The separate `Int` and `Float` cases are what let a proto `int32` render as `1` and a
`double` of the same value as `1.0`; a single numeric case could not do both.

Three further builtins come with it. `JsonSerializable` is a plain trait over `Self`,
not a generic:

```
trait_name JsonSerializable

abstract func to_json() -> JsonNode
abstract static func from_json(node: JsonNode) -> JsonResult[Self]
```

`JsonResult[T]` carries `value: T?` and `error: JsonDecodeError?`, with `ok`, `fail`,
`is_ok`, and `nested(error, key)` — which re-roots a nested failure under a key so a
field decoder can report a path from the document root. `JsonDecodeError` is
`{message: String, path: String}`, where `path` is JSONPath-like: `$.inventory.0.name`.

**`Variant` never enters the runtime or the generated surface.** Both engine
boundaries are typed. The README rule against `Variant` on generated public APIs, and
the two gates that enforce it — the `-> Variant` grep over the output directory in
`tests/foundry/run.sh`, and the whole-runtime assertion in
`internal/runtime/runtime_test.go` — hold unmodified, with no carve-out on either side.

**Why not define our own union anyway.** A second `JsonNode` would collide with a
global class name, would have to be kept in step with the engine's case ordinals to
interoperate with `JSON.stringify` at all, and would buy nothing. Consuming the engine
type also removes a cost this design had previously accepted: serialization no longer
walks two trees, because the native marshaller lowers the node tree in C++.

**Why not reuse `google.protobuf.Value`.** `Value`'s `ValueKindCase` is the same
union, but `Value` also carries `_pb_unknown_fields`, `to_bytes`, `merge_from_bytes`
and the rest of the wire surface. A JSON node owning a protobuf wire encoder is
confusing, and it would couple the JSON API to a well-known type's binding. `Value`'s
own JSON form is a straightforward mapping onto `JsonNode`.

<details>
<summary>Superseded: the runtime <code>JsonNode</code> and its <code>Variant</code> boundary</summary>

Written against engine `0.1.alpha14`, when the engine had no JSON tree type and
`JSON.stringify` / `JSON.parse_string` were the only boundary, both `Variant`-typed.

The runtime would have defined a six-case union at
`internal/runtime/data/foundry/proto/json_node.fs` — `Null`, `Bool(value: bool)`,
`Number(value: float)`, `Text(value: String)`, `List(values: Array[JsonNode])`,
`Object(fields: Dictionary[String, JsonNode])` — plus
`static func to_variant(_pb_node: JsonNode) -> Variant` and
`static func from_variant(_pb_value: Variant) -> (JsonNode?, ProtobufError)`, the one
place in the system that inspected a dynamic type. Generated messages would have
carried `to_json_node`, `to_json_string`, `merge_from_json_node`, and
`from_json_string`, all reporting through `ProtobufError`.

That required narrowing the runtime `Variant` assertion to exempt `json_node.fs` by
file name, and — discovered during #71 — a matching by-name carve-out in
`tests/foundry/run.sh`, since anvil does copy the runtime into the output directory
that gate scans. Both carve-outs are unnecessary now.

The reason for a union over a `Variant` is unchanged and is why the engine's type is
the right thing to consume: a `Variant`-typed generated surface spreads dynamic values
across every message in every project that enables JSON, where a closed union confines
them and buys exhaustiveness checking on every `match` in generated code.

One rejected alternative is worth keeping, because it is tempting and wrong. Naming
the cases after proto types — `Timestamp(String)`, `Int32Value(int)` — carries
protobuf provenance into a type that models JSON shape: a `Timestamp`, a `Duration`,
and a `FieldMask` all serialize to a JSON string, and the document does not remember
which produced it. Worse, a union keyed on proto types would have to stay open, since
every user-defined message would need a case.

</details>

## Mapping

Emitted per field, branching on kind the way the existing emitters do.

**Names.** `[json_name = "..."]` when the field carries it, otherwise the
specification's camelCase derivation from the proto name. Field options already parse
into a generic map (`internal/proto/internal/parser/messages.go`), so honoring
`json_name` costs nothing extra. The parser accepts either spelling on input.

**Output presence.** proto3 zero values are omitted. An `optional` field, a message
field, and a oneof member are emitted only when present. A wrapper that is present
but null writes an explicit `null`.

**Scalars**, in terms of the `JsonNode` case the emitter produces and the cases the
decoder accepts.

| Proto | Emit | Accept |
|---|---|---|
| `int32`, `uint32`, `fixed32`, `sfixed32`, `sint32` | `Int` | `Int`, `Str`, integral `Float` |
| `int64`, `sint64`, `sfixed64` | `Str(str(value))` | `Str` (exact), `Int`, `Float` (lossy) |
| `uint64`, `fixed64` | `Str(JsonUint64.format(value))` | `Str` via `JsonUint64.parse` (exact), non-negative `Int`, `Float` (lossy) |
| `float`, `double`, finite | `Float` | `Float`, `Int` |
| `float`, `double`, non-finite | `Str("NaN")` / `Str("Infinity")` / `Str("-Infinity")` | those three strings |
| `bool` | `Bool` | `Bool` |
| `string` | `Str` | `Str` |
| `bytes` | `Str`, base64 via `JsonBase64` | `Str`, base64 |

Three rows are forced by measured engine behavior rather than by the specification
alone.

**A non-finite float must never reach the `Float` case.**
`JSON.stringify(JsonNode.Float(NAN))` emits `null` with a warning, `+INF` emits
`1e99999`, and `-INF` emits `-1e99999`. None is valid canonical output, so the
emitter substitutes the string form itself rather than relying on the encoder.

**A 64-bit integer arriving as a bare JSON number arrives in the wrong case, not
merely rounded.** Emission is fine: `JsonNode.Int(9223372036854775807)` stringifies to
the exact digits. Parsing is not. `JSON.parse_to_node("9223372036854775807")` returns
a `Float`, because the parser produces a double and only folds a whole number into
`Int` when an `int64_t` can hold the *rounded* double — which that value cannot. Sent
as a JSON string it round-trips exactly through `String.to_int()`. So a decoder for a
64-bit field must accept `Float` and document the loss; it cannot assume `Int`.

**An unsigned 64-bit value has no signed spelling above 2^63, so neither `str()` nor
`String.to_int()` may carry it.** A Foundry `int` is signed and the binding holds a
`uint64` in one, so the widest value is the bit pattern `-1` and `str()` prints it as
`"-1"` — a document no conformant peer agrees with across the whole top half of the
range. `String.to_int("9223372036854775808")` wraps to the smallest signed value
rather than reporting, so the same trap is on the way back in. Both directions
therefore go through the `JsonUint64` runtime helper, which reads and writes the bit
pattern as the unsigned value it stands for: it splits the value into its quotient by
ten and its last digit, neither of which overflows a signed int, and reassembles it
with a shift. `uint64` and `fixed64` are the only two types this applies to; `int64`,
`sint64`, and `sfixed64` are signed, and `str()` is exact for them.

`JsonNode.Float(1.0)` stringifies as `1.0`, where protobuf's own emitters write `1`
for a whole double. Both are accepted by conformant parsers, and `1.0` is kept: it is
what the case set gives for free, and it keeps `double` and `int32` visibly distinct
in the golden corpus.

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

**Amended for engine `0.1.alpha19`.** The JSON path reports `JsonDecodeError`, not
`ProtobufError`. The wire path is unchanged.

`JsonSerializable.from_json` returns `JsonResult[Self]`, so a `ProtobufError` could
only be smuggled through a message string. `JsonDecodeError` is also the better
carrier: it records where in the document the failure was, and `JsonResult.nested`
re-roots a nested failure so a field decoder reports a path from the document root —
`$.inventory.0.name` — which a flat enum cannot express. A malformed document is
already a `JsonResult` failure out of `JSON.parse_to_node`, so both failure modes
arrive in one shape.

The five JSON cases that the foundations epic added to `ProtobufError` are now
vestigial:

```
JSON_PARSE_FAILED = 7
JSON_TYPE_MISMATCH = 8
JSON_UNKNOWN_FIELD = 9
JSON_VALUE_OUT_OF_RANGE = 10
JSON_ANY_UNSUPPORTED = 11
```

They stay, rather than being renumbered out — the enum is public and the numbering is
load-bearing for the wire cases around them. The emitter uses their names as the
leading text of a `JsonDecodeError.message`, so the categories stay greppable: a
well-formed value of the wrong shape for its field reports `JSON_TYPE_MISMATCH`, a
number outside a field's domain reports `JSON_VALUE_OUT_OF_RANGE`, and so on.

## Generator changes

- `internal/proto/api.go` grows plugin option parsing for `--foundryscript_opt=json`.
  Nothing parses plugin options today.
- New `json_serialize.go` and `json_deserialize.go` beside the existing emitters,
  driven off the same field model `plan.go` already builds.
- No new runtime source for the JSON tree: the engine owns `JsonNode`,
  `JsonSerializable`, `JsonResult`, and `JsonDecodeError`. RFC-3339, `FieldMask`
  paths, and base64 already landed in the foundations epic.
- `engine_reserved_types.gen.go` must be regenerated for the engine in use;
  `sync-foundry-engine-types.sh check` runs at the top of `tests/foundry/run.sh`, so a
  stale table fails `task foundry:test` outright.
- The four JSON builtins are *script* classes, so they do not appear in
  `extension_api.json` and the regenerated table does not contain them. They need a
  hand-maintained entry in the generator's reserved-name set, or a proto message named
  `JsonNode` would silently shadow the builtin.
- Every new runtime type also needs an entry in `runtimeTypeNames` in
  `internal/proto/internal/foundryscript/generator/names.go`, or
  `TestRuntimeTypeNamesCoverEveryExportedRuntimeType` fails.
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
number loses precision past 2^53, because the engine's parser produces a double; past
that point it does not even arrive as a `JsonNode.Int`. Emitting 64-bit integers as
strings, which the canonical mapping requires, is what keeps our own output exact.

The wire path is a separate surface and is not covered by this. A `uint64` at or
above 2^63 reads back from `to_bytes`/`from_bytes` as a negative `int`, because that
is the bit pattern the member holds; the JSON text is unsigned regardless, so the two
should not be read as the same limitation.

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
