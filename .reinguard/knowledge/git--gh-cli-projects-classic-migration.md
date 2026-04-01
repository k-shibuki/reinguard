---
id: git-gh-cli-projects-classic-migration
description: Workaround for gh CLI PR edit failures caused by Projects Classic field deprecation
triggers:
  - gh pr edit
  - Projects Classic
  - projectCards
  - GraphQL deprecation
  - pr body update
when:
  op: exists
  path: github.pull_requests.pr_exists_for_branch
---

# gh CLI and Projects Classic migration

## Symptom

`gh pr edit` may fail with an error similar to:

`Projects (classic) is being deprecated ... (repository.pullRequest.projectCards)`.

## Cause

Some `gh` versions request deprecated Projects Classic GraphQL fields during PR
edit operations. Repositories affected by this migration can see hard failures
even for simple body updates.

## Operational rule

Use REST API updates for PR body and labels:

- body update: `gh api repos/<owner>/<repo>/pulls/<N> -X PATCH --input <json>`
- label update: `gh api repos/<owner>/<repo>/issues/<N>/labels ...`

When updating body text, send JSON with a real multiline string value to avoid
literal `\n` formatting corruption.
