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

### Extension contract (runtime gates)

When adding or changing a **runtime gate** (`gate_id`):

1. **Semantics** — Document the gate’s purpose; **producer** procedure(s) that run `rgd gate record <gate-id>` after local verification; **consumer** procedure(s) or FSM rules that read `gates.<gate-id>.*` (e.g. `status`, `head_sha`). Keep recording out of versioned Semantics; artifacts stay under `.reinguard/runtime/gates/` (gitignored).
2. **FSM** — If `state eval` or `route select` references `gates.<gate-id>`, update `.reinguard/control/states/*.yaml` and/or `.reinguard/control/routes/*.yaml` and **ADR-0013** (state catalog and Adapter mapping).
3. **Freshness** — Procedures must treat `rgd gate status` outcomes per Decision §4: `stale` / `missing` / `invalid` are not proof of the current HEAD; consumers return to the producer procedure or re-verify before proceeding.
4. **Schema** — Artifacts validate against the embedded gate schema; new top-level fields require ADR-0008 / schema versioning, not ad-hoc files.
5. **Tests** — Add or extend CLI/FSM tests when gates affect state resolution (e.g. `pass` vs `stale` fallback to a residual state).

`rgd` still does not execute gate verification commands; recording remains procedure-owned.

Operational checklist: `.reinguard/knowledge/workflow--state-gate-guard-extension.md`.

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
