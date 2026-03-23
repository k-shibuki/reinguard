# Contributing to reinguard

Thank you for helping improve reinguard. This document is the **human-facing**
entry point for contributors; normative CLI behavior lives in
[`docs/cli.md`](docs/cli.md) (ADR-0008).

## Prerequisites

- **Go**: match `go.mod` / CI (toolchain **1.26.1** as of this writing).
- **golangci-lint**: optional locally; CI runs it on every PR.
- **`gh`**: required only for commands that call the GitHub API (`observe github`,
  live observation in `state eval` / `context build`, etc.). See ADR-0006.

## Quick setup

```bash
git clone https://github.com/k-shibuki/reinguard.git
cd reinguard
go build -o rgd ./cmd/rgd
```

Optional: a thin [`Makefile`](Makefile) provides `make test`, `make check`
(fmt, vet, test, `golangci-lint`), `make coverage`, etc. It is **not** normative
— CI and the shell commands below remain the source of truth.

## Checks before you push

Run the same gates CI enforces (see also
[`.cursor/rules/agent-safety.mdc`](.cursor/rules/agent-safety.mdc)):

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

## CLI and schemas

- Command tree, flags, stdout/stderr, and exit codes: **`docs/cli.md`**.
- Embedded JSON Schemas: `go run ./cmd/rgd schema export --dir /tmp/rgd-schemas`.

## Workflow and traceability

- Prefer **one GitHub Issue per implementation PR**; the PR should include
  `Closes #N` (or `Fixes`) and commits should use `Refs: #N` in the footer
  (see [`.cursor/rules/commit-format.mdc`](.cursor/rules/commit-format.mdc)).
- Fill every section of [`.github/PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md).
- Architectural intent is recorded under [`docs/adr/`](docs/adr/).

## Continuous integration

The [`ci.yaml`](.github/workflows/ci.yaml) workflow is split so **fork pull
requests** stay reliable without relying on privileged GitHub API usage from
untrusted merge commits. It can also be started manually via **`workflow_dispatch`**.

| Job | Runs on | Purpose |
|-----|---------|---------|
| `go-ci` | All pushes and PRs | Lint, vet, tests, coverage gate, basic CLI smoke |
| `rgd-dogfood` | All pushes and PRs | Validates this repo’s `.reinguard` and `rgd observe git` |
| `rgd-github-dogfood` | **Not** fork PRs | Builds `rgd` and runs `rgd observe github` with `GH_TOKEN` |

Fork PRs skip `rgd-github-dogfood` entirely (`pull_request` where the head
repository differs from the base). Pushes to `main` and same-repository PRs run
all jobs.

**Branch protection:** mark **`go-ci`** and **`rgd-dogfood`** as required checks.
Do **not** require **`rgd-github-dogfood`** if you merge contributions from
forks — a skipped job does not satisfy a required status check on GitHub.

## License

By contributing, you agree that your contributions are licensed under the same
terms as the project (Apache License 2.0 — see [`LICENSE`](LICENSE)).
