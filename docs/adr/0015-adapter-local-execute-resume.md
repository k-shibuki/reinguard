# ADR-0015: Adapter-local execute resume artifact

## Context

`rgd-next` Execute has a single approval gate, then must continue to
Per-unit Definition of Done unless an allowed stop applies. In practice,
an Adapter session may still end mid-run because of chat turn boundaries,
tool/session limits, or an implementation bug that emits a non-terminal
summary.

The system needs a durable way to recognize "this approved Execute path is
still active" on the next Adapter turn. At the same time, reinguard must
preserve the substrate boundary from ADR-0001 and ADR-0003:

- `rgd` computes repository and platform workflow position.
- `rgd` remains pull-based and stateless across invocations.
- Client-specific execution continuity must not become a new substrate
  workflow state, route, or guard signal.

ADR-0014 already permits bounded substrate-owned runtime artifacts for
verification outcomes such as `pr-readiness`. That mechanism is not a fit
for Execute continuity itself because approval continuity is Adapter-local
orchestration state, not repository workflow position. The resume artifact is
therefore proof of an Adapter approval contract, not substrate state and not a
signal that `rgd` may use to resolve workflow position.

## Decision

1. Introduce an **Adapter-local execute resume artifact** for `rgd-next`.
2. The artifact lives under `.reinguard/local/adapter/rgd-next/` (gitignored).
   Optional override: set `REINGUARD_LOCAL_DIR` to the desired `.reinguard/local`
   equivalent root (for example in tests).
3. The artifact is **Adapter-owned**, not substrate-owned:
   - it must not be merged into `rgd context build`
   - it must not become `gates.<id>.*`
   - it must not define a new `state_id`, `route_id`, or guard input
4. The artifact records only the minimum continuity contract needed to
   resume an already approved Execute path:
   - unit identity (branch, issue / PR when known)
   - **approved contract** captured at proposal time: proposal `head_sha`,
     proposed `state_id` / `route_id`, ordered remainder, completion
     condition, and a **deterministic proposal fingerprint** — SHA-256 (hex) of
     the UTF-8 newline-terminated sequence: branch, issue number (if any), PR
     number (if any), proposal `head_sha`, `state_id`, `route_id`, ordered
     remainder, completion condition, and summary, in that order (see
     `.reinguard/scripts/adapter-rgd-next-resume.sh` `compute_proposal_fingerprint`)
   - The fingerprint is a deterministic hash of the listed UTF-8 fields as written by the script (no hidden normalization beyond newline termination where the script adds it).
   - approval marker and timestamps (after the user approves Execute)
   - current lifecycle status (`pending_approval`, `active`, `done`,
     `allowed_stop`, `revoked`)
   - last observed `state_id` / `route_id`
   - terminal reason enum for non-active records
5. An Adapter implementation (for example the Cursor chat Adapter) checks
   this artifact **before** fresh `rgd-next` proposal logic. The Adapter may
   create a `pending_approval` record **before** the approval gate so unit
   identity is durable, then transition to `active` after approval.
   Resume eligibility is a **fresh-evidence** decision, not an artifact-only
   decision: an `active` artifact may resume Execute **only if** a fresh
   cross-check still proves the approved proposition.
   At minimum, the cross-check MUST fail closed unless all of the following
   still hold:
   - current branch matches the artifact branch
   - current `git rev-parse HEAD` matches `approved_contract.head_sha`
   - fresh `rgd context build --compact` still resolves to the approved
     `state_id`
   - when the approved contract recorded a `route_id`, fresh
     `rgd context build --compact` still resolves to that same `route_id`
   - any fresh observation-derived signals that materially contribute to the
     current workflow position do not contradict the approved path. "Materially
     contribute" means signals whose current values explain or block the fresh
     resolved `state_id` / `route_id` that the Adapter is about to continue.
     "Contradict" means either (a) fresh `state_id` / `route_id` no longer
     equals the approved contract, or (b) a known same-HEAD blocking signal
     now requires a different path even before a new commit exists. This must
     stay aligned with ADR-0012 facts in the current schema (for example
     bot-review facts such as `review_trigger_awaiting_ack` and
     `bot_review_trigger_awaiting_ack`)
     Example: if the user approved Execute while
     `review_trigger_awaiting_ack=false`, then a later same-HEAD
     `@coderabbitai review` causes fresh observation to report
     `review_trigger_awaiting_ack=true`, the approved path is contradicted even
     though `HEAD` did not change and resume must fail closed.
   - the stored proposal fingerprint still recomputes from the artifact fields
   An Adapter MAY layer a TTL on top as defense in depth, but TTL alone is not
   sufficient to authorize resume.
   The decision surface MUST expose machine-readable diagnostics (for example
   `resume_reason_codes[]`) and MUST distinguish at least: branch drift, head
   drift, state / route drift, stale review-trigger facts, malformed artifacts,
   unavailable fresh context, fingerprint mismatch, and TTL expiry when
   enabled.
6. Terminality remains evidence-based per `next-orchestration.md`. The
   artifact is a durable record of the Adapter contract, not an authority
   that overrides procedure semantics. In particular, the Adapter-local
   continuity contract **distinguishes temporary waits from terminal allowed
   stops**:
   - **Temporary / resumable wait states** — The **normative** FSM
     `state_id` values are defined in
     [`.reinguard/control/states/workflow.yaml`](../../.reinguard/control/states/workflow.yaml).
     This ADR and `.reinguard/scripts/adapter-rgd-next-resume.sh` enumerate the
     **wait/retry subset** aligned with that file: `waiting_ci`,
     `waiting_bot_rate_limited`, `waiting_bot_paused`, `waiting_bot_stale`,
     `waiting_bot_run`, `waiting_bot_failed`. If the control plane adds a new
     wait `state_id`, update the control rules, the script guard, and this
     Semantics text together (same-kind sweep) so the fail-closed check cannot
     drift. Procedure docs
     (`.reinguard/procedure/wait-bot-review.md`,
     `.reinguard/knowledge/review--bot-operations.md`) explain **how** to
     retry; they do not silently extend the script's guarded set. An
     Adapter's `finish` subcommand MUST fail closed when a caller attempts to
     record `allowed_stop` with reason `cannot_proceed` or
     `tooling_session_limit` while fresh `rgd context build --compact` still
     resolves to one of these states. `tooling_session_limit` is not a
     substitute for repository-state terminality when current observation
     still resolves to a wait/retry procedure. The terminal reason enum
     (`dod_satisfied`, `hard_stop`, `cannot_proceed`, `tooling_session_limit`,
     `scope_revoked`) is defined by the resume artifact schema in
     `.reinguard/scripts/adapter-rgd-next-resume.sh` (see
     `RESUME_SCHEMA_VERSION` and the script's `usage` section); the reference
     implementation there enforces the resumable-wait guard by re-running
     substrate observation at `finish` time.
   - **Terminal allowed stops** — narrowed to **HS-*** hard stops
     (procedural hard-stop invariants; SSOT in
     [`.reinguard/policy/safety--agent-invariants.md`](../../.reinguard/policy/safety--agent-invariants.md))
     and genuine unrecoverable external blocks (for example GitHub itself
     unreachable, org-level enforcement that permanently blocks the required
     bot). These are expressed as `allowed_stop` + `hard_stop` with
     evidence.
   - **Out of scope for this ADR revision:** introducing a new non-terminal
     `suspended` / `resumable_wait` artifact status, and any substrate FSM
     / observation contract changes owned by the Phase 2 substrate-alignment
     epic ([#59](https://github.com/k-shibuki/reinguard/issues/59)). Resumable
     waits remain `active` until the loop re-enters them or fresh observation
     transitions them; this Adapter-local contract tightens only the stop
     semantics, not artifact lifecycle states.
7. The artifact is allowed to answer **what was approved** for adapter-local
   audit / resume decisions, but it must still not become substrate state.
   In particular, the approved contract is evidence for the Adapter only and
   must not be promoted into `gates.*`, `state_id`, `route_id`, or guard
   inputs.
8. **Transient command payloads** (for example gate `checks[]` authoring) are
   not part of this Adapter-local contract. Prefer stdin or inline flags instead
   of writing scratch JSON under `.reinguard/local/` unless a separate bounded
   runtime contract explicitly permits it.

### Current layout

- **Durable Adapter continuity state:** `.reinguard/local/adapter/`
- **Durable Substrate gate state:** `.reinguard/local/gates/`
- **Workspace-local caches and tool homes:** `.tmp/`
- **Transient command payloads:** Decision §8 (stdin/inline; not default `.reinguard/local/` contract)

### Schema versioning

The resume artifact carries its own `schema_version`, independent of the
substrate `CurrentSchemaVersion` in `pkg/schema/embed.go`. The SSOT is
`RESUME_SCHEMA_VERSION` at the top of
`.reinguard/scripts/adapter-rgd-next-resume.sh`. The version follows semver
(`MAJOR.MINOR.PATCH`); breaking shape changes increment MAJOR.

## Migration note

Previously the resume file defaulted under `.tmp/adapter/rgd-next/` (and
`REINGUARD_LOCAL_STATE_ROOT` from `with-repo-local-state.sh`). It now defaults
to `.reinguard/local/adapter/rgd-next/execute-resume.json` so Adapter state is
not mixed with workspace tool caches under `.tmp/`.

## Consequences

- **Easier**: mid-run chat turns can resume the same approved path without
  re-opening the approval gate
- **Easier**: premature final responses become detectable as an
  Adapter-contract mismatch
- **Easier**: the artifact can explain which exact proposal / completion
  condition the user approved, not merely that approval happened
- **Easier**: substrate boundaries stay intact because repo/platform state
  still comes only from `rgd`
- **Easier**: resume decisions fail closed when fresh observation no longer
  matches the approved path, including same-HEAD observation drift
- **Harder**: Adapter commands must explicitly record start / update /
  terminal transitions
- **Harder**: each Adapter must own its own persistence details and its own
  fresh-observation revalidation instead of delegating continuity to `rgd`

## Refs

- ADR-0001 (system positioning)
- ADR-0003 (pull-based stateless invocation)
- ADR-0011 (semantic control plane structure)
- ADR-0013 (FSM workflow states and Adapter mapping)
- ADR-0014 (runtime gate artifacts)
