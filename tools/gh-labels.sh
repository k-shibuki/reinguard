#!/usr/bin/env bash
# Idempotent: create type + exception labels for PR policy (see docs/contributing.md).
# Requires: gh CLI, repo admin or label permission.
set -euo pipefail

create_label() {
  local name="$1"
  local color="$2"
  local desc="$3"
  if gh label list --json name -q ".[].name" 2>/dev/null | grep -qx "$name"; then
    echo "exists: $name"
  else
    gh label create "$name" --color "$color" --description "$desc"
    echo "created: $name"
  fi
}

# Type labels (exactly one required on PRs, per pr-policy.yaml)
create_label "feat" "A2EEEF" "Type: new feature"
create_label "fix" "D73A4A" "Type: bug fix"
create_label "refactor" "FEF2C0" "Type: refactor"
create_label "perf" "5319E7" "Type: performance"
create_label "docs" "0075CA" "Type: documentation"
create_label "test" "0E8A16" "Type: tests"
create_label "ci" "FBCA04" "Type: CI config"
create_label "build" "D4C5F9" "Type: build system"
create_label "chore" "C2E0C6" "Type: chore"
create_label "style" "BFDADC" "Type: formatting / style"
create_label "revert" "D4C5F9" "Type: revert"

# Exception labels (workflow-policy.mdc)
create_label "hotfix" "B60205" "Exception: urgent fix without normal Issue flow"
create_label "no-issue" "F9D0C4" "Exception: PR without linked Issue (justification required)"

echo "Done."
