# Foundry Engine Type Collision Design

**Issue:** [cafecito-games/Foundry-Tools#30](https://github.com/cafecito-games/Foundry-Tools/issues/30)

**Related:** [cafecito-games/Foundry#1341](https://github.com/cafecito-games/Foundry/issues/1341), [cafecito-games/Foundry-Tools#32](https://github.com/cafecito-games/Foundry-Tools/issues/32)

**Status:** Approved for implementation planning

## Summary

Foundry Script refuses a class or named enum whose generated name matches an
engine built-in type or an exposed native class. A protobuf declaration such as
`message Node` therefore generates a binding that fails Foundry lint.

Foundry Tools will detect these collisions during generation and report every
offending declaration in the current file. A schema can opt into a literal,
file-wide `(foundrytools.type_prefix)` that renames every generated type in that
file. Engine-reserved names will come from a generated Go table tied to the
Foundry release pinned in CI, so normal `anvil`, protoc, and Buf generation
remain independent of a Foundry installation.

The initial table and Foundry integration pin will target
`v0.1.0-alpha.14` (`b9a5e66c21f8f7b707a9e526ca20557485c53227`).

## Goals

- Fail generation before emitting a binding whose type declaration Foundry
  will reject.
- Cover both Foundry built-in types and exposed native classes.
- Report all collisions found while generating one protobuf file.
- Make the existing `foundrytools.type_prefix` option a complete escape hatch
  for local, nested, generated oneof, and imported type references.
- Keep normal generation deterministic and independent of Foundry.
- Detect drift between the checked-in reserved-name table and the Foundry
  release used by integration CI.
- Preserve generated names and paths for safe schemas that do not set a
  prefix.

## Non-goals

- Detect project extensions, global script classes, or autoload singletons.
  These depend on a particular Foundry project and remain lint-time concerns.
- Rename colliding types automatically.
- Require Foundry during normal CLI, protoc plugin, or Buf generation.
- Resolve field/member collisions. Uppercase protobuf field names that shadow
  engine types are tracked separately in Foundry-Tools issue #32.
- Change enum value names. Named enum values do not trigger this analyzer
  collision rule.
- Change the existing keyword and `foundry.proto` runtime-export escaping
  policy.

## Why `extension_api.json` Is Authoritative

Foundry `v0.1.0-alpha.14` reports 1,068 registered classes through
`ClassDB.get_class_list()`, but only 1,050 are exposed to Foundry Script.
Foundry's extension API dump and Foundry Script analyzer both use
`ClassDB::is_class_exposed()` to select the visible set. The 18 internal
classes omitted from `extension_api.json`, including `FSNativeClass` and
`ThemeContext`, are legal user-defined Foundry Script class names.

Consequently, an unfiltered `ClassDB.get_class_list()` would create false
positives. The generated table must use:

- `builtin_classes[].name` for built-in types;
- `classes[].name` for exposed native classes; and
- `AsyncCallable` as an analyzer-specific built-in not currently present in
  `builtin_classes`.

Foundry issue #1341 requests a lighter script-accessible exposed-class list.
When that API becomes available, the refresh implementation may use it for the
native-class portion without changing any Foundry Tools behavior. The extension
API remains the source for built-in types.

## Components

### Generated engine-reserved table

`internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go`
will contain:

- the exact Foundry version used to generate it;
- a deterministic set of built-in type names; and
- a deterministic set of exposed native class names.

The two categories remain distinct so diagnostics can say either
`built-in type` or `native class`. The file is generated, formatted Go source;
it is not parsed or rebuilt during normal code generation.

### Refresh command

The checked-in command at
`internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types/main.go`
will accept:

- the path to an `extension_api.json`;
- the normalized output of `foundry --version`; and
- an output path.

It will validate the input structure, reject empty or duplicate names, add
`AsyncCallable`, sort each category, and emit byte-for-byte deterministic Go
source.

`task gen-engine-types` will:

1. Resolve `FOUNDRY_BIN` using the existing Task convention.
2. Create a temporary directory.
3. Run the binary's `docs generate-api` command in that directory.
4. Capture the binary version.
5. Invoke the Go refresh command to replace the checked-in generated table.
6. Format the result.

The task will fail clearly when Foundry is unavailable. Temporary API dumps and
engine caches must not be written into the repository.

### File-scoped type naming

The generator will introduce a file-scoped naming object. It owns the validated
literal prefix for one protobuf file and produces declaration segments,
references, output filenames, and generated oneof-case names.

The local file and every imported `FileEntry` receive their own naming object.
The resolver registers a declaration with the naming rules of the file that
declares it, not the file that references it. This keeps direct parsing and
descriptor-based protoc generation consistent.

### Collision validation

A pre-emission validation pass will inventory current-file declarations and
imported declarations actually referenced by generated fields. It will reuse
the generator's type registry so validation and emitted references cannot
disagree.

The pass covers:

- top-level messages and enums;
- nested messages and enums;
- generated oneof-case enums; and
- referenced imported messages and enums.

Repeated uses of one offending imported declaration produce one diagnostic.
An unreferenced declaration in an imported file is not validated while
generating the importing file; it is validated when its own binding is
generated.

## Prefix Contract

`(foundrytools.type_prefix)` is an optional, file-level, non-empty string. When
present, it must match:

```text
[A-Za-z_][A-Za-z0-9_]*
```

The value is used exactly as written. It is not trimmed, case-normalized, or
sanitized. Examples:

- `"Game"` plus `Node` becomes `GameNode`.
- `"Game_"` plus `Node` becomes `Game_Node`.
- `""`, `"game-tools"`, `"game tools"`, `"game.tools"`, and `"2D"` are errors.

An invalid option fails before collision collection because final names cannot
be computed safely. The source-text parser reports the option's line and
column. The descriptor path reports the filename and option name because the
current descriptor conversion does not retain option positions.

## Naming Algorithm

For a message or enum declaration segment:

1. Apply the existing UpperCamel conversion to the protobuf identifier.
2. Prepend the declaring file's literal prefix.
3. Apply existing Foundry keyword and `foundry.proto` runtime-export escaping
   to the combined name.
4. Compare the final name, case-sensitively, with the engine-reserved table.

Engine built-in and native collisions are errors, not another automatic escape
rule.

Examples:

| Protobuf declaration | No prefix | Prefix `Game` |
| --- | --- | --- |
| `message Player` | `Player` | `GamePlayer` |
| `message Class` | `Class_` | `GameClass` |
| `message Message` | `Message_` | `GameMessage` |
| `message Node` | generation error | `GameNode` |
| `message String` | generation error | `GameString` |

Applying the prefix before existing escaping avoids artifacts such as
`GameClass_` and `GameMessage_`.

### Nested declarations

Each nested declaration segment receives the declaring file's prefix:

```protobuf
option (foundrytools.type_prefix) = "Game";

message Outer {
  message Inner {}
}
```

The generated reference is `GameOuter.GameInner`, and the top-level binding is
written to `GameOuter.pb.fs`.

### Generated oneof-case enums

Oneof-case enums retain the current flattened-owner naming convention. The
flattened name is built from the already generated owner segments:

- `Player.payload` becomes `GamePlayerPayloadCase`.
- `Outer.Inner.choice` becomes `GameOuterGameInnerChoiceCase`.

The oneof enum's output filename uses that final name.

### Cross-file references

An imported reference uses the prefix of the dependency that declares it:

```protobuf
// inventory.proto
option (foundrytools.type_prefix) = "Inventory";
message Item {}

// player.proto
import "inventory.proto";
message Player {
  Item held = 1;
}
```

The generated player binding refers to `InventoryItem`. Setting a prefix in
`player.proto` does not rename `Item`; setting it in `inventory.proto` does.
Namespaces remain unchanged.

### Unsafe prefixes

The prefix is not assumed to make a name safe. For example, prefix
`"Animation"` plus `message Node` produces `AnimationNode`, which is itself an
exposed native class. Generation reports the final collision and instructs the
user to choose another prefix.

## Diagnostics

Generation reports all reserved-name collisions found for the current file in
one deterministic error. Diagnostics are sorted by declaring filename and
fully scoped protobuf declaration name.

Each entry contains:

- declaring filename;
- line and column when the AST contains them;
- declaration kind;
- fully scoped protobuf name;
- final generated Foundry name; and
- conflicting engine category and spelling.

Example:

```text
n.proto:4:1: message probe.n.v1.Node generates Foundry type "Node",
which conflicts with native class "Node"

n.proto:8:1: enum probe.n.v1.String generates Foundry type "String",
which conflicts with built-in type "String"

set a non-empty file option such as:
  option (foundrytools.type_prefix) = "Game";
```

Descriptor-based protoc generation degrades to filename, declaration kind,
fully scoped name, and generated name when source positions are unavailable.

When the file already has a prefix, the remediation says that the current
prefix still produces reserved names and must be changed. When a referenced
dependency is responsible, the diagnostic names the dependency file and says
to set or change the option there.

The direct CLI returns this text through its normal command error. The protoc
plugin writes the same text to `CodeGeneratorResponse.Error`. No generated
files from the failing `Generate` call are returned.

## Data Flow

1. The direct parser or descriptor converter populates the file option map.
2. Generation creates the local file's naming object and validates its prefix.
3. The resolver creates naming objects for dependencies and registers their
   final generated names.
4. The validation pass inventories current declarations and resolves named
   field types, collecting and deduplicating reserved-name conflicts.
5. Generation returns the aggregated error when conflicts exist.
6. Otherwise, planning and rendering use the same naming objects for
   declarations, references, oneof enums, and output paths.

This makes it impossible for collision checking and rendering to apply the
prefix in different orders.

## Foundry Version and Drift Policy

The checked-in table describes exactly the Foundry release pinned by the
Foundry integration workflow. Foundry Tools does not claim that the table
matches arbitrary older or newer engine versions.

`task foundry:test` will create a fresh API dump from `FOUNDRY_BIN`, generate a
candidate table in a temporary directory, and compare it byte-for-byte with the
checked-in table. A mismatch reports:

- the checked-in source version;
- the current binary version; and
- the `task gen-engine-types` command needed to refresh the table.

The initial implementation updates both
`scripts/ci/install-foundry.sh` and `.github/workflows/foundry.yml` from
`v0.1.0-alpha.11` to `v0.1.0-alpha.14`. Future Foundry upgrades must update the
pin and table in the same change.

## Testing

### Unit tests

Generator tests will verify:

- safe no-prefix output remains byte-for-byte unchanged;
- valid literal prefixes, including underscores;
- rejection of empty, malformed, and non-string option values;
- prefix-before-keyword/runtime-escape ordering;
- top-level and nested message and enum names;
- flattened oneof-case enum names;
- prefixed output paths;
- local references and cross-file references using the declaring file's
  prefix;
- native-class rejection using `Node`;
- built-in rejection using `String` and `AsyncCallable`;
- acceptance of internal non-exposed names such as `FSNativeClass`;
- rejection when a prefix produces another reserved name;
- aggregation, stable ordering, and deduplication;
- source locations when available and graceful degradation when absent; and
- dependency-directed remediation text.

Refresh-command tests will use small checked-in JSON fixtures to verify:

- built-in and exposed-class extraction;
- `AsyncCallable` insertion;
- deterministic sorting and formatting;
- duplicate and malformed input rejection; and
- version recording.

### Direct CLI and plugin tests

Tests for both entry paths will cover:

- an unprefixed collision returning the same core diagnostic;
- a valid prefix generating renamed files and references; and
- a prefixed imported dependency resolving correctly.

The plugin test must verify the error is returned in
`CodeGeneratorResponse.Error` rather than as a malformed plugin response.

### Foundry integration

The Foundry fixture will include a prefixed schema with:

- a top-level native collision;
- a top-level built-in collision;
- nested colliding types;
- a oneof case enum; and
- a cross-file reference to a prefixed dependency.

The test imports the project, lints all generated files, and executes a small
round-trip using the renamed public types. This proves the bindings load and
the naming changes are applied consistently beyond string-level unit tests.

The same task performs the exact API-table drift comparison before linting.

### Required verification

```text
task ci
task integration
task foundry:test
```

## Documentation

The README's custom-options section will document:

- that `type_prefix` renames every generated type in a file;
- literal-value validation;
- prefix-before-escaping behavior;
- nested, oneof, output-path, and imported-reference examples;
- generation-time errors for built-in and exposed native collisions;
- the effect on generated public APIs; and
- the project-specific collision limitation.

`proto/foundrytools/options.proto` will gain a concise comment describing
`type_prefix` as the file-wide collision escape hatch. The generated Go stub
and embedded schema copy will be refreshed with `task gen-options`.

## Acceptance Criteria

1. An unprefixed current-file or referenced imported type named `Node`,
   `String`, or `AsyncCallable` fails generation with an actionable,
   category-specific diagnostic.
2. All collisions for one generated file are reported together without
   duplicates.
3. A valid prefix renames top-level, nested, and generated oneof types,
   references, and output files consistently.
4. A dependency's prefix controls its imported references.
5. Invalid and still-colliding prefixes fail before invalid Foundry Script is
   emitted.
6. Safe schemas without a prefix retain their existing golden output.
7. Normal generation requires no Foundry installation.
8. Foundry integration CI detects any difference between the pinned engine API
   and the checked-in table.
9. Direct CLI, protoc plugin, integration, and Foundry round-trip tests pass.
10. The documented option schema and README match the implemented behavior.
