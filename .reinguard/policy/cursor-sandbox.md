---
id: cursor-sandbox
description: Cursor repo-level sandbox.json — portable settings; user-level overrides; local CodeRabbit gate runs outside sandbox
triggers:
  - sandbox.json
  - Cursor sandbox
  - additionalReadwritePaths
  - enableSharedBuildCache
---

# Cursor sandbox (repo-level)

This policy applies to **committed** `.cursor/sandbox.json` at the repository root.

## Portable repo-level file

- Keep **only** repository-safe, machine-independent settings.
- Do **not** commit user-specific absolute paths (e.g. `/home/<user>/...`).
- **`additionalReadwritePaths`:** may include workspace-relative paths such as `".tmp"` (tool caches: pre-commit, golangci-lint, Go build cache, CodeRabbit CLI when routed via `with-repo-local-state.sh`) and `".reinguard/local"` (gitignored substrate gates, Adapter resume, scratch). Machine-specific home paths belong in user-level `~/.cursor/sandbox.json`, not in the repo.
- **`networkPolicy.allow`:** may list **portable** hostnames needed for local verification in this repo (for example `api.github.com`, `github.com`, Go module hosts, and CodeRabbit CLI endpoints). WebSocket/proxy limits can still block the CodeRabbit CLI inside the sandbox even when hosts are allowed; see `review--local-coderabbit-cli.md`.
- Prefer repo-local writable caches under `/.tmp/` (workspace-relative) for local verification commands (for example via `.reinguard/scripts/with-repo-local-state.sh`) instead of depending on user-home cache paths. Reinguard-owned local state (gates, Adapter resume) lives under `/.reinguard/local/` (see ADR-0011, ADR-0014, ADR-0015).

## User-level overrides

Per Cursor docs, optional untracked `~/.cursor/sandbox.json` merges with the repo file; user-specific paths belong there if needed.

## Related

- `coding--preflight.md` — Required local AI review (command, outside-sandbox requirement)
- `review--local-coderabbit-cli.md` — WebSocket / Bun limitation in sandboxed runs
