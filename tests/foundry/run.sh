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

(
  cd "$ROOT"
  FOUNDRY_BIN="$FOUNDRY" bash "$ROOT/scripts/ci/sync-foundry-engine-types.sh" check
)

"$ROOT/bin/anvil" proto generate \
  -I "$ROOT/tests/integration/fixtures/basic" \
  -I "$PROJECT" \
  -o "$OUT" \
  "$ROOT/tests/integration/fixtures/basic/player.proto" \
  "$ROOT/tests/integration/fixtures/basic/inventory.proto" \
  "$PROJECT/collision_dependency.proto" \
  "$PROJECT/collisions.proto"

if grep -R -n -E -e '(^|[^_])func [A-Za-z0-9_]+\(.*Variant|-> Variant' "$OUT"; then
  echo "public Variant signature found in generated Foundry Script"
  exit 1
fi

# An active `import foundry.proto` makes every runtime reference short; a
# dotted one means the emitter re-qualified something it did not need to.
if grep -R -n -E -e '\bfoundry\.proto\.[A-Z]' "$OUT"; then
  echo "redundant foundry.proto. qualification found in generated Foundry Script"
  exit 1
fi

# Build the script index first; without it every file reports unresolved namespaces.
"$FOUNDRY" --headless project import --project "$PROJECT"

if ! "$FOUNDRY" script lint --no-header --format=json --fail-on=error --project "$PROJECT" "res://"; then
  echo "Foundry Script lint reported errors in the generated project"
  exit 1
fi

# Lint proves the bindings typecheck; only running them proves the bytes are
# right, which is what main.fs asserts across every supported construct.
if ! "$FOUNDRY" --headless project run --project "$PROJECT" --script "$PROJECT/main.fs"; then
  echo "generated Foundry Script failed its round-trip checks"
  exit 1
fi
