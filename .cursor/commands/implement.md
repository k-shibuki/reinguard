# implement

## Context

- `.reinguard/policy/coding--preflight.md` — **Preflight verification** (HS-LOCAL-VERIFY, defensive checks, test design, self-review)
- `.reinguard/policy/coding--standards.md` — **Change scope** (same-kind sweep across code, `.reinguard/`, `.cursor/`)
- `.reinguard/policy/commit--format.md` — branch naming; Cursor: `commit-format.mdc`
- `.cursor/rules/test-strategy.mdc` — GWT, table tests (points at `.reinguard/knowledge/testing--*.md`)

**Already in context**: `reinguard-bridge.mdc` (HS-*, catalogs), `workflow-policy.mdc` (Issue-driven work, command separation).

**Issue metadata** (from repo root):

```bash
gh issue view <N> --json title,body,labels,state
```

If no Issue number: list candidates and wait for approval:

```bash
gh issue list --state open --limit 30 --json number,title,labels
```

**Knowledge discovery** (substrate):

```bash
rgd knowledge pack --query '<keyword from Issue>'
```

Open each `entries[].path` under `.reinguard/knowledge/` as needed. Optional full pipeline snapshot:

```bash
rgd context build
```

**Observation** (when local GitHub signals help scope work):

```bash
rgd observe
```

## Act

1. Create feature branch per `.reinguard/policy/commit--format.md` § Branch naming: `<prefix>/<issue-number>-<short-description>` (types: `tools/commit-types.txt`).
2. Search the codebase for existing paths, patterns, and dependencies; prioritize Issue **Touches** / **Go checks** sections.
3. **Doc impact**: list candidate updates (`docs/adr/`, `docs/cli.md`, `.reinguard/`). Carry this list forward to commit/PR; align finalized diffs before merge.
4. Implement per Issue **Definition of Done** and **Test plan**; include tests in the same deliverable unless the Issue explicitly defers them.
5. Same-kind sweep per coding--standards § Change scope before hand-off.
6. **Preflight** per `coding--preflight.md` before commit/push.

## Output

- Issue: `#<N>` — title (and selection rationale if auto-listed)
- Scope recap: 1–3 bullets
- Files changed: paths
- DoD progress vs Issue checklist
- Doc impact list from step 3
- Preflight result (step 6 passed / exceptions documented)

## Guard

- All work traceable to an Issue (`Refs: #N` in commits per `commit--format.md`)
- **HS-LOCAL-VERIFY**, **HS-NO-SKIP** — enforced via `coding--preflight.md` (Act step 6)
- Prefer **`rgd`** for observation/context/knowledge; use **`gh`** / **`git`** for GitHub/git inspection per `evidence-temporary.mdc`
