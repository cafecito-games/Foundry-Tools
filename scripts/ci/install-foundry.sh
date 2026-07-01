#!/usr/bin/env bash
set -euo pipefail

FOUNDRY_RELEASE_TAG="${FOUNDRY_RELEASE_TAG:-v0.1.0}"
FOUNDRY_ASSET_PATTERN="${FOUNDRY_ASSET_PATTERN:-linux.x86_64.zip}"
FOUNDRY_CACHE_DIR="${FOUNDRY_CACHE_DIR:-.cache/foundry}"
FOUNDRY_REPO="${FOUNDRY_REPO:-cafecito-games/Foundry}"

if [ -z "${GH_TOKEN:-}" ] && [ -n "${FOUNDRY_RELEASE_TOKEN:-}" ]; then
  export GH_TOKEN="$FOUNDRY_RELEASE_TOKEN"
fi

mkdir -p "$FOUNDRY_CACHE_DIR"

release_error="$(mktemp)"
release_json=""
if ! release_json="$(gh api "repos/$FOUNDRY_REPO/releases/tags/$FOUNDRY_RELEASE_TAG" 2>"$release_error")" || [ -z "$release_json" ]; then
  tag_lookup_error="$(cat "$release_error")"
  if releases_json="$(gh api "repos/$FOUNDRY_REPO/releases?per_page=100" 2>"$release_error")"; then
    release_json="$(
      jq -c --arg tag "$FOUNDRY_RELEASE_TAG" '
        [.[] | select(.tag_name == $tag)][0] // empty
      ' <<<"$releases_json"
    )"
  fi
fi

if [ -z "$release_json" ]; then
  echo "Unable to read Foundry release $FOUNDRY_REPO@$FOUNDRY_RELEASE_TAG." >&2
  echo "If this release or its assets are draft/private, set FOUNDRY_RELEASE_TOKEN or GH_TOKEN with access to the release." >&2
  if [ -n "${tag_lookup_error:-}" ]; then
    echo "$tag_lookup_error" >&2
  fi
  cat "$release_error" >&2
  rm -f "$release_error"
  exit 1
fi
rm -f "$release_error"

asset_info="$(
  jq -r --arg pattern "$FOUNDRY_ASSET_PATTERN" '
    .assets[]
    | select(.name | contains($pattern))
    | [.id, .name]
    | @tsv
  ' <<<"$release_json" | head -n 1
)"

if [ -z "$asset_info" ]; then
  echo "No Foundry release asset in $FOUNDRY_REPO@$FOUNDRY_RELEASE_TAG contains '$FOUNDRY_ASSET_PATTERN'." >&2
  exit 1
fi

asset_id="$(cut -f1 <<<"$asset_info")"
asset_name="$(cut -f2- <<<"$asset_info")"
archive="$FOUNDRY_CACHE_DIR/$asset_name"
safe_tag="$(printf '%s' "$FOUNDRY_RELEASE_TAG" | tr -c '[:alnum:]._-' '_')"
safe_asset="$(printf '%s' "${asset_name%.zip}" | tr -c '[:alnum:]._-' '_')"
extract_dir="$FOUNDRY_CACHE_DIR/$safe_tag/$safe_asset"

echo "Downloading Foundry asset $asset_name from $FOUNDRY_REPO@$FOUNDRY_RELEASE_TAG..."
gh api "repos/$FOUNDRY_REPO/releases/assets/$asset_id" \
  -H "Accept: application/octet-stream" > "$archive"

rm -rf "$extract_dir"
mkdir -p "$extract_dir"
unzip -o "$archive" -d "$extract_dir"

foundry_bin="$(
  find "$extract_dir" -type f -name 'foundry*' -perm -111 | sort | head -n 1
)"

if [ -z "$foundry_bin" ]; then
  echo "No executable foundry* binary found under $FOUNDRY_CACHE_DIR after extracting $asset_name." >&2
  exit 1
fi

if [ -n "${GITHUB_ENV:-}" ]; then
  echo "FOUNDRY_BIN=$foundry_bin" >> "$GITHUB_ENV"
else
  echo "export FOUNDRY_BIN='$foundry_bin'"
fi

echo "Foundry binary: $foundry_bin"
