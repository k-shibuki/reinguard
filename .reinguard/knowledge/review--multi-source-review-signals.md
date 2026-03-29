---
id: review-multi-source-review-signals
description: Normalize review signals from bots, humans, checks, and timeline into one actionable inbox
triggers:
  - multi source review
  - review inbox
  - dedupe review
  - bot and human review
  - checks vs review
  - review signal priority
when:
  op: eq
  path: github.pull_requests.pr_exists_for_branch
  value: true
---

# Multi-source review signals (single inbox)

Use this card with `.reinguard/procedure/review-address.md` when triaging PR feedback from **more than one source** (e.g. CodeRabbit, Codex, human reviewers, CI check logs, PR conversation summaries).

Normative disposition and resolve rules stay in `.reinguard/policy/review--consensus-protocol.md` — do not restate them here.

## Signal classes

| Class | Meaning | Typical sources |
|-------|---------|-----------------|
| **blocking** | Must clear before merge consideration (policy + branch protection) | Failing required checks; formal `CHANGES_REQUESTED`; unresolved review threads that need disposition |
| **actionable** | Should classify and reply (or fix) in this PR | Inline review comments; bot threads; human review threads |
| **informational** | Context only unless it contains a concrete finding | Clean-bill summaries; duplicate notices; meta comments |

## Source kinds

| Kind | Where it appears | Trust for thread state |
|------|------------------|-------------------------|
| **bot_thread** | Inline file/line threads from configured bot logins | Use GraphQL `reviewThreads` / disposition flow per `review--github-thread-api.md` |
| **human_thread** | Inline threads from non-bot authors | Same as bot_thread |
| **checks_text** | `gh pr checks`, job logs | Map log lines to code; if no anchor, use PR conversation comment (quote + disposition) per `review-address` |
| **timeline_summary** | PR conversation, bot walkthrough comments | Same as checks_text when no `in_reply_to` target |

## Dedupe key (same issue, one disposition)

When two sources flag the **same** issue:

1. Pick a **primary thread** (prefer the earliest inline thread with a stable anchor).
2. Apply **one** disposition category for that issue; on secondary threads reply briefly: **Duplicate of** `<location or thread>` + same disposition label.
3. Do not leave contradictory dispositions across threads for the same root cause.

## Priority when multiple classes fire

1. **Formal** `CHANGES_REQUESTED` and **unresolved** review threads (actionable inline work).
2. **Failing required checks** (fix or document exception per policy).
3. **Bot run / quota / pause / failure** (see `.reinguard/procedure/wait-bot-review.md` and `review--bot-operations.md`).
4. **CI / mergeability pending** with no open review work (`waiting_ci` in control FSM — poll checks, fix red jobs).

## “Done” for merge handoff (operational)

Align with substrate guard `merge-readiness` and FSM `merge_ready`:

- Working tree clean; **`ci-pass`** (and other required checks) green.
- `review_threads_unresolved` aggregate is **0** where observation is available; formal `CHANGES_REQUESTED` count **0**.
- Dispositions posted per policy (**HS-REVIEW-RESOLVE**).

## FSM alignment

Control plane states live under `.reinguard/control/states/workflow.yaml`. Human-actionable review states (`unresolved_threads`, `changes_requested`) are **authored to win** over bot-wait states when both signals are present, so agents address feedback instead of stalling on bot polling.

## Related

- `.reinguard/procedure/review-address.md` — execution steps
- `.reinguard/procedure/wait-bot-review.md` — bot wait / retry / re-trigger
- `.reinguard/knowledge/review--bot-operations.md` — CodeRabbit / Codex mechanics
- `.reinguard/knowledge/review--github-thread-api.md` — REST vs GraphQL for `isResolved`
- `docs/adr/0013-fsm-v1-workflow-states.md` — state catalog and Adapter mapping
