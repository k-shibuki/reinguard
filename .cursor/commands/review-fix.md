# review-fix

## Reads

- `agent-safety.mdc` (`HS-REVIEW-RESOLVE`, `HS-NO-SKIP`, `HS-LOCAL-VERIFY`)
- `AGENTS.md` (dispositions, thread policy)
- [bridle review consensus protocol](https://github.com/bridle-org/bridle/blob/main/.cursor/knowledge/review--consensus-protocol.md) (full detail)

## Sense

- `gh pr view <N> --comments` and GitHub **Files changed** / review threads.
- `gh pr checks <N>` — ensure understanding of failing checks vs review comments.

## Act

1. Classify findings: P0 / P1 / false positive / already addressed.
2. Fix code or docs for P0/P1.
3. For each review thread: post a **disposition** reply (Fixed / By design / False positive / Acknowledged), then **resolve** the thread (`HS-REVIEW-RESOLVE`).
4. Commit and push: new commit with `Refs: #<issue>` (no amend+force-push on PR head per `HS-CI-MERGE` spirit).
5. Re-trigger bot if used: `gh pr comment <N> --body "@coderabbitai review"`.
6. Re-check `gh pr checks <N>` until **`ci-pass`** green.

## Output

- Summary of fixes and thread resolutions; remaining blockers if any.

## Guard

- `HS-REVIEW-RESOLVE`, `HS-LOCAL-VERIFY`, `HS-NO-SKIP`
