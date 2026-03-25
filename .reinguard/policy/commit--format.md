---
id: commit-format
description: Conventional Commits, issue reference in body, atomic commits, and branch naming
triggers:
  - commit message
  - Conventional Commits
  - Refs body
  - branch naming
---

# Git commit message conventions

Mandatory requirements for commit messages. The Cursor Adapter rule `commit-format.mdc` points here as SSOT.

## Positioning

- Format template: `.github/gitmessage` (install via `git config commit.template .github/gitmessage`).
- Machine validation: `commit-msg` stage in `.pre-commit-config.yaml` (runs `tools/check-commit-msg.sh`).

## Prefix (Type)

Canonical type list: `tools/commit-types.txt` (shared with `check-commit-msg.sh`).

Scope is optional: `<type>(<scope>): <summary>`. See [Conventional Commits](https://www.conventionalcommits.org/).

### fix vs hotfix

They differ in **urgency**, not kind — both go through PR.

**PR title handling**: `hotfix` is a valid commit type but NOT a PR title prefix — use `fix:` as the title prefix and add the `hotfix` exception label when applicable.

## Message generation principles

- Always review uncommitted diffs (`git diff` / `git diff --cached`) before composing.
- Summarize actual changes, not Issue titles or branch names.
- AI and automation must follow the same rule: diffs as input.

## Summary (first line)

- English, imperative mood, no trailing period (not enforced by the hook;
  remains a code-review expectation).
- Length: aim ~50 characters for readability. The `commit-msg` hook warns
  above 72 characters and errors above 120 (`tools/check-commit-msg.sh`).

## Body (bullet points)

- List changes as `- ` bullet points in English.
- Include impact scope, migration steps, risks, rollback as applicable.

## Issue reference (required in body)

- **`Refs: #N`** (required): Every commit must reference the tracking Issue
  somewhere in the message body (the hook scans the full body, not only Git
  trailer lines).
- **`Closes: #N`**: Use in the final commit or PR body to auto-close the Issue.
- **`BREAKING CHANGE:`**: Required for backward-incompatible changes (or use `!` notation: `fix!:`).
- **Exception**: `hotfix`/`docs` may omit `Refs:` but must justify in body.

## Atomic commits

Split changes into **logically cohesive, minimal commits** when beneficial:

| Split when | Keep together when |
|------------|-------------------|
| Multiple unrelated fixes in one session | Tightly coupled changes that break if separated |
| Refactor + feature in same diff | Single logical change across multiple files |
| Docs update independent of code change | Code + its corresponding test |

Each commit should be independently meaningful and pass tests.
Prefer 2-4 focused commits over 1 large commit or 10+ micro-commits.
When in doubt, fewer commits is safer.

## Prohibitions

- Non-English summary
- Ambiguous summaries ("update", "fix bug")
- Body without bullet points
- Commits that only disable checks without improvement
- Using interactive shell (`git commit` without `-m`)

## Branch naming convention

Feature branches include the Issue number for traceability:

```
<prefix>/<issue-number>-<short-description>
```

Examples: `feat/42-cli-version`, `fix/15-config-validate`, `ci/50-golangci`

**Exception**: `hotfix` branches may omit the Issue number: `hotfix/<short-description>`.

## Related

- `.reinguard/policy/workflow--pr-discipline.md` — PR and Issue workflow
- `.reinguard/policy/catalog.yaml` — policy index
