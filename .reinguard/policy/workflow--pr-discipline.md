---
id: workflow-pr-discipline
description: Issue-driven work, PR discipline, branch naming, and exception policy
triggers:
  - workflow
  - PR discipline
  - branch naming
  - Closes issue
---

# Workflow and PR discipline

Repository-wide workflow rules. The Cursor Adapter rule `workflow-policy.mdc` points here as SSOT.

## Issue-driven work

- Prefer **one GitHub Issue per PR** (1 Issue ≈ 1 PR) for implementation work.
- Commit messages follow `.reinguard/policy/commit--format.md` (Conventional Commits + `Refs: #<issue>` in the message body).
- PR body includes `Closes #<issue>` (or `Fixes`) when applicable.

## Issue title patterns

- Use **Conventional Commits**-style titles where practical:
  `feat(scope): …`, `fix(scope): …`, `docs(adr): …`, `chore(test): …`.
- Title should be imperative and English (same language rule as commits).

## Recommended Issue sections

New implementation Issues SHOULD include:

1. **Context** — problem and constraints
2. **Refs: ADR** — which ADRs apply, or `none`
3. **ADR Impact** — `none`, `new`, or `amend` with target file
4. **Acceptance ↔ ADR** — how completion is verified against ADRs
5. **Definition of Done** — checklist
6. **Test plan** — concrete cases (table encouraged); not only "tests pass"

Use `.github/ISSUE_TEMPLATE/task.md` as the starting point.

## Branch naming

**SSOT**: `.reinguard/policy/commit--format.md` § Branch Naming Convention.

- Pattern: `<prefix>/<issue-number>-<short-description>` (examples: `feat/42-cli-version`, `fix/15-config-validate`, `docs/3-adr-0009`).
- **Exception**: `hotfix/<short-description>` may omit the Issue number when it meets the hotfix criteria in `.reinguard/policy/commit--format.md` and the **Exceptions** section below.

## PR discipline

- Fill **every** section of `.github/PULL_REQUEST_TEMPLATE.md` (Summary, DoD,
  Test plan, Linked issues, Exception if applicable).
- Base branch is always **`main`**.

## Exceptions

- Use repository **labels** (`no-issue` / `hotfix`) and a `## Exception` section in the PR when bypassing the normal Issue flow (document justification).

## Related

- `.reinguard/policy/commit--format.md` — commits and branch names
- `.reinguard/policy/catalog.yaml` — policy index
