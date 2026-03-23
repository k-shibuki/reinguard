## Context

Observe CI rollup for relevant PRs: check statuses, failed job names, aggregated `ci_status` enum compatible with downstream rules.

## Refs: ADR

- ADR-0005, ADR-0006

## ADR Impact

`none`

## Definition of Done

- [ ] `internal/observe/github/ci/`
- [ ] Tests with fixture JSON from GitHub API shapes
- [ ] Document filtering rules if bot statuses excluded (optional Phase 1 scope — if omitted, state “not yet” in diagnostics)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | All checks success | Normal | `ci_status==success` | — |
| TC-B-01 | Pending check | Boundary | `pending` | — |
| TC-A-01 | Any failed required check | Abnormal | `failure` + `ci_failed_jobs` populated | — |

## Dependencies

- Depends on: #9

## Notes

May depend on PR number from branch — document resolution order with #12.
