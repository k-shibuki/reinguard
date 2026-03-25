# pr-merge

## Reads

- `agent-safety.mdc` (`HS-CI-MERGE`, `HS-REVIEW-RESOLVE`, `HS-MERGE-CONSENSUS`)

## Sense

- `gh pr checks <N>` — all required checks green; especially **`ci-pass`**.
- `gh pr view <N>` — mergeable; no unresolved review conversations if branch protection requires resolution.

## Act

1. Confirm: CI success, PR policy satisfied, threads resolved (if required).
2. Merge: `gh pr merge <N> --squash` or `--merge` per [docs/contributing.md](../../docs/contributing.md) and maintainer convention for this repo.  
   Do **not** use `--admin`. Do **not** merge with failing checks.

## Output

- Confirm merged: `gh pr view <N> --json state -q .state`

## Guard

- `HS-CI-MERGE`: green checks; no `--admin`
- `HS-REVIEW-RESOLVE`: threads resolved per policy
- `HS-MERGE-CONSENSUS`: no premature auto-merge while review pending
