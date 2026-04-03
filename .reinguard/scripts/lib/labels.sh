#!/usr/bin/env bash
# Shared helpers for scripts that read label metadata from labels.yaml.

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
  local -n out="$out_name"
  mapfile -t out < <(yq -r "$query" "$labels_yaml")
}

join_with_pipe() {
  printf '%s\n' "$@" | paste -sd '|' -
}
