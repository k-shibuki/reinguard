## Context

Formalize the hybrid observation model (Go provider interfaces + config-declared signals) and clarify unified priority across mixed rule types in a single rules store.

## Refs: ADR

- ADR-0001, ADR-0002, ADR-0003, ADR-0004, ADR-0008 (context)
- **New: ADR-0009** (observation engine abstraction)
- **Amend: ADR-0004** — unified priority applies across `state` / `route` / `guard` rules in one priority space when using `.reinguard/rules/` with `type` discriminator (per Phase 1 plan)

## ADR Impact

`new` — add `docs/adr/0009-observation-engine-abstraction.md` (filename may follow repo convention).  
`amend` — update `docs/adr/0004-unified-priority-based-state-resolution.md` (or successor) with one short paragraph: single priority ordering across typed rules in shared configuration.

## Acceptance ↔ ADR

- ADR-0009 states: providers are pluggable in Go; which signals each run collects is declared in repo config; observation is side-effect free; aligns with ADR-0003 pull/stateless and ADR-0006 `gh` auth for GitHub.
- ADR-0004 amendment states: no separate merge artifact for “tiers”; same numeric priority semantics for all rule `type` values in Phase 1 layout.

## Definition of Done

- [ ] ADR-0009 merged with Context / Decision / Consequences
- [ ] ADR-0004 updated with explicit “typed rules, one priority space” wording
- [ ] `docs/design.md` or README links to ADR-0009 if needed (no normative duplication)

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | PR with ADR files only | Normal | CI passes; markdown renders | — |
| TC-A-01 | ADR-0009 missing Consequences | Abnormal | Review requests section before merge | — |

## Dependencies

- None (documentation-only). **Process:** close or supersede bootstrap Issue #1 before starting implementation PRs that assume MVP complete.

## Notes

Reference bridle `docs/agent-control/` and `tools/evidence-*.sh` as prior art; reinguard does not copy jq pipelines — document the intentional Go + schema direction.
