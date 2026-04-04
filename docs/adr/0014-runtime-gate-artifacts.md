# ADR-0014: Runtime gate artifacts for deterministic local workflow progression

## Status

Accepted.

## Context

`working_no_pr` currently collapses materially different local situations:
implementation in progress, local verification complete, self-inspection
complete, and PR-ready handoff. The FSM can only route on observable inputs, so
procedure progress that exists only in agent memory cannot justify stable state
or route separation.

At the same time, reinguard must stay within the substrate boundary from
ADR-0001: `rgd` may compute deterministic context from repository-owned inputs,
but it must not become a workflow brain or execute arbitrary repository scripts.

## Decision

1. Introduce **runtime gate artifacts** under `.reinguard/runtime/gates/`.
   These are **gitignored, substrate-owned operational state**, not Semantics
   documents.
2. Add `rgd gate` commands:
   - `rgd gate record <gate-id>` - bounded write of one validated artifact for
     the current branch HEAD
   - `rgd gate status <gate-id>` - derive `missing` / `invalid` / `stale` /
     `fail` / `pass`
   - `rgd gate show <gate-id>` - print the validated raw artifact
3. Gate artifacts are validated against an embedded JSON Schema and must record:
   `gate_id`, top-level `status`, `head_sha`, `branch`, `recorded_at`, and
   `checks[]`.
4. Freshness is deterministic:
   - `missing`: artifact file absent
   - `invalid`: file unreadable or schema-invalid
   - `stale`: artifact branch or head SHA differs from the current checkout, or
     the current checkout cannot supply comparable git identity
   - otherwise the artifact's own `status` (`pass` / `fail`)
5. Evaluation commands merge derived runtime gate signals under `gates.<id>.*`
   before state / route / guard evaluation, so future FSM rules can branch on
   gates without inventing agent-internal memory.
6. `rgd` does **not** execute verification commands for gates. Procedures or
   agents run checks such as `go test`, `go vet`, or `golangci-lint`, then
   record the resulting verified outcome into a gate artifact.

## Consequences

- **Easier**: local workflow progression becomes machine-observable without
  overloading GitHub or git signals
- **Easier**: future issues can refine `working_no_pr` using stable
  `gates.<id>.status` signals
- **Easier**: gate freshness is auditable and tied to branch HEAD
- **Harder**: procedures must explicitly record gates after verification
- **Harder**: artifacts are operational state and must stay out of versioned
  Semantics content

## Refs

- ADR-0001 (system positioning)
- ADR-0003 (pull-based, stateless invocation)
- ADR-0008 (schema versioning)
- ADR-0011 (semantic control plane structure)
- Issue #97
