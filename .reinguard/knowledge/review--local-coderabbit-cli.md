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
| Mechanism | One blocking shell command | GitHub + observation + optional polling |
| Procedure | `change-inspect`, then `pr-create` | `wait-bot-review`, `review-address` |
| Knowledge | This file | `review--bot-operations.md` |

Normative script contract and rate-limit behavior for the gate are in
`.reinguard/policy/coding--preflight.md` § Required local AI review.

## Command

From repo root:

```bash
bash .reinguard/scripts/check-local-review.sh --base main --retry-on-rate-limit
```

## Foreground wait (not polling)

- This is **one blocking process** with built-in rate-limit retry — not a
  `sleep` + `gh` polling loop.
- Treat it as a **foreground wait**: sparse output while the CLI reviews or
  sleeps for a parsed cooldown is **normal**.
- A long run is **not** failure by itself. Only terminal outcomes change control
  flow: success, explicit CLI failure, auth/tooling failure, cooldown parse
  failure, or second consecutive rate limit (per policy).
- Prefer transcript/status checks over killing the process; restart only with
  positive evidence of exit, crash, or script contract violation.

## Related

- `.reinguard/policy/coding--preflight.md` — required gate, rate-limit rules
- `.reinguard/procedure/change-inspect.md` — where local gate runs in the flow
- `.reinguard/knowledge/review--bot-operations.md` — PR-side bot review only
