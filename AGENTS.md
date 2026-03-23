# AGENTS.md

Guidelines for AI-assisted review of **reinguard** (Go). Severity model
follows the bridle project: flag **P0** and **P1** only; do not nitpick
style (gofmt, golangci-lint enforce it).

## Project context

reinguard is a spec-driven control-plane **substrate**: it reads
repo-owned configuration, performs structured observation and evaluation,
and emits typed JSON for agents. Normative architecture is in `docs/adr/`.

## P0 (blocking)

- Logic bugs in match, resolution, observation merge, or CLI contracts
- Incorrect JSON/schema shapes vs `pkg/schema/` and `docs/cli.md`
- Security: token leakage to stdout, unsafe shell when running git/gh,
  SSRF patterns in HTTP clients
- Missing tests for changed non-trivial behavior

## P1 (significant)

- ADR drift (implementation contradicts ADR without follow-up ADR)
- Missing traceability: PR should `Closes #N`; commits should `Refs: #N`
- Weak error handling (swallowed errors, ambiguous exit codes vs `docs/cli.md`)
- Incomplete boundary tests when behavior is user-visible

## Go specifics

- Prefer **small packages** under `internal/`; keep CLI thin in `cmd/rgd`.
- **Table-driven tests** and **httptest** for HTTP; no network in default tests.
- Observation and guards must respect **ADR-0005** (no agent-internal files)
  and **ADR-0006** (`gh auth token` for GitHub).

## References

- `docs/cli.md` — SSOT for CLI flags, stdout/stderr, exit codes (when present)
- `.cursor/rules/test-strategy.mdc` — test expectations
