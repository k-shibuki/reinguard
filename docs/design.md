# reinguard design (transitional note)

## Document lifecycle policy

This document is **temporary** and will be removed after ADR coverage is
complete and implementation documentation is split into stable references.

The project is now **ADR-driven**. Normative architectural decisions live
in `docs/adr/`. This file is a high-level orientation only.

## Purpose

reinguard is a spec-driven control-plane substrate for AI-agent-driven
development. It reads repository-declared control specifications, performs
structured observation and evaluation, and returns typed operational context
for the agent.

reinguard does not replace semantic judgment. It stabilizes the information
surface used by judgment.

## Decision authority

The following ADRs are the source of truth for major decisions:

- [ADR-0001](adr/0001-substrate-positioning.md): substrate positioning
- [ADR-0002](adr/0002-spec-driven-evaluation.md): match rules + named evaluators
- [ADR-0003](adr/0003-pull-based-stateless-invocation.md): pull-based stateless model
- [ADR-0004](adr/0004-unified-priority-based-state-resolution.md): unified priority resolution
- [ADR-0005](adr/0005-agent-internal-state-exclusion.md): external-observable scope only
- [ADR-0006](adr/0006-gh-cli-as-sole-authentication.md): GitHub auth source
- [ADR-0007](adr/0007-ambiguity-as-evaluation-outcome.md): ambiguity handling
- [ADR-0008](adr/0008-schema-versioning.md): synchronized semver + best-effort compatibility
- [ADR-0009](adr/0009-observation-engine-abstraction.md): observation engine
  abstraction (providers + config-declared collection)
- [ADR-0010](adr/0010-knowledge-management.md): repository knowledge format,
  manifest generation, and agent-facing delivery

If this document conflicts with an ADR, the ADR wins.

## Product boundaries (summary)

### What reinguard is

- A runtime substrate that builds operational context from repo-owned specs
- A deterministic evaluator for route candidates and guards
- A typed and auditable interface for agent-facing workflow context
- A reusable binary (`rgd`) intended to work across repositories

### What reinguard is not

- A workflow brain or orchestrator that owns semantic decisions
- A replacement for agent reasoning or human design judgment
- A tracker of agent-internal progress artifacts
- A code-generation framework

## Runtime shape (summary)

- Invocation: pull-based CLI (`rgd`), stateless per invocation
- Inputs: repo configuration plus platform observations
- Outputs: operational context JSON (effective state, route candidates,
  guard results, knowledge references, diagnostics, provenance)
- Validation: schema-backed config and output contracts

For details, see ADR-0002, ADR-0003, ADR-0004, ADR-0007, ADR-0008.

## Repository split

### Repository side

- `.reinguard/` configuration
- project-specific policies, state/route/guard rules, knowledge manifests
- fixtures and contract tests

### reinguard side

- Go implementation (`rgd`)
- observation engine
- rule/evaluator runtime
- schema tooling and CLI

The repository defines meaning; reinguard computes current status under
that meaning.

## Representative CLI surface

- `rgd observe workflow-position`
- `rgd state eval`
- `rgd route select`
- `rgd guard eval merge-readiness`
- `rgd knowledge pack`
- `rgd context build`
- `rgd config validate`
- `rgd schema export`

## Migration note

The initial generalization path comes from patterns proven in `bridle` and
moves them into a standalone substrate with configuration as semantic SSOT.
Behavioral compatibility should be guarded by fixtures and golden tests.

## Removal plan for this file

This file should be deleted once:

1. All major architectural decisions are captured in ADRs (done for the
   first decision set).
2. Stable implementation references exist (README + command docs + schema
   docs).
3. Remaining non-decision orientation text is either moved to README or no
   longer needed.

Until deletion, keep this document short and non-normative.