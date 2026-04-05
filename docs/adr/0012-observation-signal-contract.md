# ADR-0012: Observation signal contract

## Context

`rgd observe` merges provider outputs into a versioned observation document
(ADR-0009). Downstream rules, guards, and agents rely on **field names and
meanings** staying stable and honest: the substrate must not present aggregates
as if they were something else.

The GitHub **reviews** facet initially exposed REST pull-review-comment list
length under `review_threads_unresolved`. That does not match GitHub’s review
**thread** model (REST list items are per-comment rows, including replies, and
lack per-thread resolution state). Agents and the `merge-readiness` guard need
**unresolved thread counts** grounded in platform truth.

Phase 2 work (e.g. richer PR/CI/issue signals for an FSM) will add more fields;
this ADR defines **taxonomy, naming, REST vs GraphQL policy, and phase
boundaries** so extensions stay consistent.

## Decision

### 1. Signal taxonomy

- **Fact signals** — Values taken from platform APIs or explicit, documented
  aggregations over API payloads (counts, flags). Names describe observable
  facts, not agent judgment.
- **Derived signals** — Deterministic combinations of facts computed inside
  `rgd` with a fixed rule (e.g. `working_tree_clean` from porcelain line
  count). Documented in `docs/cli.md`.
- **Excluded** — Agent-internal state, session files, and semantic
  interpretation of review text (ADR-0005).

### 2. GitHub facet contracts (summary)

| Facet | Primary transport | Notes |
|-------|-------------------|--------|
| `issues` | REST (e.g. Search) | Open counts, etc. |
| `pull_requests` | REST + local git | Branch ↔ PR linkage |
| `ci` | REST | Combined status for `HEAD` |
| `reviews` | **GraphQL** for thread resolution | REST cannot express `isResolved` per thread |

Per-thread **resolution** for PR review threads requires GraphQL
`repository.pullRequest.reviewThreads` (see repository knowledge:
`.reinguard/knowledge/review--github-thread-api.md`). Authentication remains
`gh auth token` only (ADR-0006); invoking GraphQL via `gh api graphql` is
consistent with that policy.

### 3. Reviews facet fields (P1 contract)

After this ADR, the `github.reviews` subtree (under `signals.github`) SHALL
expose at least:

| Field | Meaning |
|-------|---------|
| `review_threads_total` | Count of review threads returned for the PR (after pagination completes or as documented when incomplete). |
| `review_threads_unresolved` | Count of threads where the platform reports not resolved. |
| `pagination_incomplete` | True when the engine could not read all pages of thread data (e.g. pagination truncated). |

Naming: `snake_case`; counts use `_total` / `_unresolved` as above.

### 4. Guards

Built-in and declarative guards that reference `github.reviews.*` SHALL treat
these fields according to `docs/cli.md`. The `merge-readiness` guard uses
`review_threads_unresolved == 0` as a **merge gate signal**; that value MUST
reflect unresolved **threads**, not raw comment row counts.

### 5. Schema and documentation

- `observation-document.json` keeps `signals` as open object (`additionalProperties`);
  new keys are documented in **`docs/cli.md`** (CLI I/O SSOT, ADR-0008).
- Bumping **`schema_version`** follows ADR-0008 when the repo’s declared
  contract version must move with binary-embedded schemas (minor bump when
  adding or clarifying observation fields without breaking the top-level
  envelope).

### 6. Phase boundary: P1 hotfix vs Phase 2

- **P1 (this ADR’s immediate follow-up)** — Correct reviews facet semantics
  (GraphQL threads, accurate `pagination_incomplete`, `review_threads_total`).
- **Phase 2** — Additional PR/CI/issue fields for FSM (separate issues; e.g.
  mergeability, check runs, issue labels) without changing the reviews
  contract’s meaning. **P2-1 (#70)** extends `signals.github.pull_requests` and
  `signals.github.reviews` per `docs/cli.md` (unified GraphQL PR context,
  `latestReviews` aggregates, optional `bot_reviewer_status` / `bot_review_diagnostics` with pluggable
  `enrich` names validated at provider build / `rgd config validate`).

## Consequences

- **Easier** — Agents and guards see merge/review readiness aligned with GitHub’s
  thread model; fewer false positives from comment-row counts.
- **Harder** — GraphQL pagination and rate limits must be handled explicitly;
  tests rely on HTTP/GraphQL stubs more than simple REST arrays.
- **Easier** — Phase 2 signal tables can extend the same taxonomy and `docs/cli.md`
  tables without renaming the P1 reviews fields.

## Related

- ADR-0005 (agent-internal exclusion)
- ADR-0006 (`gh` authentication)
- ADR-0008 (schema versioning)
- ADR-0009 (observation engine)
- `.reinguard/knowledge/review--github-thread-api.md`

## Amendment (2026-04, Issue #105)

Non-thread review findings use **documented, deterministic** CodeRabbit enrichment
fields on `bot_reviewer_status` and aggregate `bot_review_diagnostics.non_thread_findings_present`,
plus `signals.github.reviews.conversation_comments` and `signals.github.ci.check_runs`.
Fact vs derived semantics and merge-guard wiring are documented in `docs/cli.md`;
no semantic disposition is selected in Go beyond declared markers and counts.
