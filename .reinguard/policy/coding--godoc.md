---
id: coding-godoc
description: Godoc and package comments — semantic expectations and mechanical gate (revive via golangci-lint)
triggers:
  - godoc
  - package comment
  - exported API
  - revive
  - documentation
---

# Go documentation (godoc)

## Why

Exported APIs and packages are read by humans, `go doc`, and review automation. Missing comments hide contracts and intent; placeholder comments waste attention.

## Strategy (layers)

Docstring quality is enforced in **three layers**. CodeRabbit and other AI tools are **optional**; they do not replace policy or CI.

1. **Mechanical gate (tool-agnostic)** — `golangci-lint` / `revive` in CI (`lint-go`) and local HS-LOCAL-VERIFY enforce **presence** of doc comments on exported symbols and package comments (see § Mechanical gate). This is the substrate-level check: standard Go tooling, not vendor-specific.
2. **Semantic expectation (policy + review)** — This document defines **meaningful** comments: English, first sentence names the symbol and states purpose, behavior and errors where non-obvious. Self-inspection (`change-inspect`) and human review judge content; silence the linter with filler text is not sufficient.
3. **Optional assist (CodeRabbit)** — Repository `.coderabbit.yaml` may enable finishing touches such as docstring suggestions or follow-up PRs. That output is **suggestion only**. It must be reviewed for accuracy against ADRs, `docs/cli.md`, and repository terminology. **Mechanical insertion** (template text, restating identifiers, or bulk-generated comments without substantive meaning) is **forbidden**; reject or rewrite before merge.

Unexported helpers: `revive` does not require comments on every private symbol. Add or improve comments where complexity, invariants, or boundaries would otherwise mislead readers; skip trivial wrappers.

## Semantic expectations (must)

- **English** for all persisted comment text (see [coding--standards.md](coding--standards.md) § Language).
- **Package comment** (one file per package): first sentence should name the package and summarize its role (e.g. `Package foo implements …`). For `package main`, leading with `Command …` or `Package main …` is acceptable if the sentence clearly states what the program is.
- **Exported identifiers** (types, funcs, methods, constants, variables in non-test code): the **first sentence** of the doc comment must name the symbol and state **what it does or why it exists** — not only repeat the name.
- **Meaningful content**: prefer behavior, inputs/outputs, error semantics, or ADR references where non-obvious. **Do not** use comments whose only substance is generic filler (e.g. a single line that restates the identifier without adding information).
- **Tests**: exported `Test…` / `Benchmark…` / `Example…` should have a one-line comment describing the scenario under test when non-obvious (table-driven cases may rely on subtest names).

## Mechanical gate (substrate)

- **Tooling**: `golangci-lint` enables the `revive` linter with a **narrow** rule set configured in the repository root `.golangci.yml`:
  - `exported` — exported symbols must have a doc comment (stuttering check disabled to avoid churn on names like `resolve.ResolveState`).
  - `package-comments` — each package must have a package comment.
- **Where it runs**: the same config as local **pre-commit** (`golangci-lint run`) and CI job **`lint-go`**. There is **no** separate `docscan` binary; a custom scanner would duplicate revive and drift from standard tooling.
- **When CI fails**: read the `revive` line (file and symbol), then fix the **content** per the Semantic expectations above — not only to silence the linter with empty or generic text.

## Related

- [coding--standards.md](coding--standards.md) — language, change scope, HS-LOCAL-VERIFY alignment
- [safety--agent-invariants.md](safety--agent-invariants.md) — HS-LOCAL-VERIFY
- [review--self-inspection.md](review--self-inspection.md) — documentation impact dimension
