# Generated Foundry Docs Design

## Purpose

Generated Foundry Script should expose a documented public API. This pass adds
deterministic `##` documentation comments to generated `.pb.fs` files without
preserving schema-authored `.proto` comments.

Preserving `.proto` comments is intentionally deferred to
https://github.com/cafecito-games/Foundry-Tools/issues/2 because the parser
currently skips comments and descriptor conversion does not read `SourceCodeInfo`
comments.

## Scope

This PR documents the public API surface that the generator emits today:

- top-level message classes
- top-level enum declarations
- public field setters
- public field getters
- `from_bytes`
- `to_bytes`
- `merge_from_bytes`

Private backing fields such as `_name` are not part of the public API and should
remain undocumented.

## Documentation Format

Foundry Script follows Godot-style documentation comments: a `##` line directly
above the declaration. Generated docs should use that format and remain valid
ordinary Foundry Script comments.

Examples:

```gdscript
## Generated protobuf message binding for Player.
final class_name Player extends RefCounted

## Sets the name protobuf field.
func set_name(value: String) -> void:
	_name = value

## Returns the name protobuf field.
func get_name() -> String:
	return _name
```

Docs should be deterministic and schema-name aware, but they must not imply
human-authored semantics from missing `.proto` comments.

## Architecture

Add first-class documentation support to `internal/fsast` instead of scattering
raw `##` strings through generator code.

The renderer should support documentation on:

- `fsast.Class`
- `fsast.Func`
- a small wrapper or node type for documented raw declarations, used by current
  enum rendering

The renderer should emit each doc line at the same indentation as the declaration
it documents. Blank lines between declarations should remain consistent with the
current output style.

`internal/fsgenerator` should own the generated prose. It can derive docs from
the message type name, enum type name, field name, and method role. This keeps
`fsast` renderer-neutral.

## Generated Prose

Message classes:

```text
Generated protobuf message binding for <TypeName>.
```

Enums:

```text
Generated protobuf enum binding for <TypeName>.
```

Field setters:

```text
Sets the <field_name> protobuf field.
```

Field getters:

```text
Returns the <field_name> protobuf field.
```

Decode factory:

```text
Decodes protobuf wire data into a new <TypeName> message.
```

Serializer:

```text
Serializes this message to protobuf wire data.
```

Merge decoder:

```text
Merges protobuf wire data into this message.
```

## Testing

Use test-first implementation.

Add renderer tests proving `fsast` emits `##` comments above documented classes,
functions, and documented raw declarations with the right indentation.

Add generator tests proving generated message and enum output contains docs for
the public declarations listed in this spec and does not add docs to private
backing fields.

Refresh examples/golden output after the generator changes and keep integration
and Foundry checks passing.

## Non-Goals

- Do not parse or preserve `.proto` comments in this PR.
- Do not add CLI flags for controlling docs.
- Do not document private generated implementation details.
- Do not change generated method names or signatures.
