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

## Autonomy (extended run)

By default, **Sense → Map → Output** is one shot. When the user wants a **single upfront approval** and then continuous progress, use this loop. **Procedure bodies** (`.reinguard/procedure/*.md`) and policy (**HS-***) remain authoritative; this section only defines **how far** to go without asking again.

### 1. Scope checkpoint (once, before `Act`)

Ask the user to pick **one** autonomy scope and record it. Examples:

| Scope | Stop when |
|-------|-----------|
| `single_pass` | After one complete **Act** for the current mapped procedure and a **final** `rgd context build` (report new `state_id` / guards). |
| `to_merge_ready` | Re-observe after each push or batch of thread work; keep mapping procedures until `state_id` is `merge_ready` and `guards` allow merge prep — **do not** run `pr-merge` unless scope includes merge. |
| `through_merge` | Same as `to_merge_ready`, then execute `.reinguard/procedure/pr-merge.md` only when `merge_ready` **and** the user included merge in this scope (treat merge as opt-in even inside “full” runs if they only asked for review). |

If the user does not choose, default to **`single_pass`**.

Do **not** widen scope (e.g., from `single_pass` to `through_merge`) without a **new** explicit approval.

### 2. Loop (until scope satisfied or hard stop)

Repeat until the chosen scope’s stop condition is met, or a stop trigger fires:

1. **Sense** — `rgd context build` (same cwd / `--config-dir` as above).
2. **Parse** — `state`, `routes[0]`, `guards`, `knowledge.entries`; emit the short **Output** summary each iteration so the user can follow along in the transcript.
3. **Map** — Use the table in **Map (v2)**; if `state.kind` is not `resolved`, follow ADR-0007 (fix / handoff); do not assume a resolved route.
4. **Act** — Open the mapped procedure file(s) and follow their **Act** / **Output** / `done_when` sections only (no shortcutting HS-*).
5. **Refresh** — After any **material** remote or local change (push, merge, thread resolve batch, bot re-review trigger when procedure says so), run **`rgd context build` again** before the next Map. Prefer refresh over stale `state_id`.

**Stop triggers (always):** degraded observation you cannot fix; scope endpoint reached; an **HS-*** would be violated; procedure **escalate_when** matches; user-only action outside agent capability (e.g., org setting).

**Dirty tree + `review-address`:** Still apply the **Map (v2)** note: Step 0 in `review-address` before disposition-heavy work.

## Output (for agents)

- One short paragraph: current `state_id` / `route_id` / guard summary from JSON.
- Bullet list: which procedure file(s) to open next (repo-relative paths).
- Under **Autonomy**, repeat this block **after each** `rgd context build` in the loop (iteration label optional: e.g. “Pass 2”).
- Do **not** replace the **Output** sections inside each procedure; those remain authoritative per procedure front matter.

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-v1-workflow-states.md`](../../docs/adr/0013-fsm-v1-workflow-states.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
