## Context

Dogfood: add real `.reinguard/rules/` (and knowledge manifest) for this repo; golden tests; **GitHub Actions** on **GitHub-hosted** runners for **public OSS** free tier. Not self-hosted runners.

## Refs: ADR

- ADR-0001–0008 as applicable

## ADR Impact

`none`

## Acceptance ↔ ADR

- CI matches plan: two-tier workflows — (1) all PRs: build, lint, `go test` (mocked), **module coverage ≥80%**; (2) live API: run on `pull_request` **only if** `github.event.pull_request.head.repo.full_name == github.repository`, and on `push` to `main`; **skip (2) on fork PRs**. Use `gh` + `GITHUB_TOKEN` only. Optional `workflow_dispatch`.

## Definition of Done

- [ ] `.reinguard/` committed with meaningful Phase 1 rules (can be minimal but exercised by tests)
- [ ] Golden or integration test for `rgd context build` on this repo (with mocks where API not available)
- [ ] `.github/workflows` updated; **fork PR** does not fail due to missing secrets for live API job (job skipped or not run)
- [ ] `CONTRIBUTING.md` explains fork vs upstream PR and live API behavior
- [ ] English only

## Test Plan

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-------|
| TC-N-01 | CI workflow YAML `if:` for fork | Normal | fork PR skips live API job | Grep/assert in review |
| TC-N-02 | `push` to `main` | Normal | live API job runs | — |

## Dependencies

- Depends on: #21

## Notes

Do not add Makefile per plan. Keep jobs within public Actions free-tier expectations (minimize API volume).
