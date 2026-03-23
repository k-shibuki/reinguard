---
name: Task
about: Implementation or chore with Definition of Done and test plan
title: "type(scope): short imperative summary (Conventional Commits)"
labels: []
---

## Context

<!-- Why is this needed? -->

## Refs: ADR

<!-- e.g. ADR-0002, ADR-0008, or `none` for pure process -->

## ADR Impact

<!-- One of: none | new | amend (file name) — brief note -->

## Acceptance ↔ ADR

<!-- How you will know the change matches the ADRs cited above -->

## Definition of Done

<!-- Bullet list of completion criteria -->

-

## Test plan

Use a **test perspectives table** when helpful (Normal / Abnormal / Boundary).

| Case ID | Input / Precondition | Perspective | Expected Result | Notes |
|---------|---------------------|-------------|-----------------|-----|
| TC-N-01 | | Normal | | |

**Go checks (always):**

- `go test ./...`
- `go vet ./...`
- `golangci-lint run` (or CI)

## Linked issues

<!-- For PR: Closes #N -->

## Notes

<!-- Optional: risks, follow-ups -->
