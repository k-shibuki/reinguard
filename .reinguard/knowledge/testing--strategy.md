---
id: testing-strategy
description: Go test strategy — coverage, perspectives, table-driven tests, CLI tests, mocks
triggers:
  - test strategy
  - coverage target
  - table-driven
  - CLI test
  - mock
  - httptest
  - urfave cli
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Test strategy (reinguard / Go)

Judgment aids for writing and reviewing tests. The Cursor Adapter rule `test-strategy.mdc` points here.

For assertion style and GWT comments, see [`testing--assertions.md`](testing--assertions.md) and [`testing--given-when-then.md`](testing--given-when-then.md).

## Goals

- Keep tests **fast and deterministic**: default `go test ./...` must not
  require network access or live GitHub API (use `httptest`, fixtures,
  hermetic git repos, or build tags for integration tests).
- Align tests with **Issues**: each PR should reference an Issue. The Issue
  **Test plan** records **intent** (what to prove, boundaries to watch), not an
  exhaustive case list. Design concrete Normal / Abnormal / Boundary cases from
  the diff during implementation; verify coverage in change-inspect (dimension 4).

## Coverage

- Target **≥80% module-wide coverage** on `main` once CI enforces it
  (see repository workflow). When adding packages, bring them toward that
  bar with focused unit tests before relying only on integration tests.

## Perspectives

For each behavior change, include automated cases covering:

1. **Normal** — primary success path
2. **Abnormal** — invalid input, I/O errors, validation failures (assert
   error messages or stable substrings where helpful)
3. **Boundary** — empty collections, zero values, min/max where meaningful

Include **at least as many failure-oriented cases as happy-path cases**
for non-trivial logic (match engine, resolution, schema validation).

## Table-driven tests

Prefer table-driven tests when the **same function or entry point has two or
more scenarios** (operator matrices, resolution ties, config variants). That
includes pairing one success case with one failure case for the same helper —
use a single table with a `wantErr string` field (empty string means success;
non-empty means `strings.Contains` on the error). Use `t.Run(name, ...)` for
clear failure attribution.

A function with only **one scenario** may use a standalone test — do not
force table-driven structure on single-case tests.

For **GWT comments** with table-driven tests, see
[`testing--given-when-then.md`](testing--given-when-then.md) § Table-driven tests.

When `govet` `fieldalignment` nags on large case structs, you may use
`//nolint:govet` on the table struct with a short rationale (same pattern as
`internal/observe/engine_test.go`).

## Recursive evaluation paths

When new dependencies (registry, context, or similar) are threaded through
**recursive** helpers — for example `count` / `any` / `all` with nested `when`,
or logical combinators — extend table-driven tests with at least one row per
**distinct entry path**, not only top-level shapes. Otherwise a regression in
plumbing inside nested clauses can still pass while top-level cases stay green.

Setup error handling (never `_ =` fallible calls): see [`testing--setup-error-handling.md`](testing--setup-error-handling.md).

## CLI tests (urfave/cli v2)

1. **Package-global flags**: `urfave/cli/v2` exposes `cli.HelpFlag` and
   `cli.VersionFlag` as shared `*BoolFlag` instances. The library appends them to
   every `App` and subcommand; **concurrent `App.Run` mutates the same pointers**
   and trips the race detector. Production code uses `HideHelp: true`,
   `HideVersion: true`, per-app root clones (`newRootHelpFlag`, `newRootVersionFlag`
   with a `version` `Action` calling `cli.ShowVersion`), and `hideHelpOnCommands`
   so subcommands never append the globals (see `internal/rgdcli/rgdcli.go`).
2. **Per-`Flags` slice instances**: The library also mutates flags during
   `Apply`. **Do not register the same `*cli.BoolFlag` / `*cli.StringFlag` on
   more than one command's `Flags` slice.** Use factories (`newSerialFlag`,
   `observeFlags`, etc.) — **a new instance per slice**.
3. **Fixtures**: Shared YAML for CLI tests lives in `internal/rgdcli/fixtures_test.go`.

## Mocks and subprocesses

- **HTTP**: `net/http/httptest` for GitHub API shapes; do not hit real API
  in default tests.
- **Git**: temporary repositories with `git init` or mocked command
  runners; avoid depending on the developer's global git config.
- **`gh`**: inject fake token or stub the command runner in unit tests;
  integration tests may use build tag `integration`.

## Review expectations

PRs that change production behavior without updating or adding tests
should be rejected unless the change is documentation-only.

## Related

- [`testing--assertions.md`](testing--assertions.md)
- [`testing--given-when-then.md`](testing--given-when-then.md)
- [`testing--setup-error-handling.md`](testing--setup-error-handling.md)
- `.reinguard/knowledge/manifest.json` — index entry for this document
