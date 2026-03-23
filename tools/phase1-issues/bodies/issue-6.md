## Context

Implement the bounded match language from ADR-0002 over a signal map (`map[string]any`).

## Refs: ADR

- ADR-0002

## ADR Impact

`none`

## Acceptance ↔ ADR

- ADR-0002: operators `eq`, `ne`, `gt`, `lt`, `gte`, `lte`, `in`, `contains`, `exists`, `not_exists`, `count`, `any`, `all`, `and`, `or`, `not`; no arithmetic in match.

## Definition of Done

- [ ] Package `internal/match/` evaluates rules against signals
- [ ] Unit tests for each operator and nested logic
- [ ] Reject or document unsupported constructs clearly

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | `eq` on string field | Normal | true | — |
| TC-B-01 | Empty array with `all` | Boundary | true (vacuous) or per spec — document | — |
| TC-A-01 | `gt` on non-numeric | Abnormal | Eval error or false per spec | Document |

## Dependencies

- None (can parallelize with #5 after ADR-0009 exists for naming only — optional)

## Notes

Keep evaluator deterministic; no I/O.
