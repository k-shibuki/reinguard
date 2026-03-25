---
id: coding-standards
description: Language policy for persisted artifacts, Go defaults, and documentation authority
triggers:
  - English artifacts
  - gofmt
  - ADR authority
---

# Coding standards

Repository-wide coding and documentation rules. The Cursor Adapter rule `reinguard-bridge.mdc` § Always-active policy points here as SSOT.

## Language

- Chat with the user may follow the user's language.
- **All persisted artifacts** (code comments, commit messages, branch names, Issues, PRs, ADRs, README, CI config) must be **English**.

## Go

- Format with `gofmt` / `goimports` before commit.
- Run `go vet ./...` and `go test ./...` locally before push (see `.reinguard/policy/safety--agent-invariants.md` § **HS-LOCAL-VERIFY**).
- Prefer small packages under `internal/` and `cmd/` per repository layout conventions.

## Documentation authority

- **ADRs** under `docs/adr/` are the normative architecture record.
- `docs/cli.md` is the CLI SSOT (referenced by ADR-0008).

## Related

- `.reinguard/policy/safety--agent-invariants.md` — HS-LOCAL-VERIFY and other HS-* codes
- `.reinguard/policy/catalog.yaml` — policy index
