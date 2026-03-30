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

2. **No `.reinguard/rules/`** — Replaced by `control/` subdirectories to
   avoid ambiguous naming.

3. **Unified priority space** — ADR-0004 unchanged: all rules from the
   three `control/` subtrees share one priority namespace. Each YAML
   file's `type` field must match its subdirectory, and validation
   enforces that consistency.

4. **Placement heuristics** — When adding or moving documents:

   - Improves judgment when read → `knowledge/`
   - Must be followed as a norm → `policy/`
   - State / route / guard meaning in match YAML → `control/`
   - Repeatable agent procedure bound to state/route → `procedure/`
   - Client-specific bridge only (no SSOT prose) → Adapter layer (`.cursor/`)

5. **Adapter layer** — `.cursor/` remains thin: bridge files and commands
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

## Refs

- ADR-0001 (Adapter / Semantics / Substrate layers)
- ADR-0004 (unified priority resolution)
- ADR-0010 (knowledge format and manifest)
- ADR-0013 (FSM v1 states and Adapter mapping)
