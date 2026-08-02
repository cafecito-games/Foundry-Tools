# Inherited Engine Member Collision Escaping

## Problem

Every generated protobuf message extends `RefCounted`. A protobuf field or
oneof group whose generated spelling matches a symbol inherited from
`RefCounted` or `Object` shadows or redefines that engine symbol. Today the
member escaper knows engine type names, keywords, and generator-owned names,
but it does not know the reachable native base-class surface.

The Foundry analyzer treats these collisions according to symbol category:

- inherited methods such as `reference`, `unreference`, and `get_class` emit
  `SHADOWED_VARIABLE_BASE_CLASS` warnings;
- the inherited enum name `ConnectFlags` emits the same warning;
- inherited signals, constants, and enum values are analyzer errors; and
- matching is case-sensitive (`Reference` and `REFERENCE` do not collide with
  `reference`).

Neither `RefCounted` nor `Object` currently declares properties, but the API
category is part of the reachable class surface and must remain synchronized if
the engine adds one later.

## Design

Extend `gen-engine-reserved-types` to retain the class metadata already present
in `extension_api.json`. Starting at the same `RefCounted` base used by emitted
messages, walk `inherits` through `Object` and collect exact symbol spellings
from methods, properties, signals, constants, enum names, and enum values. Do
not inspect unrelated engine classes and do not reserve their methods globally.

The checked-in generated Go file will contain a second map beside
`foundryEngineReservedTypes`. Each inherited-symbol entry records its category
and the nearest class in the ancestry that declares it. Traversal begins at
`RefCounted`, so an override keeps the most-derived owner. The generator rejects
a missing base, a missing ancestor, or an inheritance cycle rather than
silently producing incomplete collision metadata. The emitted map is sorted so
refreshes are deterministic.

`planMemberName` will consult this generated map after the existing keyword,
generator-owned, and engine-type rules. A matching protobuf field or oneof
group receives exactly one trailing underscore and records an inherited-engine
escape reason for secondary-collision diagnostics. Oneof alternatives and
generated enum values do not inherit `RefCounted`, so their current naming
rules remain unchanged.

## Synchronization

The existing `task gen-engine-types` and
`scripts/ci/sync-foundry-engine-types.sh` flow remains the only refresh path.
One `extension_api.json` read generates both the engine type table and the
reachable base-member table, and the existing check mode compares the complete
generated file. The generated source version continues to identify the Foundry
binary that supplied the API.

## Testing

Strict red/green coverage has three layers:

1. Generator-command tests extend the compact API fixture with a multi-level
   ancestry and every supported symbol category. They assert exact-case,
   nearest-owner, deterministic rendering, and invalid ancestry handling.
2. Foundry Script generator tests assert that inherited member names escape,
   case variants do not, oneof alternatives and enum values remain unchanged,
   and an already-suffixed protobuf member produces the existing deterministic
   secondary-collision diagnostic.
3. The Foundry project fixture adds representative reachable-base members,
   round-trips them through generated code, and lints the project with
   `--fail-on=warning`. Before the implementation, that gate reproduces the
   issue's exact warning; afterward it prevents both warnings and errors from
   returning.

All existing focused Go tests, `task ci`, `task integration`, and
`task foundry:test` must pass after regeneration.
