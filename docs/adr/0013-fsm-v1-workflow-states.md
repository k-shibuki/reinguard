# ADR-0013: FSM v1 workflow states and Adapter mapping

## Status

Accepted (design note; amend when observation or Adapter contracts change).

## Context

reinguard needs a **single priority space** (ADR-0004) for declarative workflow
position from GitHub + git observation, without copying an external tool’s state
inventory. Cursor commands are **derived** from this FSM as thin Adapter bridges
(ADR-0001, Epic #59).

## Decision

### 1. State catalog (v1)

States live in `.reinguard/control/states/*.yaml`. **Lower numeric `priority`
wins** among matching rules (ADR-0004). Draft `state_id` values:

| state_id | Intent | Notes |
|----------|--------|--------|
| `bot_rate_limited` | Tracked reviewer comment signals rate limit | `any` on `github.reviews.tracked_reviewer_status` |
| `bot_review_paused` | Pause / status comment on tracked reviewer | Same array; higher priority number than rate limit |
| `changes_requested` | Blocking review decision | `github.reviews.review_decisions_changes_requested` > 0 |
| `ready_to_merge` | Coarse merge gate (clean tree, CI, threads, decisions) | Aligns with `merge-readiness` guard signals |
| `pr_open` | PR exists; work in flight | `github.pull_requests.pr_exists_for_branch` true |
| `working_no_pr` | No PR for branch (or PR facet absent) | `pr_exists_for_branch` false or path missing; not detached HEAD |

Consensus-style conditions (approved + no changes + threads resolved) are
**expressed as rules**, not a single derived observation key (see issue #72).

### 2. Routes (v1)

Routes in `.reinguard/control/routes/*.yaml` key off flattened `state.state_id`
after state resolution (same mechanism as `rgd route select` with merged state).

| route_id | Typical state_id | Procedure hint (Semantics) |
|----------|------------------|----------------------------|
| `cursor-implement` | `working_no_pr` | `implement` |
| `cursor-pr-create` | `working_no_pr` | `pr-create` (after local work) |
| `cursor-monitor-pr` | `pr_open` | `review-address` / observe |
| `cursor-address-review` | `changes_requested` | `review-address` |
| `cursor-wait-bot` | `bot_rate_limited`, `bot_review_paused` | wait / retry per bot docs |
| `cursor-merge` | `ready_to_merge` | `pr-merge` |

Multiple routes may match one state; **lowest route `priority` wins** for the
primary `route_id` in `rgd route select` output. Alternatives appear in
`route_candidates` (ADR-0004 / CLI docs).

### 3. Guards

`merge-readiness` (built-in) stays gated by declarative rules in
`control/guards/*.yaml` as today. State `ready_to_merge` is a **coarse** picture;
`guard eval merge-readiness` remains the explicit merge gate signal.

### 4. Adapter mapping (durable)

| state_id (v1) | Primary procedure (Semantics) |
|---------------|------------------------------|
| `working_no_pr` | `.reinguard/procedure/implement.md` |
| `working_no_pr` | `.reinguard/procedure/pr-create.md` (when opening a PR) |
| `pr_open` | `.reinguard/procedure/review-address.md` |
| `changes_requested` | `.reinguard/procedure/review-address.md` |
| `bot_rate_limited` / `bot_review_paused` | `.reinguard/knowledge/review--bot-operations.md` (+ `knowledge.entries`) |
| `ready_to_merge` | `.reinguard/procedure/pr-merge.md` |

Self-inspection before PR creation: `.reinguard/procedure/change-inspect.md`.
Post-review learning: `.reinguard/procedure/internalize.md`.

**Cursor entry:** `.cursor/commands/rgd-next.md` — run/read `rgd context build`,
map `state_id` → procedure paths above; no per-procedure Adapter stubs.

## Consequences

- **Easier**: One YAML-defined FSM; a single Cursor entry (`rgd-next`) routes to procedures.
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
