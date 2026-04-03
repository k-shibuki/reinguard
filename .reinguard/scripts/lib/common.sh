#!/usr/bin/env bash
# Shared helpers for repository-local shell scripts.

script_dir_from_source() {
  local source_path="$1"
  cd "$(dirname "$source_path")" && pwd
}

repo_root_from_script_dir() {
  local script_dir="$1"
  cd "$script_dir/../.." && pwd
}

fail_with() {
  local message="$1"
  local code="${2:-2}"
  echo "ERROR: $message" >&2
  exit "$code"
}

require_command() {
  local name="$1"
  local message="$2"
  local code="${3:-2}"
  if ! command -v "$name" >/dev/null 2>&1; then
    fail_with "$message" "$code"
  fi
}

require_file() {
  local path="$1"
  local message="$2"
  local code="${3:-2}"
  if [[ ! -f "$path" ]]; then
    fail_with "$message" "$code"
  fi
}

require_flag_value() {
  local flag="$1"
  local value="${2:-}"
  local message="$3"
  if [[ -z "$value" || "${value:0:1}" == "-" ]]; then
    fail_with "$message" 2
  fi
}

strip_ansi_cr() {
  printf '%s\n' "$1" | sed -E 's/\x1B\[[0-9;?]*[[:alpha:]]//g' | tr -d '\r'
}

strip_html_comments_and_blank_lines() {
  local text="$1"
  if command -v perl >/dev/null 2>&1; then
    perl -0777 -pe 's/<!--.*?-->//gs' <<< "$text" | sed '/^[[:space:]]*$/d'
  else
    sed 's/<!--[^>]*-->//g' <<< "$text" | sed '/^[[:space:]]*$/d'
  fi
}

cleanup_temp_files() {
  local path
  for path in "$@"; do
    [[ -n "$path" ]] && rm -f "$path"
  done
}
