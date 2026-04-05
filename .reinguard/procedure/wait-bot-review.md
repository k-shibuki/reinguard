---
id: procedure-wait-bot-review
purpose: Act on required bot reviewer quota, pause, failure, or in-flight states without mixing in thread disposition work.
applies_to:
  state_ids:
    - waiting_bot_rate_limited
    - waiting_bot_paused
    - waiting_bot_failed
    - waiting_bot_stale
    - waiting_bot_run
  route_ids:
    - user-wait-bot-quota
    - user-wait-bot-paused
    - user-wait-bot-failed
    - user-wait-bot-stale
    - user-wait-bot-run
reads:
  - ../knowledge/review--bot-operations.md
  - ../knowledge/review--multi-source-review-signals.md
sense:
  - rgd context build
  - rgd observe github reviews
act:
  - Classify bot tier; poll every 30s for up to 20m; backoff or re-trigger; escalate on repeated failure; re-observe.
output:
  - Bot state summary; triggers used; next poll time or handoff to review-address.
done_when: Required bots are terminal success or failure is dispositioned per bot docs; observation matches next FSM state.
escalate_when: Org policy blocks bot rerun; required bot persistently failed; rate limits repeat beyond recovery policy.
---

# wait-bot-review

## Context

Open `.reinguard/knowledge/review--bot-operations.md` for **PR-side** **CodeRabbit** and **Codex** specifics (logins, triggers, rate-limit recovery, `@coderabbitai review`, `@codex review`). Do **not** use `.reinguard/knowledge/review--local-coderabbit-cli.md` as the primary reference here — that atom is **pre-PR** only.

This procedure governs **PR-side bot waiting** after PR creation. The
repository-local CodeRabbit CLI gate (`check-local-review.sh`) runs **one**
`coderabbit review` subprocess per attempt with a supervisor (stderr heartbeat
every 30s, max 20 minutes per attempt by default); it is a single blocking gate
in `change-inspect` / `pr-create` only and is not this polling loop. See
`.reinguard/knowledge/review--local-coderabbit-cli.md`.

If **open review threads** or formal **changes requested** also apply, run `.reinguard/procedure/review-address.md` **in parallel or first** — the workflow FSM prefers human-actionable review states over bot-wait states when both are true.

**Discover aids:**

```bash
rgd context build
```

Use `knowledge.entries` (typically includes `review--bot-operations.md`, `review--github-thread-api.md`, `review--multi-source-review-signals.md`).

## Map state → first action

| `state_id` | Intent | First actions |
|------------|--------|----------------|
| `waiting_bot_run` | Required bot outcome not terminal | Poll `rgd observe github reviews` every 30s for up to 20m; avoid duplicate triggers unless policy allows. |
| `waiting_bot_rate_limited` | Bot hit quota | Prefer `rate_limit_remaining_seconds` from `bot_reviewer_status` when present; else parse wait from the **selected status comment** body (tied to `status_comment_at` / `status_comment_source`, not `latest_comment_at` alone); sleep + **one** retry path per `review--bot-operations.md`. |
| `waiting_bot_paused` | Bot paused (e.g. commit threshold) | Follow vendor resume / `@coderabbitai review` when appropriate. |
| `waiting_bot_failed` | Bot failed tier (incl. voided review) | Stabilize head; re-trigger per bot docs; if repeated failure, escalate. |
| `waiting_bot_stale` | Review is stale (head moved) | Re-trigger `@coderabbitai review`; if already triggered, poll for completion. |

## Act

1. Run `rgd observe github reviews` (or full `rgd observe`) and confirm `github.reviews.bot_reviewer_status` / `bot_review_diagnostics` match the FSM state. For substring flags and issue-comment enrichment, use `status_comment_at` / `status_comment_source` (not `latest_comment_at` alone) per `docs/cli.md`.
2. Apply the **row** for your `state_id` above; use **only** PR conversation / documented triggers — do not rely on thread replies for Codex rerun.
3. For `waiting_bot_stale`, the required bot completed its review on a **previous** HEAD. Re-trigger review per bot docs (same re-trigger path as `waiting_bot_failed` when head moved) and poll until terminal or a different FSM state applies.
4. For `waiting_bot_run`, poll every **30 seconds** for up to **20 minutes**. Stop immediately if the required bot becomes terminal, actionable review work appears, or the FSM should hand off to another procedure.
5. For `waiting_bot_rate_limited`, take **`cooldown_sec`** from `rate_limit_remaining_seconds` when present (it is **elapsed-adjusted** from **`status_comment_at`** per `docs/cli.md`); else parse wait duration from the **selected status comment** body and subtract elapsed time since **`status_comment_at`** (same source as Act step 1). **Sleep `cooldown_sec + 30` seconds** before posting `@coderabbitai review` — **30** matches the local gate default **`RATE_LIMIT_RETRY_BUFFER_SEC`** in `.reinguard/scripts/check-local-review.sh` (same formula as local `--retry-on-rate-limit`). Then follow the one-retry recovery path in `review--bot-operations.md` instead of the generic 30-second poll cadence during the cool-down window.
6. For the polling waits above, when the Adapter (the execution environment, such as Cursor) supports delegation, prefer a delegated wait owner instead of keeping the main agent in an inline sleep cycle.
7. For a single active unit, prefer foreground-first delegated wait ownership so the delegated worker blocks until review state changes.
8. Use the Adapter's configured bot-review wait template or wrapper when available; otherwise use the inline polling behavior described in steps 4-5.
9. When bots are terminal and review threads still exist, switch to `review-address.md`.

## Output

- Which bot(s) were waiting and why.
- Triggers posted (verbatim command or comment body reference).
- Next observation timestamp, elapsed polling window, or handoff to `review-address` / `pr-merge`.

## Guard

HS-MERGE-CONSENSUS, HS-REVIEW-RESOLVE (do not resolve threads without disposition when addressing findings), HS-NO-SKIP
