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
   - `guards` (e.g. `merge-readiness` for summaries during **Execute**)
   - `knowledge.entries` (filtered aids)

## Route

**Normative mapping:** [ADR-0013 § 4](../../docs/adr/0013-fsm-v1-workflow-states.md) (*Adapter mapping (durable)*). Use ADR-0013 and `.reinguard/control/` as SSOT for state and route semantics.

The table below is a **Cursor-facing heuristic** (when `state.kind` is `resolved`) with the same procedure targets as ADR-0013 § 4:

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

## Propose

After **Sense** and **Route**, present the full-path proposal **exactly once** per run, then wait for approval. There is no alternate mode (no proposal-only run and no stopping after a minimal state dump).

1. Trace forward from the current `state_id` using [ADR-0013 § 4](../../docs/adr/0013-fsm-v1-workflow-states.md) through **Per-unit Definition of Done** in [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md).
2. Follow **`next-orchestration.md` § Full-path proposal format** in full: current position, ordered remainder, gaps, completion condition.
3. Obtain **single explicit user approval** per **`next-orchestration.md` § Approval gate** (unit identity, ordered remainder, completion condition). **No per-procedure re-approval** after this gate.

**Output (for agents):** User-visible text must be the full-path proposal — including unit identity, ordered procedures through DoD, gaps, and completion condition — not merely `state_id` / `route_id` bullets. **Do not** end the turn after only a short status line. Do **not** replace the **Output** sections inside each procedure file; those remain authoritative per procedure front matter.

## Execute

After approval, **always** follow [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Post-approval execution contract** and § **Loop semantics**: drive to Per-unit DoD **without** user prompts that gate progress between iterations.

Loop (summary): **Sense** (`rgd context build`) → **Route** (ADR-0013 § 4; same rules as § Route above) → run mapped procedure(s) → **Refresh** context after material changes — per `next-orchestration.md`.

**Output (for agents):** After each `rgd context build` in the loop, emit a short paragraph: `state_id` / `route_id` / guard summary and which procedure(s) ran or are next (iteration label optional, e.g. “Pass 2”). **Final:** DoD satisfied with evidence, or allowed stop with evidence (`next-orchestration.md` § Post-approval execution contract).

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-v1-workflow-states.md`](../../docs/adr/0013-fsm-v1-workflow-states.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
