#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PROJECT="$ROOT/tests/foundry"
OUT="$PROJECT/generated"
FOUNDRY="${FOUNDRY_BIN:-$(command -v foundry || true)}"
RUN_LOG=""

if [ -z "$FOUNDRY" ] || [ ! -x "$FOUNDRY" ]; then
  echo "Foundry binary not found on PATH. Install foundry or set FOUNDRY_BIN." >&2
  exit 1
fi

cleanup() {
  rm -rf "$OUT" "$PROJECT/.foundry"
  rm -f "$PROJECT"/*.uid
  if [ -n "$RUN_LOG" ]; then
    rm -f "$RUN_LOG"
  fi
}

trap cleanup EXIT

cleanup
mkdir -p "$OUT"

# Created only now, right before it is used, so the path it reserves is never
# deleted-then-reopened by name -- the window a symlink swap would need.
RUN_LOG="$(mktemp)"

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
  "$PROJECT/collisions.proto" \
  "$PROJECT/scalars.proto" \
  "$PROJECT/packing.proto" \
  "$PROJECT/well_known_dependency.proto" \
  "$PROJECT/well_known.proto"

# The JSON option is off above, so everything generated so far covers the wire
# codec on its own. The JSON surface is generated from the schema the JSON
# golden corpus pins, which makes that corpus engine-verified rather than only
# diffed: main.fs round trips the same message the goldens describe.
"$ROOT/bin/anvil" proto generate \
  --json \
  -I "$ROOT/examples/golden-json" \
  -o "$OUT" \
  "$ROOT/examples/golden-json/json_suite.proto"

# The runtime ships the well-known bindings, so no project copy may appear.
if [ -d "$OUT/google" ]; then
  echo "a google/protobuf binding was generated into the project"
  exit 1
fi

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

if ! "$FOUNDRY" script lint --no-header --format=json --fail-on=warning --project "$PROJECT" "res://"; then
  echo "Foundry Script lint reported warnings or errors in the generated project"
  exit 1
fi

# Lint proves the bindings typecheck; only running them proves the bytes are
# right, which is what main.fs asserts across every supported construct.
#
# A SCRIPT ERROR aborts only the function that triggered it: the engine logs
# the error and unwinds to the caller, which carries on. That lets _init reach
# "round trip ok" and exit 0 even though a construct broke mid-run, so the exit
# code alone cannot be trusted -- the captured output has to be checked for a
# SCRIPT ERROR too.
if ! "$FOUNDRY" --headless project run --project "$PROJECT" --script "$PROJECT/main.fs" 2>&1 | tee "$RUN_LOG"; then
  echo "generated Foundry Script failed its round-trip checks"
  exit 1
fi

if grep -q "SCRIPT ERROR" "$RUN_LOG"; then
  echo "generated Foundry Script emitted a SCRIPT ERROR during the round-trip run"
  exit 1
fi
