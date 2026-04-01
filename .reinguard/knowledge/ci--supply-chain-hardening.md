---
id: ci-supply-chain-hardening
description: CI supply-chain hardening for runner pinning and deterministic tool execution
triggers:
  - ubuntu-latest
  - runs-on latest
  - npx latest
  - supply chain
  - ci hardening
  - deterministic ci
when:
  op: exists
  path: github.ci.ci_status
---

# CI Supply-Chain Hardening

## Rule 1: Pin runner image versions

Do not use `runs-on: ubuntu-latest` in required workflows.

Use an explicit runner image (for example, `ubuntu-24.04`) and review version
updates intentionally.

Rationale: `latest` can move without repository changes, causing unreviewed
runtime drift in CI.

## Rule 2: Avoid ad-hoc package execution with floating versions

Do not run `npx --yes <package>@latest` in CI or required local verification.

Use pinned tool sources instead:

- pinned action version in workflow
- pinned pre-commit hook revision in `.pre-commit-config.yaml`
- repository-managed script with fixed version contract

Rationale: floating package resolution introduces supply-chain risk and
non-reproducible verification.

## Rule 3: Treat hardening as policy candidates

If violations repeatedly appear in review, promote this guidance from knowledge
to policy (`.reinguard/policy/`) and reference it from safety invariants.
