#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=.reinguard/scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
# shellcheck source=.reinguard/scripts/lib/json_minimal.sh
source "$SCRIPT_DIR/lib/json_minimal.sh"

SCRIPT_NAME="adapter-rgd-next-resume.sh"

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
    printf '  "schema_version": "1",\n'
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
    if [[ -n "$reason" || -n "$artifact_branch" || -n "$artifact_issue" || -n "$artifact_pr" || -n "$artifact_summary" || -n "$artifact_approval_at" || -n "$artifact_created_at" || -n "$artifact_updated_at" || -n "$artifact_approved_head_sha" || -n "$artifact_last_state" || -n "$artifact_terminal_reason" ]]; then
      printf ',\n'
    else
      printf '\n'
    fi
    if [[ -n "$reason" ]]; then
      printf '  "reason": "%s"' "$(json_escape "$reason")"
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
  artifact_status="$status"
  artifact_terminal_reason="$reason"
  artifact_terminal_summary="$summary"
  artifact_terminal_recorded_at="$(json_now_utc)"
  artifact_updated_at="$artifact_terminal_recorded_at"
  write_artifact_file
  cat "$ARTIFACT_PATH"
}

status_cmd() {
  local raw current
  local -a missing=()
  current="$(current_branch)"

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
    emit_status_json "invalid" "false" "missing required keys: $(IFS=', '; echo "${missing[*]}")"
    return 0
  fi

  load_artifact
  if [[ -z "$artifact_approved_head_sha" || -z "$artifact_approved_state" || -z "$artifact_ordered_remainder" || -z "$artifact_completion_condition" || -z "$artifact_proposal_fingerprint" ]]; then
    emit_status_json "invalid" "false" "approved_contract is missing required fields"
    return 0
  fi
  if [[ "$(json_get_string "$raw" "artifact_type")" != "adapter_rgd_next_resume" || "$(json_get_string "$raw" "command")" != "rgd-next" ]]; then
    emit_status_json "invalid" "false" "unexpected artifact_type or command"
    return 0
  fi
  if [[ ( "$artifact_status" == "active" || "$artifact_status" == "pending_approval" ) && -n "$artifact_branch" ]]; then
    if [[ -z "$current" ]]; then
      emit_status_json "stale" "false" "detached HEAD"
      return 0
    fi
    if [[ "$artifact_branch" != "$current" ]]; then
      emit_status_json "stale" "false" "branch mismatch"
      return 0
    fi
  fi
  if [[ "$artifact_status" == "active" ]]; then
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
