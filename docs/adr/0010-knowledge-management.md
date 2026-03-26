# ADR-0010: Repository knowledge: format, manifest generation, and agent-facing delivery

## Context

reinguard is a three-layer control system whose Substrate runtime (`rgd`) stabilizes the information surface for agents (ADR-0001). Repositories declare **knowledge** under `.reinguard/knowledge/`—project-coupled guidance that agents should read when acting in that repo.

Without normative rules:

- Metadata drifts between hand-edited indexes and manifests
- Agents receive only opaque paths with no triage signal
- Selection is either “all files” or ad hoc; there is no shared contract for narrowing knowledge by task

Alternatives considered:

| Dimension | Options |
|-----------|---------|
| Manifest SSOT | Hand-maintained `manifest.json` vs generated from knowledge files |
| Selection | Agent-only triage vs keyword filter in `rgd` vs signal-based automatic selection |
| Operational context | `paths[]` only vs rich `entries[]` with metadata |
| Agent bootstrap | `rgd` only vs per-platform bridge files (Cursor rules, `AGENTS.md`, etc.) |

## Decision

1. **Knowledge file format** — Knowledge entries are **Markdown** files under `.reinguard/knowledge/` with a **YAML front matter** block (delimited by `---`). Required front matter fields: `id` (unique string), `description` (one-line summary), `triggers` (non-empty list of keyword strings). The body is free-form Markdown. Non-Markdown files are not indexed as knowledge entries.

2. **Manifest as generated artifact** — `manifest.json` is **not** hand-authored as the source of truth. **`rgd knowledge index`** scans knowledge Markdown files, reads front matter, and writes `manifest.json`. The manifest is **committed** so tools that do not invoke `rgd` (e.g. editor rules) can still read a stable catalog.

3. **Validation and freshness** — `rgd config validate` validates the manifest against its JSON Schema, checks that each `path` exists, and **fails** if the manifest does not match a re-index from front matter (stale manifest). Optional **hints** (warnings) may flag oversized files or excessive trigger counts to support atomicity discipline without claiming semantic enforcement.

4. **Atomicity** — **Structural** guarantees: valid front matter, unique `id`, resolvable paths. **Semantic** “one concern per file” is a **review and authoring** discipline, not a machine guarantee.

5. **Operational context contract** — The `knowledge` object in operational context JSON carries **`entries`**: an array of objects with at least `id` and `path`, and typically `description` and `triggers`. The legacy `paths`-only shape is **removed** in the schema line that ships with this decision (see ADR-0008 synchronized versioning).

6. **Selection model (Phase 1)** — **`rgd knowledge pack`** accepts an optional **`--query`** string. When set, `rgd` returns only entries whose **triggers** match the query (case-insensitive substring). When unset, **all** entries are returned so agents can triage using `description` and `triggers`.

7. **Selection model (future)** — Repositories may later attach **signal-based** `when:` expressions to knowledge entries, evaluated with the same match machinery as state/route rules (ADR-0002). That path is **explicitly reserved**; it is not required for Phase 1.

8. **Agent consumption** — The **platform-agnostic** contract is operational context JSON from **`rgd context build`** (and **`rgd knowledge pack`**). **Per-platform bootstrap** (e.g. Cursor `.cursor/rules/` pointing at `manifest.json`) is a **hybrid** concern: normative patterns may be documented here and exemplified in repositories; **`rgd init`-style scaffolding** may generate bridge files in a later change.

## Consequences

- **Easier**: Single SSOT for knowledge metadata (front matter); manifest drift is caught by `config validate`
- **Easier**: Agents can choose files using descriptions and triggers without opening every file first
- **Harder**: Authors must run `rgd knowledge index` after editing knowledge metadata and commit the updated manifest
- **Harder**: Breaking change for consumers of `knowledge.paths` only; they must migrate to `knowledge.entries`

## Refs

- ADR-0001 (system positioning; knowledge packing)
- ADR-0002 (spec-driven evaluation; future `when:` on knowledge)
- ADR-0008 (schema versioning; CLI SSOT in `docs/cli.md`)
