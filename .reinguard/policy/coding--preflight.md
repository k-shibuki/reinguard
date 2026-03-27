---
id: coding-preflight
description: Pre-commit and pre-handoff verification obligations — HS-LOCAL-VERIFY expansion, defensive checks, test design, self-review
triggers:
  - preflight
  - HS-LOCAL-VERIFY
  - self-review
  - before commit
  - before push
  - verification
---

# Preflight verification

Verification obligations before commit/push and hand-off. Referenced by
`implement`, `change-inspect`, and `pr-create` commands; details delegated
to Knowledge documents where noted.

## HS-LOCAL-VERIFY (conditional)

Run the applicable subset before each push:

- **Go code changed**: `go test ./... -race`, `go vet ./...`, `golangci-lint run`
- **Markdown changed**: `npx --yes markdownlint-cli2@latest '**/*.md'` (no local Node package install required)
- **Config / schemas / knowledge changed**: `rgd config validate` from repo root

**HS-NO-SKIP** applies: do not omit any applicable step without a
documented exception (PR body or review disposition).

## Defensive implementation checks

Before committing new or changed logic, verify:

1. **Nil pointer guard** — every exported function that receives a pointer
   or interface checks for `nil` at the entry point before dereferencing.
2. **No silent ignore** — when a configuration key is present, validate
   its type and value; return an error on mismatch instead of silently
   falling through.
3. **Blank / duplicate ID rejection** — enabled collection entries with
   empty or duplicate identifiers must produce an error, not a silent
   skip or overwrite.

See `.reinguard/knowledge/implementation--defensive-config-validation.md`
for Go patterns and examples.

## Test design confirmation

For each behavior change, confirm:

1. **Normal / Abnormal / Boundary** — at least one automated case per
   perspective (see `testing--strategy.md` § Perspectives).
2. **Table-driven applicability** — use table-driven format when the same
   function has two or more test scenarios; a single-scenario test may
   remain standalone (see `testing--strategy.md` § Table-driven tests).
3. **GWT comments** — add a **function-level** Given / When / Then summary on
   non-trivial tests; for **table-driven** tests, omit GWT inside the `t.Run`
   loop (see `testing--given-when-then.md`). Trivial single-assert tests may
   omit GWT.
4. **Setup error handling** — never discard errors from test setup calls
   (`_ = f(...)`); use `t.Fatal(err)` to fail fast on unexpected setup
   failures.

## Self-review checklist

Before hand-off (commit or PR creation), scan the diff for:

- [ ] No `_ = <fallible call>` in test setup
- [ ] No silent type-assertion fallthrough on config values
- [ ] Validate / walk logic stays aligned with runtime evaluation for the same
  decoded keys (no “present but wrong type” silent branch)
- [ ] Doc impact list carried forward from implementation
- [ ] Same-kind sweep completed per `coding--standards.md` § Change scope

## Related

- `.reinguard/policy/safety--agent-invariants.md` — HS-LOCAL-VERIFY, HS-NO-SKIP
- `.reinguard/policy/coding--standards.md` — Change scope
- `.reinguard/knowledge/testing--strategy.md` — Perspectives, Table-driven
- `.reinguard/knowledge/testing--given-when-then.md` — GWT format
- `.reinguard/knowledge/implementation--defensive-config-validation.md` — defensive patterns
