#!/usr/bin/env bash
# tools/check-commit-msg.sh -- Validate commit message format
# Used as a commit-msg hook via pre-commit framework.
# Usage: bash tools/check-commit-msg.sh <commit-msg-file>
set -euo pipefail

MSG_FILE="${1:?Usage: check-commit-msg.sh <commit-msg-file>}"

# Strip comment lines and trailing whitespace
MSG=$(sed '/^#/d;/^$/d' "$MSG_FILE" | head -1)
BODY=$(sed '/^#/d' "$MSG_FILE" | tail -n +2)

if [[ -z "$MSG" ]]; then
  echo "ERROR: Empty commit message." >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TYPES_FILE="$SCRIPT_DIR/commit-types.txt"

if [[ ! -f "$TYPES_FILE" ]]; then
  echo "ERROR: $TYPES_FILE not found." >&2
  exit 1
fi

TYPES=$(grep -v '^#' "$TYPES_FILE" | grep -v '^$' | paste -sd '|' -)
PATTERN="^($TYPES)(\(.+\))?!?: .+"

if ! echo "$MSG" | grep -Eq "$PATTERN"; then
  echo "ERROR: First line must match Conventional Commits format:" >&2
  echo "  <type>(<scope>): <summary>" >&2
  echo "  Types: $TYPES" >&2
  echo "  Got: $MSG" >&2
  exit 1
fi

LEN=${#MSG}
if [[ $LEN -gt 120 ]]; then
  echo "ERROR: First line is $LEN chars (max 120)." >&2
  exit 1
elif [[ $LEN -gt 72 ]]; then
  echo "WARNING: First line is $LEN chars (recommended <=72)." >&2
fi

TYPE=$(echo "$MSG" | sed -E "s/^($TYPES).*/\1/")

if [[ "$TYPE" != "hotfix" && "$TYPE" != "docs" ]]; then
  if ! echo "$BODY" | grep -Eq 'Refs: #[0-9]+'; then
    echo "ERROR: Missing 'Refs: #<number>' in commit body." >&2
    echo "  Required for type '$TYPE'. Exception: hotfix/docs may omit." >&2
    exit 1
  fi
else
  BODY_CONTENT=$(echo "$BODY" | sed '/^$/d' | tr -d '[:space:]')
  if [[ -z "$BODY_CONTENT" ]]; then
    echo "ERROR: $TYPE commits must include justification in body." >&2
    exit 1
  fi
fi

exit 0
