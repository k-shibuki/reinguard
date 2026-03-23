## Context

`rgd state eval` — load rules type `state`, run match + resolution + named evaluators (built-in subset for Phase 1).

## Refs: ADR

- ADR-0002, ADR-0004, ADR-0007

## ADR Impact

`none` unless evaluator behavior needs ADR-0002 clarification

## Definition of Done

- [ ] Reads observation from live `observe` or stdin / `--observation-file`
- [ ] Emits evaluation outcome JSON per schema
- [ ] Built-in evaluators: document which ship in Phase 1 (e.g. review consensus placeholder) — minimal real logic acceptable if covered by tests

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Fixture observation + rules → one state | Normal | `resolved` + state id | Golden file |
| TC-B-01 | Tie priority | Boundary | `ambiguous` | — |
| TC-A-01 | Malformed observation file | Abnormal | non-zero exit | — |

## Dependencies

- Depends on: #7, #8, #15, #16

## Notes

Default exit 0 for ambiguous/degraded; optional `--fail-on-non-resolved` per #15.
