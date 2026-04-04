#!/usr/bin/env bash
# Fail if go tool cover total (statements) is below MIN (default 80).
# Usage: check-coverage-threshold.sh [MIN] [COVERPROFILE]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"

min="${1:-80}"
profile="${2:-coverage.out}"
require_file "$profile" "coverage profile not found: $profile" 1
pct=$(go tool cover -func="$profile" | awk '/^total:/{gsub(/%/,"",$NF); print $NF; exit}')
awk -v p="$pct" -v m="$min" 'BEGIN {
  if (p + 0 < m + 0) {
    printf("total coverage %.1f%% is below required %.1f%%\n", p, m) > "/dev/stderr"
    exit 1
  }
  exit 0
}'
echo "total coverage: ${pct}% (minimum ${min}%)"
