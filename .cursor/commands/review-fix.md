# review-fix

## Reads

- `agent-safety.mdc` (`HS-REVIEW-RESOLVE`, `HS-NO-SKIP`, `HS-LOCAL-VERIFY`)
- `AGENTS.md` (dispositions, thread policy)
- `.reinguard/policy/review--consensus-protocol.md` — disposition categories, CodeRabbit resolution gate, no unilateral resolve
- `.reinguard/knowledge/review--bot-operations.md` — how CodeRabbit vs Codex signal agreement; re-review triggers

## Sense

- GitHub **Files changed** → open each review thread; note whether the root comment is **CodeRabbit**, **Codex / chatgpt-codex-connector**, or human.
- `gh pr view <N> --comments` and `gh pr checks <N>` — failing checks vs review findings.
- Until `rgd observe` exists, use `gh api` to list inline review threads, e.g. `gh api repos/{owner}/{repo}/pulls/<N>/comments`.
- **Outside diff range / non-inline bot findings** — CodeRabbit (and peers) may post items **only** outside the patch (label **Outside diff range**, review **summary** body, PR **conversation**, or Checks text) so they **never** appear in `pulls/<N>/comments`. **Do not** treat "zero unresolved threads" as "no review work left." Also collect:
  - `gh api repos/{owner}/{repo}/pulls/<N>/reviews` — inspect each review `body` for summary-only findings
  - `gh pr view <N> --comments` for timeline text
  - Optional: `gh api repos/{owner}/{repo}/issues/<N>/comments` when the PR is issue `#N` on the same repo
  Classify and disposition these like inline threads; if there is **no** comment id to reply under, post a **PR conversation comment** that quotes or names the finding and states the disposition, then still run `@coderabbitai review` when CodeRabbit budget/rules allow.

## Act

### 1. Classify every comment by correctness

Evaluate **every** review comment — regardless of severity label (P0, P1, nitpick, trivial, etc.). Severity does **not** exempt a comment from evaluation or reply. Classify each into exactly one category:

| Category | Meaning | Action |
|---|---|---|
| **Valid** | The finding is correct and the code should change. | Fix in code or docs. |
| **False positive** | The finding is factually wrong or based on a misunderstanding of the code. | Explain why in the reply. |
| **By design** | The current behavior is intentional; the reviewer's suggestion conflicts with an explicit design decision (ADR, spec, etc.). | Cite the rationale in the reply. |

### 2. Fix all Valid findings

Apply code or doc changes for every comment classified as **Valid**. Do not defer Valid findings unless the fix requires a separate Issue (in which case, file the Issue and cite it in the reply).

### 3. Reply on every thread — no exceptions

For each **pull review comment** (file/line thread), post a disposition reply (**Fixed** / **By design** / **False positive**) as a **threaded reply** (`gh api POST repos/{owner}/{repo}/pulls/<N>/comments` with `in_reply_to=<root_comment_database_id>`).
A generic PR body or issue-style comment **does not** substitute for a thread reply.
For **outside-diff / summary-only** findings (see Sense) with no anchor id, a **targeted PR conversation comment** (quote + disposition) is the correct substitute — not silence.

### 4. CodeRabbit threads

Follow the **CodeRabbit resolution gate** in `.reinguard/policy/review--consensus-protocol.md`: do **not** `resolve` until consensus evidence (CR auto-resolved, CR replied without objecting, or qualifying review after re-trigger). After you push fixes, trigger a new review cycle with a **PR conversation comment**:
`gh pr comment <N> --body "@coderabbitai review"`
(see `.reinguard/knowledge/review--bot-operations.md` for budget and skip rules.)

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

After bot re-review: re-check threads and `gh pr checks <N>` until **`ci-pass`** is green; resolve threads only when `HS-REVIEW-RESOLVE` is satisfied.

## Output

- Classification table: every comment → Valid / False positive / By design.
- Fixes applied; which threads got threaded replies; whether `@coderabbitai review` / `@codex review` was posted; remaining blockers.

## Guard

- `HS-REVIEW-RESOLVE`, `HS-LOCAL-VERIFY`, `HS-NO-SKIP`
- `HS-CI-MERGE` (do not merge with failing required checks)
- `HS-MERGE-CONSENSUS` (do not enable auto-merge while bot review is pending or threads unresolved)
- `HS-PR-TEMPLATE` (PR body must stay policy-complete during follow-up edits)
- `HS-PR-BASE` (if any follow-up recreates a PR: target `main` only; document stack deps in the body)
