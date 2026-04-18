---
id: procedure-pr-merge
purpose: Merge a PR after guard and GitHub checks agree with policy.
applies_to:
  state_ids:
    - merge_ready
  route_ids:
    - user-merge
reads:
  - ../policy/review--consensus-protocol.md
sense:
  - rgd observe
  - rgd guard eval merge-readiness
  - gh pr checks
act:
  - Confirm readiness; merge via gh without --admin.
output:
  - PR state JSON confirming merge.
done_when: PR merged and branch protection satisfied.
escalate_when: Checks red, threads unresolved, or policy exception required.
---

# pr-merge

## Context

- [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md) (shared review-closure contract, thread resolution, CodeRabbit gate)

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow & commit policy.

**Merge readiness (substrate):** structured check before merge. From repo root with `rgd` on PATH:

```bash
rgd observe --view summary > /tmp/obs.json
rgd guard eval --observation-file /tmp/obs.json merge-readiness
```

Interpret JSON: `"ok": true` only when CI success, zero unresolved review threads, required bot review terminal (not pending), no `bot_review_trigger_awaiting_ack` after a posted re-review trigger, no formal `CHANGES_REQUESTED`, and clean working tree — see [`docs/cli.md`](../../docs/cli.md) § `merge-readiness`.
This guard does **not** prove that non-thread findings from the current PR
review cycle have been dispositioned; that remains part of review closure
per [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md).

Optional composed pipeline: `rgd context build --compact` (observe → state → route → guard → knowledge entries).
Use `rgd context build` (without `--compact`) only when full nested observation payload is required.

**Cross-check:** `gh pr checks <N>` (especially **`ci-pass`**) and `gh pr view <N>` for mergeable state and branch protection (guard may not mirror every GitHub rule).

**Bot review status:** Verify required bots are terminal before merge:

```bash
rgd context build --compact | jq '.observation.signals.github.reviews.bot_review_diagnostics'
```

`bot_review_pending` must be **false**, `bot_review_terminal` must be **true**, and `bot_review_failed` must be **false**. If pending is true, the FSM state should be `waiting_bot_*` — follow `wait-bot-review.md` instead. If terminal is false or failed is true, do **not** merge.

## Act

1. Confirm **all** of: guard `merge-readiness` is `"ok": true`; `gh pr checks` shows CI green; required bot review is **terminal** (`bot_review_pending == false`, `bot_review_terminal == true`, `bot_review_failed == false` — HS-MERGE-CONSENSUS); threads resolved; and review closure is complete for the current PR review cycle, including non-thread findings, per `review--consensus-protocol.md`.
2. Merge: `gh pr merge <N> --squash` or `--merge` per [`.github/CONTRIBUTING.md`](../../.github/CONTRIBUTING.md) and maintainer convention for this repo.
   Do **not** use `--admin`. Do **not** merge with failing checks.

## Output

- Confirm merged: `gh pr view <N> --json state -q .state`

## Guard

HS-CI-MERGE, HS-REVIEW-RESOLVE, HS-MERGE-CONSENSUS
