## Context

Replace placeholder operational context schema; implement `rgd context build` full pipeline per `docs/cli.md` default steps.

## Refs: ADR

- ADR-0007, ADR-0008

## ADR Impact

`none` unless schema bump requires ADR-0008 cross-reference update

## Definition of Done

- [ ] `pkg/schema/` operational context is non-placeholder; `schema_version` synchronized
- [ ] `context build` runs: observe → state eval → route select → guard eval → knowledge pack → final JSON stdout
- [ ] Optional step flags deferred unless explicitly added in this PR

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Golden repo fixture | Normal | output matches golden JSON | Use httptest/mocks |
| TC-A-01 | Partial provider failure | Boundary | degraded in output, exit 0 default | — |

## Dependencies

- Depends on: #15, #16, #17, #18, #19, #20

## Notes

Module coverage ≥80% must still hold after merge (CI).
