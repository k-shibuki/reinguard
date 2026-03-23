## Context

Define provider registry and observation orchestration: collect signals per config, merge, attach diagnostics, no side effects.

## Refs: ADR

- ADR-0003
- ADR-0009 (once filed)
- ADR-0006 (GitHub path uses `gh`)

## ADR Impact

`none` if ADR-0009 covers this; else `amend` ADR-0009 with interface summary.

## Acceptance ↔ ADR

- ADR-0003: stateless invocation; no daemon; parallel invocations independent
- ADR-0005: do not read agent-internal phase files

## Definition of Done

- [ ] Interfaces for providers + registry from `reinguard.yaml`
- [ ] Merge strategy for signal namespaces (document collision rules)
- [ ] Unit tests with fake providers

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Two fake providers, disjoint keys | Normal | Merged signal map | — |
| TC-A-01 | Two providers same key | Abnormal | Deterministic error or overlay rule | Document in code + docs |

## Dependencies

- Depends on: #3

## Notes

Default parallelism for collect: parallel by provider; `--serial` at CLI layer (#16).
