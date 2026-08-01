<p align="center">
  <img src="assets/logo/anvil-banner.svg" alt="Anvil — Foundry Tooling" width="720">
</p>

# Foundry Tools

Tooling for Foundry Engine projects.

This repository ships package management and protobuf code generation for
Foundry Script:

- `anvil`: direct CLI for Foundry project tooling.
- `protoc-gen-foundryscript`: protoc and Buf plugin.

Generated `.pb.fs` files use Foundry Script namespaces, traits, generics,
nullable types, and typed collections. Public generated APIs avoid `Variant`
unless a Foundry Engine API requires it.

Supported protobuf constructs and how they map:

| Protobuf | Foundry Script |
|---|---|
| `message` | `final class_name X extends RefCounted uses Message` |
| `enum` | `enum_name X` hosting `to_wire()` / `from_wire() -> Self?` |
| singular scalar | plain field, proto3 zero-value presence |
| `optional` scalar | `String?` and friends, explicit presence |
| message field | `X?`, length-delimited submessage |
| `repeated` | `Array[T]`, packed for varint scalars |
| `map<K, V>` | `Dictionary[K, V]` |
| `oneof` | file-level tagged union, `X? = null` member |
| nested `message` / `enum` | inner `final class` / `enum`, referenced as `Outer.Inner` |
| imported type | reference plus an `import` of the dependency's namespace |
| `google.protobuf.*` | `foundry.proto.wkt.X` from the runtime, never generated |
| unknown field | kept verbatim in `_unknown_fields` and re-emitted |

proto3 enums are open, so `from_wire` returns `null` for a number the schema
has no case for. A singular or `optional` enum field keeps that number's bytes
and writes them back in the field's own position, so a decode/re-encode round
trip is lossless for a peer on a newer schema; assigning the field supersedes
the retained value. A `repeated` or map-valued enum cannot do this — the raw
number has nowhere to live in an `Array[T]` or `Dictionary[K, T]`, and parking
it in the unknown-field buffer would reorder the sequence or flip which record
wins for a duplicate map key — so an unrecognized value there takes the enum's
default instead.

## Well-known types

The `google/protobuf` well-known types ship as Foundry Script in the runtime
under `foundry.proto.wkt` rather than being generated per project. Generating
them per project would give every project its own incompatible `Timestamp`, so
a library and its consumer could not exchange one.

Importing `google/protobuf/timestamp.proto` therefore produces no output of its
own; the referencing file gets `import foundry.proto.wkt` and refers to
`Timestamp` directly. The seven supported files are `any`, `duration`, `empty`,
`field_mask`, `struct`, `timestamp`, and `wrappers`. Any other
`google/protobuf` file is rejected with a diagnostic rather than generated,
which would otherwise produce a second copy of a type the runtime already
defines.

A file is recognized as well-known by its import path — its path relative to an
include root, which is how protoc names it — and never by where the copy sits on
disk. A repo that vendors the protos and runs `anvil proto generate -I vendor
vendor/google/protobuf/timestamp.proto` is naming
`google/protobuf/timestamp.proto` and gets the runtime's bindings, while
`anvil proto generate -I . myorg/google/protobuf/timestamp.proto` is naming a
schema of its own and gets bindings generated for it. With no `-I` at all a
relative path is its own import path, as it is for protoc, so
`anvil proto generate vendor/google/protobuf/timestamp.proto` names
`vendor/google/protobuf/timestamp.proto` and generates it; passing `-I vendor`
is what names the runtime's copy instead. Only a path with no relative spelling
to fall back on — an absolute one, or one that climbs out of every include root
— is an error when it spells a `google/protobuf` path: with nothing to resolve
it against it could be either file, and both guesses fail silently.

A schema may still declare its own `Timestamp`. The import is emitted only for
a file that references a well-known type, and a local declaration shadows the
imported one, so nothing is renamed.

What a schema may not do is generate *into* a namespace the runtime ships.
`option (foundrytools.namespace) = "foundry.proto.wkt"` — or `"foundry.proto"` —
would write bindings to the same paths the runtime files occupy, and the runtime
is written last, so the schema would be silently discarded. Both are rejected
with a diagnostic naming the namespace. Only those exact namespaces are
reserved: `foundry.proto.wkt.mine` generates into its own directory and is fine,
and so is any other namespace beginning with `foundry`.

These are plain message bindings today: they round-trip correctly on the wire,
but carry none of the semantics the types imply. `Struct` will not convert to a
`Dictionary`, `Timestamp` will not convert to seconds, and `Any` will not pack
or unpack — those are designed and tracked in #43, blocked on two Foundry
retroactive-conformance gaps
([Foundry#1376](https://github.com/cafecito-games/Foundry/issues/1376),
[Foundry#1377](https://github.com/cafecito-games/Foundry/issues/1377)).

The scalar wrappers are not mapped onto nullable scalars. proto3 `optional`
already gives a scalar explicit presence, and the message form is needed anyway
wherever a wrapper appears as a map value, a repeated element, or a `oneof`
member.

## JSON

Behind `--foundryscript_opt=json` (`anvil proto generate --json`), a generated
message additionally conforms to the engine's `JsonSerializable` trait:

```
final class_name X extends RefCounted uses Message, JsonSerializable

func to_json() -> JsonNode
static func from_json(_pb_node: JsonNode) -> JsonResult[X]
```

`JsonNode`, `JsonSerializable`, `JsonResult[T]`, and `JsonDecodeError` are engine
builtins, not types this repository defines. Nothing is emitted for text
conversion: go to text with `JSON.stringify(msg, "", false)` — the third
argument turns off key sorting so members come out in field declaration order
— and parse with `JSON.parse_to_node(text)`, which returns a
`JsonResult[JsonNode]` to hand to `from_json`. The full proto3-to-JSON mapping
is specified in
[`docs/superpowers/specs/2026-07-31-proto3-canonical-json-design.md`](docs/superpowers/specs/2026-07-31-proto3-canonical-json-design.md).

A JSON round trip is lossy in ways a wire round trip is not:

- **No unknown-field preservation.** A member the schema does not recognize is
  an error on the way in and has nothing to re-emit on the way out. An
  unrecognized enum number becomes the default rather than being retained,
  unlike the wire path, which keeps it in `_pb_unknown_fields`.
- **A bare 64-bit JSON number loses precision past 2^53.** The engine's
  parser produces a double, so a large integer literal arrives as a
  `JsonNode.Float`, never an `Int`. Our own output emits 64-bit integers as
  JSON strings, which the canonical mapping requires, so output round-trips
  exactly; the loss only applies to a bare number sent by a peer.
- **`Any` has no JSON form yet** (#48).
- **Errors use a different shape than the wire path.** The JSON path reports
  `JsonDecodeError`, carrying a JSONPath-like location such as
  `$.inventory.0.name`, while the wire path keeps returning `ProtobufError`.
  `ProtobufError`'s five JSON cases (`JSON_PARSE_FAILED` through
  `JSON_ANY_UNSUPPORTED`, 7 through 11) are retained for numbering but are no
  longer returned directly; they survive only as the leading text of a
  `JsonDecodeError.message`, so the categories stay greppable.
- **A `uint64` at or above 2^63 reads back negative from `to_bytes` /
  `from_bytes`, even though its JSON text is correct and unsigned.** The JSON
  mapping emits `uint64` and `fixed64` as unsigned decimal strings (for
  example `"18446744073709551615"`), byte-identical to protobuf's own
  `JsonFormat`. The wire codec is unaffected by this and still surfaces such
  values as a negative `int`, because Foundry Script `int` is signed 64-bit —
  this is a separate, pre-existing limitation of the wire path, not something
  the JSON mapping fixes.
- **Strict input parsing is deliberate.** A quoted exponent or a
  non-canonical decimal spelling (`"1e3"`, `"007"`, `"+5"`) is refused for an
  integer field; the canonical form is a plain decimal string.

## Install

Homebrew:

```bash
brew install --cask cafecito-games/tap/foundry-tools
```

Go:

```bash
go install github.com/cafecito-games/foundry-tools/cmd/anvil@latest
go install github.com/cafecito-games/foundry-tools/cmd/protoc-gen-foundryscript@latest
```

## Direct CLI

```bash
anvil proto generate -I proto -o foundry/generated proto/player.proto
```

## Package Manager

`anvil pkg` installs Foundry packages declared in `packages.toml` into a
project's `addons/` directory and writes `packages.lock` for reproducible
installs.

Project layout:

```text
game/
  project.foundry
  packages.toml
  packages.lock
  addons/
    my_package/
```

Create a starter manifest next to `project.foundry`:

```bash
anvil pkg init
```

Add and install a package:

```bash
anvil pkg add --name my_package \
  --source git \
  --url https://github.com/org/my_package.git \
  --version v1.0.0 \
  --source-path addons/my_package
```

Install, update, remove, and list packages:

```bash
anvil pkg install
anvil pkg update
anvil pkg update my_package
anvil pkg remove my_package
anvil pkg list
```

Supported sources:

- `git`: clone a Git repository at a tag, branch, or commit SHA.
- `github-release`: download one asset from a GitHub release.
- `archive`: download a direct HTTP(S) zip, `.tar.gz`, or `.tgz` archive.

Example `packages.toml`:

```toml
[packages.my_package]
source = "git"
url = "https://github.com/org/my_package.git"
version = "v1.0.0"
source_path = "addons/my_package"
install_as = "my_package"
exclude = ["editor_only"]
```

Commit both `packages.toml` and `packages.lock`. `anvil pkg install` honors
existing lock pins when the manifest entry has not changed; `anvil pkg update`
intentionally re-resolves pins.

## protoc

```bash
protoc \
  --plugin=protoc-gen-foundryscript="$(which protoc-gen-foundryscript)" \
  --foundryscript_out=foundry/generated \
  -I proto \
  proto/player.proto
```

## Buf

```yaml
version: v2
plugins:
  - local: protoc-gen-foundryscript
    out: foundry/generated
```

Run:

```bash
buf generate
```

## Foundry Options Proto

Print the custom options schema:

```bash
anvil proto print-options-proto > proto/foundrytools/options.proto
protoc-gen-foundryscript --print-options-proto > proto/foundrytools/options.proto
```

Supported file options:

```protobuf
option (foundrytools.namespace) = "cafecito.game.v1";
option (foundrytools.type_prefix) = "Game";
option (foundrytools.emit_runtime) = true;
```

### Generated Type Prefixes

Use `type_prefix` when generated names would collide:

```protobuf
option (foundrytools.type_prefix) = "Game";
```

The value is a literal, non-empty identifier fragment matching
`[A-Za-z_][A-Za-z0-9_]*`. The generator inserts no separator and performs no
case normalization, so `Game_` stays literal. It applies the prefix uniformly
to every generated type declaration in that file and to every nested segment:
top-level and nested messages and enums, generated oneof-case enums, references
to those types, and output filenames.

| Protobuf/generated name | With `Game` prefix |
|---|---|
| `Node` | `GameNode` |
| `Outer.Inner` | `GameOuter.GameInner` |
| `Player.payload` oneof enum | `GamePlayerPayloadCase` |
| `Node.pb.fs` | `GameNode.pb.fs` |

Prefixing happens after protobuf name normalization but before Foundry
keyword/runtime escaping. For example, unprefixed `Message` becomes `Message_`,
while the `Game` prefix produces `GameMessage`. A prefix changes the public
generated API, so consumers and import references must use the new names.

Final prefixed names are still validated against Foundry built-ins and exposed
native classes. An unsafe prefix produces an error asking for another prefix.
Without a prefix, collisions fail with aggregated, actionable diagnostics. A
collision in a referenced dependency must be resolved by setting or changing
`type_prefix` in the file that declares that dependency. Project-specific
extension, global script, and autoload names remain outside static generation
and are caught by Foundry lint-time checks.

### Field member collisions

Message fields normally keep their raw protobuf names. The generator appends
one underscore when an exact, case-sensitive name matches a Foundry keyword, a
generated message method (`from_bytes`, `to_bytes`, or `merge_from_bytes`), a
built-in type, or an exposed native class. Names beginning with the
generator-reserved `_pb_` prefix are rejected instead.

| Protobuf field | Foundry member |
|---|---|
| `Node` | `Node_` |
| `String` | `String_` |
| `node` | `node` |

The escaped member is used consistently for reads, writes, serialization, and
deserialization. Protobuf field numbers and wire encoding do not change. Oneof
group storage follows the same member-escaping rule. Oneof case names and enum
values retain their existing generated-name rules; matching a built-in type or
exposed native class does not add an underscore to them. `type_prefix` changes
type declarations, not field members.

Escaping does not search for a second suffix. A secondary collision such as
fields named both `Node` and `Node_` fails generation with an instruction to
rename one protobuf declaration.

## Development

```bash
task              # local CI without Foundry
task build        # build binaries into ./bin
task test         # Go tests with -race
task integration  # protoc and Buf integration tests
task lint         # golangci-lint
```

Foundry checks require a recent Foundry editor build on your `PATH`:

```bash
task foundry:test                          # uses `which foundry`
FOUNDRY_BIN=/path/to/foundry task foundry:test
```
