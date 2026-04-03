#!/usr/bin/env bash
# Sync `.github/ISSUE_TEMPLATE/task.yml` Type dropdown from `.reinguard/labels.yaml`
# via `rgd labels list` (same SSOT). Requires `jq` and mikefarah `yq` v4
# (https://github.com/mikefarah/yq). If PATH `yq` is not mikefarah v4, a copy is
# downloaded to `.reinguard/scripts/.bin/yq` on first run.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=.reinguard/scripts/lib/yq.sh
source "$SCRIPT_DIR/lib/yq.sh"

REPO_ROOT="${REINGUARD_REPO_ROOT:-$(repo_root_from_script_dir "$SCRIPT_DIR")}"
TASK="${REINGUARD_TASK_TEMPLATE_PATH:-$REPO_ROOT/.github/ISSUE_TEMPLATE/task.yml}"
YQ_CACHE_DIR="$SCRIPT_DIR/.bin"

require_file "$TASK" "$TASK not found." 1
ensure_mikefarah_yq_cached "$YQ_CACHE_DIR" YQ_CMD

cd "$REPO_ROOT"
# Always use `go run` so we do not pick up a stale ./rgd or PATH rgd without labels commands.
if [[ -n "${REINGUARD_LABELS_NAMES_JSON:-}" ]]; then
  NAMES_JSON="$REINGUARD_LABELS_NAMES_JSON"
else
  require_command "jq" "jq is required" 1
  NAMES_JSON=$(go run ./cmd/rgd labels list --category type | jq -c '.names')
fi

TMP=$(mktemp)
TMPJSON="${TMP}.json"
trap 'cleanup_temp_files "$TMP" "$TMPJSON"' EXIT
printf '%s\n' "$NAMES_JSON" >"$TMPJSON"
# mikefarah yq: inject JSON array via load() (avoids --argjson portability issues).
"${YQ_CMD[@]}" eval ".body[0].attributes.options = load(\"$TMPJSON\")" -o yaml "$TASK" >"$TMP"
mv "$TMP" "$TASK"
echo "Updated $TASK Type dropdown from labels.yaml (via rgd labels list)."
