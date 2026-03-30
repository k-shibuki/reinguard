---
id: review-incremental-fix-flow
description: Review-driven local fixes batched into one push for the next bot incremental review
triggers:
  - incremental fix
  - review fix push
  - dirty tree review
  - uncommitted after review
  - CodeRabbit incremental
  - batch review fixes
when:
  op: eq
  path: github.pull_requests.pr_exists_for_branch
  value: true
---

# Incremental fix flow (review-sourced changes)

Use this pattern when you are **addressing review feedback** (or follow-on improvements discovered while doing so) and want **one push** to land on the PR branch so **CodeRabbit incremental review** (and other bots) can pick up the new head.

Normative disposition and resolve rules remain in `.reinguard/policy/review--consensus-protocol.md`. Execution steps for PR review work live in `.reinguard/procedure/review-address.md` — especially **Step 0** (local work gate) when the tree is dirty.

## Intent

- Incorporate review-driven fixes (and small related improvements) locally.
- **Inspect quality** before committing (`change-inspect` — same dimensions as pre-PR, scoped to the incremental diff).
- **Commit** with `Refs: #<issue>`, then run **review-address** (classify, disposition, thread replies).
- **Push once** so the next bot pass sees a stable head (see `review--bot-operations.md` for voided-review / head-moved caveats).

## Why not a special FSM state for “dirty tree”?

`git.working_tree_clean` is false in many phases (implement, pre-PR, mid-review). Encoding “dirty” as a separate `state_id` would collide with unrelated work. The FSM stays **`unresolved_threads`** / **`changes_requested`** when review work exists; **ordering** (inspect → commit before heavy disposition) is handled in **procedure Step 0**, not by renaming state.

## Signal pattern

| Observation hint | Meaning |
|------------------|---------|
| `github.pull_requests.pr_exists_for_branch` true | On a PR branch |
| `github.reviews.review_threads_unresolved` > 0 | Thread disposition work remains (typical with this flow) |
| `git.working_tree_clean` false | Uncommitted or unstaged work — run **review-address Step 0** before relying on disposition-only steps |

## Procedure outline

1. Apply fixes locally (from review threads, checks, or same-kind sweep per policy).
2. **`change-inspect`** on committed delta + staged + unstaged (see `review-address` Step 0).
3. **Commit** with `Refs: #<issue>` (no amend+force-push on PR head per `review-address`).
4. Continue **`review-address`** steps 1–7 (classify, reply, verify, push).
5. After push, if FSM shows **`waiting_bot_run`**, follow **`wait-bot-review.md`**.

## Related

- `.reinguard/procedure/review-address.md` — Step 0 gate and full Act sequence
- `.reinguard/procedure/change-inspect.md` — inspection dimensions (reuse post-review)
- `.reinguard/knowledge/review--bot-operations.md` — triggers and incremental review timing
- `.reinguard/knowledge/review--multi-source-review-signals.md` — inbox ordering across sources
