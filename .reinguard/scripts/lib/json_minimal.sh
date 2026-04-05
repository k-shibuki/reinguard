#!/usr/bin/env bash
# Minimal JSON string helpers for shell scripts.
#
# Scope: controlled, repo-authored JSON (e.g. adapter resume artifacts).
# Not a general JSON parser: nested objects in values, arbitrary nesting depth,
# and non-string scalars beyond simple unsigned integers in known fields are
# out of scope. json_get_block matches a {...} object value with balanced braces
# (strings may contain "}" when properly quoted).

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

# Decode a JSON string fragment that still contains JSON escape sequences
# (e.g. \" \\ \n). Used after json_get_string's regex capture.
json_unescape_string() {
  local s="$1"
  local out=""
  local i=0
  local len=${#s}
  while (( i < len )); do
    local c="${s:i:1}"
    if [[ "$c" != '\' ]]; then
      out+="$c"
      ((i++))
      continue
    fi
    if (( i + 1 >= len )); then
      out+='\'
      break
    fi
    local n="${s:i+1:1}"
    case "$n" in
      '"') out+='"'; ((i += 2));;
      '\') out+='\'; ((i += 2));;
      '/') out+='/'; ((i += 2));;
      'b') out+=$'\b'; ((i += 2));;
      'f') out+=$'\f'; ((i += 2));;
      'n') out+=$'\n'; ((i += 2));;
      'r') ((i += 2));;
      't') out+=$'\t'; ((i += 2));;
      'u')
        # Minimal scope: preserve \uXXXX literally if present (rare in artifacts).
        if (( i + 5 < len )); then
          out+="${s:i:6}"
          ((i += 6))
        else
          out+='\'
          ((i++))
        fi
        ;;
      *)
        out+="$n"
        ((i += 2));;
    esac
  done
  printf '%s' "$out"
}

# Find first occurrence of needle in raw; print start index or empty if missing.
json_index_of() {
  local raw="$1"
  local needle="$2"
  local nl=${#needle}
  local i=0
  local len=${#raw}
  while (( i + nl <= len )); do
    if [[ "${raw:i:nl}" == "$needle" ]]; then
      printf '%d' "$i"
      return 0
    fi
    ((i++))
  done
  return 1
}

# Extract inner content of first "key": { ... } with balanced braces.
json_get_block() {
  local raw="$1"
  local key="$2"
  local needle="\"$key\""
  local start
  if ! start="$(json_index_of "$raw" "$needle")"; then
    return 0
  fi
  local pos=$((start + ${#needle}))
  local rest="${raw:pos}"
  if [[ ! "$rest" =~ ^[[:space:]]*:[[:space:]]*\{ ]]; then
    return 0
  fi
  local matched="${BASH_REMATCH[0]}"
  local brace_start=$((pos + ${#matched} - 1))
  local i=$((brace_start + 1))
  local depth=1
  local len=${#raw}
  local in_str=0
  local esc=0
  while (( i < len && depth > 0 )); do
    local ch="${raw:i:1}"
    if (( in_str )); then
      if (( esc )); then
        esc=0
      elif [[ "$ch" == '\' ]]; then
        esc=1
      elif [[ "$ch" == '"' ]]; then
        in_str=0
      fi
    else
      if [[ "$ch" == '"' ]]; then
        in_str=1
      elif [[ "$ch" == '{' ]]; then
        ((depth++))
      elif [[ "$ch" == '}' ]]; then
        ((depth--))
      fi
    fi
    ((i++))
  done
  if (( depth != 0 )); then
    return 0
  fi
  printf '%s' "${raw:brace_start+1:i-brace_start-2}"
}

# Returns the decoded JSON string value for key (JSON string escapes applied).
json_get_string() {
  local raw="$1"
  local key="$2"
  local pattern="\"$key\"[[:space:]]*:[[:space:]]*\"(([^\"\\\\]|\\\\.)*)\""
  if [[ $raw =~ $pattern ]]; then
    json_unescape_string "${BASH_REMATCH[1]}"
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
