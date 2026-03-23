# ADR-0002: Spec-driven evaluation: match rules and named evaluators

## Context

The runtime must classify workflow position from observation signals,
evaluate guard admissibility, and select route candidates. Implementing
this entirely in imperative Go code (large switch statements, ad hoc
conditionals) makes workflow logic hard to change and invites **semantics
leakage** from repositories into the binary.

Alternative approaches considered:

1. **All logic in Go** — simple to ship but brittle; every policy change
   requires a release; the binary becomes the semantic authority.
2. **Embedded general-purpose DSL or scripting** (e.g., Lua, CEL, Rego) —
   flexible but adds a second language to learn, security and sandboxing
   concerns, and long-term maintenance burden.
3. **Plugin execution** (external commands as evaluators) — maximum
   flexibility but weak contracts for output shape, error handling, and
   security.
4. **Two-tier model** — declarative **match rules** in configuration for
   most conditions, plus **named evaluators** compiled into Go for
   algorithmic aggregation that exceeds a bounded operator set.

A prior prototype used shell and **jq** pipelines for evaluation. That
proved powerful but difficult to test in isolation and opaque to
contributors who do not read jq.

## Decision

Adopt a **two-tier evaluation model**:

**Match rules (declarative)** — Primary mechanism for state classification,
guard checks, and route selection. Rules use a bounded set of operators:

- Scalar comparison: `eq`, `ne`, `gt`, `lt`, `gte`, `lte`
- Collection: `in`, `contains`
- Existence: `exists`, `not_exists`
- Aggregation over arrays: `count`, `any`, `all`
- Logical: `and`, `or`, `not`

**Arithmetic operators are excluded by design.** Computations that need
arithmetic (e.g., summing findings across bots) belong in named
evaluators.

**Named evaluators (compiled Go)** — Referenced by name from
configuration with parameters. Used when aggregation, iteration, or
domain-specific algorithms cannot be expressed in the match syntax. The
**count** of named evaluators is a tracked design metric; unchecked growth
signals semantics leakage into the binary.

**Phase 1 scope:** User-defined conditions use match rules and operators
only. Named evaluators available in shipped configuration are limited to
a **built-in** set (e.g., review consensus, merge readiness). Extension
mechanisms such as plugin `exec` or user-supplied evaluators are **out of
scope** until real demand is validated (including against golden
fixtures migrated from the prototype).

## Consequences

- **Easier**: Policy and workflow position changes are made primarily in
  configuration and tests, not in ad hoc Go branches
- **Easier**: Golden tests can assert the contract: given config and
  observation inputs, operational context matches expected output
- **Easier**: The match engine stays small and reviewable
- **Harder**: The operator set may prove insufficient for some repos;
  adding operators or evaluators requires explicit justification
- **Harder**: The line between "belongs in match rules" and "needs a
  named evaluator" requires ongoing discipline in design and code review
- **Harder**: Named evaluators require a Go release to change behavior;
  there is no hot-reload of evaluator logic
