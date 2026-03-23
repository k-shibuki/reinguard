## Context

`rgd route select` — evaluate `route` rules given current state / signals from prior step.

## Refs: ADR

- ADR-0002, ADR-0004

## ADR Impact

`none`

## Definition of Done

- [ ] Input: observation + optional prior `state eval` output (define in `docs/cli.md`)
- [ ] Output: ordered route candidates with priorities

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | One matching route | Normal | single candidate | — |
| TC-A-01 | No route matches | Abnormal | empty list or explicit outcome | Document |

## Dependencies

- Depends on: #7, #15, #17

## Notes

Reuse match engine; do not duplicate operator parsing.
