# ADR-0001: Substrate positioning: not a workflow brain

## Context

AI agents that execute development workflows face unstable operational
surfaces: observation variance, state-recognition variance, missed
asynchronous platform signals from CI and pull requests, route
inconsistency, guard omission, and poor reachability to repository-owned
knowledge.

Two broad architectural responses exist:

1. **Orchestrator / workflow brain** — the system drives the agent
   through procedures, decides next actions, and centralizes control flow.
2. **Substrate** — the system stabilizes the information space in which
   the agent reasons: structured observation, declarative constraints,
   deterministic checks, and auditable outputs. Semantic judgment stays
   with the agent.

reinguard addresses these problems as a standalone, language-agnostic
product delivered as a Go binary (`rgd`).

## Decision

reinguard is a **control-plane substrate**, not a workflow brain.

It:

- reads control specifications declared in a repository
- observes repository and platform state
- evaluates that state into stable **operational context** (typed,
  auditable output)
- selects route **candidates**, evaluates **guards** deterministically,
  and **packs** knowledge references by declarative rules

It does **not**:

- decide architectural direction or design trade-offs
- determine whether a review comment is substantively correct
- plan work or replace the agent's reasoning
- track or depend on **agent-internal** state (phase files, task markers,
  session state)
- become the semantic authority for state meanings, transition
  semantics, review semantics, or repository policy (those remain in
  repo-owned configuration)

The primary way to constrain an agent is to design the **information
space** in which it operates, not to script its thinking.

## Responsibility layers

Three layers separate **how agents integrate**, **what the repository
means**, and **how the substrate evaluates**:

| Layer | Name | Verb | Location | SSOT for |
|-------|------|------|----------|----------|
| 3 | **Adapter** | adapt | `.cursor/`, `AGENTS.md` | Client-specific procedures, behavioral rules, bridge references |
| 2 | **Semantics** | declare | `.reinguard/` | Knowledge, policy, and control definitions (see ADR-0011) |
| 1 | **Substrate** | compute | `rgd` | Observation engine, rule/evaluator runtime, schema tooling, CLI |

**Dependency direction:** Adapter → Semantics → Substrate. Upper layers
may reference lower layers; lower layers do not depend on upper layers.

**Adapter principle:** The Adapter layer must not duplicate Semantics-layer
content as a second source of truth; it points at `.reinguard/` paths and
`rgd` commands instead.

**Change drivers:**

- **Adapter** — client tool updates, procedure changes
- **Semantics** — repository meaning (new states, policies, knowledge)
- **Substrate** — evaluator evolution (new providers, match operators)

The Semantics layer defines meaning; the Substrate computes current status
under that meaning.

## Consequences

- **Easier**: Clear product boundary; scope creep is visible when a
  feature would require semantic judgment or orchestration
- **Easier**: The substrate can be tested as ordinary software (golden
  fixtures, contract tests) independently of any specific agent
  implementation
- **Easier**: Multiple agents or sessions observing the same repository
  state can receive materially equivalent operational context when
  observation is complete (see ADR-0005)
- **Harder**: Every feature request must be checked against the substrate
  boundary; users may expect orchestration that is explicitly out of
  scope
- **Harder**: The agent retains responsibility for judgment-heavy work;
  the substrate does not pre-digest semantic disputes or exceptions
