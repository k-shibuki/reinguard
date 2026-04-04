---
id: review-bot-operations
description: PR-side bot review — trigger, detection, timing, rate-limit recovery, re-review (not pre-PR local CLI)
triggers:
  - bot review trigger
  - bot review detection
  - CodeRabbit trigger
  - Codex trigger
  - coderabbit polling
  - re-review decision
  - "@coderabbitai review"
  - "@codex review"
  - rate limit recovery
when:
  op: eq
  path: github.pull_requests.pr_exists_for_branch
  value: true
---

# Bot Review Operations (PR-side)

Operational reference for **PR-side** bot review (after a PR exists):
CodeRabbit and Codex on GitHub — trigger, detection, timing, rate limits,
re-review, and polling cadence for `wait-bot-review` / `review-address`.

**Pre-PR** local CodeRabbit CLI (`check-local-review.sh`) is **not** covered
here; see `.reinguard/knowledge/review--local-coderabbit-cli.md`.

For the consensus model (disposition, resolve, agreement), see
`.reinguard/policy/review--consensus-protocol.md`.

## Reviewers

| Role | Reviewer | When triggered | Strength |
|---|---|---|---|
| **Primary** | CodeRabbit (Pro/OSS) | Every PR (automatic + optional manual) | Walkthrough, tool integrations, incremental review |
| **Supplementary** | Codex Cloud | User instruction only | Cross-file logic, deep semantic understanding |

Both read `AGENTS.md` and apply its review guidelines.

## Trigger

- **CodeRabbit**: **Automatic** review on eligible PRs per `.coderabbit.yaml`
  (`reviews.auto_review`; this repo uses `enabled: true` on the default
  branch, with title/bot exclusions). **Incremental** re-review on new
  pushes is on by default (`auto_incremental_review`); after enough
  commits CodeRabbit may **pause** until you comment
  `@coderabbitai review` (see CodeRabbit docs).
- **Manual**: `gh pr comment <N> --body "@coderabbitai review"` — use to
  force a pass when auto-review was skipped (org/UI override, rate limit,
  voided review after head moved), after `@coderabbitai resume`, or when
  you want an explicit re-run alongside auto behavior.
- **Codex**: User instruction only — the agent never triggers Codex
  autonomously.

## Detection

Bot identities:

| Bot | GitHub login |
|---|---|
| CodeRabbit | `coderabbitai[bot]` |
| Codex | `chatgpt-codex-connector[bot]` |

### API Channels

| Channel | Endpoint | What it returns |
|---|---|---|
| Review | `pulls/<N>/reviews` | Review with state and body |
| Inline | `pulls/<N>/comments` | Line-level findings |
| PR comment | `issues/<N>/comments` | Walkthrough / clean bill / rate limit |

### Voided review (head moved during bot run)

If the branch is updated while CodeRabbit is reviewing, CR may post
"Review failed" / "The head commit changed during the review." That is
**not** a completed review for the new head. Re-trigger
`@coderabbitai review` after the push stabilizes.

## Timing

| | CodeRabbit | Codex |
|---|---|---|
| Typical completion | 2–7 min | 1–7 min |
| Polling interval | 30 s | 30 s |
| Polling window | 20 min | 20 min |

### Polling model

- Use this polling model for **PR-side bot review waits** only
  (`.reinguard/procedure/wait-bot-review.md` after PR creation).
- Poll every **30 seconds** for up to **20 minutes**.
- Exit early as soon as the required bot becomes terminal, review threads
  that need classification or reply appear, or the state changes to a
  different FSM route such as `review-address`.
- When the Adapter (the execution environment that runs these procedures,
  such as Cursor) supports delegation, prefer a delegated wait owner over
  an inline main-agent sleep loop. Inline polling is a fallback for
  environments that do not support delegation.
- Use the **PR-side** polling model only after a PR exists and the FSM routes
  to `wait-bot-review`. The **pre-PR** local CodeRabbit CLI gate is separate;
  see `.reinguard/knowledge/review--local-coderabbit-cli.md`.

## Rate-Limit Recovery

CodeRabbit often **edits** a single PR issue “Review Status” comment in place; it may also post separate short replies. `rgd` derives rate-limit cues from the **selected status comment** (`status_comment_at` / `status_comment_source` in `bot_reviewer_status`, see `docs/cli.md`), not from “newest comment only,” so a later acknowledgment does not hide an active rate-limit body in the Review Status comment.

1. Detect: selected status comment body contains "Rate limit exceeded" (or `contains_rate_limit` in observation).
2. **Cool-down duration:** when `signals.github.reviews.bot_reviewer_status[].rate_limit_remaining_seconds` is present (CodeRabbit enrichment), use it as the primary sleep budget (add any buffer your policy requires) instead of re-parsing the comment body. Otherwise parse wait time from the **selected status comment** body (minutes + seconds + 30s buffer).
3. Sleep, re-trigger same reviewer
4. Second rate limit → treat as timed out (max 1 recovery)

## Consensus and disposition (policy SSOT)

Disposition categories, CodeRabbit resolution gate, and when threads may be resolved are **normative** in `.reinguard/policy/review--consensus-protocol.md` — do not duplicate that model here.

**Operational shorthand** (after you post a disposition): check thread resolution state; CodeRabbit may confirm and auto-resolve; Codex follow-up usually needs a PR timeline comment with `@codex review` (see `.reinguard/procedure/review-address.md`).

## Re-review

| Condition | CodeRabbit | Codex |
|---|---|---|
| Push to PR branch | Usually **automatic** incremental review; if paused/skipped, `@coderabbitai review` | — |
| User instructs Codex re-review | — | `@codex review` |

All findings receive equal evaluation regardless of source.
Deduplicate when both reviewers flag the same issue.

## Related

- `.reinguard/knowledge/review--local-coderabbit-cli.md` — pre-PR local CLI gate only
- `.reinguard/policy/review--consensus-protocol.md` — disposition, resolve, consensus
- `.reinguard/policy/safety--agent-invariants.md` § **HS-REVIEW-RESOLVE**
- `.reinguard/procedure/wait-bot-review.md` — FSM routes `user-wait-bot-*` (quota, pause, failed, stale, run)
- `.reinguard/procedure/review-address.md` — thread disposition and multi-source triage
- `.reinguard/knowledge/review--multi-source-review-signals.md` — inbox model across sources
