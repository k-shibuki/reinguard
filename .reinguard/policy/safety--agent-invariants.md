---
id: safety-agent-invariants
description: HS-* safety codes for agents — CI merge, PR template/base, local verify, review resolve, merge consensus, no-dismiss
triggers:
  - HS-CI-MERGE
  - HS-PR-TEMPLATE
  - HS-PR-BASE
  - HS-LOCAL-VERIFY
  - HS-NO-SKIP
  - HS-NO-DISMISS
  - HS-REVIEW-RESOLVE
  - HS-MERGE-CONSENSUS
---

# Agent safety invariants (HS-*)

Non-negotiable rules for agents working on this repository.

## HS-CI-MERGE

Never merge a PR unless **required CI checks are green**. Use GitHub PR checks or `gh pr checks` before `gh pr merge`.

Never use `gh pr merge --admin` to bypass branch protection.

## HS-PR-TEMPLATE

Every PR must follow `.github/PULL_REQUEST_TEMPLATE.md` (all required sections present). CI job `gate-policy` enforces this (logic in `.github/scripts/pr-policy-check.js`; reusable `pr-policy.yaml` remains available).

## HS-PR-BASE

Never use `gh pr create --base <feature-branch>`. All PRs must target **`main`**. Document stack dependencies in the PR body instead.

## HS-LOCAL-VERIFY

Run the applicable subset per [`coding--preflight.md`](coding--preflight.md) § `HS-LOCAL-VERIFY (conditional)` before every push.

## HS-NO-SKIP

Run any policy-defined process to its own stated completion (verification, retry, wait, completion, review, merge). Deviation requires (a) an explicit user instruction in this turn, (b) another HS-* firing, or (c) a cited `.reinguard/policy/` clause, and must name that authority in chat. Wall-clock, turn count, response length, or "feels long" are not grounds.

## HS-NO-DISMISS

Never dismiss diagnostics as "pre-existing" or "outside diff range." Every error, warning, and finding reported by quality gates (`go test`, `go vet`, `golangci-lint`, markdownlint) or review bots (CodeRabbit, human reviewers) is a defect to address — regardless of when it was introduced or whether it falls within the current diff. The agent must not classify, skip, or deprioritize findings based on authorship or diff scope. If a gate or reviewer reports a problem, the agent resolves it or dispositions it (per `review--consensus-protocol.md`) before proceeding.

Enforcement tier: **Steering** (agent self-policing).

## HS-REVIEW-RESOLVE

Never resolve a review thread — and never treat a **non-thread review finding** (outside-diff-range comment, PR summary finding, conversation-level comment) as addressed — without a disposition reply (Fixed / By design / False positive / Acknowledged). For thread-based findings, resolve the thread after consensus. For non-thread findings, post a PR conversation comment with the quoted finding and disposition. Branch Protection **Require conversation resolution** enforces thread resolution before merge; agent discipline must also cover non-thread findings that GitHub cannot track. See `AGENTS.md` and `.reinguard/policy/review--consensus-protocol.md` for the full consensus model.

## HS-MERGE-CONSENSUS

Do **not** merge a PR (any method — direct `gh pr merge`, `gh pr merge --auto`, or manual merge) while required bot review is not terminal, has failed, is stale, review threads are unresolved, or review consensus has not been reached per `review--consensus-protocol.md`. Before merge, confirm **all** of:

- CI green (`ci-pass`)
- Required bot review is **terminal** (not pending, rate-limited, or paused)
- Required bot review has **not failed**
- Required bot review is **not stale**
- Unresolved review threads == 0
- Consensus reached on all findings (per `.reinguard/policy/review--consensus-protocol.md`)

## Related

- `.reinguard/policy/review--consensus-protocol.md` — disposition and CodeRabbit resolution gate
- `.reinguard/policy/catalog.yaml` — policy index
