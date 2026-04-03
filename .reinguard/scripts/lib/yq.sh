#!/usr/bin/env bash
# Helpers for mikefarah yq detection / caching.

is_mikefarah_yq() {
  local yq_bin="${1:-yq}"
  command -v "$yq_bin" >/dev/null 2>&1 || return 1
  local out
  out=$("$yq_bin" --version 2>&1 || true)
  grep -qE 'mikefarah|github.com/mikefarah/yq' <<< "$out" || return 1
  grep -qE 'version v[4-9]' <<< "$out" || return 1
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
    curl -sSL "$url" -o "$cached"
    chmod +x "$cached"
  fi
  out=("$cached")
}
