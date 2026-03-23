## Context

`rgd knowledge pack` — resolve knowledge references from JSON manifest under `.reinguard/knowledge/` validated by JSON Schema (#5).

## Refs: ADR

- ADR-0008

## ADR Impact

`none`

## Definition of Done

- [ ] Read manifest; output list of paths/URIs for agent consumption (do not embed file bodies unless specified in schema)
- [ ] Tests with fixture manifest

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Manifest matches current state | Normal | non-empty pack | — |
| TC-A-01 | Invalid manifest vs schema | Abnormal | validation error | — |

## Dependencies

- Depends on: #6, #15, #17

## Notes

Knowledge content is repo-owned; rgd packs references only.
