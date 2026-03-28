---
id: cli-json-contracts
description: Semantic JSON assertions and stable empty-list shape in tests and CLI output
triggers:
  - json contract
  - output shape
  - null vs array
  - unmarshal
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# JSON Contracts in Tests and CLI Output

## Rule 1: Assert JSON semantically

Prefer:

- decode JSON (`json.Unmarshal`)
- assert typed fields / keys

Avoid:

- substring-only checks like `"kind"` or `"paths"` without structure checks

Why:

- Substring checks pass on malformed or unrelated output.
- Semantic checks protect behavior, not formatting accidents.

## Rule 2: Keep empty list shape stable

When returning list fields in JSON, emit `[]` for empty lists instead of `null`.

Why:

- Downstream consumers can treat the field as list without null branching.
- Output contract remains consistent across boundary cases.

Implementation pattern:

- initialize slices as `[]any{}`
- append items conditionally

## Rule 3: Avoid whitespace-coupled assertions

Do not assert pretty-print fragments such as `"ok": true` with strict spacing.
Parse and assert logical value (`ok == true`) instead.

## Rule 4: Optional numeric fields must not use sentinel zero

When a numeric field is **optional** (present only if data/enrichment exists),
do not emit `0` as a default placeholder. Omit the key until the value is
actually derived.

Why:

- `0` becomes ambiguous (real value vs unavailable data).
- Downstream state/route logic can branch on key presence reliably.

Implementation/testing pattern:

- optional map fields: set only when source data exists
- tests should assert key absence for non-derived cases and exact value when present

## Scope: Rule 4 vs stable observation counts

Rule 4 applies to **optional** numerics whose presence means “value was
derived” (for example enrichment output). It does **not** require omitting
**documented aggregate counts** that are part of a stable observation
shape — for example `signals.github.reviews` fields such as
`review_decisions_total` documented in `docs/cli.md`, where `0` means
“no matching reviews,” not “field unavailable.”
