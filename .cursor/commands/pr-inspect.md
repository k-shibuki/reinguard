# pr-inspect

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

**Diff and metadata:**

```bash
gh pr diff <N>
gh pr view <N> --json files,title,body,labels
gh issue view <ISSUE> --json title,body,labels
```

## Act

### 1. Gather inputs

- Full diff (`gh pr diff`)
- PR metadata (title, body, labels)
- Issue metadata (Definition of Done, Test plan, Refs: ADR)
- Relevant knowledge entries (`rgd knowledge pack`)

### 2. Inspect each dimension

Open [`.reinguard/policy/review--self-inspection.md`](../../.reinguard/policy/review--self-inspection.md) and evaluate the PR against **every** dimension and subsection defined there. Do **not** restate normative criteria in this command file — that policy is SSOT (ADR-0001: Adapter points at Semantics, no duplicate body text).

### 3. Report findings

Classify each finding as **Blocking** or **Non-blocking** per `review--self-inspection.md` § Severity guidance.

### 4. Fix-and-re-inspect loop

If **Blocking** findings exist:

1. Apply fixes
2. Re-run applicable preflight steps (`go test`, `go vet`, `golangci-lint`, `markdownlint-cli2`)
3. Commit with `Refs: #<issue>`
4. Push
5. Re-run inspection (go to step 2) until no Blocking findings remain

If only **Non-blocking** findings exist, fix where practical, note remaining items in PR body, and proceed.

### 5. Declare readiness

When inspection is clean (no Blocking findings): declare **ready for external review**. External review (CodeRabbit, Codex, human) follows per `workflow-policy.mdc`.

## Output

- Dimension-by-dimension Pass/Fail with findings list and severity
- Fix commits (if any): SHA + what changed
- Final status: **Ready for external review** or **Blocking findings remain** (with list)

## Guard

HS-LOCAL-VERIFY, HS-NO-SKIP — this command reads, judges, and fixes; it does **not** merge. Merge is `pr-merge` after `review-address`.
