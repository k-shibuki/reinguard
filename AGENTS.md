# AGENTS.md

Configuration for AI reviewers (e.g. CodeRabbit) and agents using this repository. Cursor
policies live in `.cursor/rules/` and `.cursor/commands/`.

## Project context

**reinguard** is a **Go** project: spec-driven control-plane **substrate** (`rgd` CLI) that builds
operational context via observation and declarative rules — not a workflow brain (see ADR-0001).

Authoritative architecture: [docs/adr/](docs/adr/). Especially relevant for review:

- ADR-0002: spec-driven match rules and evaluators
- ADR-0003: pull-based, stateless CLI
- ADR-0006: GitHub auth via `gh` only (Phase 1)
- ADR-0008: schema versioning and embedded JSON Schema
- ADR-0009: observation engine abstraction

CI: `golangci-lint`, `go vet`, `go test -race`; PRs must pass job **`ci-pass`** (aggregates `go-ci` and `check-policy`). Branch protection should require **`ci-pass`** and **conversation resolution before merge** — see [docs/contributing.md](docs/contributing.md).

## Review guidelines

### Severity (flag P0 / P1 only)

- **P0 (blocking)**: logic bugs, incorrect control flow, security (auth, secrets, unsafe boundaries),
  missing or broken tests for changed non-trivial code.
- **P1 (significant)**: ADR drift, missing traceability (`Closes #N` in PR body, `Refs: #N` in commits),
  weak error handling, incomplete boundary tests for new APIs.
- Do **not** duplicate **gofmt** / **golangci-lint** / style nits already enforced in CI.

### Go and tests

- Table-driven tests where appropriate; use `t.Parallel()` when safe.
- Table tests: cover success, expected errors, and boundary inputs when meaningful.
- Exported functions and CLI behavior changes should have tests unless trivial wiring.

### Traceability (P1)

- PR body: `Closes #<issue>` (or exception label + `## Exception` per template).
- PR title: Conventional Commits (`<type>(<scope>): …`; types exclude `hotfix` in titles — see `tools/commit-types.txt`).

### Review threads and merge (aligns with bridle-class repos)

Before resolving a review thread (required when **Require conversation resolution** is on), leave a
short **disposition**: **Fixed** / **By design** / **False positive** / **Acknowledged** — see the
[bridle consensus protocol](https://github.com/bridle-org/bridle/blob/main/.cursor/knowledge/review--consensus-protocol.md)
for the model reinguard aims to stay compatible with for future `rgd` merge-readiness semantics.

Do **not** enable **auto-merge** while bot review is still pending or threads are unresolved.

## Long-term alignment

reinguard intends to **observe and evaluate** bridle-class workflows via `rgd`; keeping the same
merge and review **invariants** here validates the substrate against real guard semantics.
