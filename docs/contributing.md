# Contributing to reinguard

## Workflow

- **Issue-driven**: Prefer one GitHub Issue per implementation PR; the PR body must include `Closes #N` (or `Fixes` / `Resolves`), unless you use an exception label and fill `## Exception` per [.github/PULL_REQUEST_TEMPLATE.md](../.github/PULL_REQUEST_TEMPLATE.md).
- **Commits**: Conventional Commits and `Refs: #N` in the message body — see [.cursor/rules/commit-format.mdc](../.cursor/rules/commit-format.mdc).
- **Commands**: Thin procedures live under [.cursor/commands/](../.cursor/commands/) (`pr-create`, `review-fix`, `pr-merge`).

## CI and PR policy

- Workflow [.github/workflows/ci.yaml](../.github/workflows/ci.yaml) runs `go-ci`, and on pull requests calls [.github/workflows/pr-policy.yaml](../.github/workflows/pr-policy.yaml) as reusable workflow job `check-policy`.
- Job **`ci-pass`** aggregates `go-ci` and `check-policy` on PRs. Configure **branch protection** on `main` to require this single check:

### Branch protection (maintainers)

In GitHub: **Settings → Branches → Branch protection rule** for `main`:

1. **Require a pull request before merging** (recommended).
2. **Require status checks to pass**: enable **Require status checks to pass before merging** and add check **`ci-pass`** (from workflow *CI*).
3. **Require conversation resolution before merging**: **enable** — all review threads must be resolved before merge (aligns with `HS-REVIEW-RESOLVE` in [.cursor/rules/agent-safety.mdc](../.cursor/rules/agent-safety.mdc)).
4. Do not rely on bypassing checks (`gh pr merge --admin` is prohibited for agents).

### Merge strategy

- Prefer **`gh pr merge <N> --squash`** on `main` for a linear history, unless a specific PR or release note calls for a merge commit.

Observation until `rgd observe` exists: use `gh` / `git` for read-only inspection per [.cursor/rules/evidence-temporary.mdc](../.cursor/rules/evidence-temporary.mdc).

## Labels

PRs must have exactly one **type** label (`feat`, `fix`, `docs`, …). Exception PRs also need `hotfix` or `no-issue`.

From the repository root (once per org/repo):

```bash
go run ./cmd/rgd ensure-labels
```

With `rgd` on your `PATH` (for example `go install ./cmd/rgd`), run `rgd ensure-labels` instead.

### Open PRs after policy changes

After merging workflow updates to `main`, feature branches should **merge or rebase `main`** so they pick up `ci.yaml` / `pr-policy.yaml`. To add missing template sections and type labels on already-open PRs:

```bash
go run ./cmd/rgd backfill-pr-policy
```

The command shells out to `gh api` to update PR bodies and labels (some `gh` versions fail on `gh pr edit` due to deprecated Classic Projects GraphQL). With `rgd` on your `PATH` (for example `go install ./cmd/rgd`), run `rgd backfill-pr-policy` instead.

## Review threads and merge

Before merge: CI green (`ci-pass`), PR policy green, and **all review conversations resolved**. For each thread, leave a short **disposition** (e.g. Fixed / By design / False positive / Acknowledged) before resolving — see [AGENTS.md](../AGENTS.md) and the bridle consensus reference linked there.

Do **not** enable **auto-merge** while a bot review is still pending or threads are unresolved.

## Local verification (Go)

```bash
go test ./...
go vet ./...
golangci-lint run
```

Optional: `pre-commit install --hook-type commit-msg` and `git config commit.template .gitmessage`.
