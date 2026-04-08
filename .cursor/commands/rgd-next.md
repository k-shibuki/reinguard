# rgd-next

Single Cursor entry for workflow procedures: use **substrate** output to pick Semantics docs. This command does **not** embed logic that duplicates `rgd` state/route resolution (ADR-0001).

## Sense

Before fresh proposal logic, Cursor may inspect the Adapter-local resume
artifact:

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh status
```

If that reports `status: "active"` and `resume_eligible: true`, resume the
recorded approved Execute path instead of starting a new proposal cycle.
If it reports `status: "pending_approval"`, a proposal artifact exists for the
recorded unit but the user has not yet approved Execute; continue with **Propose**
(do not treat as an approved run). This artifact is **Adapter-local only**
(ADR-0015): do **not** feed it into `rgd` state / route evaluation or treat it
as substrate workflow position.

1. From repo root (or pass `--cwd` / `--config-dir` consistent with other `rgd` commands):

   ```bash
   rgd context build
   ```

   If `rgd` is not on `PATH`, from repo root: `go run ./cmd/rgd context build` (same flags).

   When the user names a PR number, prefer scoped observation:

   ```bash
   rgd context build --pr <N>
   ```

   (or `go run ./cmd/rgd context build --pr <N>` from repo root).

2. Parse stdout JSON:
   - `state.state_id`, `state.kind`
   - `routes[0].kind`, `routes[0].route_id` (when `routes[0].kind` is `resolved`)
   - `guards` (e.g. `merge-readiness` for summaries during **Execute**)
   - `knowledge.entries` (filtered aids)
   - `observation.signals.git.working_tree_clean` (dirty-tree gate for **Route** / `review-address` Step 0)

## Route

**Normative mapping:** [ADR-0013 § 4](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) (*Adapter mapping (durable)*). When `state.kind` is `resolved`, use that section’s table to choose the procedure file under `.reinguard/procedure/`. ADR-0013 and `.reinguard/control/` are the SSOT for state and route semantics (do not duplicate the mapping here).

**Dirty working tree + `review-address`:** When `observation.signals.git.working_tree_clean` is `false` and the resolved procedure is `review-address`, run **Step 0** in that procedure first (`change-inspect` → commit → refresh context). See `.reinguard/knowledge/review--incremental-fix-flow.md`.

When `state.kind` is not `resolved`, follow ADR-0007 handoff: gather observation diagnostics, fix config or observation, re-run `context build` — do not invent a winning state.

## Propose

After **Sense** and **Route**, record the Adapter-local **proposal** artifact for the unit (branch / issue / PR as known) **before** the approval gate, then present the full-path proposal **exactly once** per run, then wait for approval. There is no alternate mode (no proposal-only run and no stopping after a minimal state dump).

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh start \
  --branch <branch> \
  --state-id <state_id> \
  [--route-id <route_id>] \
  --ordered-remainder "<procedure1 -> procedure2 -> ... -> DoD>" \
  --completion-condition "<Per-unit Definition of Done>" \
  [--issue <N>] [--pr <N>] [--summary TEXT]
```

Persistent JSON defaults to `.reinguard/local/adapter/rgd-next/execute-resume.json` (ADR-0015).

This writes `status: "pending_approval"` and persists the **proposed** contract inputs (`state-id`, `ordered-remainder`, `completion-condition`, fingerprint inputs) so the artifact records **what** the user will approve. The status stays `pending_approval` until the user approves Execute (next section).

Proposal content, approval gate, and user-visible output requirements: [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Full-path proposal format** and § **Approval gate**. Trace from the current `state_id` ([ADR-0013 § 4](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md)) through **Per-unit Definition of Done** in `next-orchestration.md`. **No per-procedure re-approval** after the single gate.

**Output (for agents):** Per **`next-orchestration.md`** § **Output** and procedure front matter — not merely `state_id` / `route_id` bullets.

## Execute

After approval, **always** follow [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Post-approval execution contract** and § **Loop semantics**: drive to Per-unit DoD **without** user prompts that gate progress between iterations.

On approval, activate the existing proposal artifact (same unit as **Propose** `start`):

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh approve
```

If no `pending_approval` artifact exists (for example a fresh machine), run `start` then `approve` in one flow after approval.

Post-approval behavior, loop semantics, allowed stops, and chat output rules: [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Post-approval execution contract**, § **Loop semantics**, § **Output**.

Refresh the Adapter-local artifact with:

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh update --state-id <state_id> [--route-id <route_id>]
```

Do **not** treat per-iteration chat as required user-visible output — follow [`.cursor/rules/reinguard-bridge.mdc`](../../.cursor/rules/reinguard-bridge.mdc) § **rgd-next Execute — Cursor chat transcript** for what may appear in the Cursor chat panel. When DoD or an allowed stop is reached, close the artifact with `finish --status ... --reason ...` before returning the final user-facing report.

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md`](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
- Verification / preflight SSOT: [`.reinguard/policy/coding--preflight.md`](../../.reinguard/policy/coding--preflight.md) (local gates before push; aligns with procedure **Reads** such as `change-inspect` / `implement`)
