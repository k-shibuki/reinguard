---
id: review-consensus-protocol
description: Disposition categories, consensus flow, and thread resolution rules for review threads
triggers:
  - consensus protocol
  - disposition reply
  - review thread resolve
  - Fixed By design False positive Acknowledged
  - HS-REVIEW-RESOLVE
  - resolve thread
  - unresolved thread
---

# Review Consensus Protocol

Bidirectional agreement model for review threads. All commands that
interact with review threads (`review-address`, `pr-merge`) reference this
document as the SSOT for disposition and resolution.

## Principle

> Reach agreement with bot reviewers on ALL findings before proceeding
> to merge. Findings without agreement remain unresolved and block merge.

**Unilateral resolve is prohibited.** The agent must not resolve a thread
until the reviewer's final response confirms the disposition.

## Completeness Invariant

```text
unresolved threads == 0  ⟺  all review findings have consensus
```

GitHub Branch Protection (`required_conversation_resolution`) blocks
merge until every thread is resolved. This is a deterministic guard.

## Disposition Categories (4, exhaustive)

Every thread receives a disposition reply before being resolved (per
`HS-REVIEW-RESOLVE`). If the bot objects after the initial reply, the
agent posts a new disposition reply per round until consensus is reached.

| Category | When | Consensus requirement | Template |
|---|---|---|---|
| **Fixed** | Code change addresses the finding | Re-review confirms fix (no new finding on same lines) | `Fixed in \`<sha7>\`. <what changed>.` |
| **By design** | Intentional design decision | Bot reply does not object (acceptance or no further comment after re-review) | `By design. <rationale> (ref: <source>).` |
| **False positive** | Bot misidentified an issue | Bot reply does not object | `False positive. <why detection was wrong>.` |
| **Acknowledged** | Valid but deferred (see below) | Follow-up Issue **only** when work is a substantial separate deliverable; otherwise rationale without `Tracked in` | `Acknowledged. <brief assessment>. Tracked in #<issue>.` *or* rationale if no Issue |

### Acknowledged — in-PR resolution vs follow-up Issue

**Principle**: Prefer **Fixed**, **By design**, or **False positive** in the **same PR** when the finding is correct and the change fits the PR’s scope or is proportionate. Do **not** use **Acknowledged** + a new Issue to avoid small, in-scope edits.

When you map a finding to **Acknowledged**, explicitly **decide** how deferred work is tracked:

- **In-PR default**: If the reviewer is right and the fix is reasonable for this PR, disposition **Fixed** (or **By design** / **False positive** as appropriate) instead of **Acknowledged**.
- **New GitHub Issue** (`Tracked in #<issue>`): Use **only** when the remaining work would reasonably be a **large, separately scoped** deliverable (its own Definition of Done, multiple subsystems, or a different release intent than the current PR). Use `Refs: #…` to parent Issues/epics as appropriate; the Issue number MUST differ from the PR’s `Closes #…` target (**Acknowledged invariant** below).
- **No new Issue**: Allowed when you document **why** in the disposition reply (for example: superseded by `#M`, duplicate of existing work, explicit decision not to pursue, or the deferral is documented without needing a standalone Issue). Do **not** use `Tracked in #N` when `N` is the same as the PR’s `Closes` issue.

#### Acknowledged invariant

Any `Tracked in #<issue>` MUST refer to an Issue **other than** the one the PR closes. If the PR `Closes #N`, then `Tracked in #N` is prohibited — the finding would be lost on merge.

## Consensus Flow

```text
1. Post disposition reply on thread
2. Observe:
   a. Thread state — isResolved? (bot may auto-resolve)
   b. Thread replies — did bot confirm or object?
   c. Re-review results — new findings on same area?
3. Decide:
   ├── Bot auto-resolved thread        → consensus confirmed
   ├── Bot replied with confirmation   → resolve thread
   ├── Bot replied with objection      → address, go to 1
   ├── No response + reviewer available → trigger re-review, go to 2
   └── No response + reviewer unavailable → agent resolves (see below)
```

## CodeRabbit Resolution Gate

**Terminology**: A **turn** is one review-address cycle (observe, post a
disposition or push fixes, then observe again). A **step** is a single
atomic API or automation action. *Turn* is about workflow pacing; *step*
is about not batching incompatible actions together.

For threads whose root review comment is from **CodeRabbit**
(`coderabbitai[bot]`):

- **If consensus evidence is not confirmed, `resolveReviewThread` is
  prohibited.**
- **Same-turn resolve is allowed** only if consensus evidence already
  exists **before** the resolve action (for example, a prior bot reply in
  that turn). Do **not** resolve in the same API **step** as posting the
  disposition reply.
- **Resolve only after** at least one of:
  - CodeRabbit **auto-resolved** the thread; or
  - CodeRabbit **replied** on the thread without objecting to the
    disposition; or
  - A **qualifying pull review** after `@coderabbitai review` shows no
    new finding on the same lines.
- **Exception**: Reviewer Unavailable (see below).

Human review threads: follow Consensus Flow above (no CodeRabbit gate;
still no drive-by resolve without acceptance).

## Reviewer Unavailable

When a reviewer cannot respond (usage limit, service outage, timeout):

- Agent resolves with justification: `<Category>. <explanation>.
  Reviewer unavailable (<reason>); fix verified by <evidence>.`
- Evidence examples: other bot confirmed, code change is mechanically
  correct, independent reviewer verified.

## API Reference

### Post reply to a thread

```bash
gh api repos/{owner}/{repo}/pulls/{N}/comments -X POST \
  -f body='<disposition text>' \
  -F in_reply_to=<database_id> \
  -f commit_id=<full_sha> \
  -f path=<file_path> \
  -F line=<line_number>
```

- `database_id`: root comment's `id` from
  `gh api repos/{owner}/{repo}/pulls/{N}/comments`
- `commit_id`: full 40-char SHA of HEAD (short SHA causes 422)
- `path`, `line`: from the root comment

### Resolve a thread (after consensus)

```bash
gh api graphql -f query='
  mutation { resolveReviewThread(input: {threadId: "<node_id>"}) {
    thread { isResolved }
  }
}'
```

- `node_id`: thread's GraphQL node ID
- **CodeRabbit threads**: Call this **only after** CodeRabbit has
  reacted. Do **not** resolve in the same API **step** as posting the
  disposition reply (post the reply, complete a new observation **turn**,
  then resolve if consensus is clear).
- **Human threads**: Do not resolve without acceptance or user
  instruction unless Reviewer Unavailable applies.

### Enumerate unresolved threads

REST `pulls/{N}/comments` and `pulls/{N}/reviews` return individual comments and reviews; they do **not** expose per-thread `isResolved`. Use GraphQL `pullRequest.reviewThreads` instead (paginate with `pageInfo.endCursor` while `hasNextPage` is true):

```bash
gh api graphql -f query='
query($owner:String!, $name:String!, $number:Int!) {
  repository(owner:$owner, name:$name) {
    pullRequest(number:$number) {
      reviewThreads(first:100) {
        nodes { id isResolved isOutdated }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}' -f owner=OWNER -f name=REPO -F number=N
```

Count `nodes` where `isResolved` is false for unresolved threads.

## Edge Cases

- **CodeRabbit auto-resolve**: CR may confirm a fix and resolve the
  thread itself. This IS consensus — no further action needed.
- **`isOutdated` threads**: Still valid;
  `required_conversation_resolution` does not distinguish outdated from
  current.
- **Multiple comments in one thread**: Reply once to the root comment.
- **Human-replied threads**: Agent does not auto-resolve. Leave for the
  human reviewer to resolve, or resolve only after explicit acceptance.

## Related

- `.reinguard/knowledge/review--bot-operations.md` — trigger, detection,
  timing
- `.reinguard/policy/safety--agent-invariants.md` § **HS-REVIEW-RESOLVE** (Cursor: `reinguard-bridge.mdc` § Always-active policy)
- `.cursor/commands/review-address.md` — review-address procedure
