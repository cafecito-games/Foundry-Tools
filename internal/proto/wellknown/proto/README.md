# Vendored well-known types

`google/protobuf/*.proto` holds the seven well-known-type files whose Foundry
Script bindings ship in the runtime under `internal/runtime/data/foundry/proto/wkt/`.
They are embedded so generation needs no external protoc include path.

## Provenance

Source: <https://github.com/protocolbuffers/protobuf>
Pinned commit: `0f3dd063c6fd301cb73c3148f8ac7b570f773e94`

Files are copied verbatim from `src/google/protobuf/` at that commit:

```
google/protobuf/any.proto
google/protobuf/duration.proto
google/protobuf/empty.proto
google/protobuf/field_mask.proto
google/protobuf/struct.proto
google/protobuf/timestamp.proto
google/protobuf/wrappers.proto
```

This is the same pin as `tests/integration/fixtures/conformance/google/protobuf/`.
Move both together so the schemas the conformance fixture imports stay identical
to the ones the shipped bindings are generated from.

## Refreshing

```bash
PIN=<upstream commit sha>
BASE="https://raw.githubusercontent.com/protocolbuffers/protobuf/$PIN/src/google/protobuf"
for file in any duration empty field_mask struct timestamp wrappers; do
  curl -sfL -o "google/protobuf/$file.proto" "$BASE/$file.proto"
done
```

Then run `task gen-wkt` and review the resulting change to the checked-in
bindings.
