#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PROJECT="$ROOT/tests/foundry"
OUT="$PROJECT/generated"
FOUNDRY="${FOUNDRY_BIN:-$(command -v foundry || true)}"

if [ -z "$FOUNDRY" ] || [ ! -x "$FOUNDRY" ]; then
  echo "Foundry binary not found on PATH. Install foundry or set FOUNDRY_BIN." >&2
  exit 1
fi

cleanup() {
  rm -rf "$OUT" "$PROJECT/.foundry"
  rm -f "$PROJECT"/*.uid
}

trap cleanup EXIT

cleanup
mkdir -p "$OUT"

"$ROOT/bin/anvil" proto generate \
  -I "$ROOT/tests/integration/fixtures/basic" \
  -o "$OUT" \
  "$ROOT/tests/integration/fixtures/basic/player.proto"

if grep -R -n -E -e '(^|[^_])func [A-Za-z0-9_]+\(.*Variant|-> Variant' "$OUT"; then
  echo "public Variant signature found in generated Foundry Script"
  exit 1
fi

if grep -R -n -E -e '-> foundry\.proto\.DecodeResult\[|: foundry\.proto\.FieldRead\[|uses (foundry\.proto\.)?Message\[' "$OUT"; then
  echo "dotted runtime generic type annotation found in generated Foundry Script"
  exit 1
fi

# Build the script index first; without it every file reports unresolved namespaces.
"$FOUNDRY" --headless project import --project "$PROJECT"

if ! "$FOUNDRY" script lint --no-header --format=json --fail-on=error --project "$PROJECT" "res://"; then
  echo "Foundry Script lint reported errors in the generated project"
  exit 1
fi
