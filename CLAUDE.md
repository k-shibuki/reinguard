# reinguard bridge (Claude Code Adapter)

This file is a **concrete Adapter** for Claude Code. It points at the Semantics
layer (`.reinguard/`) and Substrate (`rgd`). Do not duplicate body text from
`.reinguard/` or review guidelines from [`AGENTS.md`](AGENTS.md).

## Shared hub

Review norms, project context, and the Adapter taxonomy live in
[`AGENTS.md`](AGENTS.md). Read that file when acting as a reviewer; this bridge
covers Claude-specific workflow entry only.

## Entry points

- **Substrate config:** [`.reinguard/reinguard.yaml`](.reinguard/reinguard.yaml) — `schema_version`, `default_branch`, `providers` (ADR-0008, ADR-0009).
- **Label SSOT:** [`.reinguard/labels.yaml`](.reinguard/labels.yaml) — GitHub label categories and `commit_prefix` (ADR-0008).
- **Knowledge catalog:** [`.reinguard/knowledge/manifest.json`](.reinguard/knowledge/manifest.json) — `entries[]` with `id`, `path`, `description`, `triggers`, `when` (ADR-0010).
- **Policy catalog:** [`.reinguard/policy/catalog.yaml`](.reinguard/policy/catalog.yaml) — normative documents; bodies under [`.reinguard/policy/`](.reinguard/policy/).
- **Control catalog:** [`.reinguard/control/catalog.yaml`](.reinguard/control/catalog.yaml) — human-maintained index; rule YAML under [`.reinguard/control/states/`](.reinguard/control/states/), [routes/](.reinguard/control/routes/), [guards/](.reinguard/control/guards/).
- **Procedures:** [`.reinguard/procedure/`](.reinguard/procedure/) — Claude: [`.claude/commands/rgd-next.md`](.claude/commands/rgd-next.md), [`.claude/commands/claude-plan.md`](.claude/commands/claude-plan.md).

## Always-active policy

Follow these policy documents in every interaction (path references only):

- [safety--agent-invariants.md](.reinguard/policy/safety--agent-invariants.md) — **HARD STOPS (HS-*)**
- [coding--standards.md](.reinguard/policy/coding--standards.md) — English artifacts, change scope, ADR/CLI authority
- [workflow--pr-discipline.md](.reinguard/policy/workflow--pr-discipline.md) — Issue-driven work, PR discipline
- [commit--format.md](.reinguard/policy/commit--format.md) — Conventional Commits, `Refs` in message body

## Workflow

Follow [`.reinguard/policy/workflow--pr-discipline.md`](.reinguard/policy/workflow--pr-discipline.md). Procedure mapping is defined in each procedure's YAML front matter (`applies_to`) under [`.reinguard/procedure/`](.reinguard/procedure/); [ADR-0013](docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) § 4 documents the mechanism.

**Claude workflow entry:** [`.claude/commands/rgd-next.md`](.claude/commands/rgd-next.md) — Sense (`rgd context build`), Route, Propose, Execute per [`.reinguard/procedure/next-orchestration.md`](.reinguard/procedure/next-orchestration.md).

**Planning entry:** [`.claude/commands/claude-plan.md`](.claude/commands/claude-plan.md) — research and design interrogation before implementation; execution handoff uses `rgd-next` loop semantics.

## GitHub and git observation

- **Primary:** `rgd observe` and `rgd context build` — see [`docs/cli.md`](docs/cli.md), ADR-0006, ADR-0009.
- **Supplementary:** `gh` and `git` for read-only facts when needed (ADR-0006).

## Knowledge retrieval

See [`docs/cli.md`](docs/cli.md) (`rgd context build`, `knowledge pack`). After changing knowledge front matter: `rgd knowledge index`, commit `manifest.json`, then `rgd config validate`.

## Normative references

- [ADR-0001](docs/adr/0001-system-positioning.md) (Adapter / Semantics / Substrate)
- [ADR-0013](docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) (FSM states, routes, Adapter mapping)
- [`docs/cli.md`](docs/cli.md)
