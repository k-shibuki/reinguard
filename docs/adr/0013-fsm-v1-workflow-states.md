# ADR-0013: FSM v1 workflow states and Adapter mapping

## Status

Accepted (design note; amend when observation or Adapter contracts change).

## Context

reinguard needs a **single priority space** (ADR-0004) for declarative workflow
position from GitHub + git observation, without copying an external tool’s state
inventory. Cursor commands are **derived** from this FSM as thin Adapter bridges
(ADR-0001, Epic #59).

Bot review position uses **`bot_reviewers`** in `reinguard.yaml` and derived
signals `github.reviews.bot_reviewer_status` plus aggregate
`github.reviews.bot_review_diagnostics` so the FSM stays **bot-agnostic** (not
hard-coded to a single vendor).

## Decision

### 1. State catalog (v2)

States live in `.reinguard/control/states/*.yaml`. **Lower numeric `priority`
wins** among matching rules (ADR-0004). `state_id` values:

| state_id | Intent | Notes |
|----------|--------|--------|
| `unresolved_threads` | Actionable review threads remain open | `github.reviews.review_threads_unresolved` > 0 (GraphQL `reviewThreads` with `isResolved` false). **Stronger than** bot-wait states when both match so agents fix threads instead of waiting on bots. |
| `changes_requested` | Formal GitHub “Request changes” on the PR | `github.reviews.review_decisions_changes_requested` > 0 (`latestReviews` with state `CHANGES_REQUESTED`). **Not** the same as open review threads; a bot may leave threads without a CHANGES_REQUESTED review. |
| `waiting_bot_rate_limited` | Required bot `status` is `rate_limited` | `op: any` on `github.reviews.bot_reviewer_status` with `$.required` and `$.status` |
| `waiting_bot_paused` | Required bot `status` is `review_paused` | Same pattern |
| `waiting_bot_failed` | Any required bot in failed tier (aggregate) | `github.reviews.bot_review_diagnostics.bot_review_failed` |
| `waiting_bot_run` | Waiting on required bot outcome | `bot_review_diagnostics.bot_review_pending` and PR exists |
| `merge_ready` | Coarse merge gate (clean tree, CI, threads, decisions) | Aligns with `merge-readiness` guard signals |
| `waiting_ci` | PR open; no thread/decision work; CI or mergeability not satisfied | Threads 0, changes 0, working tree clean; `ci_status` ≠ `success` **or** `merge_state_status` ≠ `clean` |
| `pr_open` | PR exists; residual (e.g. dirty working tree) | `github.pull_requests.pr_exists_for_branch` true |
| `working_no_pr` | No PR for branch (or PR facet absent) | `pr_exists_for_branch` false or path missing |

**Bot status tiers** (per-element `status` in `bot_reviewer_status`):

- **Reviewed (success path):** `completed` (bot finished review; issues may or may not have been found), `completed_clean` (bot finished review, explicitly reported a clean result such as "No issues found", and has a corresponding review entry)
- **Failed:** `rate_limited`, `review_paused`, `review_failed`
- **In progress:** `pending`, `not_triggered`

**Diagnostics:** `bot_review_completed` means every **required** bot is in the
Reviewed tier and none in the Failed tier; `bot_review_failed` if any required
bot is in the Failed tier; `bot_review_terminal` = failed OR completed;
`bot_review_pending` = not terminal. Optional bots (`required: false`) do not
affect aggregates.

Consensus-style conditions (approved + no changes + threads resolved) are
**expressed as rules**, not a single derived observation key (see issue #72).

### 2. Routes (v2)

Routes in `.reinguard/control/routes/*.yaml` key off flattened `state.state_id`
after state resolution (same mechanism as `rgd route select` with merged state).

| route_id | Typical state_id | Procedure hint (Semantics) |
|----------|------------------|----------------------------|
| `user-implement` | `working_no_pr` | `implement` |
| `user-monitor-pr` | `pr_open` | Observe PR / residual |
| `user-wait-ci` | `waiting_ci` | `review-address` (checks / mergeability) |
| `user-address-review` | `unresolved_threads`, `changes_requested` | `review-address` |
| `user-wait-bot-failed` | `waiting_bot_failed` | `wait-bot-review` |
| `user-wait-bot-run` | `waiting_bot_run` | `wait-bot-review` |
| `user-wait-bot-quota` | `waiting_bot_rate_limited` | `wait-bot-review` |
| `user-wait-bot-paused` | `waiting_bot_paused` | `wait-bot-review` |
| `user-merge` | `merge_ready` | `pr-merge` |

`user-*` names are **Adapter-agnostic** (not tied to a specific IDE). A given Adapter maps `rgd` output to local commands.

`pr-create` (after local work) still applies when `state_id` is `working_no_pr`; there is no separate `route_id` for it in v1.

Multiple routes may match one state; **lowest route `priority` wins** for the
primary `route_id` in `rgd route select` output. Alternatives appear in
`route_candidates` (ADR-0004 / CLI docs).

### 3. Guards

`merge-readiness` (built-in) stays gated by declarative rules in
`control/guards/*.yaml` as today. State `merge_ready` is a **coarse** picture;
`guard eval merge-readiness` remains the explicit merge gate signal.

### 4. Adapter mapping (durable)

| state_id | Primary procedure (Semantics) |
|----------|------------------------------|
| `working_no_pr` | `.reinguard/procedure/implement.md` |
| `working_no_pr` | `.reinguard/procedure/pr-create.md` (when opening a PR) |
| `pr_open` | `.reinguard/procedure/review-address.md` |
| `waiting_ci` | `.reinguard/procedure/review-address.md` |
| `unresolved_threads` | `.reinguard/procedure/review-address.md` |
| `changes_requested` | `.reinguard/procedure/review-address.md` |
| `waiting_bot_rate_limited` / `waiting_bot_paused` / `waiting_bot_failed` / `waiting_bot_run` | `.reinguard/procedure/wait-bot-review.md` (+ `review--bot-operations.md` in `knowledge.entries`) |
| `merge_ready` | `.reinguard/procedure/pr-merge.md` |

Self-inspection before PR creation: `.reinguard/procedure/change-inspect.md`.
Post-review learning: `.reinguard/procedure/internalize.md`.

**Cursor entries:** `.cursor/commands/rgd-next.md` — run/read `rgd context build`,
Route (`state_id` → procedure) per § 4 above; no per-procedure Adapter stubs.
Orchestration (mandatory after Sense and Route: single full-path **Propose**, one approval, then **Execute** to DoD):
`.reinguard/procedure/next-orchestration.md` — contract referenced from `rgd-next.md` § Propose and § Execute;
not state-mapped.
`.cursor/commands/cursor-plan.md` — Plan-mode-style interrogation (`AskQuestion` /
`CreatePlan` only); GitHub Issue creation is expressed inside the plan when
issue-first (Phase 3B content); not part of the FSM.

## Consequences

- **Easier**: One YAML-defined FSM; `rgd-next` routes substrate output to procedures.
- **Harder**: Priority authoring must stay global across states/routes/guards
  (ADR-0004).
- **Harder**: Observation gaps mean broader states (e.g. `working_no_pr` when PR
  facet is missing) — agents must not over-interpret.

## Refs

- ADR-0002 (match / `when`)
- ADR-0004 (priority resolution)
- ADR-0011 (control plane layout, `procedure/`)
- `docs/cli.md` (signal paths)
- Issue #72 (P2-3 deliverable)
