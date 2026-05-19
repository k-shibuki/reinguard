# rgd-next

Single Claude Code entry for workflow procedures: use **substrate** output to pick Semantics docs. This command does **not** embed logic that duplicates `rgd` state/route resolution (ADR-0001).

Shared review norms and the Adapter taxonomy: [`AGENTS.md`](../../AGENTS.md). Claude bridge context: [`CLAUDE.md`](../../CLAUDE.md).

## Sense

Before fresh proposal logic, inspect the Adapter-local resume artifact when present:

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh status
```

Parse the Adapter-local status JSON (`status`, `resume_eligible`, `resume_reason_codes[]`) before running substrate observation. These diagnostics come from the Adapter-local resume script, not from `rgd context build`.

If that reports `status: "active"` and `resume_eligible: true`, resume the recorded approved Execute path instead of starting a new proposal cycle. Resume eligibility: [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Resume eligibility contract** and ADR-0015.

If it reports `status: "pending_approval"`, continue with **Propose** (do not treat as an approved run). This artifact is **Adapter-local only** (ADR-0015): do **not** feed it into `rgd` state / route evaluation.

1. From repo root:

   ```bash
   rgd context build
   ```

   If `rgd` is not on `PATH`: `go run ./cmd/rgd context build` (same flags).

   When the user names a PR number:

   ```bash
   rgd context build --pr <N>
   ```

2. Parse `rgd context build` stdout JSON:
   - `state.state_id`, `state.kind`
   - `routes[0].kind`, `routes[0].route_id` (when `routes[0].kind` is `resolved`)
   - `guards` (e.g. `merge-readiness` during **Execute**)
   - `knowledge.entries`
   - `observation.signals.git.working_tree_clean` (dirty-tree gate for **Route** / `review-address` Step 0)

## Route

When `state.kind` is `resolved`, choose the procedure under `.reinguard/procedure/` whose YAML front matter `applies_to.state_ids` contains `state.state_id` (and align with `routes[0].route_id` vs `applies_to.route_ids` when procedures scope by route). Mapping is validated by `rgd config validate`; [ADR-0013](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) § 4 describes the mechanism — do not duplicate a routing table here.

**Dirty working tree + `review-address`:** When `observation.signals.git.working_tree_clean` is `false` and the mapped procedure is `review-address`, run **Step 0** in that procedure first. See `.reinguard/knowledge/review--incremental-fix-flow.md`.

When `state.kind` is not `resolved`, follow ADR-0007 handoff — do not invent a winning state.

## Propose

After **Sense** and **Route**, record the Adapter-local **proposal** artifact before the approval gate, present the full-path proposal **exactly once**, then wait for approval.

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh start \
  --branch <branch> \
  --state-id <state_id> \
  [--route-id <route_id>] \
  --ordered-remainder "<procedure1 -> procedure2 -> ... -> DoD>" \
  --completion-condition "<Per-unit Definition of Done>" \
  [--issue <N>] [--pr <N>] [--summary TEXT]
```

Proposal format and approval gate: [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Full-path proposal format** and § **Approval gate**. Trace through **Per-unit Definition of Done**. **No per-procedure re-approval** after the single gate.

## Execute

After approval, follow [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Post-approval execution contract** and § **Loop semantics**: drive to Per-unit DoD **without** user prompts that gate progress between iterations.

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh approve
```

Refresh the artifact with:

```bash
bash .reinguard/scripts/adapter-rgd-next-resume.sh update --state-id <state_id> [--route-id <route_id>]
```

When DoD or an allowed stop is reached, close with `finish --status ... --reason ...`.

**Resumable wait stops:** [`.reinguard/procedure/next-orchestration.md`](../../.reinguard/procedure/next-orchestration.md) § **Allowed stops**.

## Guard

- FSM and priorities: [`docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md`](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md)
- Adapter vs Semantics: [`docs/adr/0001-system-positioning.md`](../../docs/adr/0001-system-positioning.md)
- Preflight SSOT: [`.reinguard/policy/coding--preflight.md`](../../.reinguard/policy/coding--preflight.md)
