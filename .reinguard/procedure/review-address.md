---
id: procedure-review-address
purpose: Classify review feedback, fix as needed, post dispositions, and satisfy HS-REVIEW-RESOLVE.
applies_to:
  state_ids:
    - pr_open
    - waiting_ci
    - unresolved_threads
    - changes_requested
  route_ids:
    - user-monitor-pr
    - user-wait-ci
    - user-address-review
reads:
  - ../policy/review--disposition-categories.md
  - ../policy/review--consensus-protocol.md
  - ../policy/coding--standards.md
  - ../knowledge/review--multi-source-review-signals.md
  - ../knowledge/review--incremental-fix-flow.md
sense:
  - rgd context build
  - rgd observe github reviews
act:
  - Step 0 when tree dirty; change-inspect then commit; classify, fix, thread replies, bot re-review triggers, push, verify CI.
output:
  - Disposition ledger, fixes, remaining blockers.
done_when: Review closure is complete for the current PR review cycle; ci-pass green when required; HS-REVIEW-RESOLVE satisfied before resolve.
escalate_when: Cannot reach consensus with bot reviewers per policy.
---

# review-address

## Context

- [`../policy/review--disposition-categories.md`](../policy/review--disposition-categories.md) — shared disposition vocabulary across local and PR review
- [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md) — shared review-closure model, PR-side consensus mechanics, CodeRabbit resolution gate
- [`../policy/coding--standards.md`](../policy/coding--standards.md) § **Change scope** — same-kind drift across code, `.reinguard/`, and `.cursor/` before hand-off
- [`../knowledge/review--multi-source-review-signals.md`](../knowledge/review--multi-source-review-signals.md) — dedupe and priority across bots, humans, checks, and timeline (single inbox)
- **Bot quota / pause / in-flight only** (no open thread work): prefer [`wait-bot-review.md`](wait-bot-review.md) when FSM routes to `user-wait-bot-*`.

**Review knowledge (discover via substrate):**

```bash
rgd context build
```

Use `knowledge.entries` when a PR exists for the branch (review entries match `github.pull_requests.pr_exists_for_branch`). Open each `entries[].path` returned (typically includes `review--incremental-fix-flow.md`, `review--multi-source-review-signals.md`, `review--bot-operations.md`, `review--github-thread-api.md`). Use those docs for API channels, re-review triggers, REST vs GraphQL for `isResolved`, and outside-diff-range collection.

Optional trigger pass: `rgd observe > /tmp/rgd-observe.json` then `rgd knowledge pack --observation-file /tmp/rgd-observe.json --query "review"`.

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow & commit policy.

**Observation (initial):**

```bash
rgd observe github reviews
```

Use aggregate review / CI signals from observation JSON where helpful.

**Thread-level work:** `rgd observe github reviews` does not replace per-thread `isResolved` enumeration — use `gh api` / GraphQL per knowledge from `rgd context build` / `knowledge.entries` (or the optional `knowledge pack --observation-file … --query "review"` pass) (see `review--github-thread-api.md`).

**When `state_id` is `waiting_ci` (`user-wait-ci`):** no open thread or formal changes-request work is implied by the FSM. Focus on **`gh pr checks`**, **`ci-pass`**, and mergeability (`gh pr view`); fix failing jobs or wait for pending checks. Re-run `rgd context build` after pushes.

**Aggregate + manual checks (after `rgd observe github reviews`):**

- GitHub **Files changed** → each review thread; root comment **CodeRabbit**, **Codex / chatgpt-codex-connector**, or human.
- `gh pr view <N> --comments` and `gh pr checks <N>` — failing checks vs review findings.
- Inline threads: `gh api repos/{owner}/{repo}/pulls/<N>/comments` until `rgd observe` covers this workflow.
- **Outside diff range / non-inline bot findings** — per review knowledge from context / pack: do **not** treat "zero unresolved threads" as "no review work left." Collect summary bodies, PR conversation, Checks text. Classify and disposition like inline threads; without comment id, post a **PR conversation comment** (quote + disposition), then `@coderabbitai review` when budget/rules allow.

## Act

### 0. Local work gate (uncommitted changes)

If `observation.signals.git.working_tree_clean` is `false` (from `rgd context build` or `rgd observe`):

1. Run [`change-inspect.md`](change-inspect.md) against **committed delta + staged + unstaged** (same dimensions as pre-PR in `review--self-inspection.md`, scoped to the incremental change).
2. **Commit** fixes with `Refs: #<issue>` (no amend+force-push on the PR head).
3. Re-run `rgd context build` (or `rgd observe`) to refresh signals before continuing below.

This keeps review-sourced fixes inspected and committed before disposition-heavy steps. Full pattern: [`../knowledge/review--incremental-fix-flow.md`](../knowledge/review--incremental-fix-flow.md).

### 0.5 CodeRabbit duplicate-comment suppression (`duplicate_findings_detected`)

After `rgd context build` / `rgd observe github reviews`, check `observation.signals.github.reviews.bot_review_diagnostics.duplicate_findings_detected`.

When **true**, CodeRabbit’s latest `PullRequestReview` body listed one or more findings under **♻️ Duplicate comments (N)** — meaning the bot detected the same issue again but **did not open new inline threads** (deduplication). That is **not** evidence the issue was fixed; it often means a prior thread was resolved while the underlying problem remained.

1. Open the bot’s latest review on GitHub (or use `gh api` / GraphQL to read `latestReviews` for the configured bot) and read the collapsed duplicate section.
2. Classify and disposition each listed finding like any other review feedback (fix, or **Fixed** / **By design** / **False positive** / **Acknowledged** with threaded reply or PR conversation comment per [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md) § **Non-thread findings** when there is no new thread id).
3. Do **not** treat `review_threads_unresolved == 0` alone as “no review work” while `duplicate_findings_detected` is true.

Per-bot count is also available as `cr_duplicate_findings_count` on the matching `bot_reviewer_status` entry when `enrich` includes `coderabbit`.

### 1. Classify every finding in the current PR review cycle by correctness

Evaluate **every** review comment and non-thread finding in the current PR
review cycle — regardless of any reviewer-supplied label or tone
(severity, nitpick, trivial, etc.). Such labels do **not** exempt a
finding from evaluation or reply.

Map each finding to **exactly one** of the four disposition categories
defined in
[`../policy/review--disposition-categories.md`](../policy/review--disposition-categories.md)
(**Fixed** / **By design** / **False positive** / **Acknowledged**),
following the PR-side consensus and thread-resolution mechanics in
[`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md)
§ **PR-side application of disposition categories**. Keep the same vocabulary
that `change-inspect` uses for local review; only the PR-side closure
evidence, consensus, and thread-resolution mechanics change here. Record a
disposition ledger covering both thread and non-thread findings for the
current PR review cycle.

For **Acknowledged** specifically — **before** posting the thread reply: read `review--consensus-protocol.md` § **Acknowledged — in-PR resolution vs follow-up Issue**. Open a **new** GitHub Issue with `Tracked in #…` **only** when deferred work is a **large, separately scoped** deliverable; otherwise use **Fixed** / **By design** / **False positive**, or **Acknowledged** with a written rationale and **no** new Issue when the protocol allows. Never use `Tracked in #N` where `N` equals the PR’s `Closes` issue.

### 2. Fix all findings that will be disposition **Fixed**

Apply code or doc changes for every finding you will close with **Fixed**.
Do not silently defer valid in-scope findings; use **Acknowledged** only
per the protocol (separate Issue only for substantial follow-up work;
never `Tracked in` the PR’s `Closes` target).

### 3. Reply on every thread and disposition every non-thread finding

For each **pull review comment** (file/line thread), post a disposition reply (**Fixed** / **By design** / **False positive** / **Acknowledged**) as a **threaded reply** (`gh api POST repos/{owner}/{repo}/pulls/<N>/comments` with `in_reply_to=<root_comment_database_id>`).
A generic PR body or issue-style comment **does not** substitute for a thread reply.
For **outside-diff / summary-only** findings (see Context) with no anchor id, a **targeted PR conversation comment** (quote + disposition) is the correct substitute — not silence. Outside-diff-range findings from the current review cycle MUST be dispositioned even if the code predates this PR (`HS-NO-DISMISS`). See `review--consensus-protocol.md` § **Non-thread findings**.

### 4. CodeRabbit threads

Follow [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md) § **CodeRabbit Resolution Gate** (do **not** `resolve` until consensus evidence). After you push fixes, **incremental auto-review** often runs on its own (`.coderabbit.yaml`). If it does not (pause threshold, skip, or you need a guaranteed pass), trigger a new review cycle with a **PR conversation comment**:
`gh pr comment <N> --body "@coderabbitai review"`
(budget and skip rules: paths from `knowledge.entries` after `rgd context build`, or optional `knowledge pack --observation-file … --query "review"`.)

### 5. Codex threads

Same as step 3: post disposition as a **reply on the bot's review thread**. **Then** request a new Codex pass with a **PR conversation comment** that includes **`@codex review`** (and a one-line summary of what changed). The connector generally **does not** re-run from a thread reply alone; without `@codex review` on the PR timeline, follow-up review often does not run.
If your org limits agent-posted `@codex`, have the human post that line.

### 6. Local verify, commit, push

When Go code changed:

- `go test ./...`
- `bash .reinguard/scripts/with-repo-local-state.sh -- go vet ./...`
- `bash .reinguard/scripts/with-repo-local-state.sh -- golangci-lint run`

When Markdown changed:

- `bash .reinguard/scripts/with-repo-local-state.sh -- pre-commit run markdownlint-cli2 --all-files`

New commit with `Refs: #<issue>` (no amend+force-push on the PR head).

### 7. Post-push

After bot re-review: re-check threads and `gh pr checks <N>` until **`ci-pass`** is green; resolve threads only when **HS-REVIEW-RESOLVE** is satisfied.

## Output

- Disposition ledger: every finding in the current PR review cycle mapped to the four disposition categories in `review--disposition-categories.md`.
- Disposition posted on the correct PR channel for each finding: thread reply for thread findings; PR conversation comment for non-thread findings.
- Fixes applied; which threads got threaded replies; whether `@coderabbitai review` / `@codex review` was posted; remaining blockers.

## Guard

HS-REVIEW-RESOLVE, HS-LOCAL-VERIFY, HS-NO-SKIP, HS-CI-MERGE, HS-MERGE-CONSENSUS, HS-PR-TEMPLATE, HS-PR-BASE
