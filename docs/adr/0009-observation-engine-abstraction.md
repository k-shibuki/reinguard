# ADR-0009: Observation engine abstraction

## Context

reinguard must assemble **structured observation** from heterogeneous
sources (local Git state, GitHub Issues, pull requests, CI, reviews)
without embedding repository-specific policy in Go. A prior art stack used
shell wrappers and **jq** pipelines; reinguard intentionally moves contract
and composition into **Go interfaces**, **JSON Schemas**, and
**repository-owned configuration** (see ADR-0002, ADR-0003, ADR-0008).

Alternatives considered:

1. **Monolithic observation in CLI** — one function per signal. Simple
   initially but does not scale; cross-cutting concerns (auth, retries,
   merging) duplicate.
2. **External scripts as providers** — flexible but weak typing, harder
   sandboxing, and unclear failure semantics.
3. **Hybrid: pluggable Go providers + config-declared signal sets** —
   compile-time safety for I/O and parsing; declarative choice of which
   providers run and how outputs merge.

## Decision

Adopt a **hybrid observation model**:

1. **Go provider interface** — Each observation source implements a
   small interface (collect signals into a namespaced or flat map, report
   diagnostics). Providers are registered in the binary; adding a new
   *kind* of source requires code.

2. **Configuration declares collection** — `reinguard.yaml` (and related
   config) lists which providers are enabled and any provider-specific
   options. The engine **does not** infer which signals to collect beyond
   what configuration requests.

3. **Side-effect free observation** — A single `rgd observe` invocation
   performs read-only queries (Git, GitHub API). It does not mutate repo
   state, post comments, or merge PRs. Parallel invocations are independent
   (ADR-0003).

4. **GitHub authentication** — GitHub-backed providers obtain credentials
   only via **`gh auth token`** (or documented equivalent invocation of
   the GitHub CLI), per ADR-0006. No parallel “env-only” token path in
   Phase 1.

5. **Merge and diagnostics** — Provider outputs are merged into a single
   **observation document** (JSON) with a versioned schema. Partial
   failures are represented as **degraded** slices or diagnostics, not
   silent omission, so downstream match rules can use `depends_on`
   suppression (ADR-0004).

   **Namespaces:** The engine stores each provider’s fragment under the
   provider’s configured `id` (e.g. `signals.git`, `signals.github`).
   **Duplicate `id` values in `providers[]` are invalid** and rejected at
   config load. Within the GitHub aggregate provider, facet maps (issues,
   pull_requests, `ci`, reviews) are merged with **last-writer-wins** on
   duplicate top-level keys (`mergeSignals`); facets should use distinct
   keys.

6. **Agent-internal exclusion** — Providers MUST NOT read agent session
   files, phase trackers, or other non-repo, non-platform observables
   (ADR-0005).

## Consequences

- **Easier**: Clear extension point for new GitHub facets without changing
  the core CLI shape
- **Easier**: Tests can mock providers without network I/O
- **Harder**: Provider registry and merge semantics must stay stable as
  schemas evolve (ADR-0008)
- **Harder**: Authors must keep signal key names aligned with match rules
  and documentation (`docs/cli.md` as operational SSOT for CLI I/O)
