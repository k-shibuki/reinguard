# ADR-0013: FSM workflow states and Adapter mapping

## Context

reinguard needs a **single priority space** (ADR-0004) for declarative workflow
position from GitHub + git observation, without copying an external tool's state
inventory. Cursor commands are **derived** from this FSM as thin Adapter bridges
(ADR-0001, Epic #59).

Bot review position uses **`bot_reviewers`** in `reinguard.yaml` and derived
signals `github.reviews.bot_reviewer_status` plus aggregate
`github.reviews.bot_review_diagnostics` so the FSM stays **bot-agnostic** (not
hard-coded to a single vendor).

## Decision

### 1. State catalog

States live in `.reinguard/control/states/*.yaml`. **Lower numeric `priority`
wins** among matching rules (ADR-0004). `state_id` values:

| state_id | Intent | Notes |
|----------|--------|--------|
| `unresolved_threads` | Actionable review threads remain open | `github.reviews.review_threads_unresolved` > 0 (GraphQL `reviewThreads` with `isResolved` false). **Stronger than** bot-wait states when both match so agents fix threads instead of waiting on bots. |
| `non_thread_findings_pending` | Required bot has non-thread review work (enrichment counts) | PR exists; threads 0; pagination complete; no formal changes-requested; `github.reviews.bot_review_diagnostics.non_thread_findings_present` true (Issue #105). **Priority** between `unresolved_threads` and `changes_requested`. |
| `changes_requested` | Formal GitHub "Request changes" on the PR | `github.reviews.review_decisions_changes_requested` > 0 (`latestReviews` with state `CHANGES_REQUESTED`). **Not** the same as open review threads; a bot may leave threads without a CHANGES_REQUESTED review. |
| `waiting_bot_rate_limited` | Required bot `status` is `rate_limited` | `op: any` on `github.reviews.bot_reviewer_status` with `$.required` and `$.status` |
| `waiting_bot_paused` | Required bot `status` is `review_paused` | Same pattern |
| `waiting_bot_failed` | Any required bot in failed tier (aggregate) | `github.reviews.bot_review_diagnostics.bot_review_failed` |
| `waiting_bot_stale` | Required bot completed review on a different HEAD | `bot_review_diagnostics.bot_review_stale` true and PR exists. **Priority vs `waiting_bot_rate_limited`:** state rule `rate_limited` (priority 11) wins over `bot_stale` (priority 14) when both could match — cool down / quota recovery must finish before treating stale re-trigger as the primary action. |
| `waiting_bot_run` | Waiting on required bot outcome | `bot_review_diagnostics.bot_review_pending` and PR exists |
| `merge_ready` | Coarse merge gate (clean tree, CI, threads, decisions, bot diagnostics) | Rule includes `bot_review_diagnostics` alignment with the `merge-readiness` built-in (pending/terminal/failed/stale/non-thread), plus mergeability and CI success — see `.reinguard/control/states/workflow.yaml` |
| `waiting_ci` | PR open; no thread/decision work; CI or mergeability not satisfied | Threads 0, changes 0, working tree clean; `ci_status` != `success` **or** `merge_state_status` != `clean` |
| `pr_open` | PR exists; residual (e.g. dirty working tree) | `github.pull_requests.pr_exists_for_branch` true |
| `ready_for_pr` | No PR exists, and `pr-readiness` is fresh and passing | `pr_exists_for_branch` false (or missing) and `gates.pr-readiness.status == pass`; `pr-readiness` is the runtime gate recorded by `.reinguard/procedure/change-inspect.md` per ADR-0014 (artifact under `.reinguard/local/gates/`). Its `pass` contract is proof-carrying: it must reference the fresh passing inputs required by `workflow.runtime_gate_roles.pr_readiness.pass_requires_roles` for the same subject. In this repository’s current default config, that means `local-verification` plus `local-coderabbit`, so the FSM can keep reading one gate without treating it as an opaque marker. |
| `working_no_pr` | No PR for branch (or PR facet absent); residual before PR readiness is proven | `pr_exists_for_branch` false or path missing, with `ready_for_pr` using a lower numeric `priority` than `working_no_pr` so it wins when the gate condition matches |

**Bot status tiers** (per-element `status` in `bot_reviewer_status`):

- **Reviewed (success path):**
  - `completed` - bot finished review; review findings may or may not have been reported.
  - `completed_clean` - all of the following are true:
    1. The bot finished review.
    2. The bot emitted an explicit clean marker recognized by the observation/provider implementation for that bot (see `docs/cli.md` signal notes for the current enrichment contract).
    3. A corresponding GitHub review entry for the same bot login is observable.
  - Timing rule: if the clean marker is observed before the matching
    GitHub review entry appears in the observation pass, classify as
    `completed` on that pass and upgrade to `completed_clean` on a later
    observation pass once both are visible. If the GitHub review entry
    appears first, remain `completed` until the clean marker is also
    observed.
- **Failed:** `rate_limited`, `review_paused`, `review_failed`
- **In progress:** `pending`, `not_triggered`

**Diagnostics:** `bot_review_completed` means every **required** bot is in the
Reviewed tier and none in the Failed tier; `bot_review_failed` if any required
bot is in the Failed tier; `bot_review_terminal` = failed OR completed;
`bot_review_pending` = not terminal. Optional bots (`required: false`) do not
affect aggregates.

Consensus-style conditions (approved + no changes + threads resolved) are
**expressed as rules**, not a single derived observation key (see issue #72).

### 2. Routes

Routes in `.reinguard/control/routes/*.yaml` key off flattened `state.state_id`
after state resolution (same mechanism as `rgd route select` with merged state).

| route_id | Typical state_id | Procedure hint (Semantics) |
|----------|------------------|----------------------------|
| `user-implement` | `working_no_pr`, `ready_for_pr` | `implement` while work is still in progress; `pr-create` when `state_id` is `ready_for_pr` |
| `user-monitor-pr` | `pr_open` | Observe PR / residual |
| `user-wait-ci` | `waiting_ci` | `review-address` (checks / mergeability) |
| `user-address-review` | `unresolved_threads`, `non_thread_findings_pending`, `changes_requested` | `review-address` |
| `user-wait-bot-failed` | `waiting_bot_failed` | `wait-bot-review` |
| `user-wait-bot-stale` | `waiting_bot_stale` | `wait-bot-review` |
| `user-wait-bot-run` | `waiting_bot_run` | `wait-bot-review` |
| `user-wait-bot-quota` | `waiting_bot_rate_limited` | `wait-bot-review` |
| `user-wait-bot-paused` | `waiting_bot_paused` | `wait-bot-review` |
| `user-merge` | `merge_ready` | `pr-merge` |

`user-*` names are **Adapter-agnostic** (not tied to a specific IDE). A given Adapter maps `rgd` output to local commands.

`pr-create` (after local work) applies when `state_id` is `ready_for_pr`; there
is still no separate `route_id` for it in this FSM.

Multiple routes may match one state; **lowest route `priority` wins** for the
primary `route_id` in `rgd route select` output. Alternatives appear in
`route_candidates` (ADR-0004 / CLI docs).

### 3. Guards

`merge-readiness` (built-in) stays gated by declarative rules in
`control/guards/*.yaml` as today. State `merge_ready` is a **coarse** picture;
`guard eval merge-readiness` remains the explicit merge gate signal.

### 4. Adapter mapping (machine-readable SSOT)

The mapping from a resolved workflow `state_id` to the **primary** Adapter
procedure document is **not** maintained as a Markdown table in this ADR.

**SSOT:** `.reinguard/procedure/*.md` YAML front matter (`applies_to.state_ids`,
and optionally `applies_to.route_ids`) per ADR-0011.

**Rules:**

- Each mapped `state_id` is claimed by **exactly one** procedure’s `applies_to.state_ids`. `rgd config validate` rejects duplicates and
  `state_id` values that do not appear on any `control/states` rule.
- Adapters resolve the procedure file by matching `state.state_id` from
  `rgd context build` against that front matter (and cross-check
  `routes[0].route_id` against `applies_to.route_ids` when procedures scope by
  route). Optional `state.procedure_hint` in context JSON, when present, is a
  derived advisory field from the same SSOT; the Markdown files remain
  authoritative.

Self-inspection before PR creation remains `.reinguard/procedure/change-inspect.md`;
it prepares `ready_for_pr` by recording the configured pre-PR AI review proof
(default gate id: `local-coderabbit`) and then the proof-carrying
`pr-readiness` gate, but is not
itself a state-mapped procedure.
Post-review learning: `.reinguard/procedure/internalize.md`.

**Cursor entries:** `.cursor/commands/rgd-next.md` — Sense (`rgd context build`), Route using procedure front matter as above, then Propose / Execute per `.reinguard/procedure/next-orchestration.md` (do not duplicate a mapping table in Adapter files). `.cursor/commands/cursor-plan.md` — planning only (`AskQuestion` / `CreatePlan`); not part of the FSM.

### 5. Extension contract (state / route / Adapter)

When adding or changing FSM wiring, keep these touchpoints consistent:

1. **State catalog (this ADR)** — Add or update the row in section 1 for every new or changed `state_id`. Residual states must stay documented (observation gaps, missing facets) and their **numeric `priority`** must be explicit relative to refinements (lower wins; ADR-0004).
2. **Control state rules** — Edit `.reinguard/control/states/*.yaml`. Ensure rules do not accidentally overlap: a more specific condition (e.g. a gate-backed state) must use a **lower** numeric `priority` than its residual fallback.
3. **Routes** — Edit `.reinguard/control/routes/*.yaml` when a `state_id` needs a different primary `route_id` or when new states share an existing route with different procedure hints (section 2).
4. **Procedure mapping (`applies_to`)** — When the primary procedure for a `state_id` changes, update the affected `.reinguard/procedure/*.md` front matter (`applies_to.state_ids` / `route_ids` as appropriate). Run `rgd config validate`. Do not add a second mapping table to this ADR; section 4 documents the mechanism only.
5. **Procedures (not state-mapped)** — Procedures that are not tied to a `state_id` (e.g. `change-inspect`, `next-orchestration`) keep `applies_to.state_ids: []` and remain documented in their bodies as producers, orchestration, or prerequisites.
6. **Tests** — Extend scenario tests (e.g. `internal/rgdcli/workflow_fsm_test.go`) when resolution or fallback behavior is non-obvious (residual vs refined state, stale gate fallback).

**Guards vs states:** `guard eval` outputs (e.g. built-in `merge-readiness`) are **not** `state_id` values. FSM states may *align* with guard signals (e.g. `merge_ready` with merge-readiness); wiring belongs in `control/states/*.yaml` and this ADR, not by conflating guard JSON with state without explicit rules.

**Adapter-local resume artifacts:** approval continuity for an already approved
Execute path (ADR-0015) is **not** a workflow state. Do not model it as a new
`state_id`, `route_id`, guard input, or `gates.<id>` signal. The FSM continues
to describe repository / platform workflow position only.

Operational checklist (files, validation commands, knowledge surfacing): see `.reinguard/knowledge/workflow--state-gate-guard-extension.md` (ADR-0010 knowledge atom).

## Consequences

- **Easier**: One YAML-defined FSM; Adapters map substrate output to procedures via machine-readable Semantics (`applies_to`) validated by `rgd`.
- **Harder**: Priority authoring must stay global across states/routes/guards
  (ADR-0004).
- **Harder**: Observation providers and tests must cover the timing gap where a
  clean marker appears before the matching GitHub review entry, because
  `completed` may need to upgrade to `completed_clean` on a later pass.
  Required timing scenarios are marker-first, review-first, same-pass,
  and marker-without-review-entry.
- **Harder**: Observation gaps still require residual states (for example
  `working_no_pr` when PR readiness has not been proven yet or the PR facet is
  missing) - agents must not over-interpret.

## Refs

- ADR-0002 (match / `when`)
- ADR-0004 (priority resolution)
- ADR-0014 (runtime gate artifacts and `gates.<id>.*`)
- ADR-0011 (control plane layout, `procedure/`)
- `docs/cli.md` (signal paths)
- Issue #72 (P2-3 deliverable)
