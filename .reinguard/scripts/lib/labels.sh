#!/usr/bin/env bash
# Shared helpers for scripts that read label metadata from labels.yaml.
# Requires: source lib/common.sh before sourcing this file.

labels_yaml_path() {
  local script_dir="$1"
  printf '%s\n' "${REINGUARD_LABELS_YAML:-$script_dir/../labels.yaml}"
}

require_labels_yaml() {
  local script_dir="$1"
  local path
  path="$(labels_yaml_path "$script_dir")"
  require_file "$path" "$path not found." 2
  printf '%s\n' "$path"
}

load_label_names() {
  local labels_yaml="$1"
  local query="$2"
  local out_name="$3"
  # Requires bash 4.3+ for nameref support.
  local -n out="$out_name"
  local data
  data="$(yq -r "$query" "$labels_yaml")" || fail_with "failed to load labels from $labels_yaml" 1
  if [[ -z "$data" || "$data" == "null" ]]; then
    out=()
    return 0
  fi
  mapfile -t out <<< "$data"
  local filtered=()
  local item
  for item in "${out[@]}"; do
    [[ -n "$item" ]] && filtered+=("$item")
  done
  out=("${filtered[@]}")
}

join_with_pipe() {
  printf '%s\n' "$@" | paste -sd '|' -
}
