# reinguard

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://github.com/k-shibuki/reinguard/blob/main/LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/k-shibuki/reinguard)](https://github.com/k-shibuki/reinguard/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/k-shibuki/reinguard)](https://goreportcard.com/report/github.com/k-shibuki/reinguard)
[![CI](https://github.com/k-shibuki/reinguard/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/k-shibuki/reinguard/actions/workflows/ci.yaml?query=branch%3Amain)
[![Pre-release](https://img.shields.io/badge/status-pre--release-orange)](https://github.com/k-shibuki/reinguard#status)

**Repo-owned Semantic Control for Agentic Development** — *operational context for bot-aware PR work*.

`reinguard` makes the platform facts agentic work depends on observable before
the agent acts. It turns platform signals, review-bot state, local proof, and
repository-declared semantics into typed operational context — a deterministic
computation rather than a guess.

## The Problem

Agentic development does not fail only at code generation. It often fails
earlier, when agents act before the repository has made the relevant workflow
facts observable.

CI may have moved. A review bot may be rate-limited, stale, paused, or awaiting
acknowledgement. A branch may be locally verified but not yet PR-ready. A piece
of project knowledge may matter only in one operational state. Without a shared
observation surface for these facts, every actor reconstructs workflow position
ad hoc.

`reinguard` names that missing surface as the problem.

## The Direction

Observation comes first. `reinguard` puts the repository's operational semantics next to the work:
observation providers, workflow states, routes, guards, runtime gate roles,
knowledge `when` clauses, policies, and procedures live under version control.

Its runtime, **`rgd`**, reads those declarations, observes the current platform
state, and emits typed operational context. The goal is not to make a new agent
brain, planner, project manager, or code generator. The goal is to make the
context around agentic work deterministic, inspectable, and bot-aware before an
agent decides what to do next.

If the instinct is to solve this with an agent orchestrator, `reinguard` takes
the opposite bet: make the facts deterministic, then leave judgment with the
agent.

## What Operational Context Looks Like

When a required review bot is rate-limited and the branch is otherwise clean,
`rgd context build --compact` resolves the workflow position to a named state
and route. A trimmed excerpt of the JSON written to stdout:

```json
{
  "state": {
    "kind": "resolved",
    "state_id": "waiting_bot_rate_limited",
    "procedure_hint": {
      "procedure_id": "procedure-wait-bot-review",
      "path": "procedure/wait-bot-review.md",
      "derived_from": "state_id"
    }
  },
  "routes": [
    {
      "kind": "resolved",
      "route_id": "user-wait-bot-quota"
    }
  ]
}
```

The agent does not have to infer a stalled bot review from absence; the state
and route are named, and the procedure hint points at the repository-owned
action card. Full output also includes `schema_version`, `observation`,
`guards`, filtered `knowledge.entries`, and `diagnostics` — see
[docs/cli.md](docs/cli.md) and `pkg/schema/operational-context.json`.

## Why It Became This

The design separates observation, meaning, and action. Bots are not just peers
reading the same control surface; they are asynchronous review systems whose
state changes the appropriate path through a PR. The repository owns the
semantics for interpreting those signals; `rgd` computes the current position
under those semantics.

That is why `rgd` stays pull-based and stateless across invocations while
`reinguard` still records bounded local proof. Runtime gate artifacts under
`.reinguard/local/gates/` prove things like local verification or PR readiness
for one branch head. Adapter-local artifacts under `.reinguard/local/adapter/`
remember what execution path was approved by the user. These gitignored files
are filesystem evidence that later invocations re-observe; they are not agent
memory, workflow authority, or versioned Semantics.

Bot-aware FSM states, proof-carrying gates, and signal-filtered knowledge are
not embellishments. They are the load-bearing parts that make operational
context stable enough for agentic development.

---

## How It Is Structured

`reinguard` spans three layers, plus AI-facing reference:

- **Adapter** — client-specific procedures and bridge files (`.cursor/`)
- **Semantics** — repository meaning: knowledge, policy, states, routes, guards, and procedures (`.reinguard/`)
- **Substrate** — the `rgd` runtime that computes operational context from observation and declared rules

Additionally, [`AGENTS.md`](AGENTS.md) provides reviewer and agent configuration — read by AI bots and agents, not part of the layered runtime. See [ADR-0001](docs/adr/0001-system-positioning.md) and [ADR-0011](docs/adr/0011-semantic-control-plane-structure.md).

## `rgd` At A Glance

`rgd` provides observation, state and route evaluation, guard evaluation,
knowledge packing, operational context building, runtime gate recording and
inspection, review thread transport, schema export, and config validation.

Full command behavior, flags, stdout/stderr rules, and exit codes:
[docs/cli.md](docs/cli.md).

## Repository Runtime Expectations

- **Schema**: `0.8.0`
- **GitHub auth**: `gh` CLI
- **Go**: 1.25.8+ (CI: 1.26.1; see [`go.mod`](go.mod))
- **Build**: `go build -o rgd ./cmd/rgd`
- **Check**: `make check`

Smoke check:

```bash
go run ./cmd/rgd version
go run ./cmd/rgd config validate
go run ./cmd/rgd schema export --dir /tmp/rgd-schemas
```

Contribution and workflow policy: [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md)

## Design Rationale

- [ADR-0001](docs/adr/0001-system-positioning.md) — system positioning and layer boundaries
- [ADR-0003](docs/adr/0003-pull-based-stateless-invocation.md) — pull-based, stateless invocation
- [ADR-0010](docs/adr/0010-knowledge-management.md) — knowledge format and signal-driven delivery
- [ADR-0011](docs/adr/0011-semantic-control-plane-structure.md) — Semantics layout
- [ADR-0013](docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) — bot-aware FSM states and routes
- [ADR-0014](docs/adr/0014-runtime-gate-artifacts.md) — runtime gate artifacts
- [ADR-0015](docs/adr/0015-adapter-local-execute-resume.md) — Adapter-local execute resume

## Status / License

**Pre-release.** The CLI and schemas are still evolving. Licensed under
[Apache-2.0](LICENSE).
