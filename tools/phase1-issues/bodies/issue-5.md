## Context

Load and validate repository configuration from disk into typed Go structs for CLI and engines.

## Refs: ADR

- ADR-0008

## ADR Impact

`none`

## Acceptance ↔ ADR

- ADR-0008: loading uses embedded schemas; version field governs contract set.

## Definition of Done

- [ ] Package `internal/config/` (or agreed path) loads `reinguard.yaml` + glob `rules/*.yaml`
- [ ] Validation against embedded JSON Schema before use
- [ ] Clear errors for authors (path + schema keyword)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Fixture `.reinguard/` valid | Normal | `Load(dir)` returns non-nil config | Use testdata |
| TC-A-01 | Broken YAML in one rules file | Abnormal | Error references file name and line if available | — |
| TC-B-01 | Empty `rules/` directory | Boundary | Valid if schema allows; else documented error | Match schema |

## Dependencies

- Depends on: #5

## Notes

Respect `--config-dir` / `REINGUARD_CONFIG_DIR` from existing CLI.
