# reinguard CLI (`rgd`) — SSOT

This file is the **single source of truth** for command-line behavior,
flags, stdout/stderr, and exit codes (ADR-0008). The `rgd` implementation
must match this document; do not duplicate normative tables in the ADR
body or README.

## Global flags

| Flag | Env | Description |
|------|-----|----------------|
| `--config-dir` | `REINGUARD_CONFIG_DIR` | Path to config directory (default: `<git-root>/.reinguard`) |
| `--cwd` | — | Working directory for git/gh subprocesses (default: process CWD) |
| `-o`, `--output` | — | Reserved for future file output (optional) |
| `--serial` | — | Run observation providers sequentially (default: parallel) |
| `--fail-on-non-resolved` | — | Exit non-zero when state/route outcome is `ambiguous`, `degraded`, or `unsupported` |

With **urfave/cli v2**, place flags that must apply to a **nested** subcommand
**after** the subcommand name (e.g. `rgd state eval --config-dir DIR`), not only
before `state`.

## stdout vs stderr

- **Machine-readable JSON** (observation, evaluation, context) → **stdout** only.
- Human-readable progress, warnings, validation notes → **stderr**.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success; default even for ambiguous/degraded/unsupported evaluation unless `--fail-on-non-resolved` |
| 1 | Usage error, validation failure, or missing required flag |
| 2 | Unexpected internal error |

## Command tree

```text
rgd version
rgd config validate
rgd schema export [--dir DIR]
rgd gate record <gate-id> --status pass|fail [--checks-file FILE]
rgd gate status <gate-id>
rgd gate show <gate-id>
rgd observe [workflow-position]
rgd observe git
rgd observe github
rgd observe github issues
rgd observe github pull-requests
rgd observe github ci
rgd observe github reviews
rgd state eval [--observation-file FILE]
rgd route select [--observation-file FILE] [--state-file FILE]
rgd guard eval <guard-id> [flags...]
rgd knowledge index
rgd knowledge pack [--query STRING] [--observation-file FILE]
rgd context build [--observation-file FILE]
rgd review reply-thread --pr N --in-reply-to ID --commit-sha SHA --path FILE --line N (--body TEXT | --body-file FILE)
rgd review resolve-thread --thread-id ID
rgd ensure-labels
rgd labels list [--category TYPE]
rgd labels sync [--dry-run]
```

Phase 1 does **not** define command aliases (e.g. no `pr` for `pull-requests`).

## Maintainer / repository commands

These commands read [`.reinguard/labels.yaml`](../.reinguard/labels.yaml) and shell out to `gh` for GitHub label operations. They are **repository tooling** for this repo’s label SSOT (not the core observation/evaluation pipeline). See also [`.github/CONTRIBUTING.md`](../.github/CONTRIBUTING.md) § Labels.

| Command | Purpose |
|---------|---------|
| `rgd ensure-labels` | Create missing labels on the current repo (`gh label create`). |
| `rgd labels list` | Print type / exception / scope label names as JSON (stdout). |
| `rgd labels sync` | Align existing GitHub label color/description with `labels.yaml` (`gh label edit`); does not delete extra labels. Use `--dry-run` to preview. |

Issue/PR **policy enforcement** and related repository tooling scripts live under **`.reinguard/scripts/`** (for example `check-commit-msg.sh`, `check-pr-policy.sh`, `check-issue-policy.sh`, `sync-issue-templates.sh`, `check-coverage-threshold.sh`, `check-local-review.sh`). Invoke them with `bash .reinguard/scripts/<name>.sh` from the repository root; they are **not** part of the shipped `rgd` binary and are **not** wrapped by `Makefile` targets.

Deterministic policy validators (`check-pr-policy.sh`, `check-issue-policy.sh`, `check-commit-msg.sh`) are the only current candidates for future Go-backed `rgd` migration. External-CLI and repository-maintenance scripts (`check-local-review.sh`, `sync-issue-templates.sh`, `check-coverage-threshold.sh`) remain repository-local shell tooling. The shell script suite is exercised by Go integration tests under `internal/scripttest/` and `internal/labels/`.

## Provider IDs ↔ commands

| Provider `id` in `reinguard.yaml` | Collected by |
|-----------------------------------|--------------|
| `git` | `rgd observe git` or aggregate `observe` / `observe github` (indirect for branch) |
| `github` | `rgd observe github` (aggregate) or subcommands |

### Provider options (`reinguard.yaml`)

Each `providers[]` entry may include `options` (object). Built-in factories consume:

| Provider `id` | Key | Type | Description |
|---------------|-----|------|-------------|
| `github` | `api_base` | string | Optional GitHub REST API root override (e.g. `httptest` or a host whose REST root is `https://HOST/api/v3`); GraphQL uses `https://api.github.com/graphql` by default and maps `.../api/v3` → `.../api/graphql` for that Enterprise Server shape; leading/trailing space trimmed |
| `github` | `bot_reviewers` | array | Optional. Each element: `id` (string, required, `^[a-z0-9_]+$`, unique), `login` (string, required), `required` (boolean, required — whether this bot participates in aggregate diagnostics), `enrich` (optional string array of built-in enrichment names). Drives `signals.github.reviews.bot_reviewer_status` and `bot_review_diagnostics`. Unknown `enrich` names fail `rgd config validate` / provider build. Built-in enrichments: `coderabbit` (rate-limit seconds, CodeRabbit Review Status markers, duplicate-comment count from `PullRequestReview.body`, `StatusClassifier`). |

The `git` provider accepts `options` for forward compatibility; keys are currently unused.

Subcommands filter which facets run inside the GitHub provider for faster targeted runs.

## `rgd observe`

Runs configured providers (from `reinguard.yaml`) unless a subcommand
narrows scope. Emits one **observation document** JSON object (see schema
`observation-document.json`).

### Explicit GitHub PR scope

Live observation commands accept optional GitHub scope flags:

| Flag | Description |
|------|-------------|
| `--branch BRANCH` | Observe GitHub PR linkage for `BRANCH` instead of the checked-out branch. |
| `--pr N` | Observe GitHub pull-request / review / CI facets for pull request `N`. |

Precedence:

1. `--pr` wins for PR-scoped GitHub facets.
2. Otherwise `--branch` wins over the checked-out branch.
3. Otherwise live observation uses the local checkout branch.

When `--pr` is set without `--branch`, `signals.github.pull_requests.current_branch`
tracks the PR head branch, while `observed_scope.local_branch_at_collect`
preserves the local checkout branch that invoked `rgd`.

`--branch` / `--pr` are **live-observe only**. They are rejected when
`--observation-file` is set on higher-level commands (`state eval`,
`route select`, `knowledge pack`, `context build`) because the scope is
already fixed by the saved observation document.

### Observation document fields (reinguard-native)

| Field | Type | Description |
|-------|------|-------------|
| `schema_version` | string | Contract version (ADR-0008) |
| `signals` | object | Namespaced provider outputs (`git`, `github`, …) |
| `diagnostics` | array | Optional structured messages |
| `degraded` | boolean | True if any provider failed or returned partial data |
| `meta` | object | Optional; may include `degraded_sources` (string array) |

### Non-fatal provider failure

If one provider fails, others still run; `degraded` is true and diagnostics
record the failure. Default exit code **0** unless `--fail-on-non-resolved`
is applied at a higher-level command that interprets evaluation.

### Rate limiting (GitHub)

The GitHub client retries **429** responses with limited exponential backoff.

### Local-first vs remote-only (GitHub provider)

Observation splits **local-first** data (no GitHub API) from **remote-only** data (REST/GraphQL via `gh auth token`, ADR-0006):

- **Local-first:** `signals.github.repository` — `owner` and `name` are resolved from `git config remote.origin.url` when it points at **github.com** (SSH or HTTPS). Optional `identity_source` is `local_git` or `gh_repo_view` (fallback when origin is missing or not a standard GitHub host). Branch-level facts live under `signals.git` (git provider).
- **Remote-only:** issues, pull-requests, CI, reviews, mergeability, and other API-backed facets. Failures attach diagnostics (e.g. `github.issues`, `auth_failed`) and set `degraded` while **keeping** local-first fields when they were collected.

In restricted environments (e.g. sandboxes where `gh repo view` or GraphQL returns **Forbidden**), identity from **origin** may still succeed so `context build` can resolve `state` / `route` from git + repository identity even when remote facets are unavailable.

### `signals.github.repository` (GitHub provider)

| Field | Type | Description |
|-------|------|-------------|
| `owner` | string | Repository owner (user or org). |
| `name` | string | Repository name. |
| `identity_source` | string | `local_git` (from `remote.origin.url`) or `gh_repo_view` (fallback). |

### `signals.git` (git provider)

| Field | Type | Description |
|-------|------|-------------|
| `branch` | string | Current branch name (empty if detached) |
| `detached_head` | boolean | True when `HEAD` is not on a named branch |
| `uncommitted_files` | number | Lines from `git status --porcelain` |
| `working_tree_clean` | boolean | True when there are no uncommitted changes |
| `stash_count` | number | Lines from `git stash list` |
| `ahead_of_upstream` | number | `git rev-list --count @{upstream}..HEAD` when upstream exists, else `0` |
| `behind_of_upstream` | number | `git rev-list --count HEAD..@{upstream}` when upstream exists, else `0` |
| `has_upstream` | boolean | True when `@{upstream}` resolves for the current branch |
| `stale_remote_branches_count` | number | Count of `git branch -r --merged origin/<default_branch>` lines (excludes `HEAD ->`), `0` if `origin/<default_branch>` is missing; uses `default_branch` from `reinguard.yaml` |

### `signals.github.pull_requests` (GitHub provider, pull-requests facet)

The **pull-requests** facet always includes REST-derived fields for the current branch. When `pr_exists_for_branch` is true, the provider also runs a **GraphQL** query (same request family as the reviews facet) and merges the following fields into `pull_requests`:

| Field | Type | Description |
|-------|------|-------------|
| `open_count` | number | Open PR count for the repo (REST search). |
| `current_branch` | string | Effective branch used for PR observation. Defaults to the checked-out branch; with `--branch` (and no `--pr`), the requested branch; with `--pr` (regardless of `--branch`), the PR head branch. |
| `pr_exists_for_branch` | boolean | Whether an open PR exists for the effective scope (resolved branch or explicit `--pr`). |
| `pr_number_for_branch` | number | Resolved PR number for the effective scope, or `0`. |
| `state` | string | GraphQL PR state lowercased: `open`, `closed`, `merged`. |
| `draft` | boolean | Draft flag. |
| `title` | string | PR title. |
| `base_ref` | string | Base branch name (`baseRefName`). |
| `head_ref` | string | PR head branch name (`headRefName`). |
| `head_sha` | string | Head commit OID. |
| `mergeable` | string | `mergeable`, `conflicting`, or `unknown` (from GraphQL `mergeable`). |
| `merge_state_status` | string | GraphQL `mergeStateStatus` lowercased (e.g. `clean`, `dirty`, `blocked`, `unstable`, `behind`). |
| `labels` | string array | Label names (first page, up to 20). |
| `closing_issue_numbers` | number array | Linked issues from `closingIssuesReferences` (first page, up to 20). |
| `observed_scope` | object | Scope metadata for intent checks. See [Observed scope fields (detail)](#observed-scope-fields-detail) below. |

### `signals.github.reviews` (GitHub provider, reviews facet)

Populated when the `reviews` facet runs (see `rgd observe github reviews`). Data comes from a unified GraphQL **PR context** query: `reviewThreads` (`isResolved`), `latestReviews`, and optional PR issue `comments` for configured `bot_reviewers` (ADR-0012).

| Field | Type | Description |
|-------|------|-------------|
| `review_threads_total` | number | Threads fetched for the current PR after pagination (or up to the engine page cap). |
| `review_threads_unresolved` | number | Threads where `isResolved` is false. Used by `merge-readiness`. |
| `pagination_incomplete` | boolean | True if not all thread pages could be read (e.g. pagination capped). |
| `review_inbox` | array | Unresolved thread records for agent action. See [Review inbox fields (detail)](#review-inbox-fields-detail) below. |
| `review_decisions_total` | number | Count of nodes returned from `latestReviews` (latest decision per reviewer). |
| `review_decisions_approved` | number | Count with state `APPROVED`. |
| `review_decisions_changes_requested` | number | Count with state `CHANGES_REQUESTED`. |
| `review_decisions_truncated` | boolean | True if `latestReviews` reports `hasNextPage` (more than one page of decisions not fetched). |
| `bot_reviewer_status` | array | One object per configured `bot_reviewers` entry. See [Bot reviewer fields (detail)](#bot-reviewer-fields-detail) below. |
| `bot_review_diagnostics` | object | Aggregated booleans over **required** bots only. See [Bot review diagnostics (detail)](#bot-review-diagnostics-detail) below. |

#### Observed scope fields (detail)

`pull_requests.observed_scope` may include:

| Field | Type | Description |
|-------|------|-------------|
| `local_branch_at_collect` | string | Checked-out local branch when the command ran (empty if detached). |
| `selection` | string | Scope selection mode: `current_branch`, `explicit_branch`, `explicit_pr`, or `none` when neither a local branch nor an explicit scope could drive PR observation. |
| `requested_branch` | string | Explicit `--branch` value when provided. |
| `requested_pr_number` | number | Explicit `--pr` value when provided. |
| `effective_branch` | string | Branch that drove PR observation after precedence rules were applied. |
| `resolved_pr_number` | number | PR number selected for the effective scope, or `0`. |
| `pr_head_branch` | string | Head branch returned by the resolved PR, when available. |
| `pr_head_sha` | string | Head SHA returned by the resolved PR, when available. |

#### Review inbox fields (detail)

Each element of `review_inbox` represents one unresolved GitHub review thread.

| Field | Type | Description |
|-------|------|-------------|
| `thread_id` | string | GraphQL review thread node id (`resolveReviewThread` input). |
| `root_comment_id` | number | Root review comment database id (`in_reply_to` for REST replies). |
| `is_outdated` | boolean | Thread outdated state from GraphQL. |
| `body` | string | Root review comment body when available. |
| `author` | string | Root review comment author login when available. |
| `path` | string | Anchored file path for the root review comment. |
| `line` | number | Anchored line number when available. |
| `original_line` | number | Original line number from the root review comment when available. |
| `start_line` | number | Multi-line start line when available. |
| `original_start_line` | number | Original multi-line start line when available. |
| `commit_sha` | string | Commit SHA on the root review comment when available. |
| `original_commit_sha` | string | Original commit SHA on the root review comment when available. |

#### Bot reviewer fields (detail)

Each element of `bot_reviewer_status` includes:

- **Identity and requirement:** `id`, `login`, `required`.
- **`status`:** classified string — `not_triggered`, `pending`, `completed`, `completed_clean`, `rate_limited`, `review_paused`, `review_failed`. `completed_clean` applies when an enrichment can prove a terminal clean result (e.g. CodeRabbit “No issues found” / “no actionable comments” in issue comments), with or without a `latestReviews` entry.
- **Review linkage:** `has_review`, `review_state` (from `latestReviews`, empty if none), `review_commit_sha` (OID from the latest review by this bot when present; for `coderabbit` enrichment, may be filled from `cr_reviewed_head_sha` parsed from the bot comment body when CodeRabbit summarizes a base..head range).
- **Comment window:** `latest_comment_at` (ISO8601 from the **newest** matching PR issue comment by `updatedAt` in the fetched `comments(last: 100)` window, or empty).
- **Status source:** `status_comment_at` (ISO8601) and `status_comment_source` (`status_marker` \| `fallback_latest`) identify which issue comment body was used for substring flags and issue-comment enrichment. The builder picks a **status-bearing** comment using semantic **tier** (terminal clean/completed > rate limit > paused > failure cues > other CodeRabbit markers), then the newest `updatedAt` within the winning tier—so an edited Review Status comment is not shadowed by a later short acknowledgment, and a newer terminal-clean comment supersedes an older rate-limit body on another comment.
- **Body substring flags (case-insensitive):** `contains_rate_limit`, `contains_review_paused`, `contains_review_failed`. These flags are computed from the **selected status comment** body above, not from `latest_comment_at` alone. `contains_rate_limit` false does not prove quota cleared globally; it means the selected status body no longer contains the substring (e.g. superseded by a higher-tier completion comment).
- **Optional enrich fields** (e.g. `rate_limit_remaining_seconds`, `cr_review_processing`, `cr_walkthrough_present`, `cr_reviewed_head_sha`, `cr_duplicate_findings_count` for `coderabbit`). Duplicate count is parsed from the latest `PullRequestReview.body`, not issue comments.
- **`rate_limit_remaining_seconds` (PR / `rgd observe`):** first, seconds are parsed from the **selected status comment** body (same patterns as the local CLI cooldown parser). Then **`max(0, parsed_seconds − elapsed_seconds)`** where **`elapsed_seconds`** is the wall time from **`status_comment_at`** (the selected comment’s `updatedAt`) to the **observation time** when `rgd` built the signal. So a body that says “wait 19 minutes and 47 seconds” is interpreted as that much time **from the comment update**, not from an arbitrary later read. If CodeRabbit **edits** that issue comment in place, GitHub advances `updatedAt`; the next fetch uses the new body and new **`status_comment_at`**, so the cooldown **re-anchors** to the edit time.
- **PR recovery after `rate_limited`:** before posting `@coderabbitai review`, wait **`cooldown_sec + 30`** where **`cooldown_sec`** is **`rate_limit_remaining_seconds`** from the observation JSON (already age-adjusted as above when enrichment is enabled), or else seconds parsed from the **selected status comment** body and aged manually using **`status_comment_at`** the same way. The **30**-second tail matches the local CLI default **`RATE_LIMIT_RETRY_BUFFER_SEC`** in `.reinguard/scripts/check-local-review.sh` (same contract as `--retry-on-rate-limit`). Normative detail: `.reinguard/knowledge/review--bot-operations.md` § Rate-Limit Recovery.

#### Bot review diagnostics (detail)

Object keys:

- **`bot_review_completed`**, **`bot_review_pending`**, **`bot_review_terminal`**, **`bot_review_failed`**, **`bot_review_stale`:** booleans aggregated over **required** bots only.
- **`bot_review_stale` (fail-closed):** true when any **required** bot is considered to have completed a review but the effective review commit does not match the current PR head, **or** when a completed review is expected but `review_commit_sha` is still empty after `latestReviews` and CodeRabbit enrichment fallback. If no required bots are configured, `bot_review_stale` is vacuously false.
- **`duplicate_findings_detected`:** true when **any** configured bot (required or optional) has `cr_duplicate_findings_count` > 0 in `bot_reviewer_status` (CodeRabbit re-listed findings under duplicate suppression without new threads).

See ADR-0013.

GraphQL failures for this query are reported as diagnostics with provider **`github.pr-query`** (non-fatal to other facets unless the whole provider degrades).

## `rgd state eval`

Evaluates `type: state` rules from configuration against an observation.

Committed workflow `state_id` priorities for this repository are documented in **ADR-0013** (`docs/adr/0013-fsm-workflow-states-and-adapter-mapping.md`).

### Inputs

- **Default:** runs observation inline (same as `rgd observe`) unless:
- `--observation-file FILE` points to a JSON observation document, or
- **stdin** JSON when `-` is passed as file (optional convention).
- `--branch BRANCH` / `--pr N` use the same live GitHub scope rules as
  [`rgd observe`](#explicit-github-pr-scope) when `--observation-file` is not set.
- Runtime gate statuses from `.reinguard/runtime/gates/*.json` are merged into
  the evaluation signal map as `gates.<gate-id>.*` before state resolution.

### Output

JSON object:

| Field | Description |
|-------|-------------|
| `kind` | `resolved` \| `ambiguous` \| `degraded` \| `unsupported` (ADR-0007) |
| `state_id` | When `resolved` |
| `rule_id` | Winning rule when `resolved` |
| `candidates` | Rule ids when `ambiguous` |
| `reason` | Human-readable summary when not `resolved`, or details for `unsupported` |
| `missing_evidence` | Present for `unsupported`: machine-oriented tags (e.g. `when_evaluation`, `rule_id:…`) |
| `re_entry_hint` | Present for `unsupported`: what to do next (re-entry contract per ADR-0007) |

**`degraded` vs `unsupported`:** `degraded` means evaluation ran but no trustworthy winner (no match, all suppressed by `depends_on`, etc.). `unsupported` means the substrate refused to interpret inputs safely (invalid `when` shape, wrong `rule_type` to `Resolve`, or a winning rule missing `state_id` / `route_id`).

## `rgd route select`

Evaluates `type: route` rules using:

- `--observation-file` (required unless default live observe)
- `--state-file` optional prior `state eval` JSON (merged into signals as `state` key)
- `--branch BRANCH` / `--pr N` for live observation only (same precedence as
  [`rgd observe`](#explicit-github-pr-scope))
- Runtime gate statuses from `.reinguard/runtime/gates/*.json` (merged as
  `gates.<gate-id>.*` before route evaluation)

### Output

Same shape as state eval with `route_id` instead of `state_id` when resolved, including `unsupported` and handoff fields.

`route_candidates` is always present when at least one matching route rule has a
non-empty `route_id` after `depends_on` suppression. It lists **all** such
matches **sorted by ascending `priority`** (lower numeric value wins), then
`rule_id`. The winning rule is the first entry when `kind` is `resolved`; when
`kind` is `ambiguous`, `candidates` lists tied `rule_id` values at the best
priority and `route_candidates` still reflects the full ordered match set.

When no route rule matches, `kind` is `degraded` and `route_candidates` is omitted.

## `rgd guard eval <guard-id>`

Phase 1 uses **flags only** for guard intent (no stdin JSON for guards).

Like `state` / `route` commands, this loads `reinguard.yaml` and `control/**/*.yaml` from
`--config-dir` (or the repo’s `.reinguard`). Built-in guards (for example `merge-readiness`)
run only after declarative resolution succeeds when `control/guards/*.yaml` contains rules with
matching `guard_id`; if there are no rules for that id, the built-in runs without a resolution
step (backward compatible). Runtime gate statuses are also merged into the flat
signal map as `gates.<gate-id>.*`.

## `rgd gate`

Runtime gate artifacts live under `.reinguard/runtime/gates/` and are validated
against the embedded `gate-artifact.json` schema. These artifacts are
**gitignored operational state**, not Semantics content.

### `rgd gate record <gate-id>`

Records one validated gate artifact for the current branch HEAD.

| Flag | Required | Description |
|------|----------|-------------|
| `--status` | yes | Top-level gate outcome: `pass` or `fail` |
| `--checks-file` | no | JSON array of check objects with fields `id`, `status`, and optional `summary`; check `status` may be `pass`, `fail`, or `skipped` |

The command attaches:

- current `head_sha`
- current branch name
- `recorded_at` (RFC3339 UTC)
- `schema_version`

It refuses to record on a detached HEAD.

### `rgd gate status <gate-id>`

Derives the effective status for the current checkout:

| Status | Meaning |
|--------|---------|
| `missing` | No artifact file exists for the gate |
| `invalid` | The artifact is unreadable or schema-invalid |
| `stale` | The artifact branch / `head_sha` does not match the current checkout, or current git identity cannot be determined |
| `fail` | The artifact is fresh and its recorded top-level status is `fail` |
| `pass` | The artifact is fresh and its recorded top-level status is `pass` |

### `rgd gate show <gate-id>`

Prints the validated raw artifact JSON.

### `merge-readiness` (built-in)

| Flag | Required | Description |
|------|----------|-------------|
| `--observation-file` | yes | Observation JSON path |

Evaluates merge signals. All conditions must be true for `ok == true`:

| # | Signal path | Condition | Fail reason |
|---|-------------|-----------|-------------|
| 1 | `git.working_tree_clean` | `== true` | git working tree not clean |
| 2 | `github.ci.ci_status` | `== "success"` (case-insensitive) | ci status is "X", want success |
| 3 | `github.reviews.review_threads_unresolved` | `== 0` | unresolved review threads: N |
| 4 | `github.reviews.bot_review_diagnostics.bot_review_pending` | `== false` | required bot review still pending |
| 5 | `github.reviews.bot_review_diagnostics.bot_review_terminal` | `== true` | required bot review not terminal |
| 6 | `github.reviews.bot_review_diagnostics.bot_review_failed` | `== false` | required bot review failed |
| 7 | `github.reviews.bot_review_diagnostics.bot_review_stale` | `== false` | required bot review is stale or missing review commit SHA |
| 8 | `github.reviews.review_decisions_changes_requested` | `== 0` | changes requested: N |
| 9 | `github.reviews.pagination_incomplete` | `== false` | review thread pagination incomplete |
| 10 | `github.reviews.review_decisions_truncated` | `== false` | review decisions truncated |

All signals fail closed on missing or invalid values (guard returns `ok: false`).

`merge-readiness` is a deterministic merge gate only for the signals above.
It does **not** prove that non-thread findings from the current PR review
cycle have been dispositioned; review-closure completeness remains a
procedure / policy responsibility.

### Output

JSON `{ "guard_id": "merge-readiness", "ok": true|false, "reason": "..." }`

## `rgd knowledge index`

Scans `.reinguard/knowledge/*.md`, parses YAML front matter (`id`, `description`,
`triggers`, **`when`** — all required), and writes `.reinguard/knowledge/manifest.json` with
`schema_version` set to the binary’s current contract version (ADR-0010). Prints
a one-line summary to stdout (human-readable).

`when` is a match expression (same shape as control rules; ADR-0002). It is
copied into the manifest as-is. **`rgd knowledge index`** does not walk `when` beyond
YAML parse; **`rgd config validate`** applies the same static checks as for control rules’
`when`: registered `eval:` names, known `op` values (see `internal/match`), required operands per
`op`, `eval: constant` requires `params.value` as boolean, and every `path` must start with
`git.`, `github.`, `state.`, `gates.`, or `$` / `$.` (quantifier scope). `knowledge index` rejects **duplicate triggers** (case-insensitive) and missing `when`.

After editing knowledge metadata in front matter, run this command and commit the
updated manifest so `rgd config validate` freshness checks pass.

## `rgd knowledge pack`

Reads `.reinguard/knowledge/manifest.json` and prints JSON:

```json
{ "entries": [ { "id": "...", "path": "...", "description": "...", "triggers": ["..."], "when": { } } ] }
```

Each entry’s `when` is either one clause object or an array of clause objects (same shapes as control rules; see ADR-0002). The sample above uses an object placeholder only.

Every committed manifest entry includes `when` (schema-required). Repo-relative `path` values point at Markdown files; bodies are not embedded.

| Flag | Description |
|------|-------------|
| `--query` | Optional. Case-insensitive substring match against each entry’s `triggers`. |
| `--observation-file FILE` | Optional. Observation JSON (same shape as `rgd observe` stdout). When set, the file’s `signals` object is flattened first, and entries are kept only if `when` matches that flat signal map (not state-resolved; use `context build` for `state.*` paths in `when`). |
| `--branch BRANCH` / `--pr N` | Optional for live observation only; same precedence as [`rgd observe`](#explicit-github-pr-scope). Rejected with `--observation-file`. |

**Selection when `--observation-file` is set:** entries included if `when` matches **or** `--query` matches triggers (OR union by `id`). When `--observation-file` is omitted, `--query` remains the only filter; with an empty query, all entries are returned.

If evaluating `when` fails (e.g. malformed clause), the entry is **still included** and a **`diagnostics`** array is added to the JSON with `severity: "warning"` and `code: "knowledge_when_eval"` (safe-side for judgment aids).

## `rgd context build`

Runs the default pipeline: **observe → state eval → knowledge filter → route select → guard eval
(merge-readiness) → operational context JSON**.

Runtime gate statuses are merged into the flat signal map first as
`gates.<gate-id>.*`. After state resolution, `state.kind`, `state.state_id`,
and `state.rule_id` are merged into the same flat signal map; **knowledge
`entries`** are then filtered with `match.Eval` per entry `when`. Route and
guard steps use that flat map; they do not see route/guard outcomes inside
`when` (avoids circularity).

- **`--observation-file FILE`**: if set, skips live `observe` and uses the
  given observation document JSON as input (same shape as `rgd observe` stdout).
  Useful for tests and golden fixtures.
- **`--branch BRANCH` / `--pr N`**: when live observation runs, pass explicit
  GitHub PR scope into the observe step using the same precedence as
  [`rgd observe`](#explicit-github-pr-scope). Rejected with
  `--observation-file`.

The `knowledge` object in the output has **`entries`** (same shape as
`rgd knowledge pack` stdout), not `paths` (ADR-0010).

Optional per-step flags may be added in future issues; Phase 1 runs the full
default chain when not using `--observation-file`.

The `state` field is the state-resolution **Result** (same JSON shape as `rgd state eval` stdout).
The `routes` array contains one route-resolution **Result** (same shape as `rgd route select` stdout,
including `route_candidates` when applicable). See `pkg/schema/operational-context.json` (`resolutionResult`).

## `rgd review reply-thread`

Thin transport for one threaded PR review reply. This command does **not**
classify findings or enforce consensus policy; it only validates required
transport fields and shells out to `gh api`.

| Flag | Required | Description |
|------|----------|-------------|
| `--pr` | yes | Pull request number |
| `--in-reply-to` | yes | Root review comment database id |
| `--commit-sha` | yes | Full 40-character commit SHA |
| `--path` | yes | Review comment file path |
| `--line` | yes | Review comment line number |
| `--body` | one of body/body-file | Inline reply body |
| `--body-file` | one of body/body-file | File containing the reply body |
| `--cwd` | no | Working directory for `gh` |

Payload matches the repository’s PR review transport contract:
`POST repos/{owner}/{repo}/pulls/{N}/comments` with `body`, `in_reply_to`,
`commit_id`, `path`, and `line`.

## `rgd review resolve-thread`

Thin transport for `resolveReviewThread` after consensus.

| Flag | Required | Description |
|------|----------|-------------|
| `--thread-id` | yes | GraphQL review thread node id |
| `--cwd` | no | Working directory for `gh` |

The command shells out to `gh api graphql` with a single
`resolveReviewThread` mutation.

### Authors: extending State / Gate / Guard

- **Normative FSM and gate rules:** [ADR-0013](adr/0013-fsm-workflow-states-and-adapter-mapping.md) (states, routes, Adapter mapping), [ADR-0014](adr/0014-runtime-gate-artifacts.md) (runtime gates, `gates.*` signals, freshness).
- **Operational checklist** (which files to update, `rgd config validate`, tests, knowledge manifest): `.reinguard/knowledge/workflow--state-gate-guard-extension.md`.
- **`knowledge.entries`:** Filtered using the **merged** flat map that includes `gates.<gate-id>.*` and, after state resolution, `state.kind`, `state.state_id`, and `state.rule_id`. Use **`rgd context build`** when authoring or debugging `when` clauses that reference `gates.*` or `state.*`. `rgd knowledge pack --observation-file` alone does **not** merge state or gates unless those keys are already in the observation file—see the `knowledge pack` section above.
- **`rgd gate status` / `rgd gate record`:** Verify CLI behavior and flag order against this document when documenting new gates in procedures.

## `rgd config validate`

Validates `reinguard.yaml`, `control/{states,routes,guards}/*.yaml`, and `knowledge/manifest.json` when
present, against embedded JSON Schemas. Also **builds enabled observation providers** (same path as `rgd observe`) so invalid `providers[].options` (e.g. unknown `bot_reviewers[].enrich` names) fail validation. Non-zero exit on hard validation
errors. **Deprecated** configuration keys (marked in JSON Schema) emit **warnings
on stderr** but still exit **0** when validation succeeds.

`config validate` does **not** validate `.reinguard/runtime/`; runtime gate
artifacts use the dedicated `rgd gate` schema and commands instead.

**`when` clauses (ADR-0002):** Control rules and knowledge manifest entries are checked with the same static validator: unknown `eval:` names, unknown `op` strings, missing required keys per `op` (e.g. `eq` needs `path` and `value`), `eval: constant` requires `params.value` (boolean), and `path` strings must use allowed roots (`git.`, `github.`, `state.`, `gates.`, or `$` / `$.` for nested quantifier clauses). Named evaluators: `rgd config validate` rejects unknown `eval:` names against the built-in registry. To list built-ins, call `evaluator.DefaultRegistry().ListRegistered()` from Go (sorted names), or see `internal/evaluator/`.

**`schema_version` vs this binary (ADR-0008):** `reinguard.yaml` declares a
semver `schema_version` synchronized with embedded JSON Schemas. This `rgd`
build compares it to its contract version (`MAJOR.MINOR.PATCH`):

| Relationship | Behavior |
|----------------|----------|
| **Major** differs from the binary’s contract | **Error** (exit non-zero); do not load an incompatible major line silently. |
| **Same major**, **minor or patch** differs | **Warning on stderr**, validation and load **continue** (older or newer skew). |
| **Exact match** | No schema-skew warning from this rule. |

Skew and deprecation messages go to **stderr**; success messages go to **stdout**.

When `knowledge/manifest.json` is present, validation also:

- Ensures each `entries[].path` exists under the repository root and is a file.
- Validates `when` clauses in each entry (same static rules as control rules: `eval` registry, `op` / operands, `path` prefixes).
- Re-indexes knowledge Markdown front matter and **errors** if the committed
  manifest is stale (run `rgd knowledge index` and commit).
- May emit **warnings** on stderr for large knowledge files or many triggers per
  entry (authoring hints only).

## Agent bootstrap (Cursor and other tools)

Repositories may add editor-specific rules that point agents at
`.reinguard/knowledge/manifest.json` and describe how to use `entries`, `when`, and
`rgd context build` / optional `knowledge pack` (see ADR-0010). `rgd` does not require a particular bridge file; this
repo includes `.cursor/rules/reinguard-bridge.mdc` as an example.

## `rgd schema export`

Writes all embedded schemas from `pkg/schema/` to `--dir`.

## CI parity (`.github/workflows/ci.yaml`)

Triggers: `push` to `main`, `pull_request` to `main`, and `workflow_dispatch`.

The following commands mirror the **effective shell commands** run in CI (paths
and env are as in GitHub Actions). Fork pull requests **skip** job `dogfood-rgd-github`; see
[`CONTRIBUTING.md`](../.github/CONTRIBUTING.md).

### Job `lint-go` (after `gate-policy`)

```bash
go mod download
go mod verify
go build ./...
# golangci-lint via golangci/golangci-lint-action with: --timeout=5m ./...
go vet ./...
```

### Job `test-go` (after `lint-go`)

```bash
go mod download
go test ./... -race -coverpkg=./... -coverprofile=coverage.out -count=1
bash .reinguard/scripts/check-coverage-threshold.sh 80 coverage.out
go build -o /tmp/rgd ./cmd/rgd
/tmp/rgd version
/tmp/rgd config validate
/tmp/rgd schema export --dir /tmp/rgd-schema-smoke
```

### Job `dogfood-rgd-cli` (after `test-go`)

```bash
go build -o /tmp/rgd ./cmd/rgd
/tmp/rgd --cwd "${GITHUB_WORKSPACE}" config validate
/tmp/rgd --cwd "${GITHUB_WORKSPACE}" observe git > /tmp/observe-git.json
# CI asserts JSON shape with jq (branch, detached_head, working_tree_clean)
```

### Job `dogfood-rgd-github` (non-fork PRs and pushes to `main`)

Condition: `github.event_name != 'pull_request' || github.event.pull_request.head.repo.full_name == github.repository`.

```bash
go build -o /tmp/rgd ./cmd/rgd
gh --version
/tmp/rgd --cwd "${GITHUB_WORKSPACE}" observe github > /tmp/observe-github.json
# CI asserts repository fields and absence of auth/diagnostic failures via jq
```

(`GH_TOKEN` / `GITHUB_TOKEN` is set by Actions for `gh` and the GitHub provider.)
