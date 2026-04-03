---
id: procedure-change-inspect
purpose: Pre-PR self-inspection against policy dimensions before PR creation.
applies_to:
  state_ids: []
  route_ids: []
reads:
  - ../policy/review--self-inspection.md
  - ../policy/review--disposition-categories.md
  - ../policy/review--consensus-protocol.md
  - ../policy/coding--preflight.md
  - ../policy/coding--standards.md
sense:
  - rgd context build
  - git diff
act:
  - Gather diff and commits; run local CodeRabbit review; inspect every dimension; disposition findings; loop until PR-ready.
output:
  - Pass/fail per dimension, disposition ledger, commit-structure note, readiness declaration.
done_when: Review closure is complete for the current local review cycle; ready for `pr-create` or documented otherwise.
escalate_when: Policy interpretation conflicts, inspection scope unclear, or local review tooling cannot complete.
---

# change-inspect

Pre-PR self-inspection: evaluate the whole change against policy dimensions
**before** `pr-create`. This command judges quality; it does **not** create,
merge, or restructure commits (commit organization is `implement` step 7).

## Context

- [`../policy/review--self-inspection.md`](../policy/review--self-inspection.md) — SSOT for inspection dimensions (open the file; do not duplicate its criteria here)
- [`../policy/review--disposition-categories.md`](../policy/review--disposition-categories.md) — shared disposition vocabulary across local and PR review
- [`../policy/review--consensus-protocol.md`](../policy/review--consensus-protocol.md) — shared review-closure model; PR-only thread mechanics remain downstream
- [`../policy/coding--preflight.md`](../policy/coding--preflight.md) — prerequisite; meta-verify its obligations were met
- [`../policy/coding--standards.md`](../policy/coding--standards.md) § **Change scope** — same-kind sweep across code, `.reinguard/`, `.cursor/`
- [`../../AGENTS.md`](../../AGENTS.md) — review guidelines

**Already in context** (always-active Adapter rule): HS-* codes, catalogs, workflow & commit policy.

**Knowledge discovery** (substrate):

```bash
rgd context build
```

Use `knowledge.entries` from stdout JSON (signal-filtered). Open each `entries[].path` as needed (testing strategy, defensive patterns, etc.).

Optional: `rgd observe > /tmp/rgd-observe.json` then `rgd knowledge pack --observation-file /tmp/rgd-observe.json --query '<keyword from Issue>'` for trigger substring OR-union (`docs/cli.md`).

**Diff and metadata** (no PR required):

```bash
git diff origin/main...HEAD          # committed changes
git diff                             # unstaged changes
git diff --cached                    # staged changes
git log --oneline origin/main..HEAD  # commit structure
gh issue view <ISSUE> --json title,body,labels
```

## Act

### 1. Gather inputs

- Full diff: committed (`origin/main...HEAD`) + staged + unstaged
- Commit log (`git log --oneline origin/main..HEAD`)
- `implement` output: scope, DoD progress, doc impact, preflight result, commit status
- Issue metadata (Definition of Done, Test plan, Refs: ADR)
- Relevant knowledge entries (`rgd context build` → `knowledge.entries`)

### 2. Inspect each dimension

Open [`../policy/review--self-inspection.md`](../policy/review--self-inspection.md) and evaluate the change against **every** dimension and subsection defined there (dimensions 1–6 and 8). Do **not** restate normative criteria in this procedure — that policy is SSOT (ADR-0001: Adapter points at Semantics, no duplicate body text).

Dimension 7 (PR template substance) is deferred to `pr-create`, which fills and verifies the template.

### 3. Run required local CodeRabbit review

Run the repository-local CodeRabbit gate from the repo root:

```bash
bash .reinguard/scripts/check-local-review.sh --base main --retry-on-rate-limit
```

If the CLI reports a rate limit, the script uses the **latest** rate-limit
line in that run to parse the cooldown, adds a **safety buffer**, then
retries the review **once** (`--retry-on-rate-limit`). Treat
installation/authentication/execution failures, unparsed cooldown, or a
second consecutive rate limit, as a failed gate and do not proceed.
If the branch uses a runtime verification gate such as `local-verification`,
record or refresh it on the reviewed head after the required local checks
pass, for example:

```bash
rgd gate record local-verification --status pass
```

Review the output and disposition findings with the same four categories
defined in
[`../policy/review--disposition-categories.md`](../policy/review--disposition-categories.md).
Do not auto-dismiss them because they were surfaced locally instead of on
the PR.

### 4. Evaluate commit structure

Review the commit log for logical coherence. If commits are disorganized
(mixed concerns, WIP commits, fixups not squashed), do not proceed until
the history is restructured or an explicit `Acknowledged` contract is
recorded for that deferment, which should be rare before PR creation.

### 5. Report findings

Disposition each finding exactly once as **Fixed**, **By design**,
**False positive**, or **Acknowledged** per
[`../policy/review--disposition-categories.md`](../policy/review--disposition-categories.md)
and `review--self-inspection.md` § **Disposition guidance**. Record a
disposition ledger for the current local review cycle: finding source,
summary, disposition, evidence or rationale, and follow-up contract when
`Acknowledged` applies.

### 6. Fix-and-re-inspect loop

If review closure is not yet complete for the current local review cycle:

1. Return to `implement` for fixes (and commit restructuring if recommended)
2. Treat one local CR output as a **single batch**: classify every finding
   from that pass, fix every finding you will disposition **Fixed** on the
   current branch, and apply same-kind sweep for any fix pattern that
   extends beyond the exact commented line or file.
3. Re-run applicable preflight steps (`go test`, `go vet`, `golangci-lint`, `pre-commit run markdownlint-cli2 --all-files`)
4. Commit the stabilized batch with `Refs: #<issue>`
5. Re-run `bash .reinguard/scripts/check-local-review.sh --base main --retry-on-rate-limit` on the stabilized head
6. Re-record the runtime gate for the stabilized head when this branch uses one (for example `rgd gate record local-verification --status pass`)
7. Re-run inspection (go to step 2) until every finding in the current local review cycle is classified and closed per the shared policy

If a finding is dispositioned **Acknowledged**, record the follow-up Issue
or equally explicit deferred-work contract in the inspection output so the
PR handoff is auditable.

### 7. Declare readiness

When inspection shows review closure complete for the current local review
cycle, including local CodeRabbit findings on the latest head: declare
**ready for PR creation**. Proceed to `pr-create`.

## Output

- Dimension-by-dimension Pass/Fail with disposition ledger
- Local CodeRabbit review status: completed / gate failed, plus finding summary
- Commit structure assessment: clean / restructured / deferred with explicit `Acknowledged` contract
- Fix commits (if any): SHA + what changed
- Final status: **Ready for PR creation** or **Findings remain** (with list)

## Guard

HS-LOCAL-VERIFY, HS-NO-SKIP — this procedure reads, judges, and recommends; it does **not** merge or create PRs. Merge is `pr-merge` after `review-address`.
