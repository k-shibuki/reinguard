## Context

Observe open issues and parsed metadata (labels, test plan flags, blocked-by) via GitHub API using token from `gh auth token` (ADR-0006).

## Refs: ADR

- ADR-0005, ADR-0006

## ADR Impact

`none`

## Acceptance ↔ ADR

- ADR-0006: sole auth via `gh auth token`; no parallel env-only token path

## Definition of Done

- [ ] Subpackage under `internal/observe/github/issues/`
- [ ] Integration tests behind build tag or httptest mocks
- [ ] 429: limited exponential backoff then surface degraded (align with #16)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Mock HTTP returns 1 open issue | Normal | `open_issues_count==1` | httptest |
| TC-A-01 | 401 from API | Abnormal | Fatal or degraded with diagnostic | — |

## Dependencies

- Depends on: #9

## Notes

Rate limit: coordinate with other GitHub facets (shared client if useful).
