## Context

Observe pull requests: open list, match current branch, recent merges, mergeable field as needed for signals.

## Refs: ADR

- ADR-0005, ADR-0006

## ADR Impact

`none`

## Definition of Done

- [ ] `internal/observe/github/pullrequests/`
- [ ] Mock-based tests + documented signal keys

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Mock: one OPEN PR for current branch | Normal | `pr_exists_for_branch==true` | — |
| TC-B-01 | No open PRs | Boundary | false / empty list per schema | — |

## Dependencies

- Depends on: #9

## Notes

Align field names with `docs/cli.md` once #15 lands.
