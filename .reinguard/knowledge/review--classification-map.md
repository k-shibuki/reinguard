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

# Review classification map

Use one vocabulary for review findings across local review, pre-PR
inspection, and PR review: **Fixed / By design / False positive /
Acknowledged**.

## Terms

- **Disposition categories** or **finding dispositions** are the umbrella
  terms for review classification in this repository.
- **Fixed** is the correct label for the "addressed" outcome. Do **not**
  rename it to **Fix**: `Fix` is an action verb, while `Fixed` is the
  disposition state recorded for a finding.
- Retire legacy reviewer severity labels such as **P0 / P1** and pre-PR
  gate severity labels such as **Blocking / Non-blocking** as standing
  classification axes. Historical comments may still contain them, but new
  repository guidance should not introduce them.
- Retrospective or root-cause buckets (for example in `internalize`) are a
  separate explanatory axis. They explain **why** a finding happened; they
  do not replace the four dispositions.

## Lifecycle

### Local review before PR creation

- `change-inspect` and the local CodeRabbit CLI gate use the same four
  dispositions as PR review.
- Local findings have no GitHub thread yet, so the agent records the
  disposition in inspection output and applies fixes or rationale locally.
- Before `pr-create`, prefer `Fixed`, `By design`, or `False positive`.

### PR review after creation

- If the same issue appears again on the PR, keep the same disposition
  vocabulary in `review-address`.
- Inline thread replies and non-thread PR conversation replies use the same
  categories, with consensus and resolution rules defined by
  `.reinguard/policy/review--consensus-protocol.md`.
- Do not switch to a different label set just because the finding came from
  CodeRabbit, another bot, a human review, or check output.

## Disposition guide

| Category | Meaning | Typical evidence |
|---|---|---|
| **Fixed** | The branch now includes a change that addresses the finding. | Commit or working tree diff that removes the issue. |
| **By design** | The reported behavior is intentional for this repository. | ADR, policy, Issue scope, or explicit design rationale. |
| **False positive** | The finding premise is incorrect. | Code path, test, or repo rule showing the report is mistaken. |
| **Acknowledged** | The finding is valid but intentionally deferred. | Explicit deferred-work contract, plus rationale. |

## Pre-PR rule for `Acknowledged`

`Acknowledged` is **not** a default escape hatch before PR creation.

- Use it only when deferred work has an explicit contract outside the
  current PR, such as a separate follow-up Issue or another equally clear
  deferred-work record.
- Do **not** use it for a small, in-scope change that should be fixed on
  the current branch.
- If no such contract exists, keep working until the finding is
  dispositioned **Fixed**, **By design**, or **False positive**.

This stricter pre-PR rule keeps local review and self-inspection aligned
with the post-PR consensus model without creating a second vocabulary.

## Related

- `.reinguard/procedure/change-inspect.md` — pre-PR inspection flow
- `.reinguard/procedure/review-address.md` — PR review handling flow
- `.reinguard/policy/review--self-inspection.md` — whole-change
  self-inspection criteria
- `.reinguard/policy/review--consensus-protocol.md` — PR-thread consensus
  and resolution rules
