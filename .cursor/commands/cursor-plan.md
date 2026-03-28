# cursor-plan

Cursor-native **planning** command: exhaust design decisions with **`AskQuestion`** (recursive, with trade-offs explained for every option), then **crystallize** with **`CreatePlan` only**. Aligns with Cursor Plan mode: research first, clarify ambiguity, do not edit the repo until the plan is accepted.

When the outcome is **issue-first** (GitHub Issue before implementation), do **not** run Issue creation outside the plan artifact: **weave the Phase 3B checklist into the `CreatePlan`** (overview + todos) so acceptance is a single gate, then execute those steps after the user accepts.

Adapter layer: reference Semantics paths only for policy; do not duplicate normative policy body text (ADR-0001).

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

   Derive seed questions from Definition of Done, Touches, Test plan, and constraints.

3. **Codebase** — Use **SemanticSearch**, **Grep**, and **Glob** to locate relevant packages, tests, and config. Prefer evidence over assumptions.

4. **Knowledge (optional)** — If `rgd` is available:

   ```bash
   rgd context build
   ```

   Use `knowledge.entries` paths from stdout JSON; otherwise triage [`.reinguard/knowledge/manifest.json`](../../.reinguard/knowledge/manifest.json) by `description` / `triggers` and open only what you need.

## Phase 2 — Interrogate (recursive)

Maintain an internal **decision ledger**: each row is one design choice with status `open` or `resolved`.

1. **Identify open decisions** — From research, list what is still ambiguous (architecture, scope, API shape, migration, test strategy, rollout, etc.).
2. **Explain trade-offs** — Before each **`AskQuestion`**, describe in prose (or bullets) what each option **means**, **pros**, **cons**, and **blast radius**. Do not present bare labels.
3. **Ask in small batches** — Prefer **1–2 questions** per `AskQuestion` call (Plan mode friendly).
4. **Propagate** — New answers may create **new** open decisions; add them to the ledger and loop.
5. **Stop when** — No `open` rows remain (or the user explicitly caps scope — record that as a resolved decision).

## Phase 3 — Crystallize (`CreatePlan` only)

**Single output:** call **`CreatePlan`** once per run. Choose the **plan shape** from context (use **`AskQuestion`** if still ambiguous):

| Plan shape | When | What `CreatePlan` must contain |
|------------|------|--------------------------------|
| **Implementation** | Next step is in-repo code/docs/tests | Concise overview, file-scoped todos, test/preflight hints per repo policy |
| **Issue-first** | Next step is a **new** GitHub Issue (then later `rgd-next` / `implement`) | Same as above, but **embed Phase 3B** (below) as explicit todos and paste-ready title/body/label notes in the overview |

Do not emit a second parallel artifact (e.g. “I’ll create the Issue now” without a plan). **Issue creation runs only after the user accepts the plan**, following the todos you put in `CreatePlan`.

### Phase 3B — Issue creation (content to embed in `CreatePlan`)

When the plan shape is **Issue-first**, include these steps **inside** the `CreatePlan` todo list (and enough detail in the overview that an executor needs no guesswork):

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

- **Plan mode:** Do not modify the workspace until the user accepts **`CreatePlan`**; use read-only exploration tools only while planning.
- **`CreatePlan` is mandatory** for this command; Issue creation is **never** the standalone output—only steps **inside** the accepted plan.
- Prefer **`AskQuestion`** over open-ended “what do you want?” when discrete choices exist.
- Do not claim design decisions are “complete” while any ledger row is still `open` without user acknowledgment.

## Output (for agents)

- Short summary of **resolved decisions** (bullet list) **inside** the plan body or preamble you pass to **`CreatePlan`**.
- **Exactly one `CreatePlan` invocation** per run. For Issue-first work, that plan’s todos must include Phase 3B end-to-end (through `gh issue create` and reporting the new Issue URL/number as the final todo outcome).
