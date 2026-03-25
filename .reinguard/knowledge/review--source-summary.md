---
id: review-source-summary
description: Evidence snapshot of review corpus statistics across all PRs
triggers:
  - review corpus
  - global findings
  - all PR comments
---

# Review Corpus Summary (All PRs)

Snapshot source:
- Repository: `k-shibuki/reinguard`
- Scope: all pull requests (open + closed)
- Extraction date: 2026-03-25
- PR count: 27 (`#2` to `#50`)
- Review objects:
  - pull request reviews: 272
  - inline review comments: 371
  - root review comments: 132

Reviewer distribution on root comments:
- `coderabbitai[bot]`: 111
- `chatgpt-codex-connector[bot]`: 15
- `cursor[bot]`: 6

Most-commented paths by CodeRabbit root comments:
1. `internal/rgdcli/rgdcli.go` (7)
2. `.cursor/rules/workflow-policy.mdc` (5)
3. `internal/observe/provider_git.go` (4)
4. `internal/rgdcli/flatten_test.go` (4)
5. `tools/backfill-open-pr-policy.py` (4)

Recurring finding clusters (keyword-derived):
- JSON contract / shape assertions: high frequency
- Test rigor (Given/When/Then, boundary cases, explicit outcomes): high frequency
- Observation `signals` validation: medium frequency
- CI token permissions and aggregate gate semantics: medium frequency
- `state.*` dotted-key resolution from nested maps: medium frequency

This file is evidence-oriented. Actionable guidance lives in:
- `.reinguard/knowledge/review--index.md`
- `.reinguard/knowledge/review--trigger-index.json`
