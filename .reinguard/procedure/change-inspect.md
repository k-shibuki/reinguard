---
id: procedure-change-inspect
purpose: Pre-PR self-inspection against policy dimensions before PR creation.
applies_to:
  state_ids: []
  route_ids: []
reads:
  - ../policy/review--self-inspection.md
  - ../policy/coding--preflight.md
  - ../policy/coding--standards.md
sense:
  - rgd context build
  - git diff
act:
  - Gather diff and commits; inspect every dimension; classify findings; loop on blocking items.
output:
  - Pass/fail per dimension, commit-structure note, readiness declaration.
done_when: No blocking findings; ready for `pr-create` or documented otherwise.
escalate_when: Policy interpretation conflicts or inspection scope unclear.
---

# change-inspect

Pre-PR self-inspection: evaluate the whole change against policy dimensions
**before** `pr-create`. This command judges quality; it does **not** create,
merge, or restructure commits (commit organization is `implement` step 7).

## Context

- [`../policy/review--self-inspection.md`](../policy/review--self-inspection.md) — SSOT for inspection dimensions (open the file; do not duplicate its criteria here)
- [`../policy/coding--preflight.md`](../policy/coding--preflight.md) — prerequisite; meta-verify its obligations were met
- [`../policy/coding--standards.md`](../policy/coding--standards.md) § **Change scope** — same-kind sweep across code, `.reinguard/`, `.cursor/`
- `AGENTS.md` (severity P0/P1, review guidelines)

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

Open [`../policy/review--self-inspection.md`](../policy/review--self-inspection.md) and evaluate the change against **every** dimension and subsection defined there (dimensions 1–5 and 7). Do **not** restate normative criteria in this procedure — that policy is SSOT (ADR-0001: Adapter points at Semantics, no duplicate body text).

Dimension 6 (PR template substance) is deferred to `pr-create`, which fills and verifies the template.

### 3. Evaluate commit structure

Review the commit log for logical coherence. If commits are disorganized (mixed concerns, WIP commits, fixups not squashed), report as **Non-blocking** recommendation: "restructure commits before PR creation (return to `implement` step 7)."

### 4. Report findings

Classify each finding as **Blocking** or **Non-blocking** per `review--self-inspection.md` § Severity guidance.

### 5. Fix-and-re-inspect loop

If **Blocking** findings exist:

1. Return to `implement` for fixes (and commit restructuring if recommended)
2. Re-run applicable preflight steps (`go test`, `go vet`, `golangci-lint`, `npx --yes markdownlint-cli2@latest '**/*.md'`)
3. Commit with `Refs: #<issue>`
4. Re-run inspection (go to step 2) until no Blocking findings remain

If only **Non-blocking** findings exist, fix where practical, note remaining items for the PR body, and proceed to `pr-create`.

### 6. Declare readiness

When inspection is clean (no Blocking findings): declare **ready for PR creation**. Proceed to `pr-create`.

## Output

- Dimension-by-dimension Pass/Fail with findings list and severity
- Commit structure assessment: clean / restructuring recommended
- Fix commits (if any): SHA + what changed
- Final status: **Ready for PR creation** or **Blocking findings remain** (with list)

## Guard

HS-LOCAL-VERIFY, HS-NO-SKIP — this procedure reads, judges, and recommends; it does **not** merge or create PRs. Merge is `pr-merge` after `review-address`.
