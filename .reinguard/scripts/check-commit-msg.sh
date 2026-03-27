#!/usr/bin/env bash
# .reinguard/scripts/check-commit-msg.sh — Validate commit message format
# Used as a commit-msg hook via pre-commit framework.
# Commit types are derived from .reinguard/labels.yaml (commit_prefix: true).
# Usage: bash .reinguard/scripts/check-commit-msg.sh <commit-msg-file>
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
LABELS_FILE="$SCRIPT_DIR/../labels.yaml"

if [[ ! -f "$LABELS_FILE" ]]; then
  echo "ERROR: $LABELS_FILE not found." >&2
  exit 1
fi
if ! command -v yq >/dev/null 2>&1; then
  echo "ERROR: yq is required for commit-msg validation. See CONTRIBUTING.md." >&2
  exit 1
fi

TYPES=$(yq -r '[.categories[].labels[] | select(.commit_prefix == true)] | .[].name' "$LABELS_FILE" | paste -sd '|' -)
PATTERN="^($TYPES)(\(.+\))?!?: .+"

if ! echo "$MSG" | grep -Eq "$PATTERN"; then
  echo "ERROR: First line must match Conventional Commits format:" >&2
  echo "  <type>(<scope>): <summary>" >&2
  echo "  Types (from labels.yaml): $TYPES" >&2
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
