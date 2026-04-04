# ADR-0006: gh CLI as sole authentication source

## Context

GitHub observation requires an OAuth token or equivalent credential.
Options considered:

1. **Environment variable only** (`GITHUB_TOKEN` or similar) — no extra
   tools; users must export tokens manually in local development; CI
   systems usually provide the variable automatically.
2. **`gh auth token` only** — reinguard shells out to the GitHub CLI to
   obtain a token string, then uses standard HTTP clients against GitHub
   APIs. Users authenticate with `gh auth login` once.
3. **Hierarchical fallback** — try `GITHUB_TOKEN` first, then
   `gh auth token`, then fail. Maximizes environments covered but
   complicates debugging ("which path was used?") and doubles maintenance.

In many CI environments (including GitHub Actions), **`gh` is
pre-installed** and automatically uses the job's `GITHUB_TOKEN` when
present, so `gh auth token` continues to work without a separate code path.

## Decision

Use **`gh auth token`** as the **sole** credential retrieval mechanism
for GitHub API access in the initial design:

- The **GitHub CLI (`gh`)** is a **runtime prerequisite** for GitHub
  platform observation.
- reinguard executes `gh auth token`, uses the returned token with REST
  and/or GraphQL clients, and does **not** implement a separate
  "token from environment only" code path.

Rationale: one familiar login flow for interactive development, minimal
token handling inside reinguard, and consistent behavior in CI where `gh`
is standard.

This ADR does not prescribe HTTP vs GraphQL for individual calls; see API
strategy in implementation. It only fixes **how credentials are
obtained**.

**Repository identity** (which GitHub `owner/name` the working tree is associated with) is **not** tied to this credential path:

- Resolve it **local-first** from `git` (`remote.origin.url` for github.com remotes)
- Fall back to `gh repo view` when local git configuration is unavailable or ambiguous
- Keep **`gh auth token`** as the authentication path for all HTTP/GitHub API observation

## Consequences

- **Easier**: Single authentication story; no parallel token plumbing in
  reinguard
- **Easier**: Token rotation and login UX delegated to `gh`
- **Harder**: Environments without `gh` cannot use GitHub observation
  without installing the CLI
- **Harder**: A subprocess call is required to obtain credentials (latency,
  dependency on `gh` binary availability and version)
