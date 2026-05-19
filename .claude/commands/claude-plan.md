# claude-plan

Claude Code **planning** entry: exhaust design decisions with clarifying
questions (recursive, with trade-offs explained for every option), then
crystallize into a single written plan before editing the repo.

When the outcome is **issue-first** (GitHub Issue before implementation), embed
the Issue-creation checklist in the plan todos so acceptance is a single gate,
then execute those steps after the user accepts.

Adapter layer: reference Semantics paths only for policy; do not duplicate
normative policy body text (ADR-0001). Shared review norms: [`AGENTS.md`](../../AGENTS.md).

## Context (open as needed)

- [`.reinguard/policy/workflow--pr-discipline.md`](../../.reinguard/policy/workflow--pr-discipline.md) — Issue sections, PR body constraints
- [`.reinguard/policy/coding--standards.md`](../../.reinguard/policy/coding--standards.md) — change scope, ADR/CLI authority
- [`.reinguard/policy/safety--agent-invariants.md`](../../.reinguard/policy/safety--agent-invariants.md) — HS-*

## Phase 1 — Gather

1. **User goal** — If the prompt is vague, ask one narrowing question before heavy research.
2. **Existing Issue (optional)** — If `#N` is given:

   ```bash
   gh issue view <N> --json title,body,labels,state
   ```

3. **Codebase** — Search the repository for relevant packages, tests, and config. Prefer evidence over assumptions.
4. **Knowledge** — When building an implementation plan:

   ```bash
   rgd context build
   ```

   Use `knowledge.entries` from stdout JSON. Otherwise triage [`.reinguard/knowledge/manifest.json`](../../.reinguard/knowledge/manifest.json) by `description` / `triggers`.

## Phase 2 — Interrogate (recursive)

Maintain an internal **decision ledger**: each row is one design choice with status `open` or `resolved`.

1. **Identify open decisions** — architecture, scope, API shape, migration, test strategy, rollout, etc.
2. **Explain trade-offs** — before each question batch, describe what each option means, pros, cons, and blast radius.
3. **Ask in small batches** — prefer 1–2 questions at a time.
4. **Propagate** — new answers may create new open decisions; loop until resolved or user caps scope.

## Phase 3 — Crystallize (single plan)

**Single output:** one written plan per run. Choose the plan shape from context:

| Plan shape | When | Plan must contain |
|------------|------|-------------------|
| **Implementation** | Next step is in-repo code/docs/tests | Overview, file-scoped todos, test/preflight hints |
| **Issue-first** | Next step is a new GitHub Issue | Same as above, plus Phase 3B Issue-creation todos |

Do not create Issues or edit the repo until the user accepts the plan.

### Phase 3B — Issue creation (embed in plan todos)

When **Issue-first**, embed end-to-end Issue creation in plan todos. Required sections, labels, templates: [`.reinguard/policy/workflow--pr-discipline.md`](../../.reinguard/policy/workflow--pr-discipline.md). Labels: [`.reinguard/labels.yaml`](../../.reinguard/labels.yaml). Script: [`.reinguard/scripts/check-issue-policy.sh`](../../.reinguard/scripts/check-issue-policy.sh).

## Guard

- **Plan mode:** Do not modify the workspace until the user accepts the plan; read-only exploration only.
- **Execution handoff:** After plan acceptance, follow [`CLAUDE.md`](../../CLAUDE.md) § **Workflow** (`rgd-next` + `next-orchestration.md` loop).
- Issue creation is **never** the standalone output — only steps inside the accepted plan.

## Output

- Summary of **resolved decisions** in the plan body.
- **Exactly one plan artifact** per run. For Issue-first work, final todo outcome is the new Issue URL/number.
