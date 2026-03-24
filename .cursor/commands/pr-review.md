# pr-review

## Reads

- `AGENTS.md` (severity, P0/P1 scope)
- `.github/PULL_REQUEST_TEMPLATE.md` (traceability expectations)

## Sense

- `gh pr diff <N>` and `gh pr view <N> --json files,title,body`.
- Optional: CodeRabbit / human review comments on the PR.

## Act

- Produce a **review judgment** only: P0/P1 findings, ADR alignment, test gaps.  
- **Do not** merge, **do not** push commits (implementation belongs in `review-fix` / feature branch).

## Output

- Structured list of findings with severity; recommendation: approve / changes requested.

## Guard

- This command is **read / judge only** — merge is `pr-merge` after `review-fix` when required.
