---
id: procedure-next-orchestration
purpose: "Single-unit orchestration: full-path proposal, one approval gate, autonomous execution to Per-unit DoD."
applies_to:
  state_ids: []
  route_ids: []
reads:
  - ../policy/safety--agent-invariants.md
sense:
  - rgd context build
act:
  - Present full-path proposal; obtain single approval; execute loop to DoD.
output:
  - Per-iteration state summary; final DoD report.
done_when: "Per-unit Definition of Done satisfied (merge + branch cleanup) or allowed stop."
escalate_when: "HS-* violation; genuine cannot-proceed with evidence."
---

# next-orchestration

**SSOT** for autonomous execution after `rgd-next` has **Sense** and **Map**: what to show the user before acting, **one** approval gate per run, and post-approval work through **Per-unit Definition of Done**.

**Not a Cursor slash command** — the invocable Adapter entry is [`.cursor/commands/rgd-next.md`](../../.cursor/commands/rgd-next.md) § **Orchestrate**.

**Design alignment**: [ADR-0001](../../docs/adr/0001-system-positioning.md) — routing and state mapping stay in `rgd-next.md` and ADR-0013; this document holds **orchestration** (proposal, approval, execution contract, loop).

## Context

- [`../policy/safety--agent-invariants.md`](../policy/safety--agent-invariants.md) — **HS-*** hard stops
- [ADR-0013](../../docs/adr/0013-fsm-v1-workflow-states.md) — FSM states and Adapter procedure mapping
- [`.cursor/commands/rgd-next.md`](../../.cursor/commands/rgd-next.md) — **Map (v2)** routing table

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow and commit policy.

## Per-unit Definition of Done

Use this **verbatim** (or point here by path) in every user-facing approval summary:

For the unit at hand, the agent runs the full loop:

**implement → change-inspect → pr-create → (CI / bot review cycle as state dictates) → review-address as needed → pr-merge → delete remote branch when safe → clean up local tracking branches.**

**Hard boundary:** “Done” means **through local tracking cleanup**, not “stop after `gh pr merge`.”

**Single-unit run complete** when this loop is finished for that unit, or when an **allowed stop** (below) applies with evidence.

Concrete gates: HS-* invariants, `merge-readiness` / `rgd guard eval merge-readiness` per [`.reinguard/procedure/pr-merge.md`](pr-merge.md), and procedure **Guard** sections.

## Full-path proposal format

Before the approval gate, present:

1. **Current position** — `state_id`, `route_id` (when resolved), and brief evidence basis from `rgd context build` JSON.
2. **Ordered remainder** — Trace **forward** from the current `state_id` using the **Map (v2)** table in `rgd-next.md` through **Per-unit Definition of Done** above. List the **sequence of procedures** you expect (e.g. `review-address` → `wait-bot-review` → `pr-merge` → branch cleanup). Include `change-inspect` and `pr-create` on the path from `working_no_pr` when applicable.
3. **Gaps** — State honestly what is unknown until the next observation (e.g. “PR not opened yet — review steps are projected”).
4. **Completion condition** — Reference **Per-unit Definition of Done** (this section).

## Approval gate

Present **once**:

- (a) **Unit identity** — Issue #, PR # (if any), branch name.
- (b) **Ordered remainder** — From current state through Per-unit Definition of Done.
- (c) **Completion condition** — Per-unit Definition of Done (verbatim or by path).

Obtain **explicit user approval** to execute through that completion condition. **No per-procedure re-approval** after this gate (except Hard Stops and genuine blocks below).

If the user requests **proposal only**, do not execute; stop after the proposal.

## Post-approval execution contract

After approval, the agent **must** drive toward **Per-unit Definition of Done** **without** inserting user interaction that **gates** progress.

**Forbidden** until DoD is met or an allowed stop applies:

- Asking whether to continue, which option to take, or “should I do X next” (including soft closers that function as a gate).
- Stopping with only a roadmap while DoD remains unmet when the agent could still act.

**Allowed stops:**

- **Hard Stops** (**HS-***) in [`../policy/safety--agent-invariants.md`](../policy/safety--agent-invariants.md).
- **Genuine cannot proceed** — missing credentials, org enforcement, unrecoverable GitHub block — report with **evidence** and stop.
- **Proposal-only** run — user declared no execution up front.
- **Tooling / session limits** — chat session ended, tooling unavailable, or context limits make further tool use impossible **in this session**. Long CI or bot duration is **not** an excuse to exit the path; follow the mapped procedure. On tooling/session limits only, **resume the same approved path** on the next turn **without** re-opening the approval gate (unless the user revokes or changes scope).

## Loop semantics (after approval)

Repeat until Per-unit Definition of Done is satisfied or an **allowed stop** fires:

1. **Sense** — `rgd context build` (same cwd / `--config-dir` as `rgd-next.md` § Sense).
2. **Parse** — `state`, `routes[0]`, `guards`, `knowledge.entries`; emit a short summary each iteration for the transcript.
3. **Map** — Use **Map (v2)** in `rgd-next.md`. If `state.kind` is not `resolved`, follow ADR-0007 handoff; do not invent a winning state.
4. **Execute** — Open the mapped procedure file(s) and **follow each procedure in full** (Context, Reads, Sense, Act, Output, Guard, front-matter `done_when` / `escalate_when` as applicable). Do not shortcut HS-*.
5. **Refresh** — After any **material** remote or local change (push, merge, thread resolve batch, bot re-review when the procedure says so), run **`rgd context build` again** before the next Map.

**Dirty working tree + `review-address`:** When `rgd-next.md` **Map (v2)** says so, run **Step 0** in `review-address.md` before disposition-heavy work. See `.reinguard/knowledge/review--incremental-fix-flow.md`.

**Stop triggers (always):** observation you cannot fix; an **HS-*** would be violated; procedure **escalate_when** matches; user-only action outside agent capability.

## CI and bot wait

For `waiting_ci` and `waiting_bot_*` states, follow the **mapped** procedure (e.g. [`.reinguard/procedure/wait-bot-review.md`](wait-bot-review.md), [`.reinguard/procedure/review-address.md`](review-address.md)) in full, then **Sense** again. This repository does not yet define a separate delegation/subagent policy; do not duplicate wait logic here.

## Output

- After each loop iteration: current `state_id` / `route_id` / guard summary; which procedure(s) were executed or are next.
- Final: DoD satisfied with evidence, or allowed stop with evidence.

## Guard

All **HS-*** invariants apply. Procedures and policy remain authoritative; this document does not replace them.
