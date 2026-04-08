# ADR-0015: Adapter-local execute resume artifact

## Status

Accepted.

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
orchestration state, not repository workflow position.

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
   identity is durable, then transition to `active` after approval. When the
   artifact says an approved unit is still `active` and matches the current
   branch, the Adapter resumes Execute instead of starting a new proposal
   cycle.
6. Terminality remains evidence-based per `next-orchestration.md`. The
   artifact is a durable record of the Adapter contract, not an authority
   that overrides procedure semantics.
7. The artifact is allowed to answer **what was approved** for adapter-local
   audit / resume decisions, but it must still not become substrate state.
   In particular, the approved contract is evidence for the Adapter only and
   must not be promoted into `gates.*`, `state_id`, `route_id`, or guard
   inputs.

## Migration note

Previously the resume file defaulted under `.tmp/adapter/rgd-next/` (and
`REINGUARD_LOCAL_STATE_ROOT` from `with-repo-local-state.sh`). It now defaults
to `.reinguard/local/adapter/rgd-next/execute-resume.json` so Adapter state is
not mixed with workspace tool caches under `/.tmp/`.

## Consequences

- **Easier**: mid-run chat turns can resume the same approved path without
  re-opening the approval gate
- **Easier**: premature final responses become detectable as an
  Adapter-contract mismatch
- **Easier**: the artifact can explain which exact proposal / completion
  condition the user approved, not merely that approval happened
- **Easier**: substrate boundaries stay intact because repo/platform state
  still comes only from `rgd`
- **Harder**: Adapter commands must explicitly record start / update /
  terminal transitions
- **Harder**: each Adapter must own its own persistence details instead of
  delegating continuity to `rgd`

## Refs

- ADR-0001 (system positioning)
- ADR-0003 (pull-based stateless invocation)
- ADR-0011 (semantic control plane structure)
- ADR-0013 (FSM workflow states and Adapter mapping)
- ADR-0014 (runtime gate artifacts)
