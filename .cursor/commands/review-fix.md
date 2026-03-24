# review-fix

## Reads

- `agent-safety.mdc` (`HS-REVIEW-RESOLVE`, `HS-NO-SKIP`, `HS-LOCAL-VERIFY`)
- `AGENTS.md` (dispositions, thread policy)
- [bridle `review--consensus-protocol.md`](https://github.com/bridle-org/bridle/blob/main/.cursor/knowledge/review--consensus-protocol.md) — disposition categories, CodeRabbit resolution gate, no unilateral resolve
- [bridle `review--bot-operations.md`](https://github.com/bridle-org/bridle/blob/main/.cursor/knowledge/review--bot-operations.md) — how CodeRabbit vs Codex signal agreement; re-review triggers
- [bridle `review--disposition-reply.md` template](https://github.com/bridle-org/bridle/blob/main/.cursor/templates/review--disposition-reply.md) — copy-paste dispositions and **`gh api` reply/resolve** (`in_reply_to`, full `commit_id`)

## Sense

- GitHub **Files changed** → open each review thread; note whether the root comment is **CodeRabbit**, **Codex / chatgpt-codex-connector**, or human.
- `gh pr view <N> --comments` and `gh pr checks <N>` — failing checks vs review findings.
- Until `rgd observe` exists, use `gh api` to list inline review threads, e.g. `gh api repos/{owner}/{repo}/pulls/<N>/comments`.
- **Outside diff range / non-inline bot findings** — CodeRabbit (and peers) may post items **only** outside the patch (label **Outside diff range**, review **summary** body, PR **conversation**, or Checks text) so they **never** appear in `pulls/<N>/comments`. **Do not** treat “zero unresolved threads” as “no review work left.” Also collect:
  - `gh api repos/{owner}/{repo}/pulls/<N>/reviews` — inspect each review `body` for summary-only findings
  - `gh pr view <N> --comments` for timeline text
  - Optional: `gh api repos/{owner}/{repo}/issues/<N>/comments` when the PR is issue `#N` on the same repo
  Classify and disposition these like inline threads; if there is **no** comment id to reply under, post a **PR conversation comment** that quotes or names the finding and states **Fixed / By design / False positive / Acknowledged**, then still run `@coderabbitai review` when CodeRabbit budget/rules allow.

## Act

1. Classify each finding: P0 / P1 / false positive / already addressed.
2. Fix P0/P1 in code or docs.
3. **Dispositions: use threaded replies, not only top-level PR comments**  
   For each **pull review comment** (file/line thread), post the disposition (**Fixed / By design / False positive / Acknowledged**) as a **Reply on that thread** (GitHub UI **Reply**, or `gh api` `POST repos/{owner}/{repo}/pulls/<N>/comments` with `in_reply_to=<root_comment_database_id>` per the bridle disposition template).  
   A generic PR body or issue-style comment **does not** substitute for a thread reply when the bot left a review comment on the diff.  
   For **outside-diff / summary-only** findings (Sense above) with no anchor id, a **targeted PR conversation comment** (quote + disposition) is the correct substitute—not silence.

4. **CodeRabbit threads** — follow [bridle § CodeRabbit resolution gate](https://github.com/bridle-org/bridle/blob/main/.cursor/knowledge/review--consensus-protocol.md#coderabbit-resolution-gate-resolvereviewthread): do **not** `resolve` until consensus evidence (CR auto-resolved, CR replied without objecting, or qualifying review after re-trigger). After you push fixes, trigger a new review cycle with a **PR conversation comment**:  
   `gh pr comment <N> --body "@coderabbitai review"`  
   (budget / skip rules: `review--bot-operations.md`.)

5. **Codex (e.g. chatgpt-codex-connector) threads** — same as step 3: post disposition as a **reply on the bot’s review thread**. **Then** request a new Codex pass with a **PR conversation comment** that includes **`@codex review`** (and a one-line summary of what changed). The connector generally **does not** re-run from a thread reply alone; without `@codex review` on the PR timeline, follow-up review often does not run.  
   (Bridles strict rule is “Codex only on user instruction”; if your org limits agent-posted `@codex`, have the human post that line.)

6. **Local verify** (when Go code changed), then commit and push:
   - `go test ./...`
   - `go vet ./...`
   - `golangci-lint run` (or document in the PR why relying on CI-only is acceptable)
   - New commit with `Refs: #<issue>` (no amend+force-push on the PR head).

7. After bot re-review: re-check threads and `gh pr checks <N>` until **`ci-pass`** is green; resolve threads only when `HS-REVIEW-RESOLVE` is satisfied.

## Output

- Findings addressed by severity; which threads got threaded replies; whether `@coderabbitai review` / `@codex review` was posted; remaining blockers.

## Guard

- `HS-REVIEW-RESOLVE`, `HS-LOCAL-VERIFY`, `HS-NO-SKIP`
- `HS-CI-MERGE` (do not merge with failing required checks)
- `HS-MERGE-CONSENSUS` (do not enable auto-merge while bot review is pending or threads unresolved)
- `HS-PR-TEMPLATE` (PR body must stay policy-complete during follow-up edits)
- `HS-PR-BASE` (if any follow-up recreates a PR: target `main` only; document stack deps in the body)
