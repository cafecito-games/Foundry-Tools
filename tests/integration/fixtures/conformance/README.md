# Conformance fixture

`test_messages_proto3.proto` is upstream protobuf's exhaustive proto3 schema —
the same file the official conformance runner drives implementations with. It
answers "which parts of proto3 do we support" with a test rather than with a
reading of the generator, and because we do not maintain it, it stays a fixed
target.

`google/protobuf/` holds the seven well-known-type files it imports.

`any_protojson/` holds canonical Any JSON documents exercised against the Go
protobuf implementation as an independent oracle. The corpus covers an
ordinary upstream message, a preserved foreign type-URL prefix, each JSON
shape category, nested Any, and malformed inputs. `empty.json` pins Foundry
Tools' approved Empty exception: Empty stays an ordinary inline object without
the `value` envelope that Go protojson gives custom-form well-known types.

## Provenance

Source: <https://github.com/protocolbuffers/protobuf>
Pinned commit: `0f3dd063c6fd301cb73c3148f8ac7b570f773e94`

Files are copied verbatim from `src/google/protobuf/` at that commit:

```
test_messages_proto3.proto
google/protobuf/any.proto
google/protobuf/duration.proto
google/protobuf/empty.proto
google/protobuf/field_mask.proto
google/protobuf/struct.proto
google/protobuf/timestamp.proto
google/protobuf/wrappers.proto
```

## Refreshing

Refreshing is deliberate: a fixture that silently tracks upstream stops being a
fixed target, and a schema change would land as an unexplained diff in generated
output. To move the pin, re-fetch every file at one commit so the set stays
internally consistent, update the commit above, and review the resulting change
to the generated goldens.

```bash
PIN=<upstream commit sha>
BASE="https://raw.githubusercontent.com/protocolbuffers/protobuf/$PIN/src/google/protobuf"
curl -sfL -o test_messages_proto3.proto "$BASE/test_messages_proto3.proto"
for file in any duration empty field_mask struct timestamp wrappers; do
  curl -sfL -o "google/protobuf/$file.proto" "$BASE/$file.proto"
done
```

## Coverage

All fifteen proto3 scalars; `repeated` in its default, `[packed = true]` and
`[packed = false]` spellings as parallel field sets; the full map key/value
matrix; a ten-member `oneof` carrying a nested message; direct and mutual
recursion; a negative enum case and an `allow_alias` enum; and references to
every well-known type.

It does not use proto3 `optional`, so `examples/example.proto` remains worth
keeping for explicit presence.
