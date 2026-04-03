---
id: review-disposition-categories
description: Normative disposition categories for local review, self-inspection, and PR review
triggers:
  - disposition categories
  - finding dispositions
  - Fixed By design False positive Acknowledged
  - pre-PR acknowledged
---

# Review disposition categories

Normative disposition categories for review findings across local review,
self-inspection, and PR review.

## Principle

Use one disposition vocabulary everywhere review findings are classified:
**Fixed / By design / False positive / Acknowledged**.

Legacy reviewer severity labels such as **P0 / P1** and pre-PR gate labels
such as **Blocking / Non-blocking** are retired as standing classification
axes. Historical comments may still contain them, but new repository
guidance must use the four disposition categories instead.

## Categories

| Category | Meaning | Typical evidence |
|---|---|---|
| **Fixed** | The branch includes a change that addresses the finding. | Commit or working tree diff that removes the issue. |
| **By design** | The reported behavior is intentional for this repository. | ADR, policy, Issue scope, or explicit design rationale. |
| **False positive** | The finding premise is incorrect. | Code path, test, or repo rule showing the report is mistaken. |
| **Acknowledged** | The finding is valid but intentionally deferred. | Explicit deferred-work contract, plus rationale. |

`Fixed` is the correct disposition label. Do **not** rename it to `Fix`:
`Fix` is an action verb, while `Fixed` is the recorded outcome/state of a
finding.

Retrospective or root-cause buckets (for example in `internalize`) remain a
separate explanatory axis. They explain **why** a finding happened; they do
not replace the four dispositions.

## Pre-PR rule for `Acknowledged`

Before PR creation, `Acknowledged` is **not** a default escape hatch.

- Use it only when deferred work has an explicit contract outside the
  current branch, such as a separate follow-up Issue or another equally
  clear deferred-work record.
- Do **not** use it for a small, in-scope change that should be fixed on
  the current branch.
- If no such contract exists, keep working until the finding is
  dispositioned **Fixed**, **By design**, or **False positive**.

## PR-thread rule for `Acknowledged`

After PR creation, apply the same vocabulary with the PR-side consensus and
thread-resolution mechanics in
`.reinguard/policy/review--consensus-protocol.md`.

## Related

- `.reinguard/policy/review--consensus-protocol.md` — PR-side consensus and
  thread resolution
- `.reinguard/policy/review--self-inspection.md` — pre-PR inspection
  criteria
- `.reinguard/procedure/change-inspect.md` — pre-PR inspection procedure
- `.reinguard/procedure/review-address.md` — PR review handling procedure
