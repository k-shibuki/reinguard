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
- Until `rgd observe` exists, use `gh api` to list review comments if needed, e.g. `gh api repos/{owner}/{repo}/pulls/<N>/comments`.

## Act

1. Classify each finding: P0 / P1 / false positive / already addressed.
2. Fix P0/P1 in code or docs.
3. **Dispositions: use threaded replies, not only top-level PR comments**  
   For each **pull review comment** (file/line thread), post the disposition (**Fixed / By design / False positive / Acknowledged**) as a **Reply on that thread** (GitHub UI **Reply**, or `gh api` `POST repos/{owner}/{repo}/pulls/<N>/comments` with `in_reply_to=<root_comment_database_id>` per the bridle disposition template).  
   A generic PR body or issue-style comment **does not** substitute for a thread reply when the bot left a review comment on the diff.

4. **CodeRabbit threads** — follow [bridle § CodeRabbit resolution gate](https://github.com/bridle-org/bridle/blob/main/.cursor/knowledge/review--consensus-protocol.md#coderabbit-resolution-gate-resolvereviewthread): do **not** `resolve` until consensus evidence (CR auto-resolved, CR replied without objecting, or qualifying review after re-trigger). After you push fixes, trigger a new review cycle with a **PR conversation comment**:  
   `gh pr comment <N> --body "@coderabbitai review"`  
   (budget / skip rules: `review--bot-operations.md`.)

5. **Codex (e.g. chatgpt-codex-connector) threads** — same as step 3: post disposition as a **reply on the bot’s review thread**. **Then** request a new Codex pass with a **PR conversation comment** that includes **`@codex review`** (and a one-line summary of what changed). The connector generally **does not** re-run from a thread reply alone; without `@codex review` on the PR timeline, follow-up review often does not run.  
   (Bridles strict rule is “Codex only on user instruction”; if your org limits agent-posted `@codex`, have the human post that line.)

6. Commit and push: new commit with `Refs: #<issue>` (no amend+force-push on the PR head).

7. After bot re-review: re-check threads and `gh pr checks <N>` until **`ci-pass`** is green; resolve threads only when `HS-REVIEW-RESOLVE` is satisfied.

## Output

- Findings addressed by severity; which threads got threaded replies; whether `@coderabbitai review` / `@codex review` was posted; remaining blockers.

## Guard

- `HS-REVIEW-RESOLVE`, `HS-LOCAL-VERIFY`, `HS-NO-SKIP`
