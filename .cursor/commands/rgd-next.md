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
   - `routes[0].kind`, `routes[0].route_id` (when `routes[0].kind` is `resolved`)
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

After **Sense** and **Map**, **always** follow [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) (orchestration SSOT). There is no alternate mode (no proposal-only run and no stopping after a minimal state dump).

1. **Propose** — Trace forward through the routing table from the current `state_id` to **Per-unit Definition of Done** (defined in `next-orchestration.md`). Present the **Full-path proposal format** from that document: current position, ordered remainder, gaps, completion condition. **Do not** end the turn after only a short status line; the Propose block must satisfy `next-orchestration.md` § Full-path proposal format before waiting for approval.
2. **Approve** — Single explicit user approval for the full path and completion condition.
3. **Execute** — Post-approval loop: Sense → Map → follow the mapped procedure → Refresh, per `next-orchestration.md` § Loop semantics. Do not prompt the user between iterations (§ Post-approval execution contract).

## Output (for agents)

- **Propose:** User-visible output is the full-path proposal (see `next-orchestration.md` § Full-path proposal format), including unit identity, ordered procedures through DoD, gaps, and completion condition — not merely `state_id` / `route_id` bullets.
- **Execute loop:** After each `rgd context build` in the loop, emit a short paragraph: `state_id` / `route_id` / guard summary and which procedure(s) ran or are next (iteration label optional, e.g. “Pass 2”).
- **Final:** DoD satisfied with evidence, or allowed stop with evidence (`next-orchestration.md` § Post-approval execution contract).
- Do **not** replace the **Output** sections inside each procedure file; those remain authoritative per procedure front matter.

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-v1-workflow-states.md`](../../docs/adr/0013-fsm-v1-workflow-states.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
