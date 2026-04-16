---
id: procedure-next-orchestration
purpose: "Mandatory single-unit orchestration for rgd-next: full-path proposal, one approval gate, autonomous execution to Per-unit DoD."
applies_to:
  state_ids: []
  route_ids: []
reads:
  - ../policy/safety--agent-invariants.md
sense:
  - rgd context build
act:
  - Always present full-path proposal; obtain single approval; execute loop to DoD (no alternate modes).
output:
  - Agent-internal iteration context each loop; final user-facing DoD or allowed-stop report.
done_when: "Per-unit Definition of Done satisfied (merge + branch cleanup) or allowed stop."
escalate_when: "HS-* violation; genuine cannot-proceed with evidence."
---

# next-orchestration

**SSOT** for workflow after `rgd-next` has **Sense** and **Route**: every invocation uses this contract — what to show the user before acting, **one** approval gate per run, and post-approval work through **Per-unit Definition of Done** (no optional modes).

**Not a Cursor slash command** — the invocable Adapter entry is [`.cursor/commands/rgd-next.md`](../../.cursor/commands/rgd-next.md) (Propose → Execute after approval).

**Design alignment**: [ADR-0001](../../docs/adr/0001-system-positioning.md) — `state_id` → procedure routing uses `.reinguard/procedure/*.md` front matter (`applies_to`), validated by `rgd config validate`; [ADR-0013](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) § 4 documents the mechanism and FSM semantics. This document holds **orchestration** (proposal, approval, execution contract, loop).

## Context

- [`../policy/safety--agent-invariants.md`](../policy/safety--agent-invariants.md) — **HS-*** hard stops
- [ADR-0013](../../docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md) — FSM states; **§ 4** procedure mapping mechanism (`applies_to` SSOT)

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
2. **Ordered remainder** — Trace **forward** from the current `state_id` using the mapped primary procedure (procedure front matter per ADR-0013 § 4) through **Per-unit Definition of Done** above. List the **sequence of procedures** you expect (e.g. `review-address` → `wait-bot-review` → `pr-merge` → branch cleanup). Include `change-inspect` and `pr-create` on the path from `working_no_pr` when applicable.
3. **Gaps** — State honestly what is unknown until the next observation (e.g. “PR not opened yet — review steps are projected”).
4. **Completion condition** — Reference **Per-unit Definition of Done** (this section).

## Approval gate

The Adapter may persist a **`pending_approval`** artifact before this gate and
transition it to **`active`** after explicit approval (ADR-0015; see
[`.cursor/commands/rgd-next.md`](../../.cursor/commands/rgd-next.md) § Propose /
§ Execute).

Present **once**:

- (a) **Unit identity** — Issue #, PR # (if any), branch name.
- (a2) **Current position** — proposed `state_id` / `route_id` and proposal
  subject (`head_sha` if the Adapter persists one — same meaning as
  `subject.head_sha` in runtime gate artifacts, ADR-0014 / ADR-0015)
- (b) **Ordered remainder** — From current state through Per-unit Definition of Done.
- (c) **Completion condition** — Per-unit Definition of Done (verbatim or by path).

If the Adapter persists approval continuity, record the same contract —
identity, ordered remainder, completion condition, and a deterministic
proposal fingerprint (SHA-256 hex of the newline-terminated fields written by
`adapter-rgd-next-resume.sh` `compute_proposal_fingerprint`, ADR-0015) — so the
artifact can later answer **what was approved**.

Obtain **explicit user approval** to execute through that completion condition. **No per-procedure re-approval** after this gate (except Hard Stops and genuine blocks below).

## Post-approval execution contract

After approval, the agent **must** drive toward **Per-unit Definition of Done** **without** inserting user interaction that **gates** progress.

**Forbidden** until DoD is met or an allowed stop applies:

- Asking whether to continue, which option to take, or “should I do X next” (including soft closers that function as a gate).
- Stopping with only a roadmap while DoD remains unmet when the agent could still act.
- **Implicit stop (forbidden).** Ending the turn, “handing off,” or treating work as complete **without** (a) Per-unit Definition of Done satisfied **or** (b) an **allowed stop** (below) **with explicit evidence** is prohibited. **Do not** infer completion from habit, a “natural” break, “reasonable” end-of-message, residual uncertainty, long CI or bot wait, or a user-facing “next steps” summary. If the mapped procedure still applies or the next loop iteration is the same Sense → Route → Act → Refresh path, **continue** until DoD or an allowed stop. Naming the violation does not satisfy the contract — only DoD or an allowed stop does.

**Allowed stops:**

- **Hard Stops** (**HS-***) in [`../policy/safety--agent-invariants.md`](../policy/safety--agent-invariants.md).
- **Genuine cannot proceed** — missing credentials, org enforcement, unrecoverable GitHub block — report with **evidence** and stop.
- **Tooling / session limits** — chat session ended, tooling unavailable, or context limits make further tool use impossible **in this session**. Long CI or bot duration is **not** an excuse to exit the path; follow the mapped procedure. On tooling/session limits only, **resume the same approved path** on the next turn **without** re-opening the approval gate (unless the user revokes or changes scope).

Adapters may persist this approval continuity locally so the next turn can
resume the same approved path. Such persistence is **Adapter-local** and must
not be promoted into substrate workflow state, routes, guards, or
`gates.<id>.*` signals (ADR-0015).

## Loop semantics (after approval)

Repeat until Per-unit Definition of Done is satisfied or an **allowed stop** fires:

1. **Sense** — `rgd context build --compact` by default (same cwd / `--config-dir` as the workflow’s initial context build from repo root). Fall back to full `rgd context build` only when you need nested observation details that `--compact` intentionally trims (for example `check_runs`, `review_inbox`, `conversation_comments`) or when you are debugging observation / guard behavior against the untrimmed payload.
2. **Parse** — `state`, `routes[0]` (interpret `routes[0].route_id` only when `routes[0].kind` is `resolved`), `guards`, `knowledge.entries`; record a short iteration context **agent-internally** (e.g. tool logs or internal notes). Whether any of that appears in a user-facing channel is defined by the Adapter (see [`../../.cursor/rules/reinguard-bridge.mdc`](../../.cursor/rules/reinguard-bridge.mdc) § **rgd-next Execute — Cursor chat transcript**); Semantics does not require per-iteration user-visible output.
3. **Route** — Resolve procedure(s) from `.reinguard/procedure/` front matter (`applies_to`) consistent with ADR-0013 § 4. If `state.kind` is not `resolved`, follow ADR-0007 handoff; do not invent a winning state.
4. **Act (procedure)** — Open the mapped procedure file(s) and **follow each procedure in full** (Context, Reads, Sense, Act, Output, Guard, front-matter `done_when` / `escalate_when` as applicable). Treat any “confirm” / “verify” language in mapped procedures as **agent self-checks** (evidence-backed), not a new user-approval gate, unless an **allowed stop** applies. Do not shortcut HS-*.
5. **Refresh** — After any **material** remote or local change (push, merge, thread resolve batch, bot re-review when the procedure says so), run **`rgd context build --compact` again** before the next Route.

**Dirty working tree + `review-address`:** When `observation.signals.git.working_tree_clean` is `false` and the mapped procedure is `review-address`, run **Step 0** in `review-address.md` before disposition-heavy work. See `.reinguard/knowledge/review--incremental-fix-flow.md`.

**Stop triggers (always):** observation you cannot fix; an **HS-*** would be violated; procedure **escalate_when** matches; user-only action outside agent capability.

## CI and bot wait

For `waiting_ci` and `waiting_bot_*` states, follow the **mapped** procedure (e.g. [`.reinguard/procedure/wait-bot-review.md`](wait-bot-review.md), [`.reinguard/procedure/review-address.md`](review-address.md)) in full, then **Sense** again. This repository does not yet define a separate delegation/subagent policy; do not duplicate wait logic here.

## Output

- **Per iteration (agent-internal):** current `state_id` / `route_id` / guard summary; which procedure(s) were executed or are next. This is **not necessarily user-facing**; it is the minimum context the agent must retain to drive the loop. Adapter rules define what may appear in the Cursor chat panel during Execute.
- **Final (user-facing when using the Cursor Adapter):** DoD satisfied with evidence, or allowed stop with evidence.

## Guard

All **HS-*** invariants apply. Procedures and policy remain authoritative; this document does not replace them.
