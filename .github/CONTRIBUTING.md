# Contributing to reinguard (rgd)

Thank you for helping improve **reinguard**. The shipped CLI binary is **`rgd`**
(`go build -o rgd ./cmd/rgd`). **This file is the single contributor guide** for this
repository; normative CLI behavior is **[`docs/cli.md`](../docs/cli.md)** (see ADR-0008).

Review routing for touched areas is defined in **[`CODEOWNERS`](CODEOWNERS)**.

## Prerequisites

- **Go**: match `go.mod` / CI (toolchain **1.26.1** as of this writing).
- **golangci-lint**: optional locally; CI runs it on every PR.
- **`yq`**: **[mikefarah/yq](https://github.com/mikefarah/yq) v4** — required for local runs of `.reinguard/scripts/check-commit-msg.sh`, `check-pr-policy.sh`, `check-issue-policy.sh`, and for `.reinguard/scripts/sync-issue-templates.sh` (CI installs a pinned binary in workflows).
- **`gh`**: required only when using commands that call the GitHub API (e.g.
  `rgd observe github`, or live observation in `rgd state eval` / `rgd context build`
  without `--observation-file`). See ADR-0006.

## Quick setup

```bash
git clone https://github.com/k-shibuki/reinguard.git
cd reinguard
go build -o rgd ./cmd/rgd
./rgd version
```

Optional: the repo **[`Makefile`](../Makefile)** provides `make test`, `make check`
(fmt, vet, lint, test — lint is `golangci-lint`), `make coverage`, `make build`, etc.
It is **not** normative — CI and the shell commands below are the source of truth.

Optional commit message template (see [`.reinguard/policy/commit--format.md`](../.reinguard/policy/commit--format.md); Cursor: [`.cursor/rules/commit-format.mdc`](../.cursor/rules/commit-format.mdc)):

```bash
git config commit.template .github/gitmessage
```

## Checks before you push

Same gates as CI (see also [`.reinguard/policy/safety--agent-invariants.md`](../.reinguard/policy/safety--agent-invariants.md); Cursor: [`.cursor/rules/reinguard-bridge.mdc`](../.cursor/rules/reinguard-bridge.mdc) § Always-active policy):

```bash
go test ./... -race -count=1
go vet ./...
golangci-lint run --timeout=5m ./...
npx --yes markdownlint-cli2@latest '**/*.md'
```

**Coverage** (module-wide threshold **80%**, same as CI):

```bash
go test ./... -race -coverpkg=./... -coverprofile=coverage.out -count=1
bash .reinguard/scripts/check-coverage-threshold.sh 80 coverage.out
```

Optional: `pre-commit install --hook-type commit-msg` and `pre-commit install` (see [`.pre-commit-config.yaml`](../.pre-commit-config.yaml)).

## rgd, schemas, and layout

| Area | Role |
|------|------|
| [`cmd/rgd/`](../cmd/rgd) | `main`, thin entry — delegates to `internal/rgdcli` |
| [`internal/rgdcli/`](../internal/rgdcli) | Command tree, flags, JSON output |
| [`pkg/schema/`](../pkg/schema) | Embedded JSON Schemas; `rgd schema export` writes them to disk |
| [`internal/config/`](../internal/config) | Loads `.reinguard` / `control/{states,routes,guards}/*.yaml`; `rgd config validate` |
| [`internal/observe/`](../internal/observe) | Observation providers (`git`, `github`, …) |

- Command tree, flags, stdout/stderr, exit codes: **`docs/cli.md`**.
- Export schemas for inspection: `./rgd schema export --dir /tmp/rgd-schemas` (after `go build`).

## Workflow

- **Issue-driven**: Prefer one GitHub Issue per implementation PR; the PR body must include `Closes #N` (or `Fixes` / `Resolves`), unless you use an exception label and fill `## Exception` per [`.github/PULL_REQUEST_TEMPLATE.md`](PULL_REQUEST_TEMPLATE.md).
- **Commits**: Conventional Commits and `Refs: #N` in the message body — see [`.reinguard/policy/commit--format.md`](../.reinguard/policy/commit--format.md).
- **Commands**: Workflow procedures live under [`.reinguard/procedure/`](../.reinguard/procedure/). Cursor: [`rgd-next`](../.cursor/commands/rgd-next.md) (`rgd context build` → mapped procedure); [`cursor-plan`](../.cursor/commands/cursor-plan.md) (design interrogation + `CreatePlan` or Issue creation via Phase 3B).

## CI and PR policy

- Workflow [`ci.yaml`](workflows/ci.yaml) runs jobs in dependency order: **`gate-policy`** (always; PR body runs [`.github/scripts/pr-policy-check.js`](../.github/scripts/pr-policy-check.js), non-PR events no-op), **`lint-markdown`**, **`lint-go`** (after policy), **`test-go`** (after lint-go), **`dogfood-rgd-cli`** / **`dogfood-rgd-github`** (after `test-go`). [`pr-policy.yaml`](workflows/pr-policy.yaml) remains a **reusable** workflow with the same script for callers outside this file.
- Job **`ci-pass`** aggregates all of the above. Configure **branch protection** on `main` to require this single check.

### Local PR policy pre-flight (optional)

Before `gh pr create`, you can run the same checks locally (title, body sections, type label, base branch):

```bash
bash .reinguard/scripts/check-pr-policy.sh \
  --title "<type>(<scope>): <summary>" \
  --body-file /path/to/pr-body.md \
  --label <type> \
  --base main
```

**Scope:** [`.reinguard/scripts/check-pr-policy.sh`](../.reinguard/scripts/check-pr-policy.sh) is **repository tooling for developing reinguard** (this GitHub repo’s PR template and labels). It is **not** part of the shipped **`rgd` CLI** and is not a general product feature. That boundary matches **[ADR-0001](../docs/adr/0001-system-positioning.md)**: reinguard does not become the semantic authority for repository policy; this repo keeps PR discipline in CI plus optional local helpers like this script.

Agents: see also [`.reinguard/procedure/pr-create.md`](../.reinguard/procedure/pr-create.md).

### Branch protection (maintainers)

In GitHub: **Settings → Branches → Branch protection rule** for `main`:

1. **Require a pull request before merging** (recommended).
2. **Require status checks to pass**: enable **Require status checks to pass before merging** and add check **`ci-pass`** (from workflow *CI*).
3. **Require conversation resolution before merging**: **enable** — all review threads must be resolved before merge (aligns with `HS-REVIEW-RESOLVE` in [`.reinguard/policy/safety--agent-invariants.md`](../.reinguard/policy/safety--agent-invariants.md)).
4. Do not rely on bypassing checks (`gh pr merge --admin` is prohibited for agents).

### Merge strategy

- Prefer **`gh pr merge <N> --squash`** on `main` for a linear history, unless a specific PR or release note calls for a merge commit.

Observation until `rgd observe` exists: use `gh` / `git` for read-only inspection per [`.cursor/rules/evidence-temporary.mdc`](../.cursor/rules/evidence-temporary.mdc).

## Labels

**SSOT:** [`.reinguard/labels.yaml`](../.reinguard/labels.yaml) — type / exception / scope labels, colors, and `commit_prefix` (commit types for hooks).

PRs must have exactly one **type** label (`feat`, `fix`, `docs`, …). Exception PRs also need `hotfix` or `no-issue`.

From the repository root (once per org/repo):

```bash
go run ./cmd/rgd ensure-labels
go run ./cmd/rgd labels sync
```

With `rgd` on your `PATH` (for example `go install ./cmd/rgd`), run `rgd ensure-labels` / `rgd labels sync` instead.

**Issue bodies** (agents): validate before `gh issue create` with:

```bash
bash .reinguard/scripts/check-issue-policy.sh \
  --title "feat(scope): summary" \
  --body-file /path/to/issue-body.md \
  --label feat \
  --template task
```

Regenerate the Task Issue Form **Type** dropdown after editing `labels.yaml`:

```bash
bash .reinguard/scripts/sync-issue-templates.sh
```

### Open PRs after policy changes

After merging workflow updates to `main`, feature branches should **merge or rebase `main`** so they pick up `ci.yaml` / `pr-policy.yaml`. To add missing template sections and type labels on already-open PRs:

```bash
go run ./cmd/rgd backfill-pr-policy
```

The command shells out to `gh api` to update PR bodies and labels (some `gh` versions fail on `gh pr edit` due to deprecated Classic Projects GraphQL). With `rgd` on your `PATH` (for example `go install ./cmd/rgd`), run `rgd backfill-pr-policy` instead.

## Review threads and merge

Before merge: CI green (`ci-pass`), PR policy green, and **all review conversations resolved**. For each thread, leave a short **disposition** (e.g. Fixed / By design / False positive / Acknowledged) before resolving — see [`AGENTS.md`](../AGENTS.md) and [`.reinguard/policy/review--consensus-protocol.md`](../.reinguard/policy/review--consensus-protocol.md).

Do **not** enable **auto-merge** while a bot review is still pending or threads are unresolved.

## Continuous integration (job reference)

[`ci.yaml`](workflows/ci.yaml) orders jobs so **PR template failures skip Go work**, **Go lint failures skip tests**, and **fork pull requests** skip GitHub API dogfood. Manual runs: **`workflow_dispatch`**.

**DAG (summary):** `gate-policy` → `lint-go` → `test-go` → `dogfood-rgd-cli` / `dogfood-rgd-github`; `lint-markdown` runs in parallel with `gate-policy` (no Go dependency).

| Job ID | `name` (UI) | Runs on | Purpose |
|--------|-------------|---------|---------|
| `gate-policy` | Gate — PR policy | All events | PR: template/labels/title/base checks; push: no-op success |
| `lint-markdown` | Lint — Markdown | All events | `markdownlint-cli2` |
| `lint-go` | Lint — Go | After `gate-policy` | `go mod`, `golangci-lint`, `go vet` |
| `test-go` | Test — Go | After `lint-go` | `go test -race`, coverage gate, `rgd` smoke |
| `dogfood-rgd-cli` | Dogfood — rgd (CLI) | After `test-go` | `config validate` + `observe git` |
| `dogfood-rgd-github` | Dogfood — rgd (GitHub) | After `test-go`; **not** fork PRs | `observe github` with token |
| `ci-pass` | Gate — Pass | Always (aggregate) | Fails if any required job failed (`dogfood-rgd-github` may be `skipped`) |

**Required check for merge:** aggregate job **`ci-pass`** only (see [Branch protection](#branch-protection-maintainers) above).

## License

By contributing, you agree that your contributions are licensed under the same
terms as the project (Apache License 2.0 — see [`LICENSE`](../LICENSE)).
