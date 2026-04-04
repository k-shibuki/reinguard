#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

HOME_SUBDIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --home-subdir)
      require_flag_value "--home-subdir" "${2:-}" "--home-subdir requires a non-empty directory name"
      HOME_SUBDIR="$2"
      shift 2
      ;;
    --)
      shift
      break
      ;;
    *)
      break
      ;;
  esac
done

if [[ $# -eq 0 ]]; then
  fail_with "with-repo-local-state.sh requires a command to run." 2
fi

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  fail_with "with-repo-local-state.sh must run inside a Git repository." 2
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
STATE_ROOT="$REPO_ROOT/.tmp"
mkdir -p \
  "$STATE_ROOT/xdg-cache" \
  "$STATE_ROOT/pre-commit-cache" \
  "$STATE_ROOT/go-build-cache" \
  "$STATE_ROOT/golangci-lint-cache"

export REINGUARD_LOCAL_STATE_ROOT="$STATE_ROOT"
# Do not set TMPDIR to a path under the repo root: Go's t.TempDir() would then
# fall inside this git work tree and break tests that expect a non-git directory.
export XDG_CACHE_HOME="$STATE_ROOT/xdg-cache"
export PRE_COMMIT_HOME="$STATE_ROOT/pre-commit-cache"
export GOCACHE="$STATE_ROOT/go-build-cache"
export GOLANGCI_LINT_CACHE="$STATE_ROOT/golangci-lint-cache"

if [[ -n "$HOME_SUBDIR" ]]; then
  mkdir -p "$STATE_ROOT/$HOME_SUBDIR"
  export HOME="$STATE_ROOT/$HOME_SUBDIR"
fi

exec "$@"
