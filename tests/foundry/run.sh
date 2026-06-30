#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/tests/foundry/generated"
FOUNDRY="${FOUNDRY_BIN:-$HOME/.foundry/bin/foundry.macos.editor.dev.arm64}"

rm -rf "$OUT"
mkdir -p "$OUT"

"$ROOT/bin/foundry-tools" proto generate \
  -I "$ROOT/tests/integration/fixtures/basic" \
  -o "$OUT" \
  "$ROOT/tests/integration/fixtures/basic/player.proto"

if rg -n '(^|[^_])func [A-Za-z0-9_]+\(.*Variant|-> Variant' "$OUT"; then
  echo "public Variant signature found in generated Foundry Script"
  exit 1
fi

"$FOUNDRY" --headless --check-only --script "$ROOT/tests/foundry/main.fs" --path "$ROOT/tests/foundry"
