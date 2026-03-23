## Context

Define `.reinguard/` layout, embed JSON Schemas, and bind **one `schema_version`** across config, observation document, intermediate evaluation payloads, and operational context (ADR-0008 strict reading).

## Refs: ADR

- ADR-0002 (rules shape implied)
- ADR-0004, ADR-0007 (outcomes in JSON)
- ADR-0008

## ADR Impact

`none` unless schema decisions require ADR-0008 clarifications (then `amend` in same PR with justification).

## Acceptance ↔ ADR

- ADR-0008: synchronized semver; schemas ship in binary; validate/export path exists for consumers.

## Definition of Done

- [ ] Document structure under `.reinguard/`: `reinguard.yaml` (required: `schema_version`, `default_branch`, providers, defaults), `rules/*.yaml`, `knowledge/*.json` manifest location
- [ ] JSON Schema files in `pkg/schema/` (embed) for: root config, rules entry, knowledge manifest, observation document, operational context (placeholders acceptable only if versioned and tested)
- [ ] `schema export` lists new artifacts (follow-up in #22 for strict validate)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Valid minimal `reinguard.yaml` + empty `rules/` | Equivalence | Passes schema validation | Provide fixture path |
| TC-A-01 | Missing `default_branch` | Boundary | Validation error naming field | — |
| TC-A-02 | Unknown top-level key with unsafe interpretation | Abnormal | Fail or warn per ADR-0008 policy (document choice) | — |

## Dependencies

- Depends on: #3 (ADR-0009 merged so provider/signal vocabulary is stable)

## Notes

Semver: additive=PATCH, semantic change=MINOR, breaking=MAJOR per plan.
