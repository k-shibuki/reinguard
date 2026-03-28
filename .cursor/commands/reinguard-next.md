# reinguard-next

Thin orchestrator: use **substrate** output to pick Semantics procedures. This command does **not** embed workflow logic that duplicates `rgd` state/route resolution (ADR-0001).

## Sense

1. From repo root (or pass `--cwd` / `--config-dir` consistent with other `rgd` commands):

   ```bash
   rgd context build
   ```

2. Parse stdout JSON:
   - `state.state_id`, `state.kind`
   - `routes[0].route_id` (when `kind` is `resolved`)
   - `knowledge.entries` (filtered aids)

## Map (v1)

Use ADR-0013 and `.reinguard/control/` as SSOT. Heuristic table (when `state.kind` is `resolved`):

| `state_id` | Open procedure |
|------------|----------------|
| `working_no_pr` | `.reinguard/procedure/implement.md` (or `pr-create.md` when opening a PR) |
| `pr_open` | `.reinguard/procedure/review-address.md` |
| `changes_requested` | `.reinguard/procedure/review-address.md` |
| `bot_rate_limited` / `bot_review_paused` | `.reinguard/knowledge/review--bot-operations.md` (and bot-specific docs from `knowledge.entries`) |
| `ready_to_merge` | `.cursor/commands/pr-merge.md` |

When `state.kind` is not `resolved`, follow ADR-0007 handoff: gather observation diagnostics, fix config or observation, re-run `context build` — do not invent a winning state.

## Output (for agents)

- One short paragraph: current `state_id` / `route_id` / guard summary from JSON.
- Bullet list: which procedure file(s) to open next (repo-relative paths).
- Do **not** replace the **Output** sections inside each procedure; those remain authoritative per procedure front matter.

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-v1-workflow-states.md`](../../docs/adr/0013-fsm-v1-workflow-states.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
