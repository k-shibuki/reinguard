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

Before pushing Go changes:

- `go test ./...`
- `go vet ./...`
- `golangci-lint run` (or rely on CI, but local run is strongly preferred)

Before pushing Markdown changes:

- `pre-commit run markdownlint-cli2 --all-files` (pinned hook version from `.pre-commit-config.yaml`)

## HS-NO-SKIP

Do not skip verification steps or merge with failing checks without an explicit documented exception.

## HS-NO-DISMISS

Never dismiss diagnostics as "pre-existing" or "outside diff range." Every error, warning, and finding reported by quality gates (`go test`, `go vet`, `golangci-lint`, markdownlint) or review bots (CodeRabbit, human reviewers) is a defect to address — regardless of when it was introduced or whether it falls within the current diff. The agent must not classify, skip, or deprioritize findings based on authorship or diff scope. If a gate or reviewer reports a problem, the agent resolves it or dispositions it (per `review--consensus-protocol.md`) before proceeding.

Enforcement tier: **Steering** (agent self-policing).

## HS-REVIEW-RESOLVE

Never resolve a review thread — and never treat a **non-thread review finding** (outside-diff-range comment, PR summary finding, conversation-level comment) as addressed — without a disposition reply (Fixed / By design / False positive / Acknowledged). For thread-based findings, resolve the thread after consensus. For non-thread findings, post a PR conversation comment with the quoted finding and disposition. Branch Protection **Require conversation resolution** enforces thread resolution before merge; agent discipline must also cover non-thread findings that GitHub cannot track. See `AGENTS.md` and `.reinguard/policy/review--consensus-protocol.md` for the full consensus model.

## HS-MERGE-CONSENSUS

Do **not** merge a PR (any method — direct `gh pr merge`, `gh pr merge --auto`, or manual merge) while required bot review is not terminal, review threads are unresolved, or consensus has not been reached. Confirm CI green and merge policy (`ci-pass`) before merge.

## Related

- `.reinguard/policy/review--consensus-protocol.md` — disposition and CodeRabbit resolution gate
- `.reinguard/policy/catalog.yaml` — policy index
