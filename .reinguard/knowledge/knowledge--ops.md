---
id: knowledge-ops
description: How rgd indexes knowledge, validates manifests, and how agents retrieve review docs
triggers:
  - knowledge operations
  - read flow
  - review flow
  - retrieval strategy
  - rgd knowledge index
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Knowledge Operations for reinguard

## Implemented behavior (ADR-0010, ADR-0011)

- Each knowledge file is Markdown under `.reinguard/knowledge/` with YAML front matter
  (`id`, `description`, `triggers`, **`when`** — all required).
- **Policy** lives under `.reinguard/policy/` — not indexed here; use `.reinguard/policy/catalog.yaml` to list entries, then open Markdown by path.
- **Control** match rules live under `.reinguard/control/{states,routes,guards}/`
  (`*.yaml` loaded by `config.Load` / `rgd config validate`). `.reinguard/control/catalog.yaml`
  is a human-maintained index only (not read by validate today). Not `knowledge pack`.
- **`rgd knowledge index`** scans `knowledge/*.md`, reads front matter, and writes
  `.reinguard/knowledge/manifest.json` (committed; run after changing metadata). Rejects duplicate triggers (case-insensitive) and missing `when`.
- **`rgd config validate`** checks the manifest schema, that paths exist, that the manifest
  matches front matter (freshness), statically validates each `when` (known `op` and operands,
  `eval:` registry, `path` prefixes `git.` / `github.` / `state.` / `$`), validates control YAML, and emits optional size/trigger-count hints.
- **`rgd context build`** emits **`knowledge.entries`** filtered by each entry’s `when` against observation + merged **`state.*`** signals (`docs/cli.md`).
- **`rgd knowledge pack`** lists manifest entries; with **`--observation-file`**, applies `when` against nested **`signals`** only; optional **`--query`** OR-unions trigger substring matches (`docs/cli.md`).

## Practical retrieval flow

1. **Default:** run **`rgd context build`** and read **`knowledge.entries`** from stdout JSON (`id`, `path`, `description`, `triggers`, `when`).
2. **Catalog without running observe:** open `.reinguard/knowledge/manifest.json` for the same fields (entries are not signal-filtered until you run `context build` or `pack --observation-file`).
3. **Optional keyword pass:** `rgd observe > /tmp/rgd-observe.json` then `rgd knowledge pack --observation-file /tmp/rgd-observe.json --query '<keyword>'`.
4. Read only the Markdown paths you need for the current task.

Until `rgd observe` fully covers every workflow signal, **Adapter** guidance for ad-hoc `gh` / `git` inspection lives in `.cursor/rules/evidence-temporary.mdc` (see also repo `AGENTS.md`).

## Authoring rules for new review knowledge

- Keep each file atomic — see [`.reinguard/README.md` § Atomicity](../README.md#atomicity).
- Use required front matter: `id`, `description`, `triggers` (non-empty, unique case-insensitively), **`when`** (match when this entry should surface — e.g. PR-scoped review docs use `github.pull_requests.pr_exists_for_branch`).
- Prefer stable guidance over PR-specific details or evidence-only snapshots.

## Review/update loop

1. Periodically extract PR review comments and refresh recurring patterns into atomic knowledge files (one concern per file).
2. Update knowledge docs as needed.
3. Run **`rgd knowledge index`** and commit `manifest.json`.
4. Validate with **`rgd config validate`**.
