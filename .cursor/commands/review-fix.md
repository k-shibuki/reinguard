# review-fix

## Context

- `.reinguard/policy/review--consensus-protocol.md` — disposition categories, CodeRabbit resolution gate, no unilateral resolve

**Review knowledge (discover via substrate):**

```bash
rgd knowledge pack --query "review"
```

Open each `entries[].path` returned (typically includes `review--bot-operations.md`, `review--github-thread-api.md`). Use those docs for API channels, re-review triggers, REST vs GraphQL for `isResolved`, and outside-diff-range collection.

**Already in context**: `reinguard-bridge.mdc` (HS-*).

**Observation (initial):**

```bash
rgd observe github reviews
```

Use aggregate review / CI signals from observation JSON where helpful.

**Thread-level work:** `rgd observe github reviews` does not replace per-thread `isResolved` enumeration — use `gh api` / GraphQL per knowledge from `rgd knowledge pack --query "review"` (see `review--github-thread-api.md`).

**Aggregate + manual checks (after `rgd observe github reviews`):**

- GitHub **Files changed** → each review thread; root comment **CodeRabbit**, **Codex / chatgpt-codex-connector**, or human.
- `gh pr view <N> --comments` and `gh pr checks <N>` — failing checks vs review findings.
- Inline threads: `gh api repos/{owner}/{repo}/pulls/<N>/comments` until `rgd observe` covers this workflow.
- **Outside diff range / non-inline bot findings** — per knowledge from `knowledge pack`: do **not** treat "zero unresolved threads" as "no review work left." Collect summary bodies, PR conversation, Checks text. Classify and disposition like inline threads; without comment id, post a **PR conversation comment** (quote + disposition), then `@coderabbitai review` when budget/rules allow.

## Act

### 1. Classify every comment by correctness

Evaluate **every** review comment — regardless of severity label (P0, P1, nitpick, trivial, etc.). Severity does **not** exempt a comment from evaluation or reply.

Map each finding to **exactly one** of the four disposition categories in `.reinguard/policy/review--consensus-protocol.md` § **Disposition Categories (4, exhaustive)** (**Fixed** / **By design** / **False positive** / **Acknowledged**). When planning fixes, treat a finding as "valid for this PR" when the disposition will be **Fixed**; treat as out-of-scope when it will be **Acknowledged** (with a tracking Issue per that protocol).

### 2. Fix all findings that will be disposition **Fixed**

Apply code or doc changes for every comment you will close with **Fixed**. Do not silently defer valid in-scope findings; use **Acknowledged** only per the protocol (tracking Issue distinct from the PR's `Closes` target).

### 3. Reply on every thread — no exceptions

For each **pull review comment** (file/line thread), post a disposition reply (**Fixed** / **By design** / **False positive** / **Acknowledged**) as a **threaded reply** (`gh api POST repos/{owner}/{repo}/pulls/<N>/comments` with `in_reply_to=<root_comment_database_id>`).
A generic PR body or issue-style comment **does not** substitute for a thread reply.
For **outside-diff / summary-only** findings (see Context) with no anchor id, a **targeted PR conversation comment** (quote + disposition) is the correct substitute — not silence.

### 4. CodeRabbit threads

Follow `.reinguard/policy/review--consensus-protocol.md` § **CodeRabbit Resolution Gate** (do **not** `resolve` until consensus evidence). After you push fixes, trigger a new review cycle with a **PR conversation comment**:
`gh pr comment <N> --body "@coderabbitai review"`
(budget and skip rules: paths from `rgd knowledge pack --query "review"`.)

### 5. Codex threads

Same as step 3: post disposition as a **reply on the bot's review thread**. **Then** request a new Codex pass with a **PR conversation comment** that includes **`@codex review`** (and a one-line summary of what changed). The connector generally **does not** re-run from a thread reply alone; without `@codex review` on the PR timeline, follow-up review often does not run.
If your org limits agent-posted `@codex`, have the human post that line.

### 6. Local verify, commit, push

When Go code changed:
- `go test ./...`
- `go vet ./...`
- `golangci-lint run` (or document in the PR why relying on CI-only is acceptable)
- New commit with `Refs: #<issue>` (no amend+force-push on the PR head).

### 7. Post-push

After bot re-review: re-check threads and `gh pr checks <N>` until **`ci-pass`** is green; resolve threads only when **HS-REVIEW-RESOLVE** is satisfied.

## Output

- Classification: every comment mapped to the four disposition categories in `review--consensus-protocol.md`.
- Disposition posted per thread: Fixed / By design / False positive / Acknowledged.
- Fixes applied; which threads got threaded replies; whether `@coderabbitai review` / `@codex review` was posted; remaining blockers.

## Guard

HS-REVIEW-RESOLVE, HS-LOCAL-VERIFY, HS-NO-SKIP, HS-CI-MERGE, HS-MERGE-CONSENSUS, HS-PR-TEMPLATE, HS-PR-BASE
