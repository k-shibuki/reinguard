---
id: safety-agent-invariants
description: HS-* safety codes for agents — CI merge, PR template/base, local verify, review resolve, merge consensus
triggers:
  - HS-CI-MERGE
  - HS-PR-TEMPLATE
  - HS-PR-BASE
  - HS-LOCAL-VERIFY
  - HS-NO-SKIP
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

- `npx --yes markdownlint-cli2@latest '**/*.md'` (or `pre-commit run markdownlint-cli2 --all-files`)

## HS-NO-SKIP

Do not skip verification steps or merge with failing checks without an explicit documented exception.

## HS-REVIEW-RESOLVE

Never resolve a review thread without a **disposition reply** (Fixed / By design / False positive / Acknowledged) when substantive review comments exist. Branch Protection **Require conversation resolution** enforces resolution before merge; agent discipline must match. See `AGENTS.md` and `.reinguard/policy/review--consensus-protocol.md` for the full consensus model.

## HS-MERGE-CONSENSUS

Do **not** enable **auto-merge** while bot review is pending or review threads are unresolved. Confirm CI green and merge policy (`ci-pass`) before merge.

## Related

- `.reinguard/policy/review--consensus-protocol.md` — disposition and CodeRabbit resolution gate
- `.reinguard/policy/catalog.yaml` — policy index
