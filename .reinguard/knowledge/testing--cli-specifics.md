---
id: testing-cli-specifics
description: "urfave/cli v2 test pitfalls — global flag mutation, per-Flags instances, fixtures"
triggers:
  - urfave cli
  - CLI test
  - HelpFlag race
  - cli flag mutation
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# CLI tests (urfave/cli v2)

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

## Related

- `.reinguard/knowledge/testing--strategy.md` — general test strategy
