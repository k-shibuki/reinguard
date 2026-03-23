## Context

Implement `rgd observe` per `docs/cli.md`: all providers, `observe git`, `observe github`, `observe github <issues|pull-requests|ci|reviews>`. Default parallel collect; `--serial`. Required aggregate `observe github`.

## Refs: ADR

- ADR-0003, ADR-0006, ADR-0009

## ADR Impact

`none`

## Acceptance ↔ ADR

- ADR-0003: stateless; JSON to stdout; progress to stderr only

## Definition of Done

- [ ] All subcommands and flags from `docs/cli.md`
- [ ] 429 limited retry with backoff
- [ ] CI smoke commands listed in `docs/cli.md` match `.github/workflows/ci.yaml` updates (coordinate #23)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Mock providers, `observe` | Normal | Valid JSON on stdout | — |
| TC-B-01 | One provider fails non-fatal | Boundary | degraded in JSON, exit 0 per plan | — |
| TC-A-01 | Unknown subcommand | Abnormal | exit non-zero, stderr message | — |

## Dependencies

- Depends on: #6, #9, #10, #11, #12, #13, #14, #15

## Notes

Never print raw tokens.
