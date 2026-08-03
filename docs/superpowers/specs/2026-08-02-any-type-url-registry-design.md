# Any Type-URL Registry Design

**Status:** Approved

**Parent epic:** [cafecito-games/Foundry-Tools#48](https://github.com/cafecito-games/Foundry-Tools/issues/48)

**Native sub-issues:**

- [#98 — Generate protobuf message identity and class-handle construction](https://github.com/cafecito-games/Foundry-Tools/issues/98)
- [#99 — Add the explicit typed Any registry and deterministic errors](https://github.com/cafecito-games/Foundry-Tools/issues/99)
- [#100 — Implement Any pack, type checks, and registry-driven unpack](https://github.com/cafecito-games/Foundry-Tools/issues/100)
- [#101 — Implement ordinary-message Any ProtoJSON](https://github.com/cafecito-games/Foundry-Tools/issues/101)
- [#102 — Implement custom-WKT and nested-Any ProtoJSON](https://github.com/cafecito-games/Foundry-Tools/issues/102)
- [#103 — Complete Any integration, conformance, regeneration, and documentation](https://github.com/cafecito-games/Foundry-Tools/issues/103)

## Context

`google.protobuf.Any` stores a serialized protobuf message beside a type URL.
The current Foundry Script binding exposes only those two generated fields and
deliberately reports `JSON_ANY_UNSUPPORTED` from its JSON surface. Packing,
checking, unpacking, and canonical ProtoJSON all require a runtime mapping from
the type name in the URL to a generated message class.

Foundry engine issue
[cafecito-games/Foundry#1524](https://github.com/cafecito-games/Foundry/issues/1524)
added nested `Type[T]` support and preserved class-handle semantics through
typed containers. The installed binary used by this repository is:

```text
/Users/christian/bin/foundry
0.1.dev.custom_build.c11e3a080
```

That build includes commit `c11e3a0809`, whose end-to-end fixtures cover nested
class handles in analyzed source, tooling, persistence, and bytecode.

## Engine Capability Evidence

Focused probes against the installed binary established the exact surface this
design may rely on:

| Question | Result |
|---|---|
| Can a generated message handle enter `Dictionary[String, Type[Message]]`? | Yes. |
| Does a retrieved value remain the original class handle? | Yes. |
| Can `Type[Message].new()` construct the concrete type? | No. The analyzer reports `Cannot construct trait "Message"`. |
| Can a trait-declared abstract static factory construct it? | Yes, and the returned `Message` retains the concrete dynamic class. |
| Can `Type[Message]` directly narrow to the unrelated `Type[JsonSerializable]`? | No. The analyzer rejects the direct test and cast. |
| Can a private dynamic boundary check and narrow the handle? | Yes. A handle copied temporarily through `Variant` can be tested and cast to `Type[JsonSerializable]`, then invoke the static `from_json` requirement. |
| What does a missing dictionary lookup produce? | `null`; the missing value fails a `Type[Message]` test and casts to null. |
| Are unrelated handles and instances rejected by the typed dictionary? | Yes. Runtime validation separately reports an incompatible class handle and an instance where a handle is required. |
| Can static and instance functions both be named `type_name`? | No. Foundry rejects the duplicate name. |
| Can a separate static identity feed the instance identity without duplicating the string? | Yes. |

The registry therefore uses real class handles throughout. A narrow private
`Variant` temporary is permitted only at the JSON cross-trait capability seam;
it is not stored, returned, accepted by a public API, or used as a substitute
for `Dictionary[String, Type[Message]]`.

## Goals

- Give every generated message its fully qualified protobuf identity.
- Register generated message class handles explicitly at runtime.
- Pack exact message bytes into a canonical type URL.
- Check Any identity without decoding.
- Resolve, construct, and decode a concrete registered message without the
  caller naming or instantiating the destination.
- Implement canonical ProtoJSON for ordinary messages and custom-form
  well-known types.
- Preserve supplied foreign type-URL prefixes on JSON input.
- Make malformed, missing, conflicting, unsupported, and corrupt cases
  deterministic and transactional.
- Cover generation, runtime embedding, Foundry behavior, integration,
  conformance, regeneration, and documentation.

## Non-goals

- Network fetching or a protobuf type server.
- Automatic class discovery or generated load-time self-registration.
- A general reflection system.
- Prototype instances retained as factories.
- Per-message `Callable` factories or string-to-`Callable` adapters.
- A public or stored `Variant` registry.
- A caller-named `unpack_into(target)` API.
- Changes to the Foundry engine unless a separate minimal reproduction proves
  a new engine defect blocks this design. No such blocker exists now.
- Changing ordinary wire or ProtoJSON behavior outside the generated identity
  and construction members required by Any.

## Registry Model

### Selected: runtime-wide registry with explicit registration

The runtime ships `foundry.proto.AnyTypeRegistry`. Applications populate it
with class handles before performing an operation that needs resolution:

```foundryscript
AnyTypeRegistry.clear()
assert(AnyTypeRegistry.register(Player) == ProtobufError.OK)
assert(AnyTypeRegistry.register(Duration) == ProtobufError.OK)
```

The registry starts empty and owns one static mapping:

```foundryscript
Dictionary[String, Type[Message]]
```

Its key is the fully qualified protobuf name, not the complete type URL. The
registration call derives the key from the handle rather than accepting an
unverified string.

This model keeps the existing parameterless `JsonSerializable.to_json()` and
static `from_json()` surface usable. Registration order is explicit, class
loading is caused by the application's handle references, and tests can reset
global state deliberately.

### Rejected: generated self-registration

Self-registration would add a static initializer or load hook to every
generated message. Only bindings the engine happened to load would register,
initialization order would become observable, generated files would acquire
hidden global side effects, and test isolation would depend on script loading.
These costs provide no benefit now that explicit class-handle registration is
concise.

### Rejected: caller-owned registry

A caller-owned registry is locally isolated but cannot satisfy ordinary
`JSON.stringify(any)` or `Any.from_json(node)`: neither `JsonSerializable`
method accepts context. Adding parallel methods with registry parameters would
leave the standard JSON entry points broken. A hidden current-registry slot
would merely recreate the selected global model with ambiguous ownership.

## Generated Message Contract

The `foundry.proto.Message` trait gains three requirements:

```foundryscript
abstract static func create_message() -> Self
abstract static func protobuf_type_name() -> String
abstract func type_name() -> String
```

The existing wire requirements remain unchanged. Every generated message,
including nested messages, implements the new requirements:

```foundryscript
static func create_message() -> Player:
	return Player.new()

static func protobuf_type_name() -> String:
	return "example.Player"

func type_name() -> String:
	return Player.protobuf_type_name()
```

The descriptor's full name is the sole identity source. A nested message uses
protobuf notation such as `example.Outer.Inner`, regardless of how its Foundry
Script class is hosted or escaped. The instance method required by #48 remains
available, while the differently named static method lets a class handle expose
the same identity. The literal appears only in the static implementation.
The `Message` contract requires this identity to be a valid protobuf full name;
generated conformers satisfy that contract by construction, while registration
validates hand-written conformers at runtime.

The factory is required because `Type[Message].new()` attempts to construct the
trait rather than the represented class. A static trait requirement preserves
the concrete `Self` return at each generated implementation and widens soundly
to `Message` after registry resolution.

`create_message`, `protobuf_type_name`, and `type_name` join the generator's
reserved member inventory so schema fields cannot replace them. Runtime names
such as `AnyTypeRegistry` are discovered and reserved through the existing
runtime-export mechanism.

## Registry API and State

The public surface is intentionally small:

```foundryscript
static func register(message_type: Type[Message]) -> ProtobufError
static func clear() -> void
```

Resolution helpers remain private runtime implementation details.

Registration proceeds as follows:

1. Read `message_type.protobuf_type_name()`.
2. Validate it as a protobuf full name.
3. If the name is absent, insert the class handle.
4. If the same handle already occupies the name, return `OK` without mutation.
5. If a different handle occupies the name, return
   `ANY_REGISTRY_CONFLICT` without mutation.

A valid protobuf full name is one or more dot-separated identifiers. Each
identifier begins with an ASCII letter or underscore and continues with ASCII
letters, digits, or underscores. Empty segments, leading or trailing dots, and
other characters are invalid. Generated names already satisfy this rule; the
runtime validation protects hand-written `Message` conformers.

`clear()` replaces the mapping with an empty typed dictionary. There is no
unregister operation: removing one entry adds API and state transitions without
a demonstrated use case. Applications that need a new registration scope clear
and rebuild it explicitly.

Pack and `is_type` do not need registry membership. Unpack and both Any JSON
directions do. Well-known bindings are not privileged or registered
automatically; an application explicitly registers each WKT it plans to
resolve.

## Type-URL Rules

Packing writes:

```text
type.googleapis.com/<fully-qualified-protobuf-name>
```

Resolution extracts the substring after the final `/`. A URL with no slash is
a bare protobuf name and is valid. Any prefix before the last slash is accepted
and ignored for identity. The extracted name must pass the same protobuf-name
validation as registration.

The following are distinct:

| Input | Result |
|---|---|
| `type.googleapis.com/example.Player` | Resolves `example.Player`. |
| `https://peer.example/types/example.Player` | Resolves `example.Player`. |
| `example.Player` | Resolves `example.Player`. |
| empty string | Invalid type URL. |
| `peer.example/` | Invalid type URL. |
| `peer.example/example..Player` | Invalid type URL. |
| valid but absent `example.Missing` | Valid URL, unregistered type. |

JSON encoding emits the `Any.type_url` already stored on the message. JSON
decoding resolves by the trailing name but preserves the supplied string
verbatim in the decoded `Any`. Only `pack` canonicalizes the prefix.

## Wire API

The generated `foundry.proto.wkt.Any` class gains regenerable semantic helpers:

```foundryscript
static func pack(message: Message) -> Any
func is_type(message_type: Type[Message]) -> bool
func unpack() -> (Message?, ProtobufError)
```

### Pack

`pack` constructs a new Any, writes the canonical URL from
`message.type_name()`, and copies the exact result of `message.to_bytes()` into
`value`. It does not require the type to be registered. Its non-error return
relies on the `Message` identity contract; a hand-written conformer that reports
an invalid name produces an Any that later URL-based operations reject.

### Type check

`is_type` validates and extracts the URL's trailing name, then compares it with
`message_type.protobuf_type_name()`. It returns false for malformed URLs or a
different name. It never reads or decodes `value`, so corrupt bytes cannot
affect the result.

### Unpack

`unpack` validates the URL, resolves its trailing name, invokes the registered
handle's static factory, and calls `merge_from_bytes(value)` on that fresh
instance. Success returns `(message, OK)`. The static return type is `Message?`,
the strongest type sound for a runtime-selected class; the object's dynamic
type is the registered generated class.

An invalid URL or missing registration returns null with the corresponding Any
error. A corrupt payload returns null with the exact existing wire error from
`merge_from_bytes`. The partially decoded fresh instance is discarded. The
source Any and every caller-owned object remain unchanged.

## JSON Capability Resolution

JSON generation remains optional. A wire-only generated class uses `Message`;
a JSON-enabled generated class continues to use both `Message` and the engine's
`JsonSerializable` trait.

Foundry does not allow a direct cast between unrelated `Type[Message]` and
`Type[JsonSerializable]`. The registry therefore uses a private checked seam:

1. Resolve the value as `Type[Message]` from the typed dictionary.
2. Copy that handle into a local `Variant`.
3. Test and cast the local value to `Type[JsonSerializable]`.
4. Discard the temporary after the operation.

Encoding similarly narrows the freshly unpacked instance through a private
local dynamic check before calling `to_json()`.

The temporary is not a registry model or a public bridge. The stored mapping,
registration parameter, wire APIs, and returned objects all retain their
strong types. A registered wire-only message therefore supports pack, type
checks, and unpack while reporting `ANY_JSON_UNSUPPORTED` from Any JSON
conversion.

A combined `JsonMessage` subtrait was considered. Although a local probe can
use it, global named-trait class-handle compatibility requires unrelated
instance-type context to resolve reliably in the installed analyzer. The
direct `Message, JsonSerializable` conformance plus a private checked narrowing
has predictable behavior and does not require engine work.

## Canonical Any ProtoJSON

The official ProtoJSON mapping divides embedded messages into ordinary object
forms and well-known types with custom forms.

### Encoding ordinary messages

For an ordinary registered JSON-enabled message:

1. Validate and resolve `type_url`.
2. Confirm the registered handle is JSON-capable.
3. Unpack the wire bytes into a fresh concrete object.
4. Call its canonical `to_json()`.
5. Require an object result.
6. Build a new object with `"@type"` inserted first, followed by the embedded
   object's entries in their existing order.

The embedded object is not mutated. Example:

```json
{"@type":"type.googleapis.com/example.Player","name":"Ada"}
```

### Encoding custom-form well-known types

The existing `wellKnownJSONForms` table remains the sole source for deciding
whether an Any payload uses a `value` member. The generator derives the Any
special-form name set from that table rather than maintaining a second list.

The forms are:

| Embedded type | Any payload form |
|---|---|
| `Timestamp`, `Duration`, `FieldMask` | `value` contains their canonical string. |
| Scalar wrappers | `value` contains the canonical scalar. |
| `Struct` | `value` contains its object. |
| `Value` | `value` contains the represented JSON value, including null. |
| `ListValue` | `value` contains its array. |
| `Any` | `value` contains the inner canonical Any object. |
| `Empty` | Ordinary form; no `value` member. |

Example:

```json
{"@type":"type.googleapis.com/google.protobuf.Duration","value":"1.250s"}
```

Nested Any therefore has an outer object whose `value` is the complete inner
Any object.

### Decoding

`Any.from_json(node)` and `_pb_merge_from_json(node)` follow one transactional
path:

1. `JsonNode.Null` produces an empty Any, matching the repository's existing
   message-null convention.
2. An empty object produces an empty Any, matching protobuf conformance
   behavior for an unset Any.
3. A nonempty object must contain exactly one representable `@type` member with
   a nonempty string.
4. Validate the trailing protobuf name and resolve its registered handle.
5. Confirm the handle is JSON-capable.
6. For an ordinary message, copy every entry except `@type` into a payload
   object and pass it to the concrete static `from_json`.
7. For a custom-form WKT, require `value`, reject other members, and pass the
   value node to the concrete static `from_json`.
8. Require a successful, non-null concrete result and serialize it with
   `to_bytes()`.
9. Only now assign the supplied URL and bytes to the destination Any.

An ordinary message may legitimately have a protobuf field whose JSON name is
`value`; it remains in the copied payload. A special-form object reserves only
`@type` and `value`.

Identical duplicate keys cannot survive the engine's text-to-`JsonNode.Object`
boundary. The binding rejects duplicates wherever the node representation
retains them, and documentation states that identical raw object-key
duplicates cannot be diagnosed after parsing. There is no alternate spelling
for `@type`.

## JSON Error Paths

Any errors use paths that identify the failing part of the canonical document:

| Failure | Path |
|---|---|
| Root is neither null nor object | `$` |
| Missing, non-string, empty, malformed, or unregistered `@type` | `$["@type"]` |
| Registered type lacks JSON support | `$["@type"]` |
| Missing custom-form `value` | `$.value` |
| Custom-form scalar or message error | Existing path re-rooted below `$.value` |
| Ordinary embedded field error | Original embedded root path, such as `$.level` |
| Unknown extra custom-form member | That member's root path |
| Nested Any type error | Recursive path such as `$.value["@type"]` |

The ordinary payload stays at the Any object's root, so its decoder errors do
not gain an artificial parent segment. The custom payload actually lives under
`value`, so `JsonResult.nested` re-roots its errors there.

## Error Model and Transactionality

Existing numeric values remain fixed. In particular,
`JSON_ANY_UNSUPPORTED = 11` stays in the enum but becomes deprecated and unused.
The current subsequent values also remain fixed:

```text
STRUCT_KEY_NOT_STRING = 12
STRUCT_VALUE_UNREPRESENTABLE = 13
WELL_KNOWN_TIME_OUT_OF_RANGE = 14
```

New values append after them:

```text
ANY_TYPE_NAME_INVALID = 15
ANY_REGISTRY_CONFLICT = 16
ANY_TYPE_URL_INVALID = 17
ANY_TYPE_NOT_REGISTERED = 18
ANY_JSON_UNSUPPORTED = 19
```

Their uses are:

| Error | Use |
|---|---|
| `ANY_TYPE_NAME_INVALID` | A hand-written `Message` handle reports an invalid protobuf name during registration. |
| `ANY_REGISTRY_CONFLICT` | A different handle is already registered for the same name. |
| `ANY_TYPE_URL_INVALID` | A wire or JSON operation receives an empty, trailing-slash, or invalid trailing name. |
| `ANY_TYPE_NOT_REGISTERED` | The URL is valid but no handle is registered for its name. |
| `ANY_JSON_UNSUPPORTED` | The registered message lacks `JsonSerializable`. |

Registry and wire APIs return the relevant enum values directly. JSON decoding returns
`JsonDecodeError` whose message begins with the relevant category and whose
path follows the table above. JSON encoding has no error return in the engine's
trait, so it calls `push_error` with the category and returns `JsonNode.Null`.

The JSON encoder's deterministic validation order is URL, registration, JSON
capability, wire decode, then embedded JSON rendering. Consequently a
wire-only registered type reports `ANY_JSON_UNSUPPORTED` before inspecting its
payload bytes.

Every mutating operation is transactional:

- Failed registration leaves the registry unchanged.
- Failed unpack discards its fresh partial instance.
- Failed JSON merge leaves the destination Any unchanged.
- Encoding never mutates the source Any or embedded message.
- Unknown types preserve their original URL and bytes as opaque Any fields.

## Generator and Runtime Integration

- `message.fs` gains the new abstract requirements.
- The general message generator emits identity and construction witnesses for
  every message, regardless of the JSON option.
- Generated member collision logic reserves the new witness names.
- `any_type_registry.fs` is a checked-in runtime component under
  `foundry/proto`.
- Runtime embedding and export tests cover the new component and public name.
- `wellknown_semantics.go` emits Any wire helpers and any native semantic
  helpers that must remain regenerable.
- `json_serialize.go` and `json_deserialize.go` replace the unsupported Any
  branches with the approved behavior.
- `json_wellknown.go` exposes a generator helper that derives Any's `value`
  wrapper classification from `wellKnownJSONForms`.
- `task gen-wkt` is the only mechanism used to change checked-in
  `internal/runtime/data/foundry/proto/wkt/*.pb.fs` files.
- Every intentionally affected example and golden regenerates from the
  generator; no generated file is edited manually.

## Test Strategy

Implementation follows TDD within each native child issue. The first child
adds a focused Foundry fixture expressing the public handle API and records its
failure before production changes. Each later child adds its focused Go and
Foundry failures before its implementation slice.

Coverage includes:

- Message identity for packages, top-level messages, and nested messages.
- Generated factory construction and concrete dynamic types.
- Same-handle idempotence, conflicting registration, invalid identity,
  missing lookup, and clear/reset isolation.
- Typed-container rejection of unrelated handles and instances.
- Canonical pack bytes and URL, foreign prefixes, bare names, malformed URLs,
  and unregistered types.
- `is_type` true and false without payload decoding.
- Registry-driven unpack and exact corrupt-wire errors.
- Ordinary-message Any JSON in both directions.
- Timestamp, Duration, wrappers, Struct, Value, ListValue, FieldMask, Empty,
  and nested Any in both directions.
- Missing, empty, non-string, malformed, unknown, and unregistered `@type`.
- Missing and malformed special-form `value`.
- Embedded unknown-field, duplicate-where-representable, type, range, and
  oneof errors with exact paths.
- A registered wire-only message that lacks `JsonSerializable`.
- Empty and null Any JSON input.
- Stable WKT regeneration, runtime embedding, examples, and goldens.
- No changes to ordinary non-Any wire or JSON behavior.

The official
[ProtoJSON specification](https://protobuf.dev/programming-guides/json/) and
upstream protobuf conformance cases are authoritative for canonical output and
accepted input. Repository-specific choices such as strict unknown fields and
the `JsonNode` duplicate-key boundary remain documented.

## Epic Delivery

Issue #48 is the native parent epic. Its immutable child set is #98 through
#103. Dependencies are deliberately sequential where files and semantics
overlap:

1. #98 establishes identity and construction.
2. #99 builds the registry on that contract.
3. #100 adds wire Any behavior using both foundations.
4. #101 replaces ordinary Any JSON unsupported behavior.
5. #102 extends that JSON path to custom WKTs and nested Any.
6. #103 supplies cross-cutting conformance, regeneration, documentation, and
   final review convergence.

Each child has its own executable TDD plan:

- [#98: message identity and construction](../plans/2026-08-02-any-message-identity.md)
- [#99: explicit type registry](../plans/2026-08-02-any-type-registry.md)
- [#100: wire API](../plans/2026-08-02-any-wire-api.md)
- [#101: ordinary-message ProtoJSON](../plans/2026-08-02-any-ordinary-protojson.md)
- [#102: WKT and nested-Any ProtoJSON](../plans/2026-08-02-any-wkt-protojson.md)
- [#103: conformance, regeneration, and documentation](../plans/2026-08-02-any-conformance-docs.md)

Every child owns focused tests for its behavior. #103 is a completion gate,
not a place to defer unit coverage from earlier children.

The design document merges before implementation children begin so each child
branches from a base containing the approved contract. Each child is delivered
through a focused Conventional Commit and PR that closes only that child.
Issue #48 closes only after all six native children merge, final verification
passes on the resulting base branch, both independent reviews converge, and all
child worktrees and branches are cleaned up.

Final fresh verification includes:

```text
task gen-wkt
task gen-wkt
git diff --exit-code
task fmt
task ci
task integration
FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test
git diff --check origin/main...HEAD
```

The second generation and clean diff prove regeneration stability.
