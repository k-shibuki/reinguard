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
`.reinguard/procedure/` action cards (`implement`, `change-inspect`, `pr-create`,
etc.); details delegated to Knowledge documents where noted.

## HS-LOCAL-VERIFY (conditional)

Run the applicable subset before each push:

- **Go code changed**: `go test ./... -race`, `go vet ./...`, `golangci-lint run`
- **Markdown changed**: `pre-commit run markdownlint-cli2 --all-files` (pinned hook version from `.pre-commit-config.yaml`; no ad-hoc `npx @latest` installs)
- **Config / schemas / knowledge changed**: `rgd config validate` from repo root

**HS-NO-SKIP** applies: do not omit any applicable step without a
documented exception (PR body or review disposition).

## Required local AI review (before PR creation)

After applicable HS-LOCAL-VERIFY steps pass and before `change-inspect`
declares the change ready for `pr-create`, run the repository-local
CodeRabbit gate from the repo root:

```bash
bash .reinguard/scripts/check-local-review.sh --base main --retry-on-rate-limit
```

- This is a **required pre-PR gate** for this repository; it does **not**
  replace PR-based CodeRabbit review or merge consensus.
- The script standardizes installation/authentication checks and executes
  the CLI review against the repository's `.coderabbit.yaml`. On a rate
  limit, it parses the cooldown **only from the latest rate-limit line in
  that CLI run** (so earlier log text does not affect the wait), adds a
  small **safety buffer** (default 30s; override with
  `RATE_LIMIT_RETRY_BUFFER_SEC`), then retries the review **once**
  automatically. If the cooldown cannot be parsed from that line, or a
  second consecutive rate limit occurs, treat the gate as failed.
- If the script cannot run (CLI missing, auth missing, second consecutive
  rate limit after retry, execution error), treat that as a failed gate and
  do not proceed to `pr-create`.
- Review findings are dispositioned in `change-inspect` using the shared
  four-category model from
  `.reinguard/policy/review--disposition-categories.md`; do not
  auto-dismiss them just because they came from the local CLI instead of
  the PR bot. Handle one local CR output as a batch before rerunning the
  CLI: fix all in-scope material findings, apply same-kind sweep where the
  fix pattern repeats, then rerun the gate on the stabilized head.

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
3. **GWT comments** — non-trivial tests **must** have a function-level
   Given / When / Then summary (see `testing--given-when-then.md` for
   format and table-driven rules).
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
- `.reinguard/policy/review--disposition-categories.md` — shared
  disposition vocabulary for review findings
- `.reinguard/knowledge/testing--strategy.md` — Perspectives, Table-driven
- `.reinguard/knowledge/testing--given-when-then.md` — GWT format
- `.reinguard/knowledge/implementation--defensive-config-validation.md` — defensive patterns
