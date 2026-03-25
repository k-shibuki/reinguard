---
id: review-ops
description: How rgd indexes knowledge, validates manifests, and how agents retrieve review docs
triggers:
  - knowledge operations
  - read flow
  - review flow
  - retrieval strategy
  - rgd knowledge index
---

# Knowledge Operations for reinguard

## Implemented behavior (ADR-0010)

- Each knowledge file is Markdown under `.reinguard/knowledge/` with YAML front matter
  (`id`, `description`, `triggers`).
- **`rgd knowledge index`** scans `*.md`, reads front matter, and writes
  `.reinguard/knowledge/manifest.json` (committed; run after changing metadata).
- **`rgd config validate`** checks the manifest schema, that paths exist, that the manifest
  matches front matter (freshness), and emits optional size/trigger-count hints.
- **`rgd knowledge pack`** prints JSON `{ "entries": [...] }` with full metadata; optional
  **`--query`** filters by substring match on triggers.
- **`rgd context build`** includes `knowledge.entries` in operational context JSON.

## Practical retrieval flow

1. Run `rgd knowledge pack` or `rgd context build`.
2. Open `.reinguard/knowledge/manifest.json` (or use `entries` from JSON) for id, path,
   description, and triggers.
3. Optionally use `rgd knowledge pack --query <keyword>` to narrow entries.
4. Read only the Markdown files you need for the current task.

## Authoring rules for new review knowledge

- Keep each file atomic (one concern per file).
- Use required front matter: `id`, `description`, `triggers` (non-empty list).
- Prefer stable guidance over PR-specific details.
- Put evidence snapshots in `review--source-summary.md`, not in atomic rule docs.

## Review/update loop

1. Periodically extract PR review comments and refresh patterns in `review--source-summary.md`.
2. Update atomic docs as needed.
3. Run **`rgd knowledge index`** and commit `manifest.json`.
4. Validate with **`rgd config validate`**.
