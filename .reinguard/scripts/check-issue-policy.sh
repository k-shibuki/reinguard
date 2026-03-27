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
LABELS_YAML="$SCRIPT_DIR/../labels.yaml"
if [[ ! -f "$LABELS_YAML" ]]; then
  echo "ERROR: $LABELS_YAML not found." >&2
  exit 2
fi
if ! command -v yq >/dev/null 2>&1; then
  echo "ERROR: yq is required (mikefarah/yq v4). Install: https://github.com/mikefarah/yq" >&2
  exit 2
fi

mapfile -t TYPE_LABELS < <(yq -r '.categories.type.labels[].name' "$LABELS_YAML")
mapfile -t SCOPE_LABELS < <(yq -r '.categories.scope.labels[].name' "$LABELS_YAML")
TYPE_PATTERN=$(printf '%s\n' "${TYPE_LABELS[@]}" | paste -sd '|' -)

TITLE=""
BODY_FILE=""
TEMPLATE="task"
LABELS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --title)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --title requires a non-empty value" >&2
        exit 2
      }
      TITLE="$2"
      shift 2
      ;;
    --body-file)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --body-file requires a non-empty path" >&2
        exit 2
      }
      BODY_FILE="$2"
      shift 2
      ;;
    --template)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --template requires task or epic" >&2
        exit 2
      }
      TEMPLATE="$2"
      shift 2
      ;;
    --label)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --label requires a non-empty value" >&2
        exit 2
      }
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

if [[ ! -f "$BODY_FILE" ]]; then
  echo "ERROR: body file not found: $BODY_FILE" >&2
  exit 2
fi

BODY=$(cat "$BODY_FILE")
ERRORS=()

strip_comments() {
  # shellcheck disable=SC2001
  sed 's/<!--[^>]*-->//g' <<< "$1" | sed '/^[[:space:]]*$/d'
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
