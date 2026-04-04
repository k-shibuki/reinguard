---
id: review-classification-map
description: Unified disposition categories for local review, self-inspection, and PR review flows
triggers:
  - review classification map
  - disposition categories
  - finding dispositions
  - local review disposition
  - pre-PR acknowledged
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Review classification lifecycle

Explanatory map for how the shared review disposition vocabulary flows
through local review, pre-PR inspection, and PR review.

The **normative** definition of the four categories lives in
`.reinguard/policy/review--disposition-categories.md`. This knowledge file
is a retrieval aid and lifecycle explainer, not the policy source of
truth.

## Terms

- **Disposition categories** or **finding dispositions** are the umbrella
  terms for review classification in this repository.
- **Finding** means one review-reported concern that requires a
  disposition, regardless of whether it came from local CodeRabbit,
  self-inspection, a PR thread, or a non-thread PR comment.
- Retrospective or root-cause buckets (for example in `internalize`) are a
  separate explanatory axis. They explain **why** a finding happened; they
  do not replace the four dispositions.

## Lifecycle

### Local review before PR creation

- `change-inspect` and the local CodeRabbit CLI gate use the same four
  categories defined in
  `.reinguard/policy/review--disposition-categories.md`.
- Local findings have no GitHub thread yet, so the agent records the
  disposition in an inspection ledger and applies fixes or rationale
  locally.
- Before `pr-create`, batch one local CR pass's actionable findings,
  apply same-kind sweep where the fix pattern extends beyond the exact
  comment, and then rerun the local gate on the stabilized head.

### PR review after creation

- If the same issue appears again on the PR, keep the same disposition
  vocabulary in `review-address`.
- Inline thread replies and non-thread PR conversation replies use the same
  categories, with consensus and resolution rules defined by
  `.reinguard/policy/review--consensus-protocol.md`.
- PR review closes findings through PR evidence channels rather than local
  inspection output: thread findings need thread reply plus consensus;
  non-thread findings need a PR conversation comment.
- Do not switch to a different label set just because the finding came from
  CodeRabbit, another bot, a human review, or check output.

## Why this file still exists

- It helps retrieval through `rgd context build` / knowledge packing when
  agents need orientation about the lifecycle of local vs PR review.
- It points readers to the normative policy files without duplicating their
  body text as a competing source of truth.

## Related

- `.reinguard/policy/review--disposition-categories.md` — normative
  category definitions
- `.reinguard/procedure/change-inspect.md` — pre-PR inspection flow
- `.reinguard/procedure/review-address.md` — PR review handling flow
- `.reinguard/policy/review--self-inspection.md` — whole-change
  self-inspection criteria
- `.reinguard/policy/review--consensus-protocol.md` — PR-thread consensus
  and resolution rules
