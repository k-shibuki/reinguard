## Context

Upgrade `rgd config validate` from MVP stub to JSON Schema validation (warnings for deprecated fields per ADR-0008).

## Refs: ADR

- ADR-0008

## ADR Impact

`none`

## Definition of Done

- [ ] Validates `reinguard.yaml` + rules + knowledge manifest as defined in #5
- [ ] Exit non-zero on hard validation failure; warnings to stderr

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Valid `.reinguard/` | Normal | exit 0, "config OK" | — |
| TC-A-01 | Wrong `schema_version` major | Abnormal | clear error | — |

## Dependencies

- Depends on: #5

## Notes

Coordinate with `schema export` output paths.
