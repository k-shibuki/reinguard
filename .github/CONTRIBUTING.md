# Contributing to reinguard (rgd)

Thank you for helping improve **reinguard**. The shipped CLI binary is **`rgd`**
(`go build -o rgd ./cmd/rgd`). This file is the human-facing contributor guide;
normative CLI behavior is **[`docs/cli.md`](../docs/cli.md)** (see ADR-0008).

Review routing for touched areas is defined in **[`CODEOWNERS`](CODEOWNERS)**.

## Prerequisites

- **Go**: match `go.mod` / CI (toolchain **1.26.1** as of this writing).
- **golangci-lint**: optional locally; CI runs it on every PR.
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

Optional commit message template (see [`.cursor/rules/commit-format.mdc`](../.cursor/rules/commit-format.mdc)):

```bash
git config commit.template .github/gitmessage
```

## Checks before you push

Same gates as CI (see also [`.cursor/rules/agent-safety.mdc`](../.cursor/rules/agent-safety.mdc)):

```bash
go test ./... -race -count=1
go vet ./...
golangci-lint run --timeout=5m ./...
```

**Coverage** (module-wide threshold **80%**, same as CI):

```bash
go test ./... -race -coverpkg=./... -coverprofile=coverage.out -count=1
bash tools/check-coverage-threshold.sh 80 coverage.out
```

## rgd, schemas, and layout

| Area | Role |
|------|------|
| [`cmd/rgd/`](../cmd/rgd) | `main`, thin entry — delegates to `internal/rgdcli` |
| [`internal/rgdcli/`](../internal/rgdcli) | Command tree, flags, JSON output |
| [`pkg/schema/`](../pkg/schema) | Embedded JSON Schemas; `rgd schema export` writes them to disk |
| [`internal/config/`](../internal/config) | Loads `.reinguard` / `rules/*.yaml`; `rgd config validate` |
| [`internal/observe/`](../internal/observe) | Observation providers (`git`, `github`, …) |

- Command tree, flags, stdout/stderr, exit codes: **`docs/cli.md`**.
- Export schemas for inspection: `./rgd schema export --dir /tmp/rgd-schemas` (after `go build`).

## Workflow and traceability

- Prefer **one GitHub Issue per implementation PR**; the PR should include
  `Closes #N` (or `Fixes`) and commits should use `Refs: #N` in the footer
  (see [`.cursor/rules/commit-format.mdc`](../.cursor/rules/commit-format.mdc)).
- Fill every section of [`.github/PULL_REQUEST_TEMPLATE.md`](PULL_REQUEST_TEMPLATE.md).
- Architectural intent lives under [`docs/adr/`](../docs/adr/).

## Continuous integration

[`ci.yaml`](workflows/ci.yaml) is split so **fork pull requests** stay safe without
running GitHub API collection on untrusted merge commits. Manual runs:
**`workflow_dispatch`**.

| Job | Runs on | Purpose |
|-----|---------|---------|
| `go-ci` | All pushes and PRs | Lint, vet, tests, coverage gate, basic `rgd` smoke |
| `rgd-dogfood` | All pushes and PRs | This repo’s `.reinguard` + `rgd config validate` + `rgd observe git` |
| `rgd-github-dogfood` | **Not** fork PRs | `rgd observe github` with `GH_TOKEN` |

Fork PRs skip `rgd-github-dogfood` when the head repo is not the base repo.

**Branch protection:** mark **`go-ci`** and **`rgd-dogfood`** as required checks.
Do **not** require **`rgd-github-dogfood`** if you merge from forks — a skipped job
does not satisfy a required check on GitHub.

## License

By contributing, you agree that your contributions are licensed under the same
terms as the project (Apache License 2.0 — see [`LICENSE`](../LICENSE)).
