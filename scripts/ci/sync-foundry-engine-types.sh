#!/usr/bin/env bash

set -euo pipefail

MODE="${1:-check}"
case "$MODE" in
  write | check) ;;
  *)
    echo "Usage: $0 [write|check]" >&2
    exit 2
    ;;
esac

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FOUNDRY="${FOUNDRY_BIN:-}"
if [[ -z "$FOUNDRY" ]]; then
  FOUNDRY="$(command -v foundry || true)"
fi
if [[ -z "$FOUNDRY" || ! -x "$FOUNDRY" ]]; then
  echo "Foundry binary not found on PATH. Install foundry or set FOUNDRY_BIN." >&2
  exit 1
fi
FOUNDRY_DIR="$(cd "$(dirname "$FOUNDRY")" && pwd)"
FOUNDRY="$FOUNDRY_DIR/$(basename "$FOUNDRY")"

TARGET="$ROOT/internal/proto/internal/foundryscript/generator/engine_reserved_types.gen.go"
SYNC_DIR="$(mktemp -d)"
if [[ -z "$SYNC_DIR" || ! -d "$SYNC_DIR" ]]; then
  echo "Failed to create synchronization directory." >&2
  exit 1
fi

cleanup() {
  if [[ -n "${SYNC_DIR:-}" && -d "$SYNC_DIR" ]]; then
    rm -rf -- "$SYNC_DIR"
  fi
}
trap cleanup EXIT

(
  cd "$SYNC_DIR"
  "$FOUNDRY" --headless --no-header docs generate-api
)

VERSION_OUTPUT="$("$FOUNDRY" --version)"
VERSION="${VERSION_OUTPUT%%$'\n'*}"
CANDIDATE="$SYNC_DIR/engine_reserved_types.gen.go"
go run "$ROOT/internal/proto/internal/foundryscript/generator/cmd/gen-engine-reserved-types" \
  --api "$SYNC_DIR/extension_api.json" \
  --version "$VERSION" \
  --output "$CANDIDATE"

if [[ "$MODE" == "write" ]]; then
  cp "$CANDIDATE" "$TARGET"
  exit 0
fi

if ! cmp -s "$CANDIDATE" "$TARGET"; then
  echo "Foundry engine type table is stale for $VERSION." >&2
  echo "Run: task gen-engine-types" >&2
  diff -u "$TARGET" "$CANDIDATE" || true
  exit 1
fi
