# ADR-0003: Pull-based stateless invocation

## Context

The substrate must define how agents obtain operational context.

Alternatives:

1. **Daemon process** — long-running service subscribed to webhooks or
   polling; pushes events to the agent.
2. **Pull-based CLI** — the agent invokes a command when it needs fresh
   context; each run reads current state and exits.
3. **Hybrid** — pull-based CLI with optional local cache or background
   sync.

A daemon introduces state: cache invalidation, TTL management, failure
modes when the daemon falls behind, and coordination when multiple agent
sessions run concurrently. A prototype workflow used **stateless** shell
targets for observation; that model matched the "refresh when needed"
mental model of agent-driven development.

## Decision

Use **pull-based, stateless** invocation.

- The agent invokes the `rgd` CLI when it needs operational context.
- reinguard does **not** run as a daemon, does **not** subscribe to
  webhooks, and does **not** push events.
- Each invocation is **stateless**: no durable state is carried between
  invocations inside reinguard.
- **Phase 1** explicitly excludes: a caching layer inside the substrate,
  event subscription, and background process management owned by
  reinguard.

Two invocations at different times may return different results if
repository or platform state changed in between. That is acceptable
because the agent controls when to re-invoke.

Future extensions (e.g., explicit snapshot commands for audit or replay,
optional caching) are **not** part of the initial design but are **not**
ruled out by this ADR.

**Concurrency:** There is no shared state between concurrent
invocations. Parallel sessions each perform a full observation cycle
independently.

## Consequences

- **Easier**: Simplest correct mental model; no cache invalidation bugs
  inside the substrate; each run is independently testable
- **Easier**: No process lifecycle to manage in development or CI
- **Harder**: Redundant work when invoked frequently (full observation
  each time)
- **Harder**: No proactive notification; the agent must poll by invoking
  when it needs freshness
