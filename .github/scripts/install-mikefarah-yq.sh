#!/usr/bin/env bash
# Install mikefarah yq for GitHub Actions (Linux) with SHA-256 verification against
# the release checksums file (https://github.com/mikefarah/yq/releases).
#
# Env: YQ_VER (default 4.45.1). Optional first argument overrides the version.
set -euo pipefail

YQ_VER="${YQ_VER:-${1:-4.45.1}}"
ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
x86_64) ARCH=amd64 ;;
aarch64 | arm64) ARCH=arm64 ;;
*)
  echo "ERROR: unsupported runner arch for yq: $ARCH_RAW" >&2
  exit 1
  ;;
esac

ASSET="yq_linux_${ARCH}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

BASE="https://github.com/mikefarah/yq/releases/download/v${YQ_VER}"
curl -fsSL "$BASE/checksums" -o "$TMP/checksums"
curl -fsSL "$BASE/checksums_hashes_order" -o "$TMP/checksums_hashes_order"
curl -fsSL "$BASE/extract-checksum.sh" -o "$TMP/extract-checksum.sh"
chmod +x "$TMP/extract-checksum.sh"
curl -fsSL "$BASE/${ASSET}" -o /tmp/yq

(
  cd "$TMP"
  ./extract-checksum.sh SHA-256 "$ASSET" | awk '{print $2 "  /tmp/yq"}' | sha256sum -c -
)

sudo install -m 0755 /tmp/yq /usr/local/bin/yq
yq --version
