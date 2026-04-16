# reinguard

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://github.com/k-shibuki/reinguard/blob/main/LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/k-shibuki/reinguard)](https://github.com/k-shibuki/reinguard/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/k-shibuki/reinguard)](https://goreportcard.com/report/github.com/k-shibuki/reinguard)
[![CI](https://github.com/k-shibuki/reinguard/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/k-shibuki/reinguard/actions/workflows/ci.yaml?query=branch%3Amain)
[![Pre-release](https://img.shields.io/badge/status-pre--release-orange)](https://github.com/k-shibuki/reinguard#status)

**reinguard** is a repo-owned control system for agentic development. It stabilizes the information space in which AI agents reason—without turning into a workflow brain.

Its runtime, **`rgd`**, is a stateless CLI that computes operational context from repository-declared semantics and platform observation.

## Why it exists

AI agents often fail not because they cannot generate code, but because they reason over unstable inputs:

- incomplete or inconsistent observation
- missing guard checks
- ambiguous workflow position
- poor reachability to repository knowledge and policy

reinguard addresses that problem by making repository meaning explicit, version-controlled, and auditable.

## How it is structured

reinguard spans three layers:

- **Adapter** — client-specific procedures and bridge files (`.cursor/`, `AGENTS.md`)
- **Semantics** — repository meaning: knowledge, policy, states, routes, and guards (`.reinguard/`)
- **Substrate** — the runtime (`rgd`) that computes operational context from observation and declared rules

`rgd` is intentionally **pull-based and stateless**: each invocation observes current repository and platform state, evaluates it, and exits. Improvement happens through **design-time feedback** into the Semantics layer, not through runtime memory or online adaptation.

## What `rgd` provides

Representative commands:

```text
rgd observe workflow-position
rgd gate status local-verification
rgd state eval
rgd route select
rgd guard eval merge-readiness
rgd knowledge pack
rgd context build
rgd config validate
rgd schema export
```

Command behavior, flags, and exit codes: [docs/cli.md](docs/cli.md)

## What reinguard is not

- a workflow orchestrator or autonomous planner
- a code generator or application framework
- a project management tool
- a substitute for human or agent judgment on design or review substance
- a system that tracks agent-internal progress files or session state

## Design

Key design decisions are recorded as ADRs under [docs/adr/](docs/adr/).

Start with:

- [ADR-0001](docs/adr/0001-system-positioning.md) — system positioning: not a workflow brain
- [ADR-0003](docs/adr/0003-pull-based-stateless-invocation.md) — pull-based stateless invocation
- [ADR-0010](docs/adr/0010-knowledge-management.md) — repository knowledge and agent-facing delivery
- [ADR-0011](docs/adr/0011-semantic-control-plane-structure.md) — semantic control plane layout

## Development

- **Go**: 1.25.8+ (CI: 1.26.1; see [`go.mod`](go.mod))
- **Build**: `go build -o rgd ./cmd/rgd`
- **Test / vet / lint**: `make check` (uses `.reinguard/scripts/with-repo-local-state.sh` so caches stay under repo-local `.tmp/`; see [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md))

Smoke check:

```bash
go run ./cmd/rgd version
go run ./cmd/rgd config validate
go run ./cmd/rgd gate status local-verification
go run ./cmd/rgd schema export --dir /tmp/rgd-schemas
```

Contribution and workflow policy: [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md)

## Status

**Pre-release.** The CLI and schemas are still evolving. ADRs are the source of truth for architectural intent.

## License

Apache License 2.0. See [LICENSE](LICENSE).
