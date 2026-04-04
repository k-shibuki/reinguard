#!/usr/bin/env bash
# .reinguard/scripts/check-local-review.sh — Required local CodeRabbit CLI review
# before PR creation. This script standardizes installation/auth checks and
# review invocation. Finding triage remains part of change-inspect.
#
# Usage:
#   bash .reinguard/scripts/check-local-review.sh [--base main] [--mode plain|prompt-only|agent] [--type all|committed|uncommitted] [--retry-on-rate-limit]
# Optional env: RATE_LIMIT_RETRY_BUFFER_SEC (default 30) — seconds after parsed cooldown before automatic retry.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

# Seconds added after the CLI-reported cooldown so the retry does not land on the boundary.
RATE_LIMIT_RETRY_BUFFER_SEC="${RATE_LIMIT_RETRY_BUFFER_SEC:-30}"
if ! [[ "$RATE_LIMIT_RETRY_BUFFER_SEC" =~ ^[0-9]+$ ]]; then
  fail_with "RATE_LIMIT_RETRY_BUFFER_SEC must be a non-negative integer. Got: $RATE_LIMIT_RETRY_BUFFER_SEC" 2
fi

BASE="main"
MODE="plain"
REVIEW_TYPE="all"
RETRY_ON_RATE_LIMIT=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base)
      require_flag_value "--base" "${2:-}" "--base requires a non-empty branch name"
      BASE="$2"
      shift 2
      ;;
    --mode)
      require_flag_value "--mode" "${2:-}" "--mode requires one of: plain, prompt-only, agent"
      MODE="$2"
      shift 2
      ;;
    --type)
      require_flag_value "--type" "${2:-}" "--type requires one of: all, committed, uncommitted"
      REVIEW_TYPE="$2"
      shift 2
      ;;
    --retry-on-rate-limit)
      RETRY_ON_RATE_LIMIT=1
      shift
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
  fail_with "check-local-review.sh must run inside a Git repository." 2
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

CONFIG_FILE=".coderabbit.yaml"
require_file "$CONFIG_FILE" "$CONFIG_FILE is required for the repository-local CodeRabbit gate." 2

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
  coderabbit auth login
  # or: cr auth login
EOF
  exit 2
fi

# CodeRabbit CLI currently exposes human-readable auth status. Treat only known
# "logged in" output as success and fail closed on unrecognized text until a
# documented machine-readable mode exists.
AUTH_STATUS_RC=0
AUTH_STATUS_OUTPUT="$("$CR_BIN" auth status 2>&1)" || AUTH_STATUS_RC=$?
AUTH_STATUS_CLEAN="$(strip_ansi_cr "$AUTH_STATUS_OUTPUT")"
if [[ $AUTH_STATUS_RC -ne 0 ]]; then
  echo "ERROR: CodeRabbit CLI auth status failed:" >&2
  printf '%s\n' "$AUTH_STATUS_CLEAN" >&2
  exit 2
fi
# Reject explicit unauthenticated / negated phrasing before any positive match.
if grep -Eqi "unauthenticated|not[[:space:]]+logged[[:space:]]+in|not[[:space:]]+currently[[:space:]]+logged[[:space:]]+in" <<< "$AUTH_STATUS_CLEAN"; then
  echo "ERROR: CodeRabbit CLI is not authenticated. Run: $CR_BIN auth login" >&2
  exit 2
fi
# Do not treat bare "logged in" as success — it matches negated phrases (e.g. "not currently logged in").
if ! grep -Eqi "authentication:[[:space:]]*logged in" <<< "$AUTH_STATUS_CLEAN"; then
  echo "ERROR: CodeRabbit CLI auth status output was not recognized as authenticated." >&2
  printf '%s\n' "$AUTH_STATUS_CLEAN" >&2
  exit 2
fi

echo "Running CodeRabbit local review..."
echo "  Base branch: $BASE"
echo "  Review type: $REVIEW_TYPE"
echo "  Output mode: $MODE"
echo "  Config file: $CONFIG_FILE"
echo

# The last line containing "rate limit exceeded" (case-insensitive). Cooldown is parsed
# from this line only so unrelated footer text (e.g. "finished in N seconds") cannot
# satisfy extraction when the rate-limit line itself is unparseable.
last_rate_limit_line_only() {
  local text="$1"
  local -a lines
  local i start=-1

  mapfile -t lines <<< "$(printf '%s\n' "$text")"
  for ((i = 0; i < ${#lines[@]}; i++)); do
    if grep -qi "rate limit exceeded" <<< "${lines[$i]}"; then
      start=$i
    fi
  done
  if [[ $start -lt 0 ]]; then
    return 1
  fi
  printf '%s\n' "${lines[$start]}"
}

# Parse hours/minutes/seconds from a single rate-limit snippet (one CLI message block).
extract_rate_limit_seconds() {
  local text="$1"
  local lower parse_target hours minutes seconds total matched_any

  lower="$(printf '%s\n' "$text" | tr '[:upper:]' '[:lower:]')"
  parse_target="$lower"
  if [[ $parse_target == *"try after "* ]]; then
    parse_target="${parse_target#*try after }"
  elif [[ $parse_target == *"try again in "* ]]; then
    parse_target="${parse_target#*try again in }"
  elif [[ $parse_target == *"retry in "* ]]; then
    parse_target="${parse_target#*retry in }"
  fi
  hours=0
  minutes=0
  seconds=0
  matched_any=0

  if [[ $parse_target =~ ([0-9]+)[[:space:]]*hours? ]]; then
    hours="${BASH_REMATCH[1]}"
    matched_any=1
  fi
  if [[ $parse_target =~ ([0-9]+)[[:space:]]*minutes? ]]; then
    minutes="${BASH_REMATCH[1]}"
    matched_any=1
  fi
  if [[ $parse_target =~ ([0-9]+)[[:space:]]*seconds? ]]; then
    seconds="${BASH_REMATCH[1]}"
    matched_any=1
  fi

  if [[ $matched_any -eq 0 ]]; then
    return 1
  fi

  total=$((hours * 3600 + minutes * 60 + seconds))
  printf '%s\n' "$total"
}

attempt=1
max_attempts=1
if [[ $RETRY_ON_RATE_LIMIT -eq 1 ]]; then
  max_attempts=2
fi

while true; do
  REVIEW_RC=0
  REVIEW_OUTPUT="$("$CR_BIN" review "--$MODE" --type "$REVIEW_TYPE" --base "$BASE" -c "$CONFIG_FILE" --no-color 2>&1)" || REVIEW_RC=$?
  printf '%s\n' "$REVIEW_OUTPUT"
  if [[ $REVIEW_RC -eq 0 ]]; then
    break
  fi

  REVIEW_OUTPUT_CLEAN="$(strip_ansi_cr "$REVIEW_OUTPUT")"
  if grep -qi "rate limit exceeded" <<< "$REVIEW_OUTPUT_CLEAN"; then
    if [[ $RETRY_ON_RATE_LIMIT -eq 1 && $attempt -lt $max_attempts ]]; then
      LATEST_RL_LINE=""
      if LATEST_RL_LINE="$(last_rate_limit_line_only "$REVIEW_OUTPUT_CLEAN")"; then
        wait_seconds=""
        if wait_seconds="$(extract_rate_limit_seconds "$LATEST_RL_LINE")" && [[ "$wait_seconds" =~ ^[0-9]+$ ]]; then
          total_sleep=$((wait_seconds + RATE_LIMIT_RETRY_BUFFER_SEC))
          echo "" >&2
          echo "Rate limit detected on attempt ${attempt}/${max_attempts} (using latest rate-limit line from this CLI run only; ignoring earlier text)." >&2
          echo "Parsed cooldown: ${wait_seconds}s; safety buffer: ${RATE_LIMIT_RETRY_BUFFER_SEC}s; sleeping ${total_sleep}s before one automatic retry..." >&2
          sleep "$total_sleep"
          echo "Retrying CodeRabbit local review (attempt $((attempt + 1))/${max_attempts})..." >&2
          attempt=$((attempt + 1))
          continue
        fi
      fi
      echo "ERROR: CodeRabbit CLI reported rate limit but cooldown could not be parsed from the latest rate-limit line in this CLI run. Re-run after cooldown or check CLI output." >&2
      exit 2
    else
      if [[ $RETRY_ON_RATE_LIMIT -eq 1 && $attempt -eq $max_attempts ]]; then
        echo "ERROR: CodeRabbit CLI is rate limited again after automatic retry (second consecutive). Wait for the reported cooldown and rerun manually." >&2
      else
        echo "ERROR: CodeRabbit CLI is rate limited. Pass --retry-on-rate-limit for one automatic cooldown wait, or wait and rerun manually." >&2
      fi
    fi
  else
    echo "ERROR: CodeRabbit local review failed. Resolve the CLI error and rerun before PR creation." >&2
  fi
  exit 2
done

cat <<'EOF'

CodeRabbit local review completed.
Disposition findings in change-inspect using Fixed / By design / False
positive / Acknowledged. Before PR creation, Acknowledged requires a
follow-up Issue or another explicit deferred-work contract.
EOF
