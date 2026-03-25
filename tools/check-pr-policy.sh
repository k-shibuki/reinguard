#!/usr/bin/env bash
# tools/check-pr-policy.sh -- Local pre-flight check for PR policy compliance.
# Mirrors the checks in .github/workflows/pr-policy.yaml so issues are caught
# before `gh pr create`.
#
# Usage:
#   bash tools/check-pr-policy.sh --title "chore(scope): summary" \
#       --body-file /tmp/pr-body.md --label chore [--base main]
#
# --title, --body-file, and at least one --label are required.
# --base defaults to main (must be main or master to match check-policy CI).
set -euo pipefail

TITLE=""
BODY_FILE=""
BASE="main"
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
    --base)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --base requires a non-empty branch name" >&2
        exit 2
      }
      BASE="$2"
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
  echo "Usage: check-pr-policy.sh --title <title> --body-file <file> --label <type> [--label ...] [--base main]" >&2
  exit 2
fi

if [[ ! -f "$BODY_FILE" ]]; then
  echo "ERROR: body file not found: $BODY_FILE" >&2
  exit 2
fi

BODY=$(cat "$BODY_FILE")
ERRORS=()
WARNINGS=()

strip_comments() {
  # shellcheck disable=SC2001
  # HTML comment strip needs sed regex; not replaceable by bash ${var//}.
  sed 's/<!--[^>]*-->//g' <<< "$1" | sed '/^[[:space:]]*$/d'
}

# 1. Issue linkage
if ! grep -qiE '(closes|fixes|resolves)\s+#[0-9]+' <<< "$BODY"; then
  EXCEPTION_LABELS=("no-issue" "hotfix")
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
TEST_PLAN=$(sed -n '/## Test [Pp]lan/,/^## /p' <<< "$BODY" | tail -n +2)
TEST_PLAN_CLEAN=$(strip_comments "$TEST_PLAN")
if [[ -z "$TEST_PLAN" ]]; then
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
RISK_CLEAN=$(strip_comments "$RISK")
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
ROLLBACK_CLEAN=$(strip_comments "$ROLLBACK")
if [[ -z "$ROLLBACK" ]]; then
  ERRORS+=("Rollback Plan: section missing from body.")
elif [[ ${#ROLLBACK_CLEAN} -lt 3 ]]; then
  ERRORS+=("Rollback Plan: section exists but appears empty.")
fi

# 8. PR title format (Conventional Commits)
TITLE_RE='^(feat|fix|refactor|perf|test|docs|build|ci|chore|style|revert)(\(.+\))?!?: .+$'
if ! grep -qE "$TITLE_RE" <<< "$TITLE"; then
  ERRORS+=("PR title: must match Conventional Commits: <type>(<scope>): <summary>. Got: $TITLE")
fi

# 9. Type label (exactly one)
TYPE_LABELS=(feat fix refactor perf docs test ci build chore style revert)
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
if [[ "$BASE" != "main" && "$BASE" != "master" ]]; then
  ERRORS+=("Base branch: PR must target main (or master). Got: $BASE. Document stack deps in the PR body instead of using --base feat/...")
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
