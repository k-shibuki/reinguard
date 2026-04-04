---
id: review-local-coderabbit-cli
description: Pre-PR local CodeRabbit CLI gate (check-local-review.sh) — foreground wait, rate limits; distinct from PR-side bot review
triggers:
  - check-local-review
  - local CodeRabbit
  - pre-PR review
  - local CLI gate
  - change-inspect CodeRabbit
when:
  and:
    - op: eq
      path: git.detached_head
      value: false
    - or:
        - op: eq
          path: github.pull_requests.pr_exists_for_branch
          value: false
        - op: not_exists
          path: github.pull_requests.pr_exists_for_branch
---

# Local CodeRabbit CLI (pre-PR)

This document covers the **repository-local CodeRabbit CLI** gate used **before**
a PR exists (`change-inspect` → `pr-create`). It is **not** the same workflow as
**PR-side** CodeRabbit review (automatic review on PRs, polling, thread
resolution) — that is `.reinguard/knowledge/review--bot-operations.md`.

## Scope

| Topic | Local CLI (this doc) | PR-side bot review |
|------|------------------------|---------------------|
| When | No open PR for branch (pre-PR) | `pr_exists_for_branch` true |
| Mechanism | One `coderabbit review` run per attempt, supervised (stderr heartbeat every 30s, max 20m per attempt; env `LOCAL_CR_*`) | GitHub + observation; poll every 30s up to 20m per `wait-bot-review` |
| Procedure | `change-inspect`, then `pr-create` | `wait-bot-review`, `review-address` |
| Knowledge | This file | `review--bot-operations.md` |

Normative script contract and rate-limit behavior for the gate are in
`.reinguard/policy/coding--preflight.md` § Required local AI review.

## Command

From repo root:

```bash
bash .reinguard/scripts/check-local-review.sh --base main --retry-on-rate-limit
```

## Supervised wait (aligned cadence with PR-side)

- The script runs **one** `coderabbit review` subprocess per attempt (no
  restart every 30s). A parent supervisor prints a **stderr heartbeat** every
  **30 seconds** while that subprocess runs and enforces a **maximum wall-clock
  time of 20 minutes** per attempt (default `LOCAL_CR_MAX_WAIT_SEC=1200`,
  `LOCAL_CR_HEARTBEAT_SEC=30`). This matches the **30s / 20m** polling budget
  described for PR-side bot waits in `review--bot-operations.md` (different
  mechanism: subprocess supervision vs `rgd observe` / GitHub polling).
- Built-in **rate-limit retry** remains: one automatic retry when
  `--retry-on-rate-limit` is set and the CLI reports a parseable cooldown.
- Sparse **stdout** from the CLI while it works is **normal**; heartbeats go to
  **stderr** so transcripts stay readable.
- Only terminal outcomes change control flow: success, explicit CLI failure,
  supervisor timeout, auth/tooling failure, cooldown parse failure, or second
  consecutive rate limit (per policy).
- Do not kill the subprocess without positive evidence of hang beyond the
  supervisor limit; the supervisor already bounds worst-case wait.

## Related

- `.reinguard/policy/coding--preflight.md` — required gate, rate-limit rules
- `.reinguard/procedure/change-inspect.md` — where local gate runs in the flow
- `.reinguard/knowledge/review--bot-operations.md` — PR-side bot review only
