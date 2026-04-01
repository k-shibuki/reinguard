---
id: review-self-inspection
description: Whole-change self-inspection dimensions for PR readiness — complements line-level preflight
triggers:
  - self-inspection
  - change-inspect
  - self-review
  - inspection dimensions
  - whole-change review
  - pre-PR inspection
---

# Self-inspection dimensions

Whole-change inspection criteria applied **after** `coding--preflight.md`
(line/file-level) and **before** PR creation. The Adapter command
`change-inspect` references this document as SSOT for what to check.

## Relationship to preflight

`coding--preflight.md` covers mechanical, line-level checks (nil guards,
silent ignores, setup error handling, `go vet`, `npx --yes markdownlint-cli2@latest`).
Self-inspection operates at the **whole-change** level: coherence,
alignment, and coverage that only become visible when reviewing the
complete diff against the Issue and architecture.

Preflight is a prerequisite; self-inspection is not a re-run of
preflight but a meta-verification that preflight obligations were met
plus higher-order checks.

## Dimensions (7)

### 1. Issue alignment

Verify the diff satisfies the Issue's **Definition of Done** and
**Test plan** items. Flag:

- DoD items not addressed by the diff
- Diff changes not traceable to any DoD item (scope creep)
- Test plan steps that cannot be executed with the current change

### 2. ADR compliance

For each ADR listed in the Issue's **Refs: ADR** (or discovered via
`rgd context build` / `rgd knowledge pack`), confirm the implementation respects the
**Decision** and **Consequences** sections. Flag:

- Contradictions (e.g. orchestration logic in the substrate)
- Missing consequences handling (e.g. new boundary not tested)

### 3. Defensive implementation (meta-verification)

Confirm `coding--preflight.md` § Defensive implementation checks were
applied across the diff:

- Every new exported function receiving a pointer/interface has a nil
  check at entry
- No silent fallthrough on config value type mismatches
- Blank/duplicate ID rejection in collection entries

This is a **diff-level scan**, not re-running tools — verify the
patterns are present in changed code.

### 4. Test adequacy

For each **behavior change in the diff** (not the Issue checklist alone),
confirm:

- Coverage across **Normal / Abnormal / Boundary** perspectives where
  meaningful (see `testing--strategy.md` § Perspectives)
- Table-driven format when two or more scenarios test the same function
  (see `testing--strategy.md` § Table-driven tests)
- **GWT**: non-trivial tests **must** have a function-level summary; not
  required inside table-driven loop bodies (see `testing--given-when-then.md`)
- No `_ = <fallible call>` in test setup (immediate fail-fast with
  `t.Fatal`)

### 5. Same-kind sweep

Verify `coding--standards.md` § Change scope was completed:

- Search the diff for parallel wording, config keys, or call sites in
  **code**, **`.reinguard/`**, and **`.cursor/`**
- Any intentional gap has an explicit rationale (PR body or inline
  comment), not silent omission

### 6. PR template substance

Check each section of `.github/PULL_REQUEST_TEMPLATE.md`:

- **Summary** — describes *why*, not just *what*
- **Traceability** — `Closes #N` present and correct
- **Definition of Done** — checklist items checked or explained
- **Test plan** — concrete steps (not "tests pass")
- **Risk / Impact** — non-placeholder content
- **Rollback Plan** — non-placeholder or justified "N/A"

### 7. Documentation impact

Verify the doc impact list from `implement` (Act step 3) is reflected:

- ADR amendments committed if behavior changed
- `docs/cli.md` updated if CLI surface changed
- `.reinguard/` knowledge or policy updated if operational meaning
  changed
- Intentional deferrals documented in the PR body

## Severity guidance

When reporting findings, use:

- **Blocking** — must fix before external review (Issue misalignment,
  ADR violation, missing tests for new behavior, broken defensive
  pattern)
- **Non-blocking** — should fix but may proceed (template wording,
  minor doc gap, sweep miss with low risk)

## Related

- `.reinguard/policy/coding--preflight.md` — line-level preflight
  (prerequisite)
- `.reinguard/policy/coding--standards.md` — Change scope, language
  policy
- `.reinguard/knowledge/testing--strategy.md` — test perspectives,
  table-driven
- `.reinguard/knowledge/testing--given-when-then.md` — GWT format
- `.reinguard/procedure/change-inspect.md` — procedure that executes this
  inspection (pre-PR; dimension 6 is deferred to `pr-create`; enter via
  `.cursor/commands/rgd-next.md` + `rgd context build`)
