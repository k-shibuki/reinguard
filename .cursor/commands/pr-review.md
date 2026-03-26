# pr-review

## Context

- `AGENTS.md` (severity P0/P1, review guidelines, traceability)

**Already in context**: `reinguard-bridge.mdc`, `workflow-policy.mdc`. PR title/body come from `gh pr view` (no need to list `.github/PULL_REQUEST_TEMPLATE.md` here).

**Diff and metadata:**

```bash
gh pr diff <N>
gh pr view <N> --json files,title,body
```

Optional: CodeRabbit / human review comments on the PR.

## Act

- Produce a **review judgment** only: P0/P1 findings, ADR alignment, test gaps.
- **Do not** merge, **do not** push commits (implementation belongs in `review-fix` / feature branch).

## Output

- Structured list of findings with severity; recommendation: approve / changes requested.

## Guard

This command is **read / judge only** — merge is `pr-merge` after `review-fix` when required.
