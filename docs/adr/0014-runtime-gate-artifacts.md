# ADR-0014: Runtime gate artifacts for deterministic local workflow progression

## Context

`working_no_pr` currently collapses materially different local situations:
implementation in progress, local verification complete, self-inspection
complete, and PR-ready handoff. The FSM can only route on observable inputs, so
procedure progress that exists only in agent memory cannot justify stable state
or route separation.

At the same time, reinguard must stay within the substrate boundary from
ADR-0001: `rgd` may compute deterministic context from repository-owned inputs,
but it must not become a workflow brain or execute arbitrary repository scripts.

## Decision

1. Introduce **runtime gate artifacts** under `.reinguard/local/gates/`.
   These are **gitignored, substrate-owned operational state**, not Semantics
   documents.
2. Add `rgd gate` commands:
   - `rgd gate record <gate-id>` - bounded write of one validated artifact for
     the current branch HEAD
   - `rgd gate status <gate-id>` - derive `missing` / `invalid` / `stale` /
     `fail` / `pass`
   - `rgd gate show <gate-id>` - print the validated raw artifact
3. Gate artifacts are validated against an embedded JSON Schema and must record:
   `gate_id`, top-level `status`, `recorded_at`, **`subject`** (`branch`,
   `head_sha`), **`producer`** (`procedure`, `tool`), **`inputs[]`** (upstream
   gate proofs when a gate depends on another proof), and `checks[]`.
4. Freshness is deterministic:
   - `missing`: artifact file absent
   - `invalid`: file unreadable or schema-invalid
   - `stale`: artifact branch or head SHA differs from the current checkout, or
     the current checkout cannot supply comparable git identity
   - otherwise the artifact's own `status` (`pass` / `fail`)
5. Evaluation commands merge derived runtime gate signals under `gates.<id>.*`
   before state / route / guard evaluation, so future FSM rules can branch on
   gates without inventing agent-internal memory.
6. `rgd` does **not** execute verification commands for gates. Procedures or
   agents run checks such as `go test`, `go vet`, or `golangci-lint`, then
   record the resulting verified outcome into a gate artifact. Checks can be
   supplied inline via the repeatable `--check id:status:summary` flag
   (preferred for simple cases) or via `--checks-file` for bulk / evidence-
   rich payloads. The two may be combined; entries are merged.
7. Runtime gates may define **gate-specific proof contracts** on top of the
   common schema:
   - `local-verification` proves HS-LOCAL-VERIFY work on one `subject`
   - the configured **pre-PR AI review** gate role proves one local AI-review
     pass on one `subject` (default binding in this repository:
     `local-coderabbit`)
   - `pr-readiness` proves **local review closure** for one `subject`: the
     producer has dispositioned every finding from the current local review
     cycle (per `change-inspect`) and recorded that fact in `checks[]` (check id
     `review-closure`). For `pass` it must also carry the fresh passing
     `inputs[]` required by the repo’s configured runtime gate roles (default
     here: `local-verification` and `local-coderabbit`). It does **not** prove
     GitHub thread resolution or human approval; those remain PR-side signals.
8. `ready_for_pr` may continue to consume `gates.pr-readiness.status == pass`,
   but that status is no longer an opaque boolean: the `pr-readiness` artifact
   must itself prove which upstream local checks were consumed for the same
   subject.

**Runtime gate role configuration** is repository Semantics in
`.reinguard/reinguard.yaml` under `workflow.runtime_gate_roles` (validated by
`pkg/schema/reinguard-config.json`). It binds semantic roles
(`local_verification`, `pre_pr_ai_review`, `pr_readiness`) to concrete
`gate_id` values, optional `required` flags, `pass_check_ids`, and
`pr_readiness.pass_requires_roles`. Inspect the effective contract with
`rgd config validate` and the checked-in `reinguard.yaml`; changes that alter
workflow meaning should be traceable (issue / ADR per repository policy).

### Extension contract (runtime gates)

When adding or changing a **runtime gate** (`gate_id`):

1. **Semantics** — Document the gate’s purpose; **subject** (which branch head
   the proof applies to); **producer** procedure(s) that run `rgd gate record
   <gate-id>` after local verification; **consumer** procedure(s) or FSM rules
   that read `gates.<gate-id>.*`; and any required **inputs[]**. Keep
   recording out of versioned Semantics; artifacts stay under
   `.reinguard/local/gates/` (gitignored).
2. **FSM** — If `state eval` or `route select` references `gates.<gate-id>`, update `.reinguard/control/states/*.yaml` and/or `.reinguard/control/routes/*.yaml` and **ADR-0013** (state catalog and Adapter mapping).
3. **Freshness** — Procedures must treat `rgd gate status` outcomes per Decision §4: `stale` / `missing` / `invalid` are not proof of the current HEAD; consumers return to the producer procedure or re-verify before proceeding.
4. **Schema** — Artifacts validate against the embedded gate schema; common
   provenance fields (`subject`, `producer`, `inputs`, `checks`) belong in the
   shared schema. Gate-specific proof rules live in code / docs, not ad-hoc
   side files. New top-level fields require ADR-0008 / schema versioning.
5. **Tests** — Add or extend CLI/FSM tests when gates affect state resolution (e.g. `pass` vs `stale` fallback to a residual state).

`rgd` still does not execute gate verification commands; recording remains procedure-owned.

Operational checklist: `.reinguard/knowledge/workflow--state-gate-guard-extension.md`.

## Migration note

Gate files on disk moved from `.reinguard/runtime/gates/` to
`.reinguard/local/gates/` (breaking). Rationale: consolidate reinguard-owned
local state under `.reinguard/local/` and reserve `/.tmp/` for tool caches only.
Re-run `rgd gate record <gate-id>` on the current HEAD after upgrading `rgd`.

The gate artifact JSON shape also moved **`branch` / `head_sha` under
`subject`** and added `producer`, `inputs`, and structured `checks`. Older
artifacts with top-level `branch` / `head_sha` only are **schema-invalid**;
delete them or re-record after upgrade — `rgd gate status` treats unreadable or
invalid files as `invalid` / `missing` proof.

If you work across several branches, each checkout has its own
`.reinguard/local/gates/*.json` under that tree: after upgrading `rgd`, switch
to a branch and re-run `rgd gate record <gate-id>` on that branch’s HEAD when
you need a valid proof there. There is no mixed-schema grace period — stale
files simply read as `invalid` / `stale` until replaced.

## Consequences

- **Easier**: local workflow progression becomes machine-observable without
  overloading GitHub or git signals
- **Easier**: future issues can refine `working_no_pr` using stable
  `gates.<id>.status` signals
- **Easier**: gate freshness is auditable and tied to branch HEAD
- **Easier**: `pass` can carry its own provenance instead of acting as an
  opaque hand-written marker
- **Harder**: procedures must explicitly record gates after verification
- **Harder**: procedures must supply structured `checks[]`, `producer`, and
  `inputs[]` instead of treating `rgd gate record` as a bare status toggle
- **Harder**: artifacts are operational state and must stay out of versioned
  Semantics content

## Refs

- ADR-0001 (system positioning)
- ADR-0003 (pull-based, stateless invocation)
- ADR-0008 (schema versioning)
- ADR-0011 (semantic control plane structure)
- ADR-0013 (FSM state catalog and Adapter mapping)
- Issue #97
