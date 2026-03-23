# ADR-0007: Ambiguity as evaluation outcome

## Context

Some situations cannot be collapsed into a single workflow state without
**semantic interpretation** or **additional evidence**: observation may be
incomplete, signals may conflict, or multiple state rules may match in a
way that does not admit a unique winner (see ADR-0004 for same-priority
matches).

Two modeling approaches:

1. **Ambiguity as a normal workflow state** — e.g., a dedicated state ID
   meaning "uncertain," with transitions when the agent resolves it.
2. **Ambiguity as an evaluation outcome** — classification carries a
   status field (`resolved`, `ambiguous`, `degraded`, `unsupported`)
   plus structured metadata: candidate interpretations, missing
   evidence, suppressed candidates, and a **re-entry contract** telling
   the agent what to do next.

Option 1 conflates **position in a workflow** with **confidence in
classification**. It also encourages the substrate to pretend it has a
conventional state when the underlying evidence does not support one.

## Decision

Represent uncertainty as **evaluation outcomes** attached to operational
context, not as a routine workflow state ID for every kind of confusion.

Operational context includes at least:

- A **classification status** distinguishing resolved results from
  ambiguous, degraded, or unsupported outcomes
- When ambiguous or degraded: **why** (e.g., conflicting rules,
  suppressed candidates, missing sources)
- **Reasoning handoff** metadata: candidate interpretations, missing
  evidence, and an explicit **re-entry contract** (e.g., refresh a
  specific observation, provide a human decision, narrow candidates)

When observation dependencies fail such that **no** state candidate can be
trusted, prefer **degraded** (or equivalent) with structured reasons over
guessing a nominal state.

When multiple rules match at the same best priority (ADR-0004), emit
**ambiguous** rather than picking silently.

The substrate **must not** fabricate certainty from incomplete or
conflicting evidence.

## Consequences

- **Easier**: Honest outputs; agents and humans can see what is unknown
- **Easier**: Clear separation between "where we think we are" and "how
  confident we are"
- **Harder**: Agents must implement handling for non-resolved
  classification status
- **Harder**: More fields in the operational context contract and more
  test cases for handoff scenarios
