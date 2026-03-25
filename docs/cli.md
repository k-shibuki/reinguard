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
| `--fail-on-non-resolved` | — | Exit non-zero when state/route outcome is `ambiguous` or `degraded` |

With **urfave/cli v2**, place flags that must apply to a **nested** subcommand
**after** the subcommand name (e.g. `rgd state eval --config-dir DIR`), not only
before `state`.

## stdout vs stderr

- **Machine-readable JSON** (observation, evaluation, context) → **stdout** only.
- Human-readable progress, warnings, validation notes → **stderr**.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success; default even for ambiguous/degraded evaluation unless `--fail-on-non-resolved` |
| 1 | Usage error, validation failure, or missing required flag |
| 2 | Unexpected internal error |

## Command tree

```text
rgd version
rgd config validate
rgd schema export [--dir DIR]
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
rgd knowledge pack [--query STRING]
rgd context build [--observation-file FILE]
```

Phase 1 does **not** define command aliases (e.g. no `pr` for `pull-requests`).

## Provider IDs ↔ commands

| Provider `id` in `reinguard.yaml` | Collected by |
|-----------------------------------|--------------|
| `git` | `rgd observe git` or aggregate `observe` / `observe github` (indirect for branch) |
| `github` | `rgd observe github` (aggregate) or subcommands |

Subcommands filter which facets run inside the GitHub provider for faster targeted runs.

## `rgd observe`

Runs configured providers (from `reinguard.yaml`) unless a subcommand
narrows scope. Emits one **observation document** JSON object (see schema
`observation-document.json`).

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

## `rgd state eval`

Evaluates `type: state` rules from configuration against an observation.

### Inputs

- **Default:** runs observation inline (same as `rgd observe`) unless:
- `--observation-file FILE` points to a JSON observation document, or
- **stdin** JSON when `-` is passed as file (optional convention).

### Output

JSON object:

| Field | Description |
|-------|-------------|
| `kind` | `resolved` \| `ambiguous` \| `degraded` |
| `state_id` | When `resolved` |
| `rule_id` | Winning rule when `resolved` |
| `candidates` | Rule ids when `ambiguous` |
| `reason` | Human-readable when not `resolved` |

## `rgd route select`

Evaluates `type: route` rules using:

- `--observation-file` (required unless default live observe)
- `--state-file` optional prior `state eval` JSON (merged into signals as `state` key)

### Output

Same shape as state eval with `route_id` instead of `state_id` when resolved.

`route_candidates` is always present when at least one matching route rule has a
non-empty `route_id` after `depends_on` suppression. It lists **all** such
matches **sorted by ascending `priority`** (lower numeric value wins), then
`rule_id`. The winning rule is the first entry when `kind` is `resolved`; when
`kind` is `ambiguous`, `candidates` lists tied `rule_id` values at the best
priority and `route_candidates` still reflects the full ordered match set.

When no route rule matches, `kind` is `degraded` and `route_candidates` is omitted.

## `rgd guard eval <guard-id>`

Phase 1 uses **flags only** for guard intent (no stdin JSON for guards).

### `merge-readiness` (built-in)

| Flag | Required | Description |
|------|----------|-------------|
| `--observation-file` | yes | Observation JSON path |

Evaluates coarse signals: `github.ci.ci_status == success`,
`github.reviews.review_threads_unresolved == 0`, and `git.working_tree_clean == true`.

### Output

JSON `{ "guard_id": "merge-readiness", "ok": true|false, "reason": "..." }`

## `rgd knowledge index`

Scans `.reinguard/knowledge/*.md`, parses YAML front matter (`id`, `description`,
`triggers`), and writes `.reinguard/knowledge/manifest.json` with
`schema_version` set to the binary’s current contract version (ADR-0010). Prints
a one-line summary to stdout (human-readable).

After editing knowledge metadata in front matter, run this command and commit the
updated manifest so `rgd config validate` freshness checks pass.

## `rgd knowledge pack`

Reads `.reinguard/knowledge/manifest.json` and prints JSON:

```json
{ "entries": [ { "id": "...", "path": "...", "description": "...", "triggers": ["..."] } ] }
```

Repo-relative `path` values point at Markdown files; bodies are not embedded.

| Flag | Description |
|------|-------------|
| `--query` | Optional. Case-insensitive substring match against each entry’s `triggers`; only matching entries are returned. If omitted, all entries are returned. |

## `rgd context build`

Runs the default pipeline: **observe → state eval → route select → guard eval
(merge-readiness) → knowledge pack → operational context JSON**.

- **`--observation-file FILE`**: if set, skips live `observe` and uses the
  given observation document JSON as input (same shape as `rgd observe` stdout).
  Useful for tests and golden fixtures.

The `knowledge` object in the output has **`entries`** (same shape as
`rgd knowledge pack` stdout), not `paths` (ADR-0010).

Optional per-step flags may be added in future issues; Phase 1 runs the full
default chain when not using `--observation-file`.

## `rgd config validate`

Validates `reinguard.yaml`, `rules/*.yaml`, and `knowledge/manifest.json` when
present, against embedded JSON Schemas. Non-zero exit on hard validation
errors. **Deprecated** configuration keys (marked in JSON Schema) emit **warnings
on stderr** but still exit **0** when validation succeeds.

When `knowledge/manifest.json` is present, validation also:

- Ensures each `entries[].path` exists under the repository root and is a file.
- Re-indexes knowledge Markdown front matter and **errors** if the committed
  manifest is stale (run `rgd knowledge index` and commit).
- May emit **warnings** on stderr for large knowledge files or many triggers per
  entry (authoring hints only).

## Agent bootstrap (Cursor and other tools)

Repositories may add editor-specific rules that point agents at
`.reinguard/knowledge/manifest.json` and describe how to use `entries` and
`--query` (see ADR-0010). `rgd` does not require a particular bridge file; this
repo includes `.cursor/rules/knowledge-bridge.mdc` as an example.

## `rgd schema export`

Writes all embedded schemas from `pkg/schema/` to `--dir`.

## CI parity (`.github/workflows/ci.yaml`)

Triggers: `push` to `main`, `pull_request` to `main`, and `workflow_dispatch`.

The following commands mirror the **effective shell commands** run in CI (paths
and env are as in GitHub Actions). Fork pull requests **skip** job (3); see
[`CONTRIBUTING.md`](../.github/CONTRIBUTING.md).

### Job `go-ci` (all PRs and pushes)

```bash
go mod download
go mod verify
go build ./...
# golangci-lint via golangci/golangci-lint-action with: --timeout=5m ./...
go vet ./...
go test ./... -race -coverpkg=./... -coverprofile=coverage.out -count=1
bash tools/check-coverage-threshold.sh 80 coverage.out
go build -o /tmp/rgd ./cmd/rgd
/tmp/rgd version
/tmp/rgd config validate
/tmp/rgd schema export --dir /tmp/rgd-schema-smoke
```

### Job `rgd-dogfood` (after `go-ci`)

```bash
go build -o /tmp/rgd ./cmd/rgd
/tmp/rgd --cwd "${GITHUB_WORKSPACE}" config validate
/tmp/rgd --cwd "${GITHUB_WORKSPACE}" observe git > /tmp/observe-git.json
grep -q '"schema_version"' /tmp/observe-git.json
```

### Job `rgd-github-dogfood` (non-fork PRs and pushes to `main`)

Condition: `github.event_name != 'pull_request' || github.event.pull_request.head.repo.full_name == github.repository`.

```bash
go build -o /tmp/rgd ./cmd/rgd
gh --version
/tmp/rgd --cwd "${GITHUB_WORKSPACE}" observe github > /tmp/observe-github.json
grep -q '"schema_version"' /tmp/observe-github.json
```

(`GH_TOKEN` / `GITHUB_TOKEN` is set by Actions for `gh` and the GitHub provider.)
