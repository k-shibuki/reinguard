---
id: review-bot-operations
description: Bot reviewer trigger, detection, timing, rate-limit recovery, and re-review procedures
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
---

# Bot Review Operations

Operational reference for AI code reviewers (CodeRabbit and Codex).
Covers trigger, detection, timing, rate limits, and re-review.

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
| Timeout | 20 min | 20 min |

## Rate-Limit Recovery

1. Detect: PR comment from bot containing "Rate limit exceeded"
2. Parse wait time from the message (minutes + seconds + 30s buffer)
3. Sleep, re-trigger same reviewer
4. Second rate limit → treat as timed out (max 1 recovery)

## Agreement Mechanics

How each bot expresses agreement after a disposition reply:

| Behavior | CodeRabbit | Codex |
|---|---|---|
| Reads disposition replies | **Yes** — checks referenced commit | **No** |
| Confirms fix | Replies with confirmation + **auto-resolves thread** | N/A |
| Objects | Replies with objection, thread stays unresolved | N/A |
| Agreement via re-review | Also works (no new finding = agreement) | **Only method** (no new finding = implicit agreement) |

**Implication**: After posting a disposition reply, check thread state
first. If CodeRabbit already confirmed and auto-resolved, no re-review
is needed. For Codex, re-review is the only path to consensus.

## Re-review

| Condition | CodeRabbit | Codex |
|---|---|---|
| Push to PR branch | Usually **automatic** incremental review; if paused/skipped, `@coderabbitai review` | — |
| User instructs Codex re-review | — | `@codex review` |

All findings receive equal evaluation regardless of source.
Deduplicate when both reviewers flag the same issue.

## Related

- `.reinguard/policy/review--consensus-protocol.md` — disposition, resolve, consensus
- `.reinguard/policy/safety--agent-invariants.md` § **HS-REVIEW-RESOLVE** (Cursor: `reinguard-bridge.mdc` § Always-active policy)
- `.cursor/commands/review-fix.md`
