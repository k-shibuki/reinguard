# ADR-0011: Semantic control plane directory structure

## Context

The Semantics layer (`.reinguard/`, see ADR-0001) holds repository-owned
meaning: knowledge, policy, control rules, and substrate configuration.
Without explicit structure:

- Knowledge, policy, and procedural content are easy to conflate (for
  example, normative review rules indexed as "knowledge" for `rgd
  knowledge pack`).
- The name `rules/` is overloaded (policy rules, control rules, linter
  rules, interaction rules).
- The Adapter layer (`.cursor/`) may grow thick with duplicated semantics.

## Decision

1. **Semantics layer layout** — Under `.reinguard/`:

   - `reinguard.yaml` — substrate configuration (schema version, default
     branch, providers).
   - `knowledge/` — knowledge atoms only: Markdown with YAML front matter,
     plus generated `manifest.json` (ADR-0010).
   - `policy/` — normative documents (invariants, review/merge
     discipline, exception policy). Not part of `knowledge pack` unless
     explicitly copied into a future catalog; Adapter references paths
     directly.
   - `control/` — machine-readable state, route, and guard match rules:
     - `control/states/*.yaml` — `type: state` rules only
     - `control/routes/*.yaml` — `type: route` rules only
     - `control/guards/*.yaml` — `type: guard` rules only
   - `procedure/` — agent **action-card** bodies (Markdown with YAML front
     matter: `id`, `purpose`, `applies_to`, `reads`, `sense`, `act`, `output`,
     `done_when`, `escalate_when`). SSOT for procedural steps lives here
     (ADR-0013). Cursor: `rgd-next` maps substrate output to these paths
     (`.cursor/commands/rgd-next.md`); `cursor-plan` handles deep planning via
     `CreatePlan` only, embedding Issue-creation steps when issue-first
     (`.cursor/commands/cursor-plan.md`).
   - `local/` — **gitignored operational state** written by the Substrate or
     Adapter when a bounded runtime contract explicitly allows it (for example
     runtime gate artifacts and Adapter resume state; see ADR-0014 and
     ADR-0015). This directory is **not** Semantics content, is not indexed as
     knowledge, and is not part of control-rule validation. Physical placement
     under `.reinguard/` is for repository-local discovery only; semantic
     ownership remains Substrate/runtime or Adapter/runtime, not Semantics.
     Transient command inputs, scratch payloads, and tool caches do **not**
     belong here; use workspace-relative paths such as `.tmp/` for those
     artifacts unless a bounded runtime contract explicitly says otherwise.

1. **No `.reinguard/rules/`** — Replaced by `control/` subdirectories to
   avoid ambiguous naming.

1. **Unified priority space** — ADR-0004 unchanged: all rules from the
   three `control/` subtrees share one priority namespace. Each YAML
   file's `type` field must match its subdirectory, and validation
   enforces that consistency.

1. **Placement heuristics** — When adding or moving documents:

   - Improves judgment when read → `knowledge/`
   - Must be followed as a norm → `policy/`
   - State / route / guard meaning in match YAML → `control/`
   - Repeatable agent procedure bound to state/route → `procedure/`
   - Substrate or Adapter operational state under bounded contract → `local/`
   - Client-specific bridge only (no SSOT prose) → Adapter layer (`.cursor/`)

1. **Adapter layer** — `.cursor/` remains thin: bridge files and commands
   reference `.reinguard/` paths; they do not restate Semantics-layer body
   text as SSOT.

## Consequences

- **Easier**: Clear boundaries between packable knowledge, human-facing
  policy, and evaluable control YAML
- **Easier**: `rgd knowledge index` / `knowledge pack` stay scoped to
  judgment aids, not merge policy prose
- **Harder**: Authors must choose the correct subtree; misplaced files need
  review
- **Harder**: Downstream repos that used `.reinguard/rules/` must migrate
  paths (breaking layout change for configuration discovery)
- **Harder**: `.reinguard/` now contains both Semantics content and an explicit
  gitignored local-state enclave; tooling and docs must keep that boundary clear

## Migration note

The on-disk directory for substrate-owned gate artifacts was renamed from
`runtime/` to `local/` so that reinguard-owned local state (gates and Adapter
resume) lives under one gitignored tree (`.reinguard/local/`) and stays
distinct from workspace-relative tool caches (`.tmp/`). Scratch artifacts
previously written to `runtime/` other than gate records are no longer
supported and can be safely deleted. Existing
`.reinguard/runtime/gates/*.json` files are not read by updated tooling;
re-record them with `rgd gate record` after updating `rgd`.

## Refs

- ADR-0001 (Adapter / Semantics / Substrate layers)
- ADR-0004 (unified priority resolution)
- ADR-0010 (knowledge format and manifest)
- ADR-0013 (FSM workflow states and Adapter mapping)
- ADR-0014 (runtime gate artifacts)
- ADR-0015 (Adapter-local execute resume)
