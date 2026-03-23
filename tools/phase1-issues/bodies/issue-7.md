## Context

Resolve competing rule matches using unified float priority, `depends_on` degradation, and explicit ambiguous/degraded outcomes (ADR-0004, ADR-0007).

## Refs: ADR

- ADR-0004
- ADR-0007

## ADR Impact

`none`

## Acceptance ↔ ADR

- ADR-0004: suppress on failed deps; min priority wins; tie at best → ambiguous; duplicate priority → warning
- ADR-0007: outcomes as structured evaluation status, not fake state IDs

## Definition of Done

- [ ] Package `internal/resolve/` (or integrated in `match`) implements resolution API consumed by `state eval`
- [ ] Tests: tie priority → ambiguous; all suppressed → degraded; happy path → resolved

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Two rules, priorities 10 and 20 | Normal | 10 wins | Lower = higher precedence |
| TC-B-01 | Two matches both priority 1.0 | Boundary | `ambiguous` outcome | — |
| TC-A-01 | Match depends_on missing signal source | Abnormal | suppressed / degraded per spec | — |

## Dependencies

- Depends on: #7

## Notes

Float equality for duplicate-priority warning: use normalized comparison per ADR-0004 consequences.
