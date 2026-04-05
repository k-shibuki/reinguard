#!/usr/bin/env bash
# Minimal JSON string helpers for shell scripts.
#
# Scope: controlled, repo-authored JSON (e.g. adapter resume artifacts).
# Not a general JSON parser: nested objects in values, arbitrary nesting depth,
# and non-string scalars beyond simple unsigned integers in known fields are
# out of scope. json_get_block matches a single-level {...} body (no nested
# braces inside the block).

# shellcheck shell=bash

json_now_utc() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

json_escape() {
  local value="$1"
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//$'\n'/\\n}
  value=${value//$'\r'/}
  value=${value//$'\t'/\\t}
  printf '%s' "$value"
}

json_get_block() {
  local raw="$1"
  local key="$2"
  local pattern="\"$key\"[[:space:]]*:[[:space:]]*\\{([^}]*)\\}"
  if [[ $raw =~ $pattern ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  fi
}

json_get_string() {
  local raw="$1"
  local key="$2"
  local pattern="\"$key\"[[:space:]]*:[[:space:]]*\"(([^\"\\\\]|\\\\.)*)\""
  if [[ $raw =~ $pattern ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  fi
}

json_get_number() {
  local raw="$1"
  local key="$2"
  local pattern="\"$key\"[[:space:]]*:[[:space:]]*([0-9]+)"
  if [[ $raw =~ $pattern ]]; then
    printf '%s' "${BASH_REMATCH[1]}"
  fi
}

json_has_key() {
  local raw="$1"
  local key="$2"
  local pattern="\"$key\"[[:space:]]*:"
  [[ $raw =~ $pattern ]]
}
