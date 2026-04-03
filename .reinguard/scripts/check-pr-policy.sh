#!/usr/bin/env bash
# .reinguard/scripts/check-pr-policy.sh — Local pre-flight check for PR policy compliance.
# Mirrors the checks in .github/workflows/pr-policy.yaml so issues are caught
# before `gh pr create`. Type labels are read from .reinguard/labels.yaml (requires yq).
#
# Usage:
#   bash .reinguard/scripts/check-pr-policy.sh --title "chore(scope): summary" \
#       --body-file /tmp/pr-body.md --label chore [--base main]
#
# --title, --body-file, and at least one --label are required.
# --base defaults to main (must be main to match gate-policy CI).
set -euo pipefail

# labels.sh uses `local -n` (Bash 4.3+). Fail fast with guidance for macOS /bin/bash 3.2.
if [ "${BASH_VERSINFO[0]:-0}" -lt 4 ] || { [ "${BASH_VERSINFO[0]:-0}" -eq 4 ] && [ "${BASH_VERSINFO[1]:-0}" -lt 3 ]; }; then
  echo "check-pr-policy.sh requires Bash 4.3+ (uses local -n via labels.sh). Current: ${BASH_VERSION:-unknown}" >&2
  echo "Install a newer bash (e.g. brew install bash) and run: bash $0 ..." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=lib/labels.sh
source "$SCRIPT_DIR/lib/labels.sh"

LABELS_YAML="$(require_labels_yaml "$SCRIPT_DIR")"
require_command "yq" "yq is required. Install: https://github.com/mikefarah/yq" 2
load_label_names "$LABELS_YAML" '.categories.type.labels[].name' TYPE_LABELS
load_label_names "$LABELS_YAML" '.categories.exception.labels[].name' EXCEPTION_LABELS
TYPE_PATTERN="$(join_with_pipe "${TYPE_LABELS[@]}")"

TITLE=""
BODY_FILE=""
BASE="main"
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
    --base)
      require_flag_value "--base" "${2:-}" "--base requires a non-empty branch name"
      BASE="$2"
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
  echo "Usage: check-pr-policy.sh --title <title> --body-file <file> --label <type> [--label ...] [--base main]" >&2
  exit 2
fi

require_file "$BODY_FILE" "body file not found: $BODY_FILE" 2

BODY=$(cat "$BODY_FILE")
ERRORS=()
WARNINGS=()

# 1. Issue linkage
if ! grep -qiE '(closes|fixes|resolves)\s+#[0-9]+' <<< "$BODY"; then
  IS_EXCEPTION=false
  for el in "${EXCEPTION_LABELS[@]}"; do
    for l in "${LABELS[@]}"; do
      [[ "$l" == "$el" ]] && IS_EXCEPTION=true
    done
  done
  if $IS_EXCEPTION; then
    JUST=$(sed -n '/## Exception/,/^## /p' <<< "$BODY" | grep -i 'Justification:' | sed 's/.*Justification:\s*//' | sed 's/<!--[^>]*-->//g' | xargs)
    if [[ ${#JUST} -lt 20 ]]; then
      ERRORS+=("Issue linkage: Exception label set but justification missing or too short (min 20 chars).")
    else
      WARNINGS+=("Issue linkage: Exception — no Issue linked. Justification accepted.")
    fi
  else
    ERRORS+=("Issue linkage: body must contain 'Closes #N' (or Fixes/Resolves). Or add no-issue/hotfix label + Exception section.")
  fi
fi

# 2. Summary section
if ! grep -qiE '^#{1,2} Summary' <<< "$BODY"; then
  ERRORS+=("Summary: section missing from body.")
fi

# 3. Traceability section
if ! grep -qi '## Traceability' <<< "$BODY"; then
  ERRORS+=("Traceability: section missing from body.")
fi

# 4. Definition of Done section
if ! grep -qiE '## (Acceptance Criteria|Definition of Done)' <<< "$BODY"; then
  ERRORS+=("Definition of Done: section missing from body.")
fi

# 5. Test plan section (non-empty)
HAS_TEST_PLAN_SECTION=false
if grep -qiE '^## Test [Pp]lan([[:space:]]*)$' <<< "$BODY"; then
  HAS_TEST_PLAN_SECTION=true
fi
TEST_PLAN=$(awk '
  BEGIN { on = 0 }
  /^##[[:space:]]+/ {
    rest = $0
    sub(/^##[[:space:]]+/, "", rest)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", rest)
    rl = tolower(rest)
    gsub(/[[:space:]]+/, " ", rl)
    if (rl == "test plan") { on = 1; next }
    if (on) exit
    next
  }
  on { print }
' <<< "$BODY")
TEST_PLAN_CLEAN=$(strip_html_comments_and_blank_lines "$TEST_PLAN")
if [[ "$HAS_TEST_PLAN_SECTION" != true ]]; then
  ERRORS+=("Test plan: section missing from body.")
elif [[ ${#TEST_PLAN_CLEAN} -lt 5 ]]; then
  ERRORS+=("Test plan: section exists but appears empty.")
fi

# 6. Risk / Impact section (non-empty; case-insensitive like pr-policy.yaml)
RISK=$(awk -v want="risk / impact" '
  /^## / {
    rest = $0
    sub(/^##[[:space:]]+/, "", rest)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", rest)
    rl = tolower(rest)
    gsub(/[[:space:]]+/, " ", rl)
    if (rl == want) { on = 1; next }
    if (on) { on = 0 }
    next
  }
  on { print }
' <<< "$BODY")
RISK_CLEAN=$(strip_html_comments_and_blank_lines "$RISK")
if [[ -z "$RISK" ]]; then
  ERRORS+=("Risk / Impact: section missing from body.")
elif [[ ${#RISK_CLEAN} -lt 5 ]]; then
  ERRORS+=("Risk / Impact: section exists but appears empty.")
fi

# 7. Rollback Plan section (non-empty; case-insensitive like pr-policy.yaml)
ROLLBACK=$(awk -v want="rollback plan" '
  /^## / {
    rest = $0
    sub(/^##[[:space:]]+/, "", rest)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", rest)
    rl = tolower(rest)
    gsub(/[[:space:]]+/, " ", rl)
    if (rl == want) { on = 1; next }
    if (on) { on = 0 }
    next
  }
  on { print }
' <<< "$BODY")
ROLLBACK_CLEAN=$(strip_html_comments_and_blank_lines "$ROLLBACK")
if [[ -z "$ROLLBACK" ]]; then
  ERRORS+=("Rollback Plan: section missing from body.")
elif [[ ${#ROLLBACK_CLEAN} -lt 3 ]]; then
  ERRORS+=("Rollback Plan: section exists but appears empty.")
fi

# 8. PR title format (Conventional Commits; types from labels.yaml categories.type)
TITLE_RE="^($TYPE_PATTERN)(\\(.+\\))?!?: .+$"
if ! grep -qE "$TITLE_RE" <<< "$TITLE"; then
  ERRORS+=("PR title: must match Conventional Commits: <type>(<scope>): <summary>. Got: $TITLE")
fi

# 9. Type label (exactly one)
HITS=()
for tl in "${TYPE_LABELS[@]}"; do
  for l in "${LABELS[@]}"; do
    [[ "$l" == "$tl" ]] && HITS+=("$tl")
  done
done
if [[ ${#HITS[@]} -eq 0 ]]; then
  ERRORS+=("Type label: must have exactly one type label. Got none.")
elif [[ ${#HITS[@]} -gt 1 ]]; then
  ERRORS+=("Type label: multiple type labels (${HITS[*]}). Keep exactly one.")
fi

# 10. Base branch (HS-PR-BASE; mirrors pr-policy.yaml)
if [[ "$BASE" != "main" ]]; then
  ERRORS+=("Base branch: PR must target main. Got: $BASE. Document stack deps in the PR body instead of using --base feat/...")
fi

# Report
if [[ ${#ERRORS[@]} -gt 0 ]]; then
  echo "PR policy pre-flight FAILED:" >&2
  for e in "${ERRORS[@]}"; do
    echo "  ERROR: $e" >&2
  done
  for w in "${WARNINGS[@]}"; do
    echo "  WARNING: $w" >&2
  done
  exit 1
fi

if [[ ${#WARNINGS[@]} -gt 0 ]]; then
  for w in "${WARNINGS[@]}"; do
    echo "  WARNING: $w" >&2
  done
fi
echo "PR policy pre-flight OK."
