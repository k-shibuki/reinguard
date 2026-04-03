#!/usr/bin/env bash
# .reinguard/scripts/check-local-review.sh — Required local CodeRabbit CLI review
# before PR creation. This script standardizes installation/auth checks and
# review invocation. Finding triage remains part of change-inspect.
#
# Usage:
#   bash .reinguard/scripts/check-local-review.sh [--base main] [--mode plain|prompt-only|agent] [--type all|committed|uncommitted]
set -euo pipefail

BASE="main"
MODE="plain"
REVIEW_TYPE="all"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --base requires a non-empty branch name" >&2
        exit 2
      }
      BASE="$2"
      shift 2
      ;;
    --mode)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --mode requires one of: plain, prompt-only, agent" >&2
        exit 2
      }
      MODE="$2"
      shift 2
      ;;
    --type)
      [[ $# -ge 2 && -n "${2:-}" && "${2:0:1}" != "-" ]] || {
        echo "ERROR: --type requires one of: all, committed, uncommitted" >&2
        exit 2
      }
      REVIEW_TYPE="$2"
      shift 2
      ;;
    *)
      echo "Unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

case "$MODE" in
  plain|prompt-only|agent) ;;
  *)
    echo "ERROR: --mode must be plain, prompt-only, or agent. Got: $MODE" >&2
    exit 2
    ;;
esac

case "$REVIEW_TYPE" in
  all|committed|uncommitted) ;;
  *)
    echo "ERROR: --type must be all, committed, or uncommitted. Got: $REVIEW_TYPE" >&2
    exit 2
    ;;
esac

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "ERROR: check-local-review.sh must run inside a Git repository." >&2
  exit 2
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

CONFIG_FILE=".coderabbit.yaml"
if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "ERROR: $CONFIG_FILE is required for the repository-local CodeRabbit gate." >&2
  exit 2
fi

if command -v coderabbit >/dev/null 2>&1; then
  CR_BIN="coderabbit"
elif command -v cr >/dev/null 2>&1; then
  CR_BIN="cr"
else
  cat >&2 <<'EOF'
ERROR: CodeRabbit CLI is not installed.
Install one of:
  curl -fsSL https://cli.coderabbit.ai/install.sh | sh
  brew install coderabbit
Then authenticate with:
  cr auth login
EOF
  exit 2
fi

# CodeRabbit CLI currently exposes human-readable auth status. Keep the parsing
# conservative and fail closed until a documented machine-readable mode exists.
AUTH_STATUS_RC=0
AUTH_STATUS_OUTPUT="$("$CR_BIN" auth status 2>&1)" || AUTH_STATUS_RC=$?
AUTH_STATUS_CLEAN="$(printf '%s\n' "$AUTH_STATUS_OUTPUT" | sed -E 's/\x1B\[[0-9;?]*[[:alpha:]]//g' | tr -d '\r')"
if [[ $AUTH_STATUS_RC -ne 0 ]]; then
  echo "ERROR: CodeRabbit CLI is not authenticated. Run: $CR_BIN auth login" >&2
  exit 2
fi
if grep -Eqi "not logged in|unauthenticated" <<< "$AUTH_STATUS_CLEAN"; then
  echo "ERROR: CodeRabbit CLI is not authenticated. Run: $CR_BIN auth login" >&2
  exit 2
fi

echo "Running CodeRabbit local review..."
echo "  Base branch: $BASE"
echo "  Review type: $REVIEW_TYPE"
echo "  Output mode: $MODE"
echo "  Config file: $CONFIG_FILE"
echo

REVIEW_RC=0
REVIEW_OUTPUT="$("$CR_BIN" review "--$MODE" --type "$REVIEW_TYPE" --base "$BASE" -c "$CONFIG_FILE" --no-color 2>&1)" || REVIEW_RC=$?
printf '%s\n' "$REVIEW_OUTPUT"
if [[ $REVIEW_RC -ne 0 ]]; then
  REVIEW_OUTPUT_CLEAN="$(printf '%s\n' "$REVIEW_OUTPUT" | sed -E 's/\x1B\[[0-9;?]*[[:alpha:]]//g' | tr -d '\r')"
  if grep -qi "rate limit exceeded" <<< "$REVIEW_OUTPUT_CLEAN"; then
    echo "ERROR: CodeRabbit CLI is rate limited. Wait for the reported cooldown and retry before PR creation." >&2
  else
    echo "ERROR: CodeRabbit local review failed. Resolve the CLI error and rerun before PR creation." >&2
  fi
  exit 2
fi

cat <<'EOF'

CodeRabbit local review completed.
Address Blocking findings before PR creation and disposition any remaining
non-blocking findings during change-inspect.
EOF
