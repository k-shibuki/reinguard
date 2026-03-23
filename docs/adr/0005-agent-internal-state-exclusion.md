# ADR-0005: Agent-internal state exclusion

## Context

It is tempting to model fine-grained development phases (e.g., "tests
written but not run") as distinct workflow states, including states
inferred from **files written by the agent** or other **self-reported**
markers.

That approach has three problems:

1. **Verifiability** — Agent-generated artifacts are not externally
   auditable the same way git remotes, GitHub API responses, or CI
   statuses are.
2. **Stability** — Two agents (or two sessions) observing the **same**
   repository checkout could receive **different** operational context
   if internal phase files differ, violating the goal that operational
   context be stable for a given observable repo and platform state.
3. **Separation of concerns** — Distinguishing "implementation sub-phase"
   is **semantic judgment** about the agent's own progress; it belongs to
   the agent, not to a substrate that claims to compute platform-grounded
   context.

A prototype workflow enumerated more states than needed because four of
them depended on agent-local markers. Collapsing those into a coarser
externally observable state reduces false precision and aligns the
product with verifiable signals only.

## Decision

reinguard **does not** observe, track, or depend on **agent-internal**
state:

- No reading of agent-generated phase files, task markers, or progress
  indicators
- No reading of agent session state
- Observation scope is limited to what can be derived from **git**,
  **GitHub** (and similar platform APIs), **CI**, and other **external**
  platform sources defined in configuration

If a signal cannot be derived from those sources, it is **outside**
observation scope.

The substrate may return coarse observable facts (e.g., feature branch
with uncommitted changes and no open pull request). The **agent**
interprets what phase of its own work that represents.

When migrating from a richer state catalog, states that existed only to
encode agent-internal sub-phases are **merged** into a single externally
grounded state rather than preserved as distinct IDs.

## Consequences

- **Easier**: Stronger guarantee that operational context is a function
  of verifiable platform and repository state
- **Easier**: Simpler observation engine (no scanning for agent-specific
  file conventions)
- **Harder**: Agents must track fine-grained progress internally; the
  substrate will not encode it
- **Harder**: Users accustomed to many fine-grained states must adapt
  workflows or keep human-facing documentation outside the substrate
