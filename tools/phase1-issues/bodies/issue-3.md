## Context

Establish Go testing conventions and Issue quality bars so Phase 1 work stays traceable to ADRs (Refs, ADR Impact, Acceptance line).

## Refs: ADR

- ADR-0008 (contracts tested via schema/tests)
- `none` for pure process — cite coding-policy / workflow-policy

## ADR Impact

`none` (adds `.cursor` rules and templates only).

## Acceptance ↔ ADR

- Contributors can write Issues with required sections; tests use GWT comments and table-driven style suitable for `go test`.

## Definition of Done

- [ ] `.cursor/rules/test-strategy.mdc` — Go-focused (GWT, branches, failure cases, coverage target reference ≥80% module-wide per plan)
- [ ] `.cursor/knowledge/test--given-when-then.md` — Go `testing` example
- [ ] `.github/ISSUE_TEMPLATE/` updated: Title guide, `Refs: ADR-…`, `ADR Impact`, `Acceptance ↔ ADR` examples
- [ ] `.cursor/rules/workflow-policy.mdc` — Issue title patterns and required Issue sections (bridle-inspired)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | New test file following template | Normal | `go test ./...` passes; GWT comments present | — |
| TC-A-01 | Test with only happy path | Abnormal | Review rejects per strategy (failure cases required) | Policy |

## Dependencies

- None

## Notes

English only for persisted artifacts per `coding-policy.mdc`.
