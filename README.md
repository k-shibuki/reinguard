# reinguard

[![CI](https://github.com/k-shibuki/reinguard/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/k-shibuki/reinguard/actions/workflows/ci.yaml?query=branch%3Amain)

**reinguard** is a three-layer control system (Adapter / Semantics /
Substrate) that stabilizes the information space in which AI agents
reason—without taking over semantic judgment. Its runtime, `rgd`, is a
stateless CLI that computes operational context from repository-declared
specifications and platform observation.

## What reinguard is

reinguard is the **complete control system** spanning three layers
(see [ADR-0001](docs/adr/0001-system-positioning.md)):

- **Spec-driven**: workflow position, guards, routes, knowledge, and
  policy are expressed in version-controlled configuration under
  `.reinguard/`, not hard-coded in the binary.
- **Feedback-aware**: review findings flow back into the Semantics layer
  through `internalize`, improving future operational context without
  making the runtime stateful (design-time correction, not online
  adaptation).
- **Layered**: Adapter (`.cursor/`, `AGENTS.md`) adapts for specific
  clients; Semantics (`.reinguard/`) declares repository meaning;
  Substrate (`rgd`) computes.

## What `rgd` does

`rgd` is the **Substrate layer**—a single Go binary that agents invoke
on demand:

- **Pull-based and stateless**: each invocation observes current
  repository and platform state, evaluates it, and exits. No durable
  state is carried between runs.
- Stabilizes the **information surface** (structured observation,
  declarative rules, deterministic guards) that agents use to decide
  what to do next.
- **Auditable**: outputs are typed, versioned, and intended to support
  golden testing and CI validation.

## What reinguard is not

- A **workflow orchestrator** or autonomous planner
- A **code generator** or application framework
- A **project management** or issue tracker
- A substitute for **human or agent judgment** on design, review
  substance, or exception policy
- A system that **tracks agent-internal** progress files or session state

## CLI (representative)

These illustrate the intended command surface; behavior is defined by the
implementation and configuration.

```text
rgd observe workflow-position
rgd state eval
rgd route select
rgd guard eval merge-readiness
rgd knowledge pack
rgd context build
rgd config validate
rgd schema export
```

## CLI reference

Command behavior, flags, and exit codes: [docs/cli.md](docs/cli.md).

## Architecture decisions

Authoritative decisions are recorded as ADRs under [docs/adr/](docs/adr/):

| ADR | Title |
|-----|--------|
| [ADR-0001](docs/adr/0001-system-positioning.md) | System positioning: not a workflow brain |
| [ADR-0002](docs/adr/0002-spec-driven-evaluation.md) | Spec-driven evaluation: match rules and named evaluators |
| [ADR-0003](docs/adr/0003-pull-based-stateless-invocation.md) | Pull-based stateless invocation |
| [ADR-0004](docs/adr/0004-unified-priority-based-state-resolution.md) | Unified priority-based state resolution |
| [ADR-0005](docs/adr/0005-agent-internal-state-exclusion.md) | Agent-internal state exclusion |
| [ADR-0006](docs/adr/0006-gh-cli-as-sole-authentication.md) | gh CLI as sole authentication source |
| [ADR-0007](docs/adr/0007-ambiguity-as-evaluation-outcome.md) | Ambiguity as evaluation outcome |
| [ADR-0008](docs/adr/0008-schema-versioning.md) | Schema versioning: synchronized semver with best-effort compatibility |
| [ADR-0009](docs/adr/0009-observation-engine-abstraction.md) | Observation engine abstraction (providers + config) |
| [ADR-0010](docs/adr/0010-knowledge-management.md) | Repository knowledge: format, manifest generation, and agent-facing delivery |
| [ADR-0011](docs/adr/0011-semantic-control-plane-structure.md) | Semantic control plane directory structure |

## Development

- **Go**: 1.25.8 or newer; CI uses 1.26.1 (see [`go.mod`](go.mod)).
- **Build**: `go build -o rgd ./cmd/rgd`
- **Test**: `go test ./...`
- **Vet**: `go vet ./...`
- **Lint** (optional locally): install [golangci-lint](https://golangci-lint.run/) and run `golangci-lint run` (CI enforces it).

### CLI smoke (from repository root)

```bash
go run ./cmd/rgd version
go run ./cmd/rgd config validate
go run ./cmd/rgd schema export --dir /tmp/rgd-schemas
```

## Contributing

See **[.github/CONTRIBUTING.md](.github/CONTRIBUTING.md)** for local checks, CI behavior (including
fork PRs), branch protection, labels, PR policy, and review-thread rules.

- Follow **Issue-driven** workflow: open an Issue, then a PR that `Closes #N`
  (SSOT: `.reinguard/policy/workflow--pr-discipline.md`; Cursor: `workflow-policy.mdc`).
- **Commit format** follows `.reinguard/policy/commit--format.md`
  (Conventional Commits + `Refs: #N` in the body; Cursor: `commit-format.mdc`). Optional local setup:
  - `git config commit.template .github/gitmessage`
  - `pip install pre-commit && pre-commit install --hook-type commit-msg`
- Use the PR template at `.github/PULL_REQUEST_TEMPLATE.md`.
- Architecture decisions belong in `docs/adr/` (ADR).

## License

Distributed under the Apache License 2.0. See [LICENSE](LICENSE).

## Status

**Pre-release.** The CLI and schemas are under active design; ADRs are the
source of truth for architectural intent.
