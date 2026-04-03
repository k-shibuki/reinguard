#!/usr/bin/env bash
# Helpers for mikefarah yq detection / caching.
# Requires: source lib/common.sh before sourcing this file.

is_mikefarah_yq() {
  local yq_bin="${1:-yq}"
  command -v "$yq_bin" >/dev/null 2>&1 || return 1
  local out
  out=$("$yq_bin" --version 2>&1 || true)
  grep -qE 'mikefarah|github.com/mikefarah/yq' <<< "$out" || return 1
  grep -qE 'version v([4-9]|[1-9][0-9]+)' <<< "$out" || return 1
  return 0
}

ensure_mikefarah_yq_cached() {
  local cache_dir="$1"
  local out_name="$2"
  local -n out="$out_name"
  local cached="$cache_dir/yq"

  if is_mikefarah_yq yq; then
    out=(yq)
    return 0
  fi

  mkdir -p "$cache_dir"
  if [[ ! -x "$cached" ]]; then
    echo "Installing mikefarah yq to $cached (one-time)..." >&2
    local ver=4.45.1
    local arch
    arch=$(uname -m)
    case "$arch" in
      x86_64) arch=amd64 ;;
      aarch64 | arm64) arch=arm64 ;;
      *)
        fail_with "unsupported arch for bundled yq: $arch" 1
        ;;
    esac
    local url="https://github.com/mikefarah/yq/releases/download/v${ver}/yq_linux_${arch}"
    if ! curl -fsSL "$url" -o "$cached"; then
      rm -f "$cached"
      fail_with "failed to download yq from $url" 1
    fi
    chmod +x "$cached"
    if ! is_mikefarah_yq "$cached"; then
      rm -f "$cached"
      fail_with "downloaded yq binary failed validation" 1
    fi
  fi
  out=("$cached")
}
