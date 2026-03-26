# change-inspect

Pre-PR self-inspection: evaluate the whole change against policy dimensions
**before** `pr-create`. This command judges quality; it does **not** create,
merge, or restructure commits (commit organization is `implement` step 7).

## Context

- `.reinguard/policy/review--self-inspection.md` — SSOT for inspection dimensions (open the file; do not duplicate its criteria here)
- `.reinguard/policy/coding--preflight.md` — prerequisite; meta-verify its obligations were met
- `.reinguard/policy/coding--standards.md` § **Change scope** — same-kind sweep across code, `.reinguard/`, `.cursor/`
- `AGENTS.md` (severity P0/P1, review guidelines)

**Already in context**: `reinguard-bridge.mdc` (HS-*), `workflow-policy.mdc` (command separation).

**Knowledge discovery** (substrate):

```bash
rgd knowledge pack --query '<keyword from Issue>'
```

Open each `entries[].path` under `.reinguard/knowledge/` as needed (testing strategy, defensive patterns, etc.).

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
- Relevant knowledge entries (`rgd knowledge pack`)

### 2. Inspect each dimension

Open [`.reinguard/policy/review--self-inspection.md`](../../.reinguard/policy/review--self-inspection.md) and evaluate the change against **every** dimension and subsection defined there (dimensions 1–5 and 7). Do **not** restate normative criteria in this command file — that policy is SSOT (ADR-0001: Adapter points at Semantics, no duplicate body text).

Dimension 6 (PR template substance) is deferred to `pr-create`, which fills and verifies the template.

### 3. Evaluate commit structure

Review the commit log for logical coherence. If commits are disorganized (mixed concerns, WIP commits, fixups not squashed), report as **Non-blocking** recommendation: "restructure commits before PR creation (return to `implement` step 7)."

### 4. Report findings

Classify each finding as **Blocking** or **Non-blocking** per `review--self-inspection.md` § Severity guidance.

### 5. Fix-and-re-inspect loop

If **Blocking** findings exist:

1. Return to `implement` for fixes (and commit restructuring if recommended)
2. Re-run applicable preflight steps (`go test`, `go vet`, `golangci-lint`, `markdownlint-cli2`)
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

HS-LOCAL-VERIFY, HS-NO-SKIP — this command reads, judges, and recommends; it does **not** merge or create PRs. Merge is `pr-merge` after `review-address`.
