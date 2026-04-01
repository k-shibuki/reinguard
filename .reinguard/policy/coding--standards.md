---
id: coding-standards
description: Language policy for persisted artifacts, Go defaults, change scope, and documentation authority
triggers:
  - English artifacts
  - gofmt
  - ADR authority
  - same-kind drift
  - change scope
---

# Coding standards

Repository-wide coding and documentation rules.

## Language

- Chat with the user may follow the user's language.
- **All persisted artifacts** (code comments, commit messages, branch names, Issues, PRs, ADRs, README, CI config) must be **English**.

## Go

- Format with `gofmt` / `goimports` before commit.
- Run `go vet ./...` and `go test ./...` locally before push (see `.reinguard/policy/safety--agent-invariants.md` § **HS-LOCAL-VERIFY**).
- Prefer small packages under `internal/` and `cmd/` per repository layout conventions.
- **Documentation**: meaningful godoc for exported APIs and package comments; mechanical enforcement via `golangci-lint` / `revive` — see [coding--godoc.md](coding--godoc.md).
- **Complexity**: `golangci-lint` / `gocyclo` with `min-complexity: 15` (aligned with Go Report Card); refactor or extract helpers when a function exceeds the threshold.

## Markdown

- Lint Markdown with `pre-commit run markdownlint-cli2 --all-files` before commit (pinned by `.pre-commit-config.yaml`, no ad-hoc package install).
- Pre-commit hook and CI job `lint-markdown` enforce the same rules.

## Change scope

- Before hand-off, **search for same-kind** occurrences (parallel wording, config, or call sites), including **`.reinguard/`** and **`.cursor/`**, and reconcile them in the **same deliverable** when in scope for the task.
- Intentional gaps need an **explicit rationale** (PR body or review disposition), not silent omission.
- Before hand-off, verify per `.reinguard/policy/coding--preflight.md` (defensive checks, test design, self-review).

## Documentation authority

- **ADRs** under `docs/adr/` are the normative architecture record.
- `docs/cli.md` is the CLI SSOT (referenced by ADR-0008).

## Related

- `.reinguard/policy/coding--godoc.md` — godoc semantics and revive gate
- `.reinguard/policy/safety--agent-invariants.md` — HS-LOCAL-VERIFY and other HS-* codes
- `.reinguard/policy/catalog.yaml` — policy index
