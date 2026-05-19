# AGENTS.md

**agents.md hub** for tools that read `AGENTS.md` directly (for example Codex,
CodeRabbit, and other agents.md-native clients). Shared cross-agent metadata:
project context, review guidelines, and path-only entry to Semantics and
Substrate. Client-specific workflow bridges live under concrete Adapter
surfaces (see **Client adapters** below).

## Architecture

**reinguard** provides repo-owned semantic control for agentic development:
shared operational context for agents, humans, and the bots reviewing their
work. Its architecture remains three-layered (Adapter / Semantics / Substrate):
repository-owned Semantics define meaning, and `rgd` is the Substrate layer, a
stateless Go CLI runtime that computes operational context via observation and
declarative rules (see [ADR-0001](docs/adr/0001-system-positioning.md)).

Authoritative architecture: [docs/adr/](docs/adr/). Especially relevant for review:

- ADR-0002: spec-driven match rules and evaluators
- ADR-0003: pull-based, stateless CLI
- ADR-0006: GitHub auth via `gh` only (Phase 1)
- ADR-0008: schema versioning and embedded JSON Schema
- ADR-0009: observation engine abstraction
- ADR-0010: repository knowledge format, manifest generation, and agent-facing delivery
- ADR-0011: semantic control plane directory structure (`.reinguard/` layout)
- ADR-0013: FSM workflow states and Adapter mapping
- ADR-0014: runtime gate artifacts for local proof
- ADR-0015: Adapter-local execute resume artifact

CI: `golangci-lint`, `go vet`, `go test -race`; PRs must pass job **`ci-pass`** (aggregates `gate-policy`, `lint-markdown`, `lint-go`, `test-go`, and dogfood jobs). Branch protection should require **`ci-pass`** and **conversation resolution before merge** — see [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md).

## Semantics entry points

Path-only references to the Semantics layer (SSOT). Do not duplicate policy or
procedure bodies here.

- **Policy catalog:** [`.reinguard/policy/catalog.yaml`](.reinguard/policy/catalog.yaml)
- **Knowledge catalog:** [`.reinguard/knowledge/manifest.json`](.reinguard/knowledge/manifest.json)
- **Agent procedures:** [`.reinguard/procedure/`](.reinguard/procedure/) — FSM mapping mechanism: [ADR-0013](docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md)

## Substrate (`rgd`)

Operational context, observation, guards, and knowledge filtering are computed
by the Substrate CLI. Command surface: [`docs/cli.md`](docs/cli.md).

## Review guidelines

Shared review norms for AI reviewers and agents. CodeRabbit applies these via
the agents.md convention; concrete Adapters reference this section rather than
duplicating it.

### Finding scope

- Flag logic bugs, incorrect control flow, security (auth, secrets, unsafe boundaries),
  missing or broken tests for changed non-trivial code.
- Flag ADR drift, missing traceability (`Closes #N` in PR body, `Refs: #N` in commits),
  weak error handling, and incomplete boundary tests for new APIs.
- Do **not** duplicate **gofmt** / **golangci-lint** / style nits already enforced in CI.

### Go and tests

- Table-driven tests where appropriate; use `t.Parallel()` when safe.
- Table tests: cover success, expected errors, and boundary inputs when meaningful.
- Exported functions and CLI behavior changes should have tests unless trivial wiring.
- Observation and guards must respect **ADR-0005** (no agent-internal files) and **ADR-0006** (`gh` for GitHub auth).

### Traceability

- PR body: `Closes #<issue>` (or exception label + `## Exception` per template).
- PR title: Conventional Commits (`<type>(<scope>): …`; types exclude `hotfix` in titles — see `.reinguard/labels.yaml` (`categories.type`, `commit_prefix`).

### Review threads and merge

Normative: [`.reinguard/policy/review--consensus-protocol.md`](.reinguard/policy/review--consensus-protocol.md) (dispositions, thread and non-thread resolution, [**HS-REVIEW-RESOLVE**](.reinguard/policy/safety--agent-invariants.md)); [`.reinguard/policy/review--disposition-categories.md`](.reinguard/policy/review--disposition-categories.md) (classification for local review, self-inspection, and PR review).

[**HS-NO-DISMISS**](.reinguard/policy/safety--agent-invariants.md) · [**HS-MERGE-CONSENSUS**](.reinguard/policy/safety--agent-invariants.md)

## Client adapters

| Surface | Primary consumers | Role |
|---------|-------------------|------|
| `AGENTS.md` | Codex, CodeRabbit, agents.md tools | **Hub** (this file): review guidelines, project context, Semantics/Substrate entry |
| `.coderabbit.yaml` | CodeRabbit | PR review binding and path-specific bot instructions; review norms in `AGENTS.md` |
| `.cursor/` | Cursor | **Concrete** Adapter: [`reinguard-bridge.mdc`](.cursor/rules/reinguard-bridge.mdc), [`rgd-next`](.cursor/commands/rgd-next.md), [`cursor-plan`](.cursor/commands/cursor-plan.md) |
| [`CLAUDE.md`](CLAUDE.md) / [`.claude/`](.claude/) | Claude Code | **Concrete** Adapter: bridge + workflow entry |

Codex may consume this hub without a dedicated concrete Adapter surface;
sufficiency validation is tracked separately ([#163](https://github.com/k-shibuki/reinguard/issues/163)).

## Workflow by client

- **Cursor:** [`.cursor/commands/rgd-next.md`](.cursor/commands/rgd-next.md) — Sense (`rgd context build`), Route, Propose, Execute per [`.reinguard/procedure/next-orchestration.md`](.reinguard/procedure/next-orchestration.md). Planning: [`.cursor/commands/cursor-plan.md`](.cursor/commands/cursor-plan.md).
- **Claude Code:** [`.claude/commands/rgd-next.md`](.claude/commands/rgd-next.md) — same Semantics loop; see [`CLAUDE.md`](CLAUDE.md). Planning: [`.claude/commands/claude-plan.md`](.claude/commands/claude-plan.md).
- **agents.md-native (no workflow client):** run `rgd context build`, open the mapped procedure under [`.reinguard/procedure/`](.reinguard/procedure/) per [ADR-0013](docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) § 4.
