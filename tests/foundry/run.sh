#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="$ROOT/tests/foundry/generated"
FOUNDRY="${FOUNDRY_BIN:-$HOME/.foundry/bin/foundry.macos.editor.dev.arm64}"

cleanup() {
  rm -rf "$OUT" "$ROOT/tests/foundry/.foundry"
  rm -f "$ROOT"/tests/foundry/*.uid
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

run_foundry_checked() {
  local foundry_output
  local foundry_status

  set +e
  foundry_output="$("$FOUNDRY" "$@" 2>&1)"
  foundry_status=$?
  set -e

  printf '%s\n' "$foundry_output"

  if [ "$foundry_status" -ne 0 ]; then
    exit "$foundry_status"
  fi

  if grep -E 'SCRIPT ERROR|Parse Error|Failed to load script' <<<"$foundry_output"; then
    echo "Foundry reported script parse/load diagnostics"
    exit 1
  fi
}

run_foundry_checked --headless --import --path "$ROOT/tests/foundry"
run_foundry_checked --headless --check-only --script "$ROOT/tests/foundry/main.fs" --path "$ROOT/tests/foundry"
