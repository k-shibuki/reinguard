# AGENTS.md

Configuration for AI reviewers (e.g. CodeRabbit) and agents using this repository.

- **Semantics (SSOT)**: `.reinguard/` — policy index: [`.reinguard/policy/catalog.yaml`](.reinguard/policy/catalog.yaml); knowledge index: [`.reinguard/knowledge/manifest.json`](.reinguard/knowledge/manifest.json); agent procedures: [`.reinguard/procedure/`](.reinguard/procedure/) (workflow Cursor entry: [`.cursor/commands/rgd-next.md`](.cursor/commands/rgd-next.md); FSM: [ADR-0013](docs/adr/0013-fsm-v1-workflow-states.md)).
- **Adapter (Cursor)**: `.cursor/rules/` and `.cursor/commands/` point at Semantics; see [ADR-0001](docs/adr/0001-system-positioning.md).

## Project context

**reinguard** is a three-layer control system (Adapter / Semantics / Substrate) that stabilizes the
information space for AI agents — not a workflow brain. `rgd` is its Substrate layer: a stateless Go
CLI runtime that computes operational context via observation and declarative rules (see ADR-0001).

Authoritative architecture: [docs/adr/](docs/adr/). Especially relevant for review:

- ADR-0002: spec-driven match rules and evaluators
- ADR-0003: pull-based, stateless CLI
- ADR-0006: GitHub auth via `gh` only (Phase 1)
- ADR-0008: schema versioning and embedded JSON Schema
- ADR-0009: observation engine abstraction
- ADR-0010: repository knowledge format, manifest generation, and agent-facing delivery
- ADR-0011: semantic control plane directory structure (`.reinguard/` layout)

CI: `golangci-lint`, `go vet`, `go test -race`; PRs must pass job **`ci-pass`** (aggregates `gate-policy`, `lint-markdown`, `lint-go`, `test-go`, and dogfood jobs). Branch protection should require **`ci-pass`** and **conversation resolution before merge** — see [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md).

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
- Observation and guards must respect **ADR-0005** (no agent-internal files) and **ADR-0006** (`gh` for GitHub auth).

### Traceability (P1)

- PR body: `Closes #<issue>` (or exception label + `## Exception` per template).
- PR title: Conventional Commits (`<type>(<scope>): …`; types exclude `hotfix` in titles — see `.reinguard/labels.yaml` (`categories.type`, `commit_prefix`).

### Review threads and merge

Before resolving a review thread (required when **Require conversation resolution** is on), leave a
short **disposition**: **Fixed** / **By design** / **False positive** / **Acknowledged** — see
[`.reinguard/policy/review--consensus-protocol.md`](.reinguard/policy/review--consensus-protocol.md)
for the full consensus model and resolution rules.

Do **not** enable **auto-merge** while bot review is still pending or threads are unresolved ([**HS-MERGE-CONSENSUS**](.reinguard/policy/safety--agent-invariants.md)).
