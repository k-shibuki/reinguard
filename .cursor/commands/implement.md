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

1. **Read the Issue** (`gh issue view …` above): pull **1–3 concrete search terms** from the **title and body** — subsystem or package names, ADR/issue refs, CLI subcommands, domain nouns, error messages, or phrases from **Touches** / **Definition of Done**. These are not generic words (“fix”, “bug”); `rgd knowledge pack --query` matches **triggers** only (case-insensitive substring), not `description` text — use `description` in the manifest for human context when choosing terms.
1. **Run retrieval** with each term (separate runs or one combined phrase if it reads naturally):

```bash
rgd knowledge pack --query '<term derived from Issue title/body>'
```

Repeat for other salient terms if the first pack is thin or off-topic.

1. Open each returned `entries[].path` under `.reinguard/knowledge/` as needed. Optional full pipeline snapshot:

```bash
rgd context build
```

**Observation** (when local GitHub signals help scope work):

```bash
rgd observe
```

## Act

1. Create feature branch per `.reinguard/policy/commit--format.md` § Branch naming: `<prefix>/<issue-number>-<short-description>` (type prefixes: `.reinguard/labels.yaml` `categories.type` with `commit_prefix: true`).
2. Search the codebase for existing paths, patterns, and dependencies; prioritize Issue **Touches** / **Go checks** sections.
3. **Doc impact**: list candidate updates (`docs/adr/`, `docs/cli.md`, `.reinguard/`). Carry this list forward to commit/PR; align finalized diffs before merge.
4. Implement per Issue **Definition of Done** and **Test plan** (the Issue Test plan states **intent**, not an exhaustive case-ID list — derive concrete Normal / Abnormal / Boundary cases from the diff; include tests in the same deliverable unless the Issue explicitly defers them).
5. Same-kind sweep per coding--standards § Change scope before hand-off.
6. **Preflight** per `coding--preflight.md` before commit/push.
7. **Commit organization**: review the commit history (`git log origin/main..HEAD`) and organize into logical, self-contained commits where needed (interactive rebase, amend, or squash). Each commit should represent one coherent change. This step is the primary location for commit restructuring — `change-inspect` may *recommend* restructuring but does not execute it.

## Output

- Issue: `#<N>` — title (and selection rationale if auto-listed)
- Scope recap: 1–3 bullets
- Files changed: paths
- DoD progress vs Issue checklist
- Doc impact list from step 3
- Preflight result (step 6 passed / exceptions documented)
- Commit status: organized / needs restructuring (carried forward to `change-inspect`)

## Guard

- All work traceable to an Issue (`Refs: #N` in commits per `commit--format.md`)
- **HS-LOCAL-VERIFY**, **HS-NO-SKIP** — enforced via `coding--preflight.md` (Act step 6)
- Prefer **`rgd`** for observation/context/knowledge; use **`gh`** / **`git`** for GitHub/git inspection per `evidence-temporary.mdc`
