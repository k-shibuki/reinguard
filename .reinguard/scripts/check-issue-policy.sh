#!/usr/bin/env bash
# .reinguard/scripts/check-issue-policy.sh — Local pre-flight for Issue body/title/labels.
# Type and scope labels are read from .reinguard/labels.yaml (requires mikefarah yq v4).
#
# Usage:
#   bash .reinguard/scripts/check-issue-policy.sh \
#     --title "feat(scope): summary" \
#     --body-file /tmp/issue.md \
#     --label feat \
#     [--template task|epic]
#
# --title, --body-file, and at least one --label are required.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=.reinguard/scripts/lib/labels.sh
source "$SCRIPT_DIR/lib/labels.sh"

LABELS_YAML="$(require_labels_yaml "$SCRIPT_DIR")"
require_command "yq" "yq is required (mikefarah/yq v4). Install: https://github.com/mikefarah/yq" 2

load_label_names "$LABELS_YAML" '.categories.type.labels[].name' TYPE_LABELS
load_label_names "$LABELS_YAML" '.categories.exception.labels[].name' EXCEPTION_LABELS
load_label_names "$LABELS_YAML" '.categories.scope.labels[].name' SCOPE_LABELS
TYPE_PATTERN="$(join_with_pipe "${TYPE_LABELS[@]}")"

TITLE=""
BODY_FILE=""
TEMPLATE="task"
LABELS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --title)
      require_flag_value "--title" "${2:-}" "--title requires a non-empty value"
      TITLE="$2"
      shift 2
      ;;
    --body-file)
      require_flag_value "--body-file" "${2:-}" "--body-file requires a non-empty path"
      BODY_FILE="$2"
      shift 2
      ;;
    --template)
      require_flag_value "--template" "${2:-}" "--template requires task or epic"
      TEMPLATE="$2"
      shift 2
      ;;
    --label)
      require_flag_value "--label" "${2:-}" "--label requires a non-empty value"
      LABELS+=("$2")
      shift 2
      ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$TITLE" || -z "$BODY_FILE" || ${#LABELS[@]} -eq 0 ]]; then
  echo "Usage: check-issue-policy.sh --title <title> --body-file <file> --label <name> [--label ...] [--template task|epic]" >&2
  exit 2
fi

if [[ "$TEMPLATE" != "task" && "$TEMPLATE" != "epic" ]]; then
  echo "ERROR: --template must be task or epic" >&2
  exit 2
fi

require_file "$BODY_FILE" "body file not found: $BODY_FILE" 2

BODY=$(cat "$BODY_FILE")
ERRORS=()

for l in "${LABELS[@]}"; do
  for el in "${EXCEPTION_LABELS[@]}"; do
    if [[ "$l" == "$el" ]]; then
      ERRORS+=("Label: \`$el\` is PR-only (exception category); do not use on Issues.")
    fi
  done
done

strip_comments() {
  strip_html_comments_and_blank_lines "$1"
}

# Match ## or ### heading (Issue Form markdown or hand-authored).
has_section() {
  local title="$1"
  grep -qiF "## ${title}" <<< "$BODY" || grep -qiF "### ${title}" <<< "$BODY"
}

extract_title_type() {
  # Conventional Commits: type(scope)!: or type!:
  sed -E 's/^([a-z]+)(\([^)]+\))?(!)?:.*/\1/' <<< "$1"
}

if [[ "$TEMPLATE" == "epic" ]]; then
  EPIC_HIT=false
  for l in "${LABELS[@]}"; do
    [[ "$l" == "epic" ]] && EPIC_HIT=true
  done
  if ! $EPIC_HIT; then
    ERRORS+=("Label: epic template requires label \`epic\`.")
  fi
  for tl in "${TYPE_LABELS[@]}"; do
    for l in "${LABELS[@]}"; do
      [[ "$l" == "$tl" ]] && ERRORS+=("Label: epic Issues must not use a type label ($tl). Use --template task for implementation Issues.")
    done
  done

  has_section "Summary" || ERRORS+=("Summary: section missing from body (use ## or ### Summary).")
  has_section "Background" || ERRORS+=("Background: section missing from body.")
  has_section "Verification baseline" || ERRORS+=("Verification baseline: section missing from body.")
  if ! grep -qiE '(^|\n)#{1,3}[[:space:]]+Child work items' <<< "$BODY"; then
    ERRORS+=("Child work items: section missing from body (## or ### heading starting with \"Child work items\").")
  fi
else
  TITLE_RE="^($TYPE_PATTERN)(\\(.+\\))?(!)?: .+$"
  if ! grep -qE "$TITLE_RE" <<< "$TITLE"; then
    ERRORS+=("Issue title: must match Conventional Commits: <type>(<scope>): <summary>. Valid types: ${TYPE_LABELS[*]}.")
  fi

  TYPE_FROM_TITLE=$(extract_title_type "$TITLE")

  HITS=()
  for tl in "${TYPE_LABELS[@]}"; do
    for l in "${LABELS[@]}"; do
      [[ "$l" == "$tl" ]] && HITS+=("$tl")
    done
  done
  if [[ ${#HITS[@]} -eq 0 ]]; then
    ERRORS+=("Type label: must have exactly one type label (${TYPE_LABELS[*]}).")
  elif [[ ${#HITS[@]} -gt 1 ]]; then
    ERRORS+=("Type label: multiple type labels (${HITS[*]}). Keep exactly one.")
  fi

  if [[ ${#HITS[@]} -eq 1 && -n "$TYPE_FROM_TITLE" && "$TYPE_FROM_TITLE" != "${HITS[0]}" ]]; then
    ERRORS+=("Type label (${HITS[0]}) must match title type ($TYPE_FROM_TITLE).")
  fi

  for l in "${LABELS[@]}"; do
    for sl in "${SCOPE_LABELS[@]}"; do
      [[ "$l" == "$sl" ]] && ERRORS+=("Label: task Issues must not use scope-only label \`$sl\` (use --template epic for epics).")
    done
  done

  has_section "Context" || ERRORS+=("Context: section missing from body.")
  has_section "Refs: ADR" || ERRORS+=("Refs: ADR: section missing from body.")
  has_section "ADR Impact" || ERRORS+=("ADR Impact: section missing from body.")
  has_section "Acceptance ↔ ADR" || ERRORS+=("Acceptance ↔ ADR: section missing from body.")
  has_section "Definition of Done" || ERRORS+=("Definition of Done: section missing from body.")
  has_section "Test plan" || ERRORS+=("Test plan: section missing from body.")

  DOD=$(awk '
    /^#{1,3}[[:space:]]+Definition of Done/ { on=1; next }
    /^#{1,3}[[:space:]]+/ { if (on) exit }
    on { print }
  ' <<< "$BODY")
  DOD_CLEAN=$(strip_comments "$DOD")
  if [[ ${#DOD_CLEAN} -lt 3 ]]; then
    ERRORS+=("Definition of Done: section exists but appears empty.")
  fi
  TP=$(awk '
    /^#{1,3}[[:space:]]+Test [Pp]lan/ { on=1; next }
    /^#{1,3}[[:space:]]+/ { if (on) exit }
    on { print }
  ' <<< "$BODY")
  TP_CLEAN=$(strip_comments "$TP")
  if [[ ${#TP_CLEAN} -lt 5 ]]; then
    ERRORS+=("Test plan: section exists but appears empty.")
  fi
fi

if [[ ${#ERRORS[@]} -gt 0 ]]; then
  echo "Issue policy pre-flight FAILED:" >&2
  for e in "${ERRORS[@]}"; do
    echo "  ERROR: $e" >&2
  done
  exit 1
fi

echo "Issue policy pre-flight OK."
