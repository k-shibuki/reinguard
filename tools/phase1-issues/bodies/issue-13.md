## Context

Observe review threads, disposition, and minimal bot-related fields required for named evaluators (Phase 1 subset).

## Refs: ADR

- ADR-0005, ADR-0006, ADR-0007

## ADR Impact

`none`

## Definition of Done

- [ ] `internal/observe/github/reviews/` with GraphQL or REST per implementation choice
- [ ] Tests with recorded fixtures (no network in default `go test`)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Fixture: 0 unresolved threads | Normal | `review_threads_unresolved==0` | — |
| TC-B-01 | Pagination truncated flag | Boundary | diagnostic if incomplete counts | If applicable |

## Dependencies

- Depends on: #9

## Notes

Keep Phase 1 scope smaller than bridle’s full bot matrix unless #17 demands parity.
