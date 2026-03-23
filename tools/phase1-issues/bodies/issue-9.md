## Context

Git-local signals via `git` subprocess: branch, dirtiness, stale branches, ahead count, stash.

## Refs: ADR

- ADR-0005 (external signals only)
- ADR-0006 (N/A for git)

## ADR Impact

`none`

## Acceptance ↔ ADR

- ADR-0005: only verifiable git state; no agent files

## Definition of Done

- [ ] Provider `git` implementing registry interface
- [ ] Tests with mocked `git` or hermetic test repo setup
- [ ] Respect `--cwd` / working directory from CLI contract (#15)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Temp repo on `main`, clean | Normal | `on_main==true`, `uncommitted_files==0` | — |
| TC-B-01 | Detached HEAD | Boundary | Documented signal values | — |
| TC-A-01 | `git` not in PATH | Abnormal | degraded/fatal diagnostic | — |

## Dependencies

- Depends on: #9

## Notes

Subprocess isolation; no shell injection.
