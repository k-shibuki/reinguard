# cursor-plan

Cursor-native **planning** command: exhaust design decisions with **`AskQuestion`** (recursive, with trade-offs explained for every option), then **crystallize** into either **`CreatePlan`** (implementation plan) or **GitHub Issue creation** (Phase 3B). Aligns with Cursor Plan mode: research first, clarify ambiguity, do not edit the repo until the plan is accepted.

Adapter layer: reference Semantics paths only for policy; do not duplicate normative policy body text (ADR-0001).

## Context (open as needed)

- [`.reinguard/policy/workflow--pr-discipline.md`](../.reinguard/policy/workflow--pr-discipline.md) — Issue sections, PR body constraints
- [`.reinguard/policy/coding--standards.md`](../.reinguard/policy/coding--standards.md) — change scope, ADR/CLI authority
- [`.reinguard/policy/safety--agent-invariants.md`](../.reinguard/policy/safety--agent-invariants.md) — HS-*

## Phase 1 — Gather

1. **User goal** — If the prompt is vague, ask one narrowing question before heavy research.
2. **Existing Issue (optional)** — If `#N` is given:

   ```bash
   gh issue view <N> --json title,body,labels,state
   ```

   Derive seed questions from Definition of Done, Touches, Test plan, and constraints.

3. **Codebase** — Use **SemanticSearch**, **Grep**, and **Glob** to locate relevant packages, tests, and config. Prefer evidence over assumptions.

4. **Knowledge (optional)** — If `rgd` is available:

   ```bash
   rgd context build
   ```

   Use `knowledge.entries` paths from stdout JSON; otherwise triage [`.reinguard/knowledge/manifest.json`](../.reinguard/knowledge/manifest.json) by `description` / `triggers` and open only what you need.

## Phase 2 — Interrogate (recursive)

Maintain an internal **decision ledger**: each row is one design choice with status `open` or `resolved`.

1. **Identify open decisions** — From research, list what is still ambiguous (architecture, scope, API shape, migration, test strategy, rollout, etc.).
2. **Explain trade-offs** — Before each **`AskQuestion`**, describe in prose (or bullets) what each option **means**, **pros**, **cons**, and **blast radius**. Do not present bare labels.
3. **Ask in small batches** — Prefer **1–2 questions** per `AskQuestion` call (Plan mode friendly).
4. **Propagate** — New answers may create **new** open decisions; add them to the ledger and loop.
5. **Stop when** — No `open` rows remain (or the user explicitly caps scope — record that as a resolved decision).

## Phase 3 — Crystallize (branch)

Ask one final **`AskQuestion`** (or confirm from context) which output is primary:

| Branch | When | Action |
|--------|------|--------|
| **3A — Implementation plan** | User will implement in-repo next | Call **`CreatePlan`**: concise, file-scoped todos, no repo edits in Plan mode |
| **3B — New Issue** | Work should be tracked as a GitHub Issue first | Follow **Issue creation** below, then stop (or hand off to `rgd-next` / `.reinguard/procedure/implement.md`) |

You may document both in narrative form, but **execute exactly one** primary artifact per run unless the user asked for both explicitly.

### Phase 3B — Create GitHub Issue (inline)

Use when the outcome is **issue-first** tracking (replaces the old `issue-create` command).

1. **Choose template**
   - **Task** — implementation or chore; Conventional Commits title + type label. Starting point: `.github/ISSUE_TEMPLATE/task.yml`.
   - **Epic** — phased parent work; label `epic`. Starting point: `.github/ISSUE_TEMPLATE/epic.yml`.

2. **Build the body file** — Markdown file for `--body-file` with sections required by the template (see `workflow--pr-discipline.md` § Recommended Issue sections). Use real newlines (same constraints as PR bodies in `workflow--pr-discipline.md` § PR body updates).

3. **Choose labels** — SSOT: `.reinguard/labels.yaml`.
   - **Task**: exactly one **type** label (`feat`, `fix`, …) from `categories.type`.
   - **Epic**: label **`epic`** only (no type label).

4. **Pre-flight validation**

   ```bash
   bash .reinguard/scripts/check-issue-policy.sh \
     --title "<title>" \
     --body-file /path/to/issue-body.md \
     --label <feat|…|epic> \
     [--template task|epic]
   ```

   Fix errors until it prints `Issue policy pre-flight OK.`

5. **Create the Issue**

   ```bash
   gh issue create --title "<title>" --body-file /path/to/issue-body.md --label "<label>"
   ```

   For multiple labels: repeat `--label` or `gh issue edit` after create.

**Related:** `.reinguard/scripts/check-issue-policy.sh`, `.reinguard/policy/workflow--pr-discipline.md`, `.reinguard/procedure/implement.md` (branch naming shares `labels.yaml` type vocabulary).

## Guard

- **Plan mode:** Do not modify the workspace for **3A** until the user accepts the plan; use read-only exploration tools only.
- **3B (`gh issue create`)** creates remote state on GitHub — that is **not** a repo file edit; it is allowed when Issue creation is the chosen outcome.
- Prefer **`AskQuestion`** over open-ended “what do you want?” when discrete choices exist.
- Do not claim design decisions are “complete” while any ledger row is still `open` without user acknowledgment.

## Output (for agents)

- Short summary of **resolved decisions** (bullet list).
- Either **CreatePlan** result (3A) or **Issue URL / number** (3B), never neither when the user expected a deliverable.
