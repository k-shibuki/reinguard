---
id: review-ci-permissions-gates
description: GitHub Actions permissions, aggregate CI gates, and least privilege
triggers:
  - github actions permissions
  - least privilege
  - ci-pass
  - required checks
---

# CI Permissions and Aggregate Gates

## Rule 1: Minimize workflow default token scope

At workflow-level `permissions`, keep the narrowest shared baseline.

Grant broader scopes only on jobs that actually need them
(including reusable workflow call jobs).

## Rule 2: Keep required gate aligned with policy intent

If branch protection relies on aggregate check `ci-pass`, ensure all
must-pass jobs feed that aggregate outcome.

Typical pitfall:
- adding new validation jobs without adding them to aggregate `needs`

## Rule 3: Preserve explicit skip semantics

For conditionally skipped jobs (e.g. non-fork-only jobs), aggregate logic should
allow expected `skipped` where policy intends it, and fail otherwise.
