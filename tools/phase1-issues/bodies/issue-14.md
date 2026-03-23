## Context

Authoritative CLI and machine I/O contract for `rgd` (subcommands, flags, exit codes, stdout/stderr, observation JSON shape). Same PR adds ADR-0008 pointer.

## Refs: ADR

- ADR-0003, ADR-0007, ADR-0008

## ADR Impact

`amend` — add to `docs/adr/0008-schema-versioning.md` (or successor): **SSOT for CLI and machine-readable I/O is `docs/cli.md`**, not duplicated in ADR body.

## Acceptance ↔ ADR

- ADR-0008: contracts versioned; CLI JSON fields align with embedded schemas from #5
- ADR-0007: non-resolved outcomes in JSON; default exit 0; `--fail-on-non-resolved` documented

## Definition of Done

- [ ] New file `docs/cli.md` with: full command tree; provider ID ↔ CLI mapping; global flags (`--config-dir`, `--cwd`, `-o`, `--serial`, `--fail-on-non-resolved`); exit code table; stdout vs stderr rules; observation document field list (reinguard-native, not bridle `_meta` clone); stdin / `--observation-file` contract for `state eval` pipeline; guard-by-guard required flags table (stub rows OK if guards unimplemented)
- [ ] ADR-0008 amended in **this** PR
- [ ] README links to `docs/cli.md` only (no duplicate normative tables)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | Each top-level command has Examples section | Normal | Review checklist passes | Manual |
| TC-A-01 | README duplicates exit code table | Abnormal | PR rejected — single SSOT | Process |

## Dependencies

- Depends on: #5 (schema + naming stable)

## Notes

English only. Phase 1: no CLI aliases (`pull-requests` not `pr`).
