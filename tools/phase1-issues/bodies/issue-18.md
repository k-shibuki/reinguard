## Context

`rgd guard eval <guard-id>` — Phase 1 uses **flags only** for intent (no stdin JSON). Tables in `docs/cli.md` are SSOT for required flags per guard.

## Refs: ADR

- ADR-0001 (substrate does not infer semantic intent)
- ADR-0002

## ADR Impact

`none`

## Definition of Done

- [ ] At least one guard (e.g. `merge-readiness`) with documented flags
- [ ] Guard `id` matches rules config and `docs/cli.md`

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Valid flags + observation | Normal | pass/fail JSON | — |
| TC-A-01 | Missing required flag | Abnormal | exit non-zero, stderr | — |

## Dependencies

- Depends on: #7, #15, #17

## Notes

Align with substrate boundary: no implicit intent from observation alone.
