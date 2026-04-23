#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=.reinguard/scripts/lib/json_minimal.sh
source "$SCRIPT_DIR/lib/json_minimal.sh"

SCRIPT_NAME="adapter-rgd-next-resume.sh"
# SSOT for the resume artifact schema version (ADR-0015).
RESUME_SCHEMA_VERSION="1.0.0"

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  fail_with "$SCRIPT_NAME must run inside a Git repository." 2
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
# Adapter-local state under .reinguard/local (not tool caches in .tmp/).
# REINGUARD_LOCAL_DIR overrides the root (for tests); default is $REPO_ROOT/.reinguard/local
LOCAL_ROOT="${REINGUARD_LOCAL_DIR:-$REPO_ROOT/.reinguard/local}"
ARTIFACT_DIR="$LOCAL_ROOT/adapter/rgd-next"
ARTIFACT_PATH="$ARTIFACT_DIR/execute-resume.json"

artifact_branch=""
artifact_created_at=""
artifact_updated_at=""
artifact_approval_at=""
artifact_status=""
artifact_issue=""
artifact_pr=""
artifact_summary=""
artifact_approved_head_sha=""
artifact_approved_state=""
artifact_approved_route=""
artifact_ordered_remainder=""
artifact_completion_condition=""
artifact_proposal_fingerprint=""
artifact_last_state=""
artifact_last_route=""
artifact_last_recorded_at=""
artifact_terminal_reason=""
artifact_terminal_summary=""
artifact_terminal_recorded_at=""
status_reason_codes=()

current_branch() {
  local branch
  branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
  printf '%s' "$branch"
}

current_head_sha() {
  local sha
  sha="$(git rev-parse HEAD 2>/dev/null || true)"
  printf '%s' "$sha"
}

reset_status_reason_codes() {
  status_reason_codes=()
}

append_status_reason_code() {
  local code="$1"
  local existing
  for existing in ${status_reason_codes[@]+"${status_reason_codes[@]}"}; do
    if [[ "$existing" == "$code" ]]; then
      return 0
    fi
  done
  status_reason_codes+=("$code")
}

compute_proposal_fingerprint() {
  printf '%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n' \
    "$artifact_branch" \
    "$artifact_issue" \
    "$artifact_pr" \
    "$artifact_approved_head_sha" \
    "$artifact_approved_state" \
    "$artifact_approved_route" \
    "$artifact_ordered_remainder" \
    "$artifact_completion_condition" \
    "$artifact_summary" | sha256sum | awk '{print $1}'
}

emit_json_string_array() {
  local key="$1"
  shift
  local values=("$@")
  local i
  printf '  "%s": [' "$(json_escape "$key")"
  for i in "${!values[@]}"; do
    if (( i > 0 )); then
      printf ', '
    fi
    printf '"%s"' "$(json_escape "${values[$i]}")"
  done
  printf ']'
}

build_resume_context_json() {
  if [[ -n "${REINGUARD_RGD_NEXT_CONTEXT_BUILD_FILE:-}" ]]; then
    require_file "$REINGUARD_RGD_NEXT_CONTEXT_BUILD_FILE" "resume context file is missing" 2
    cat "$REINGUARD_RGD_NEXT_CONTEXT_BUILD_FILE"
    return 0
  fi
  if command -v rgd >/dev/null 2>&1; then
    (
      cd "$REPO_ROOT" &&
      rgd context build --compact
    )
    return
  fi
  if [[ -f "$REPO_ROOT/cmd/rgd/main.go" ]]; then
    (
      cd "$REPO_ROOT" &&
      go run ./cmd/rgd context build --compact
    )
    return
  fi
  return 1
}

is_resumable_wait_state() {
  # Resumable wait states per next-orchestration.md § Allowed stops and
  # .reinguard/control/states/workflow.yaml. Keep in sync with that SSOT.
  case "$1" in
    waiting_ci|waiting_bot_rate_limited|waiting_bot_paused|waiting_bot_stale|waiting_bot_run|waiting_bot_failed)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

parse_resume_context_fields() {
  local raw="$1"
  python3 -c '
import json
import shlex
import sys

fields = {
    "fresh_context_parse_error": "",
    "fresh_state_kind": "",
    "fresh_state_id": "",
    "fresh_route_kind": "",
    "fresh_route_id": "",
    "fresh_review_trigger_awaiting_ack": "false",
    "fresh_bot_review_trigger_awaiting_ack": "false",
}

try:
    payload = json.load(sys.stdin)
except Exception as exc:
    fields["fresh_context_parse_error"] = str(exc)
else:
    state = payload.get("state") or {}
    routes = payload.get("routes") or []
    route = routes[0] if routes and isinstance(routes[0], dict) else {}
    signals = (payload.get("observation") or {}).get("signals") or {}
    github = signals.get("github") or {}
    reviews = github.get("reviews") or {}
    fields["fresh_state_kind"] = str(state.get("kind") or "")
    fields["fresh_state_id"] = str(state.get("state_id") or "")
    fields["fresh_route_kind"] = str(route.get("kind") or "")
    fields["fresh_route_id"] = str(route.get("route_id") or "")
    fields["fresh_review_trigger_awaiting_ack"] = "true" if bool(reviews.get("review_trigger_awaiting_ack")) else "false"
    fields["fresh_bot_review_trigger_awaiting_ack"] = "true" if bool(reviews.get("bot_review_trigger_awaiting_ack")) else "false"

for key, value in fields.items():
    print(f"{key}={shlex.quote(value)}")
' <<<"$raw"
}

resume_ttl_state() {
  local approval_at="$1"
  local ttl_seconds="$2"
  python3 - "$approval_at" "$ttl_seconds" <<'PY'
from datetime import datetime, timezone
import sys

approval_at = sys.argv[1]
ttl_seconds = int(sys.argv[2])

try:
    approved = datetime.strptime(approval_at, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
except ValueError:
    print("parse_error")
    raise SystemExit(0)

age = (datetime.now(timezone.utc) - approved).total_seconds()
print("expired" if age > ttl_seconds else "valid")
PY
}

require_positive_integer() {
  local flag="$1"
  local value="$2"
  if [[ ! $value =~ ^[0-9]+$ ]] || (( value <= 0 )); then
    fail_with "$flag must be a positive integer" 2
  fi
}

ensure_artifact_branch_matches_current() {
  local cur
  cur="$(current_branch)"
  [[ -n "$cur" ]] || fail_with "current branch is unavailable (detached HEAD)" 2
  [[ -n "$artifact_branch" ]] || fail_with "artifact has no branch" 2
  [[ "$artifact_branch" == "$cur" ]] || fail_with "artifact belongs to branch '$artifact_branch' (current: '$cur')" 2
}

load_artifact() {
  local raw approved_contract last_iteration terminal
  raw="$(<"$ARTIFACT_PATH")"

  artifact_branch="$(json_get_string "$raw" "branch")"
  artifact_created_at="$(json_get_string "$raw" "created_at")"
  artifact_updated_at="$(json_get_string "$raw" "updated_at")"
  artifact_approval_at="$(json_get_string "$raw" "approval_recorded_at")"
  artifact_status="$(json_get_string "$raw" "status")"
  artifact_issue="$(json_get_number "$raw" "issue_number")"
  artifact_pr="$(json_get_number "$raw" "pr_number")"
  artifact_summary="$(json_get_string "$raw" "summary")"

  approved_contract="$(json_get_block "$raw" "approved_contract")"
  artifact_approved_head_sha="$(json_get_string "$approved_contract" "head_sha")"
  artifact_approved_state="$(json_get_string "$approved_contract" "state_id")"
  artifact_approved_route="$(json_get_string "$approved_contract" "route_id")"
  artifact_ordered_remainder="$(json_get_string "$approved_contract" "ordered_remainder")"
  artifact_completion_condition="$(json_get_string "$approved_contract" "completion_condition")"
  artifact_proposal_fingerprint="$(json_get_string "$approved_contract" "proposal_fingerprint")"

  last_iteration="$(json_get_block "$raw" "last_iteration")"
  artifact_last_state="$(json_get_string "$last_iteration" "state_id")"
  artifact_last_route="$(json_get_string "$last_iteration" "route_id")"
  artifact_last_recorded_at="$(json_get_string "$last_iteration" "recorded_at")"

  terminal="$(json_get_block "$raw" "terminal")"
  artifact_terminal_reason="$(json_get_string "$terminal" "reason")"
  artifact_terminal_summary="$(json_get_string "$terminal" "summary")"
  artifact_terminal_recorded_at="$(json_get_string "$terminal" "recorded_at")"
}

write_artifact_file() {
  mkdir -p "$ARTIFACT_DIR"
  {
    printf '{\n'
    printf '  "schema_version": "%s",\n' "$RESUME_SCHEMA_VERSION"
    printf '  "artifact_type": "adapter_rgd_next_resume",\n'
    printf '  "command": "rgd-next",\n'
    printf '  "status": "%s",\n' "$(json_escape "$artifact_status")"
    printf '  "branch": "%s",\n' "$(json_escape "$artifact_branch")"
    printf '  "approval_recorded_at": "%s",\n' "$(json_escape "$artifact_approval_at")"
    printf '  "created_at": "%s",\n' "$(json_escape "$artifact_created_at")"
    if [[ -n "$artifact_issue" ]]; then
      printf '  "issue_number": %s,\n' "$artifact_issue"
    fi
    if [[ -n "$artifact_pr" ]]; then
      printf '  "pr_number": %s,\n' "$artifact_pr"
    fi
    if [[ -n "$artifact_summary" ]]; then
      printf '  "summary": "%s",\n' "$(json_escape "$artifact_summary")"
    fi
    printf '  "updated_at": "%s",\n' "$(json_escape "$artifact_updated_at")"
    printf '  "approved_contract": {\n'
    printf '    "head_sha": "%s",\n' "$(json_escape "$artifact_approved_head_sha")"
    printf '    "state_id": "%s",\n' "$(json_escape "$artifact_approved_state")"
    if [[ -n "$artifact_approved_route" ]]; then
      printf '    "route_id": "%s",\n' "$(json_escape "$artifact_approved_route")"
    fi
    printf '    "ordered_remainder": "%s",\n' "$(json_escape "$artifact_ordered_remainder")"
    printf '    "completion_condition": "%s",\n' "$(json_escape "$artifact_completion_condition")"
    printf '    "proposal_fingerprint": "%s"\n' "$(json_escape "$artifact_proposal_fingerprint")"
    printf '  }'
    if [[ -n "$artifact_last_state" || -n "$artifact_terminal_reason" ]]; then
      printf ',\n'
    else
      printf '\n'
    fi
    if [[ -n "$artifact_last_state" ]]; then
      printf '  "last_iteration": {\n'
      printf '    "state_id": "%s",\n' "$(json_escape "$artifact_last_state")"
      if [[ -n "$artifact_last_route" ]]; then
        printf '    "route_id": "%s",\n' "$(json_escape "$artifact_last_route")"
      fi
      printf '    "recorded_at": "%s"\n' "$(json_escape "$artifact_last_recorded_at")"
      printf '  }'
      if [[ -n "$artifact_terminal_reason" ]]; then
        printf ',\n'
      else
        printf '\n'
      fi
    fi
    if [[ -n "$artifact_terminal_reason" ]]; then
      printf '  "terminal": {\n'
      printf '    "reason": "%s",\n' "$(json_escape "$artifact_terminal_reason")"
      if [[ -n "$artifact_terminal_summary" ]]; then
        printf '    "summary": "%s",\n' "$(json_escape "$artifact_terminal_summary")"
      fi
      printf '    "recorded_at": "%s"\n' "$(json_escape "$artifact_terminal_recorded_at")"
      printf '  }\n'
    fi
    printf '}\n'
  } >"$ARTIFACT_PATH"
}

emit_status_json() {
  local status="$1"
  local resume_eligible="$2"
  local reason="${3:-}"
  local current
  current="$(current_branch)"

  {
    printf '{\n'
    printf '  "artifact_path": "%s",\n' "$(json_escape "$ARTIFACT_PATH")"
    printf '  "current_branch": "%s",\n' "$(json_escape "$current")"
    printf '  "status": "%s",\n' "$(json_escape "$status")"
    printf '  "resume_eligible": %s' "$resume_eligible"
    if [[ -n "$reason" || ${#status_reason_codes[@]} -gt 0 || -n "$artifact_branch" || -n "$artifact_issue" || -n "$artifact_pr" || -n "$artifact_summary" || -n "$artifact_approval_at" || -n "$artifact_created_at" || -n "$artifact_updated_at" || -n "$artifact_approved_head_sha" || -n "$artifact_last_state" || -n "$artifact_terminal_reason" ]]; then
      printf ',\n'
    else
      printf '\n'
    fi
    if [[ -n "$reason" ]]; then
      printf '  "reason": "%s"' "$(json_escape "$reason")"
      if [[ ${#status_reason_codes[@]} -gt 0 || -n "$artifact_branch" || -n "$artifact_issue" || -n "$artifact_pr" || -n "$artifact_summary" || -n "$artifact_approval_at" || -n "$artifact_created_at" || -n "$artifact_updated_at" || -n "$artifact_approved_head_sha" || -n "$artifact_last_state" || -n "$artifact_terminal_reason" ]]; then
        printf ',\n'
      else
        printf '\n'
      fi
    fi
    if [[ ${#status_reason_codes[@]} -gt 0 ]]; then
      emit_json_string_array "resume_reason_codes" "${status_reason_codes[@]}"
      if [[ -n "$artifact_branch" || -n "$artifact_issue" || -n "$artifact_pr" || -n "$artifact_summary" || -n "$artifact_approval_at" || -n "$artifact_created_at" || -n "$artifact_updated_at" || -n "$artifact_approved_head_sha" || -n "$artifact_last_state" || -n "$artifact_terminal_reason" ]]; then
        printf ',\n'
      else
        printf '\n'
      fi
    fi
    if [[ -n "$artifact_branch" ]]; then
      printf '  "branch": "%s",\n' "$(json_escape "$artifact_branch")"
    fi
    if [[ -n "$artifact_issue" ]]; then
      printf '  "issue_number": %s,\n' "$artifact_issue"
    fi
    if [[ -n "$artifact_pr" ]]; then
      printf '  "pr_number": %s,\n' "$artifact_pr"
    fi
    if [[ -n "$artifact_summary" ]]; then
      printf '  "summary": "%s",\n' "$(json_escape "$artifact_summary")"
    fi
    if [[ -n "$artifact_approval_at" ]]; then
      printf '  "approval_recorded_at": "%s",\n' "$(json_escape "$artifact_approval_at")"
    fi
    if [[ -n "$artifact_created_at" ]]; then
      printf '  "created_at": "%s",\n' "$(json_escape "$artifact_created_at")"
    fi
    if [[ -n "$artifact_updated_at" ]]; then
      printf '  "updated_at": "%s",\n' "$(json_escape "$artifact_updated_at")"
      printf '  "approved_contract": {\n'
      printf '    "head_sha": "%s",\n' "$(json_escape "$artifact_approved_head_sha")"
      printf '    "state_id": "%s",\n' "$(json_escape "$artifact_approved_state")"
      if [[ -n "$artifact_approved_route" ]]; then
        printf '    "route_id": "%s",\n' "$(json_escape "$artifact_approved_route")"
      fi
      printf '    "ordered_remainder": "%s",\n' "$(json_escape "$artifact_ordered_remainder")"
      printf '    "completion_condition": "%s",\n' "$(json_escape "$artifact_completion_condition")"
      printf '    "proposal_fingerprint": "%s"\n' "$(json_escape "$artifact_proposal_fingerprint")"
      printf '  }'
      if [[ -n "$artifact_last_state" || -n "$artifact_terminal_reason" ]]; then
        printf ',\n'
      else
        printf '\n'
      fi
    fi
    if [[ -n "$artifact_last_state" ]]; then
      printf '  "last_iteration": {\n'
      printf '    "state_id": "%s",\n' "$(json_escape "$artifact_last_state")"
      if [[ -n "$artifact_last_route" ]]; then
        printf '    "route_id": "%s",\n' "$(json_escape "$artifact_last_route")"
      fi
      printf '    "recorded_at": "%s"\n' "$(json_escape "$artifact_last_recorded_at")"
      printf '  }'
      if [[ -n "$artifact_terminal_reason" ]]; then
        printf ',\n'
      else
        printf '\n'
      fi
    fi
    if [[ -n "$artifact_terminal_reason" ]]; then
      printf '  "terminal": {\n'
      printf '    "reason": "%s",\n' "$(json_escape "$artifact_terminal_reason")"
      if [[ -n "$artifact_terminal_summary" ]]; then
        printf '    "summary": "%s",\n' "$(json_escape "$artifact_terminal_summary")"
      fi
      printf '    "recorded_at": "%s"\n' "$(json_escape "$artifact_terminal_recorded_at")"
      printf '  }\n'
    fi
    printf '}\n'
  }
}

usage() {
  cat <<EOF2
Usage:
  $SCRIPT_NAME start --branch BRANCH --state-id ID --ordered-remainder TEXT --completion-condition TEXT [--route-id ID] [--issue N] [--pr N] [--summary TEXT]
  $SCRIPT_NAME approve
  $SCRIPT_NAME update --state-id ID [--route-id ID]
  $SCRIPT_NAME finish --status done|allowed_stop|revoked --reason REASON [--summary TEXT]
  $SCRIPT_NAME status
  $SCRIPT_NAME show
  $SCRIPT_NAME clear

Terminal reason enum:
  dod_satisfied
  hard_stop
  cannot_proceed
  tooling_session_limit
  scope_revoked
EOF2
}

start_cmd() {
  local branch="" state_id="" route_id="" ordered_remainder="" completion_condition="" issue="" pr="" summary="" now
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --branch)
        require_flag_value "--branch" "${2:-}" "--branch requires a non-empty branch name"
        branch="$2"
        shift 2
        ;;
      --issue)
        require_flag_value "--issue" "${2:-}" "--issue requires a positive integer"
        issue="$2"
        shift 2
        ;;
      --state-id)
        require_flag_value "--state-id" "${2:-}" "--state-id requires a non-empty value"
        state_id="$2"
        shift 2
        ;;
      --route-id)
        require_flag_value "--route-id" "${2:-}" "--route-id requires a non-empty value"
        route_id="$2"
        shift 2
        ;;
      --ordered-remainder)
        require_flag_value "--ordered-remainder" "${2:-}" "--ordered-remainder requires a non-empty value"
        ordered_remainder="$2"
        shift 2
        ;;
      --completion-condition)
        require_flag_value "--completion-condition" "${2:-}" "--completion-condition requires a non-empty value"
        completion_condition="$2"
        shift 2
        ;;
      --pr)
        require_flag_value "--pr" "${2:-}" "--pr requires a positive integer"
        pr="$2"
        shift 2
        ;;
      --summary)
        require_flag_value "--summary" "${2:-}" "--summary requires a non-empty value"
        summary="$2"
        shift 2
        ;;
      *)
        fail_with "unknown flag for start: $1" 2
        ;;
    esac
  done

  [[ -n "$branch" ]] || fail_with "--branch is required for start" 2
  [[ -n "$state_id" ]] || fail_with "--state-id is required for start" 2
  [[ -n "$ordered_remainder" ]] || fail_with "--ordered-remainder is required for start" 2
  [[ -n "$completion_condition" ]] || fail_with "--completion-condition is required for start" 2
  [[ -z "$issue" ]] || require_positive_integer "--issue" "$issue"
  [[ -z "$pr" ]] || require_positive_integer "--pr" "$pr"

  now="$(json_now_utc)"
  artifact_branch="$branch"
  artifact_status="pending_approval"
  artifact_issue="$issue"
  artifact_pr="$pr"
  artifact_summary="$summary"
  artifact_approved_head_sha="$(current_head_sha)"
  [[ -n "$artifact_approved_head_sha" ]] || fail_with "current HEAD SHA is unavailable" 2
  artifact_approved_state="$state_id"
  artifact_approved_route="$route_id"
  artifact_ordered_remainder="$ordered_remainder"
  artifact_completion_condition="$completion_condition"
  artifact_proposal_fingerprint="$(compute_proposal_fingerprint)"
  artifact_approval_at=""
  artifact_created_at="$now"
  artifact_updated_at="$now"
  artifact_last_state=""
  artifact_last_route=""
  artifact_last_recorded_at=""
  artifact_terminal_reason=""
  artifact_terminal_summary=""
  artifact_terminal_recorded_at=""
  write_artifact_file
  cat "$ARTIFACT_PATH"
}

approve_cmd() {
  require_file "$ARTIFACT_PATH" "resume artifact is missing; start must run before approve" 2
  load_artifact
  ensure_artifact_branch_matches_current
  [[ "$artifact_status" == "pending_approval" ]] || fail_with "approve requires status pending_approval (got $artifact_status)" 2
  local now
  now="$(json_now_utc)"
  artifact_status="active"
  artifact_approval_at="$now"
  artifact_updated_at="$now"
  write_artifact_file
  cat "$ARTIFACT_PATH"
}

update_cmd() {
  local state_id="" route_id=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --state-id)
        require_flag_value "--state-id" "${2:-}" "--state-id requires a non-empty value"
        state_id="$2"
        shift 2
        ;;
      --route-id)
        require_flag_value "--route-id" "${2:-}" "--route-id requires a non-empty value"
        route_id="$2"
        shift 2
        ;;
      *)
        fail_with "unknown flag for update: $1" 2
        ;;
    esac
  done

  [[ -n "$state_id" ]] || fail_with "--state-id is required for update" 2
  require_file "$ARTIFACT_PATH" "resume artifact is missing; start must run before update" 2
  load_artifact
  ensure_artifact_branch_matches_current
  [[ "$artifact_status" == "active" ]] || fail_with "only an active artifact can be updated" 2

  artifact_last_state="$state_id"
  artifact_last_route="$route_id"
  artifact_last_recorded_at="$(json_now_utc)"
  artifact_updated_at="$artifact_last_recorded_at"
  write_artifact_file
  cat "$ARTIFACT_PATH"
}

finish_cmd() {
  local status="" reason="" summary=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --status)
        require_flag_value "--status" "${2:-}" "--status requires done, allowed_stop, or revoked"
        status="$2"
        shift 2
        ;;
      --reason)
        require_flag_value "--reason" "${2:-}" "--reason requires a terminal reason value"
        reason="$2"
        shift 2
        ;;
      --summary)
        require_flag_value "--summary" "${2:-}" "--summary requires a non-empty value"
        summary="$2"
        shift 2
        ;;
      *)
        fail_with "unknown flag for finish: $1" 2
        ;;
    esac
  done

  [[ -n "$status" && -n "$reason" ]] || fail_with "--status and --reason are required for finish" 2
  case "$status" in
    done|allowed_stop|revoked) ;;
    *) fail_with "--status must be one of done, allowed_stop, revoked" 2 ;;
  esac
  case "$reason" in
    dod_satisfied|hard_stop|cannot_proceed|tooling_session_limit|scope_revoked) ;;
    *) fail_with "--reason must be one of dod_satisfied, hard_stop, cannot_proceed, tooling_session_limit, scope_revoked" 2 ;;
  esac
  [[ "$status" != "done" || "$reason" == "dod_satisfied" ]] || fail_with "done status requires reason dod_satisfied" 2
  [[ "$status" != "revoked" || "$reason" == "scope_revoked" ]] || fail_with "revoked status requires reason scope_revoked" 2
  [[ "$status" != "allowed_stop" || ( "$reason" != "dod_satisfied" && "$reason" != "scope_revoked" ) ]] || fail_with "allowed_stop requires hard_stop, cannot_proceed, or tooling_session_limit" 2

  require_file "$ARTIFACT_PATH" "resume artifact is missing; start must run before finish" 2
  load_artifact
  ensure_artifact_branch_matches_current

  # Resumable-wait guard (Issue #132, ADR-0015, next-orchestration.md § Allowed stops):
  # allowed_stop with cannot_proceed or tooling_session_limit must not silently terminalize a
  # resumable wait state. Only hard_stop is permitted for those conditions, and only with a
  # genuinely unrecoverable external block. hard_stop, dod_satisfied, and scope_revoked do not
  # need the fresh-observation check because they are not substitutes for a wait/retry procedure.
  if [[ "$status" == "allowed_stop" && ( "$reason" == "cannot_proceed" || "$reason" == "tooling_session_limit" ) ]]; then
    local fresh_context fresh_context_parse_error="" fresh_state_kind="" fresh_state_id=""
    local fresh_route_kind="" fresh_route_id=""
    local fresh_review_trigger_awaiting_ack="false" fresh_bot_review_trigger_awaiting_ack="false"
    if ! fresh_context="$(build_resume_context_json)"; then
      fail_with "finish refused: cannot verify resumable-wait guard — fresh 'rgd context build --compact' unavailable (see next-orchestration.md § Allowed stops). Run it successfully, or use --reason hard_stop with evidence of a genuinely unrecoverable external block." 2
    fi
    eval "$(parse_resume_context_fields "$fresh_context")"
    if [[ -n "$fresh_context_parse_error" ]]; then
      fail_with "finish refused: fresh 'rgd context build --compact' JSON could not be parsed ($fresh_context_parse_error); cannot verify resumable-wait guard. Re-run substrate observation, or use --reason hard_stop with evidence." 2
    fi
    if [[ "$fresh_state_kind" == "resolved" ]] && is_resumable_wait_state "$fresh_state_id"; then
      fail_with "finish refused: fresh state_id '$fresh_state_id' is a resumable wait state (waiting_ci / waiting_bot_*); allowed_stop with reason '$reason' is forbidden for resumable waits (see next-orchestration.md § Allowed stops). Continue the mapped wait procedure (cooldown + re-trigger) and refresh context, or use --reason hard_stop with evidence of a genuinely unrecoverable external block." 2
    fi
  fi

  artifact_status="$status"
  artifact_terminal_reason="$reason"
  artifact_terminal_summary="$summary"
  artifact_terminal_recorded_at="$(json_now_utc)"
  artifact_updated_at="$artifact_terminal_recorded_at"
  write_artifact_file
  cat "$ARTIFACT_PATH"
}

status_cmd() {
  local raw current current_head current_context status_ttl_seconds ttl_state
  local fresh_context_parse_error="" fresh_state_kind="" fresh_state_id="" fresh_route_kind="" fresh_route_id=""
  local fresh_review_trigger_awaiting_ack="false" fresh_bot_review_trigger_awaiting_ack="false"
  local recomputed_fingerprint=""
  local -a missing=()
  current="$(current_branch)"
  current_head="$(current_head_sha)"
  reset_status_reason_codes

  artifact_branch=""
  artifact_created_at=""
  artifact_updated_at=""
  artifact_approval_at=""
  artifact_status=""
  artifact_issue=""
  artifact_pr=""
  artifact_summary=""
  artifact_approved_head_sha=""
  artifact_approved_state=""
  artifact_approved_route=""
  artifact_ordered_remainder=""
  artifact_completion_condition=""
  artifact_proposal_fingerprint=""
  artifact_last_state=""
  artifact_last_route=""
  artifact_last_recorded_at=""
  artifact_terminal_reason=""
  artifact_terminal_summary=""
  artifact_terminal_recorded_at=""

  if [[ ! -f "$ARTIFACT_PATH" ]]; then
    append_status_reason_code "artifact_missing"
    emit_status_json "missing" "false"
    return 0
  fi

  raw="$(<"$ARTIFACT_PATH")"
  for key in schema_version artifact_type command status branch created_at updated_at approved_contract; do
    if ! json_has_key "$raw" "$key"; then
      missing+=("$key")
    fi
  done
  if (( ${#missing[@]} > 0 )); then
    append_status_reason_code "artifact_missing_required_keys"
    emit_status_json "invalid" "false" "missing required keys: $(IFS=', '; echo "${missing[*]}")"
    return 0
  fi

  load_artifact
  if [[ -z "$artifact_approved_head_sha" || -z "$artifact_approved_state" || -z "$artifact_ordered_remainder" || -z "$artifact_completion_condition" || -z "$artifact_proposal_fingerprint" ]]; then
    append_status_reason_code "approved_contract_incomplete"
    emit_status_json "invalid" "false" "approved_contract is missing required fields"
    return 0
  fi
  if [[ "$(json_get_string "$raw" "artifact_type")" != "adapter_rgd_next_resume" || "$(json_get_string "$raw" "command")" != "rgd-next" ]]; then
    append_status_reason_code "artifact_identity_mismatch"
    emit_status_json "invalid" "false" "unexpected artifact_type or command"
    return 0
  fi
  if [[ ( "$artifact_status" == "active" || "$artifact_status" == "pending_approval" ) && -n "$artifact_branch" ]]; then
    if [[ -z "$current" ]]; then
      append_status_reason_code "detached_head"
      emit_status_json "stale" "false" "detached HEAD"
      return 0
    fi
    if [[ "$artifact_branch" != "$current" ]]; then
      append_status_reason_code "branch_mismatch"
      emit_status_json "stale" "false" "branch mismatch"
      return 0
    fi
    append_status_reason_code "branch_match"
  fi
  if [[ "$artifact_status" == "active" ]]; then
    recomputed_fingerprint="$(compute_proposal_fingerprint)"
    if [[ "$recomputed_fingerprint" != "$artifact_proposal_fingerprint" ]]; then
      append_status_reason_code "proposal_fingerprint_mismatch"
      emit_status_json "invalid" "false" "proposal fingerprint mismatch"
      return 0
    fi
    append_status_reason_code "proposal_fingerprint_match"
    if [[ -z "$artifact_approval_at" ]]; then
      append_status_reason_code "approval_record_missing"
      emit_status_json "invalid" "false" "active artifact is missing approval_recorded_at"
      return 0
    fi
    if [[ -z "$current_head" ]]; then
      append_status_reason_code "current_head_unavailable"
      emit_status_json "stale" "false" "current HEAD is unavailable"
      return 0
    fi
    if [[ "$current_head" != "$artifact_approved_head_sha" ]]; then
      append_status_reason_code "head_mismatch"
      emit_status_json "stale" "false" "head mismatch"
      return 0
    fi
    append_status_reason_code "head_match"
    status_ttl_seconds="${REINGUARD_RGD_NEXT_RESUME_TTL_SECONDS:-}"
    if [[ -n "$status_ttl_seconds" ]]; then
      if [[ ! "$status_ttl_seconds" =~ ^[0-9]+$ ]] || (( status_ttl_seconds <= 0 )); then
        append_status_reason_code "ttl_config_invalid"
        emit_status_json "invalid" "false" "REINGUARD_RGD_NEXT_RESUME_TTL_SECONDS must be a positive integer"
        return 0
      fi
      ttl_state="$(resume_ttl_state "$artifact_approval_at" "$status_ttl_seconds")"
      if [[ "$ttl_state" == "parse_error" ]]; then
        append_status_reason_code "approval_record_invalid"
        emit_status_json "invalid" "false" "approval_recorded_at is not RFC3339 UTC"
        return 0
      fi
      if [[ "$ttl_state" == "expired" ]]; then
        append_status_reason_code "ttl_expired"
        emit_status_json "stale" "false" "resume TTL expired"
        return 0
      fi
      append_status_reason_code "ttl_valid"
    fi
    if ! current_context="$(build_resume_context_json)"; then
      append_status_reason_code "fresh_context_unavailable"
      emit_status_json "stale" "false" "fresh context build failed"
      return 0
    fi
    eval "$(parse_resume_context_fields "$current_context")"
    if [[ -n "$fresh_context_parse_error" ]]; then
      append_status_reason_code "fresh_context_invalid"
      emit_status_json "stale" "false" "fresh context JSON could not be parsed"
      return 0
    fi
    append_status_reason_code "fresh_context_loaded"
    if [[ "$fresh_state_kind" != "resolved" ]]; then
      append_status_reason_code "fresh_context_unresolved"
      emit_status_json "stale" "false" "fresh context state is not resolved"
      return 0
    fi
    append_status_reason_code "state_resolved"
    if [[ "$fresh_state_id" != "$artifact_approved_state" ]]; then
      append_status_reason_code "state_mismatch"
      if [[ "$fresh_review_trigger_awaiting_ack" == "true" ]]; then
        append_status_reason_code "review_trigger_awaiting_ack_true"
      fi
      if [[ "$fresh_bot_review_trigger_awaiting_ack" == "true" ]]; then
        append_status_reason_code "bot_review_trigger_awaiting_ack_true"
      fi
      emit_status_json "stale" "false" "state mismatch"
      return 0
    fi
    append_status_reason_code "state_match"
    if [[ -n "$artifact_approved_route" ]]; then
      if [[ "$fresh_route_kind" != "resolved" ]]; then
        append_status_reason_code "route_unresolved"
        emit_status_json "stale" "false" "fresh context route is not resolved"
        return 0
      fi
      if [[ "$fresh_route_id" != "$artifact_approved_route" ]]; then
        append_status_reason_code "route_mismatch"
        if [[ "$fresh_review_trigger_awaiting_ack" == "true" ]]; then
          append_status_reason_code "review_trigger_awaiting_ack_true"
        fi
        if [[ "$fresh_bot_review_trigger_awaiting_ack" == "true" ]]; then
          append_status_reason_code "bot_review_trigger_awaiting_ack_true"
        fi
        emit_status_json "stale" "false" "route mismatch"
        return 0
      fi
      append_status_reason_code "route_match"
    fi
    emit_status_json "$artifact_status" "true"
    return 0
  fi
  emit_status_json "$artifact_status" "false"
}

show_cmd() {
  require_file "$ARTIFACT_PATH" "resume artifact is missing" 2
  cat "$ARTIFACT_PATH"
}

clear_cmd() {
  rm -f "$ARTIFACT_PATH"
  printf '{\n  "artifact_path": "%s",\n  "status": "cleared"\n}\n' "$ARTIFACT_PATH"
}

if [[ $# -eq 0 ]]; then
  usage
  exit 2
fi

subcommand="$1"
shift

case "$subcommand" in
  start)
    start_cmd "$@"
    ;;
  approve)
    approve_cmd "$@"
    ;;
  update)
    update_cmd "$@"
    ;;
  finish)
    finish_cmd "$@"
    ;;
  status)
    status_cmd "$@"
    ;;
  show)
    show_cmd "$@"
    ;;
  clear)
    clear_cmd "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    fail_with "unknown subcommand: $subcommand" 2
    ;;
esac
