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
---

# Test strategy (reinguard / Go)

Judgment aids for writing and reviewing tests. The Cursor Adapter rule `test-strategy.mdc` points here.

For assertion style and GWT comments, see [`testing--assertions.md`](testing--assertions.md) and [`testing--given-when-then.md`](testing--given-when-then.md).

## Goals

- Keep tests **fast and deterministic**: default `go test ./...` must not
  require network access or live GitHub API (use `httptest`, fixtures,
  hermetic git repos, or build tags for integration tests).
- Align tests with **Issues**: each PR should reference an Issue; test cases
  should map to the Issue **Test plan** where applicable.

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

Prefer table-driven tests when the **same function has two or more test
scenarios** (operator matrices, resolution ties, config variants). Use
`t.Run(name, ...)` for clear failure attribution.

A function with only **one scenario** may use a standalone test — do not
force table-driven structure on single-case tests.

## Test setup error handling

Never discard errors from test setup or helper calls:

```go
// BAD — masks root cause
_ = r.Register("a", factory)

// GOOD — fails fast with diagnostic message
if err := r.Register("a", factory); err != nil {
    t.Fatal(err)
}
```

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
- `.reinguard/knowledge/manifest.json` — index entry for this document
