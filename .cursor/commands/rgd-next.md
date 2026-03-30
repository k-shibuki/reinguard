# rgd-next

Single Cursor entry for workflow procedures: use **substrate** output to pick Semantics docs. This command does **not** embed logic that duplicates `rgd` state/route resolution (ADR-0001).

## Sense

1. From repo root (or pass `--cwd` / `--config-dir` consistent with other `rgd` commands):

   ```bash
   rgd context build
   ```

   If `rgd` is not on `PATH`, from repo root: `go run ./cmd/rgd context build` (same flags).

2. Parse stdout JSON:
   - `state.state_id`, `state.kind`
   - `routes[0].route_id` (when `kind` is `resolved`)
   - `knowledge.entries` (filtered aids)

## Map (v2)

Use ADR-0013 and `.reinguard/control/` as SSOT. Heuristic table (when `state.kind` is `resolved`):

| `state_id` | Open procedure |
|------------|----------------|
| `working_no_pr` | `.reinguard/procedure/implement.md` (or `pr-create.md` when opening a PR) |
| `pr_open` | `.reinguard/procedure/review-address.md` (residual monitor) |
| `waiting_ci` | `.reinguard/procedure/review-address.md` (checks / mergeability) |
| `unresolved_threads` | `.reinguard/procedure/review-address.md` (thread disposition) |
| `changes_requested` | `.reinguard/procedure/review-address.md` (formal “Request changes” on the PR) |
| `waiting_bot_run` | `.reinguard/procedure/wait-bot-review.md` (+ `review--bot-operations.md` from `knowledge.entries`) |
| `waiting_bot_rate_limited` | `.reinguard/procedure/wait-bot-review.md` |
| `waiting_bot_paused` | `.reinguard/procedure/wait-bot-review.md` |
| `waiting_bot_failed` | `.reinguard/procedure/wait-bot-review.md` |
| `merge_ready` | `.reinguard/procedure/pr-merge.md` |

**Dirty working tree + `review-address`:** When `observation.signals.git.working_tree_clean` is `false` and the resolved procedure is `review-address`, run **Step 0** in that procedure first (`change-inspect` → commit → refresh context). See `.reinguard/knowledge/review--incremental-fix-flow.md`.

When `state.kind` is not `resolved`, follow ADR-0007 handoff: gather observation diagnostics, fix config or observation, re-run `context build` — do not invent a winning state.

## Orchestrate

By default, **Sense → Map → Output** is one shot (proposal-only). When the user wants **autonomous execution**, follow [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) (orchestration SSOT):

1. **Propose** — After Map, trace forward through the routing table from the current `state_id` to **Per-unit Definition of Done** (defined in `next-orchestration.md`). Present the ordered sequence of procedures, known gaps, and completion condition.
2. **Approve** — Single explicit user approval for the full path.
3. **Execute** — Post-approval loop: Sense → Map → follow the mapped procedure → Refresh, per `next-orchestration.md` § Loop semantics. Do not prompt the user between iterations (§ Post-approval execution contract).

If the user does not request autonomous execution, produce **Output** only (one-shot report).

## Output (for agents)

- One short paragraph: current `state_id` / `route_id` / guard summary from JSON.
- Bullet list: which procedure file(s) to open next (repo-relative paths).
- Under **Orchestrate**, emit the short state summary **after each** `rgd context build` in the loop (iteration label optional: e.g. “Pass 2”). Final output: DoD status or stop reason with evidence.
- Do **not** replace the **Output** sections inside each procedure; those remain authoritative per procedure front matter.

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-v1-workflow-states.md`](../../docs/adr/0013-fsm-v1-workflow-states.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
