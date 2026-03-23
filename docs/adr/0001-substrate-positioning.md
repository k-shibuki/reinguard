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

This project generalizes patterns proven in an existing agent-control
system (structured evidence, finite-state workflow position, guard
discipline) into a standalone, language-agnostic product delivered as a
Go binary.

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
