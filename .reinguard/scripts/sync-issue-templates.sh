#!/usr/bin/env bash
# Sync `.github/ISSUE_TEMPLATE/task.yml` Type dropdown from `.reinguard/labels.yaml`
# via `rgd labels list` (same SSOT). Requires `jq` and mikefarah `yq` v4
# (https://github.com/mikefarah/yq). If PATH `yq` is not mikefarah v4, a copy is
# downloaded to `.reinguard/scripts/.bin/yq` on first run (Linux or macOS), with
# SHA-256 verification against the release checksums file.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TASK="$REPO_ROOT/.github/ISSUE_TEMPLATE/task.yml"
YQ_CACHE_DIR="$SCRIPT_DIR/.bin"
YQ_CACHED="$YQ_CACHE_DIR/yq"

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq is required" >&2
  exit 1
fi

is_mikefarah_yq() {
  command -v yq >/dev/null 2>&1 || return 1
  local out
  out=$(yq --version 2>&1 || true)
  echo "$out" | grep -qE 'mikefarah|github.com/mikefarah/yq' || return 1
  echo "$out" | grep -qE 'version v[4-9]' || return 1
  return 0
}

# Download mikefarah yq for os (linux|darwin) and arch (amd64|arm64); verify SHA-256
# using the release checksums and upstream extract-checksum.sh.
download_mikefarah_yq_verified() {
  local ver="$1"
  local dest="$2"
  local os="$3"
  local arch="$4"
  local asset="yq_${os}_${arch}"
  local tmp base
  tmp=$(mktemp -d)
  base="https://github.com/mikefarah/yq/releases/download/v${ver}"
  curl -fsSL "$base/checksums" -o "$tmp/checksums"
  curl -fsSL "$base/checksums_hashes_order" -o "$tmp/checksums_hashes_order"
  curl -fsSL "$base/extract-checksum.sh" -o "$tmp/extract-checksum.sh"
  chmod +x "$tmp/extract-checksum.sh"
  curl -fsSL "$base/$asset" -o "$dest"
  (
    cd "$tmp"
    line=$(./extract-checksum.sh SHA-256 "$asset")
    exp=$(echo "$line" | awk '{print $2}')
    if command -v sha256sum >/dev/null 2>&1; then
      echo "$exp  $dest" | sha256sum -c -
    elif command -v shasum >/dev/null 2>&1; then
      echo "$exp  $dest" | shasum -a 256 -c -
    else
      echo "ERROR: need sha256sum or shasum to verify yq download" >&2
      exit 1
    fi
  )
  rm -rf "$tmp"
  chmod +x "$dest"
}

ensure_yq() {
  if is_mikefarah_yq; then
    YQ_CMD=(yq)
    return 0
  fi
  mkdir -p "$YQ_CACHE_DIR"
  if [[ ! -x "$YQ_CACHED" ]]; then
    echo "Installing mikefarah yq to $YQ_CACHED (one-time)..." >&2
    local ver=4.45.1
    local arch kernel
    kernel=$(uname -s)
    arch=$(uname -m)
    case "$arch" in
      x86_64) arch=amd64 ;;
      aarch64 | arm64) arch=arm64 ;;
      *)
        echo "ERROR: unsupported arch for bundled yq: $arch" >&2
        exit 1
        ;;
    esac
    local os
    case "$kernel" in
      Linux) os=linux ;;
      Darwin) os=darwin ;;
      *)
        echo "ERROR: unsupported OS for bundled yq: $kernel (install mikefarah yq v4 and ensure it is on PATH)" >&2
        exit 1
        ;;
    esac
    download_mikefarah_yq_verified "$ver" "$YQ_CACHED" "$os" "$arch"
  fi
  YQ_CMD=("$YQ_CACHED")
}

ensure_yq

cd "$REPO_ROOT"
# Always use `go run` so we do not pick up a stale ./rgd or PATH rgd without labels commands.
NAMES_JSON=$(go run ./cmd/rgd labels list --category type | jq -c '.names')

TMP=$(mktemp)
TMPJSON="${TMP}.json"
trap 'rm -f "$TMP" "$TMPJSON"' EXIT
printf '%s\n' "$NAMES_JSON" >"$TMPJSON"
# mikefarah yq: inject JSON array via load() (avoids --argjson portability issues).
"${YQ_CMD[@]}" eval ".body[0].attributes.options = load(\"$TMPJSON\")" -o yaml "$TASK" >"$TMP"
mv "$TMP" "$TASK"
echo "Updated $TASK Type dropdown from labels.yaml (via rgd labels list)."
