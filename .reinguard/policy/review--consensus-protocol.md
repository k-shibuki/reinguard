---
id: review-consensus-protocol
description: Shared review-closure contract, PR consensus flow, and thread resolution rules
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

Shared review-closure contract plus the bidirectional agreement model for
PR review threads. Commands that classify or resolve review findings
(`change-inspect`, `review-address`, `pr-merge`) reference this document
for closure semantics; only the PR-side sections below govern thread
resolution.

The shared disposition vocabulary across local review, self-inspection, and
PR review is defined in
`.reinguard/policy/review--disposition-categories.md`. This policy defines
how those same categories become **closed** findings locally and on a PR.

## Shared review-closure model

- **Finding** — one review-reported concern that requires classification.
  Findings include self-inspection findings, commit-structure findings,
  repository-local CodeRabbit findings, PR review threads, outside-diff
  findings, PR summary findings, and conversation-level review findings.
- **Classified** — a finding has been mapped to exactly one of
  **Fixed / By design / False positive / Acknowledged**.
- **Closed** — a classified finding also has the evidence required by its
  review channel:
  - **Local review**: the disposition is recorded in inspection output; any
    finding dispositioned **Fixed** is already addressed on the current
    branch head; and the applicable local verification / local CodeRabbit
    reruns for that stabilized head have completed.
  - **PR review**: the disposition is posted on the correct PR channel
    (thread reply or PR conversation comment), and any thread finding also
    satisfies the consensus and resolve rules in this policy.
- **Review cycle** — one bounded set of findings observed on the current
  branch or PR head. For local review, this is the current self-inspection
  pass plus the latest local CodeRabbit run on that head. For PR review,
  this is the current set of thread and non-thread findings visible after
  the latest observation / re-review turn on that head.
- **Review closure complete** — every finding in the current review cycle is
  classified and closed.

## Principle

> Reach agreement with bot reviewers on ALL findings before proceeding
> to merge. Findings without agreement remain unresolved and block merge.

**Unilateral resolve is prohibited.** The agent must not resolve a thread
until the reviewer's final response confirms the disposition.

## Completeness Invariant

```text
unresolved threads == 0  ⟹  all thread-based findings have consensus
```

GitHub Branch Protection (`required_conversation_resolution`) blocks
merge until every thread is resolved. This is a deterministic guard.

**Non-thread findings** (outside-diff-range comments, PR summary
findings, conversation-level comments) are NOT tracked by GitHub's
thread resolution mechanism. `unresolved threads == 0` is therefore
**necessary but not sufficient** for full consensus. The agent must
additionally verify that all non-thread findings have been dispositioned
via PR conversation comments (see § Non-thread findings below).
`HS-NO-DISMISS` prohibits skipping these findings as "pre-existing."

Within a PR review cycle:

```text
review closure complete
  ⟹ all thread findings are closed
  ∧ all non-thread findings are dispositioned on the PR
```

## PR-side application of disposition categories

Every thread receives a disposition reply before being resolved (per
`HS-REVIEW-RESOLVE`). Use the four categories defined in
`.reinguard/policy/review--disposition-categories.md`. If the bot objects
after the initial reply, the agent posts a new disposition reply per round
until consensus is reached.

| Category | Consensus requirement | Template |
|---|---|---|
| **Fixed** | Re-review confirms fix (no new finding on same lines) | `Fixed in \`<sha7>\`. <what changed>.` |
| **By design** | Bot reply does not object (acceptance or no further comment after re-review) | `By design. <rationale> (ref: <source>).` |
| **False positive** | Bot reply does not object | `False positive. <why detection was wrong>.` |
| **Acknowledged** | Follow-up Issue **only** when work is a substantial separate deliverable; otherwise rationale without `Tracked in` | `Acknowledged. <brief assessment>. Tracked in #<issue>.` *or* rationale if no Issue |

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

## Non-thread findings

Review bots may report findings that do not create GitHub review threads:

- **Outside-diff-range comments** — findings on lines not modified by the PR
- **PR summary findings** — items listed in the review summary but not anchored to a specific line
- **Conversation-level comments** — bot comments posted as PR conversation, not as part of a pull review

These findings are invisible to `required_conversation_resolution` and
to the `review_threads_unresolved` signal in the observation engine.

**Disposition obligation** (per `HS-REVIEW-RESOLVE` and `HS-NO-DISMISS`):
the agent must disposition every non-thread finding using a **PR
conversation comment** that quotes the finding and states the disposition
category (Fixed / By design / False positive / Acknowledged). The same
consensus model applies: prefer Fixed / By design / False positive over
Acknowledged; use Acknowledged only per § **Acknowledged — in-PR resolution
vs follow-up Issue**.

**Completeness check**: before proceeding to merge, the agent must
confirm that all non-thread findings from the current review cycle have
been dispositioned. This is a Steering obligation (agent self-policing);
no deterministic guard currently tracks it.

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

- `.reinguard/procedure/change-inspect.md` — local review closure before PR creation
- `.reinguard/knowledge/review--bot-operations.md` — trigger, detection,
  timing
- `.reinguard/policy/review--disposition-categories.md` — shared
  disposition vocabulary across local and PR review
- `.reinguard/policy/safety--agent-invariants.md` § **HS-REVIEW-RESOLVE**
- `.reinguard/procedure/review-address.md` — review-address procedure
