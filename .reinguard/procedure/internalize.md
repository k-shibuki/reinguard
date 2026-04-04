---
id: procedure-internalize
purpose: Feed review and inspection findings into Semantics (knowledge/policy) without duplicating SSOT in Adapter.
applies_to:
  state_ids: []
  route_ids: []
reads:
  - ../policy/coding--preflight.md
  - ../policy/catalog.yaml
  - ../knowledge/manifest.json
sense:
  - gh api / gh pr view
act:
  - Collect, classify root causes, map to documents, apply minimal diffs, validate.
output:
  - Root-cause table, paths updated, validate results, deferred gaps.
done_when: SSOT updated; `rgd config validate` and index steps done when knowledge changed.
escalate_when: Findings require normative policy change beyond maintainer scope.
---

# internalize

Feed review findings back into the Semantics layer so future
implementations avoid the same class of review comments.

## Context

- [`../policy/coding--preflight.md`](../policy/coding--preflight.md) — **Preflight verification** (HS-LOCAL-VERIFY); SSOT for commands — do not duplicate its checklists here
- [`../policy/catalog.yaml`](../policy/catalog.yaml) / [`../knowledge/manifest.json`](../knowledge/manifest.json) — indexes
- ADR-0001 § Responsibility layers — Adapter must not duplicate Semantics content

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow & commit policy.

**Review input** (from repo root):

```bash
gh api repos/{owner}/{repo}/pulls/<N>/comments
gh pr view <N> --json reviews,comments
```

**Disposition history**: from `review-address` output or PR thread replies.

**Self-inspection findings**: from `change-inspect` output (dimension-level findings that were fixed before PR creation).

## Act

1. **Collect**: gather all review comments, disposition replies, and `change-inspect` findings for the PR.
2. **Classify root causes** into categories:
   - Defensive implementation gap (nil, silent ignore, blank/duplicate id)
   - Test design gap (missing perspective, wrong format, setup error ignored)
   - Procedure gap (verification step missing or mis-placed in command)
   - Wording ambiguity (knowledge/policy text interpreted inconsistently)
3. **Map to documents**: for each root cause, identify the existing knowledge, policy, or procedure file that should have prevented it. If no file covers the gap, note it as a candidate for a new document.
4. **Propose changes**: for each mapped document, draft a minimal diff. For new documents, draft front matter and outline. Prefer extending existing documents over creating new ones.
5. **SSOT check**: verify that no Adapter file (`.cursor/`) duplicates Semantics content (`.reinguard/`). Adapter files should reference, not restate.
6. **Apply and validate**:
   - Edit knowledge / policy / procedure files per the diffs.
   - `rgd knowledge index` (when knowledge files changed).
   - **Preflight** per `coding--preflight.md` (covers `rgd config validate`, `bash .reinguard/scripts/with-repo-local-state.sh -- pre-commit run markdownlint-cli2 --all-files`, and Go checks when those paths change).

## Output

- Root-cause classification table (category, file, gap description)
- Documents updated or created (paths)
- `rgd config validate` result
- Remaining gaps deferred (with rationale)

## Guard

- Do not duplicate Semantics content in Adapter (ADR-0001 § Adapter principle)
- Do not automate judgment — `rgd` computes, agents reason (ADR-0001 § Decision)
- **HS-LOCAL-VERIFY**, **HS-NO-SKIP** — enforced via `coding--preflight.md` (Act step 6)
