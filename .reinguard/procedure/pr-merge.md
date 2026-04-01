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

- [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md) (disposition, thread resolution, CodeRabbit gate)

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow & commit policy.

**Merge readiness (substrate):** structured check before merge. From repo root with `rgd` on PATH:

```bash
rgd observe > /tmp/obs.json
rgd guard eval --observation-file /tmp/obs.json merge-readiness
```

Interpret JSON: `"ok": true` only when CI success, zero unresolved review threads, required bot review terminal (not pending), no formal `CHANGES_REQUESTED`, and clean working tree — see [`docs/cli.md`](../../docs/cli.md) § `merge-readiness`.

Optional full pipeline: `rgd context build` (observe → state → route → guard → knowledge entries).

**Cross-check:** `gh pr checks <N>` (especially **`ci-pass`**) and `gh pr view <N>` for mergeable state and branch protection (guard may not mirror every GitHub rule).

**Bot review status:** Verify required bots are terminal before merge:

```bash
rgd context build | jq '.observation.signals.github.reviews.bot_review_diagnostics'
```

`bot_review_pending` must be **false**. If true, the FSM state should be `waiting_bot_*` — follow `wait-bot-review.md` instead.

## Act

1. Confirm **all** of: guard `merge-readiness` is `"ok": true`; `gh pr checks` shows CI green; required bot review is **terminal** (not pending — HS-MERGE-CONSENSUS); threads resolved; consensus reached per `review--consensus-protocol.md`.
2. Merge: `gh pr merge <N> --squash` or `--merge` per [`.github/CONTRIBUTING.md`](../../.github/CONTRIBUTING.md) and maintainer convention for this repo.
   Do **not** use `--admin`. Do **not** merge with failing checks.

## Output

- Confirm merged: `gh pr view <N> --json state -q .state`

## Guard

HS-CI-MERGE, HS-REVIEW-RESOLVE, HS-MERGE-CONSENSUS
