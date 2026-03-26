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

### PR body updates (`gh api` and multiline text)

Programmatic create/edit must preserve **real newlines** in the PR body. If GitHub stores the body as one physical line containing the two-character sequence `\` + `n` instead of line breaks, **Gate — PR policy** fails (required sections such as `Summary` are not detected).

- **Preferred**: `gh pr create --body-file …` / `gh pr edit <N> --body-file …`, after `tools/check-pr-policy.sh` passes for create.
- **If using `gh api`**: send JSON with `--input` and a `body` value that is an actual multiline string — for example `jq -n --rawfile b path/to/body.md '{body: $b}' > patch.json` then `gh api repos/<owner>/<repo>/pulls/<N> -X PATCH --input patch.json`. Do **not** rely on `-f body=…` with multiline shell values or on piping `jq -Rs` into form-style fields; form encoding can turn newlines into literal `\n` and break section checks.

## Exceptions

- Use repository **labels** (`no-issue` / `hotfix`) and a `## Exception` section in the PR when bypassing the normal Issue flow (document justification).

## Related

- `.reinguard/policy/commit--format.md` — commits and branch names
- `.reinguard/policy/catalog.yaml` — policy index
