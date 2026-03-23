# ADR-0004: Unified priority-based state resolution

## Context

Workflow position must be derived from multiple observation signals. A
prototype implementation used **two tiers**: a global workflow position
and a pull-request-scoped position, merged by a dedicated step into an
**effective** identifier. That works but adds coupling between tiers,
requires maintaining merge rules separately from classification rules,
and obscures a single ordering principle.

Other tie-breaking strategies were considered:

1. **Declaration order** in configuration — first matching rule wins among
   ties. Simple but **silent**: reordering YAML changes behavior without
   validation feedback.
2. **Specificity** (CSS-like) — more specific `when` clauses win. Powerful
   but hard for authors to predict and debug.
3. **Same priority as validation error** — forbids duplicate priority
   values. Strict but inflexible when rules are authored independently.

State rules also need to interact with **degraded observation**: if a
rule depends on a source that failed or is stale, that candidate must be
suppressed without hiding the failure behind logs.

## Decision

**Unified priority resolution** in a single rule set:

1. Evaluate all state match rules against current observation signals.
2. For each matching rule, check **dependencies** (`depends_on`): if any
   declared observation source is degraded (stale, unavailable, error,
   rate-limited, etc.), **suppress** that candidate.
3. Among unsuppressed matches, select by **priority**. **Priority values
   are floating-point numbers.** Lower numeric values indicate higher
   precedence (equivalently: the "best" match has the minimum priority
   number among matches).
4. If **no** rule matches or **all** matches are suppressed, emit a
   **degraded** classification outcome (operational context reflects
   incomplete observation), not a guessed state.
5. If **more than one** unsuppressed rule matches at the **same best
   priority value**, do **not** pick a winner silently. Emit
   **ambiguous** classification and a **reasoning handoff** (see
   ADR-0007). This rejects declaration-order tie-breaking.

**Configuration validation:** If two or more rules declare the **same**
priority value, validation emits a **warning** (not necessarily a hard
error) so authors notice potential ambiguity early.

Floating-point priorities allow inserting a new rule between existing
ones without renumbering entire sections.

## Consequences

- **Easier**: One conceptual ordering mechanism; merge logic is not a
  separate artifact
- **Easier**: Explicit ambiguity is preferable to silent wrong state
- **Harder**: Authors must reason about priorities globally across all
   rules, not only within a tier
- **Harder**: Floating-point equality for "same priority" warnings needs
   careful handling (epsilon or normalized representation)
- **Harder**: Ambiguous outcomes require agent-side handling (ADR-0007)
