---
id: cli-json-contracts
description: Semantic JSON assertions and stable empty-list shape in tests and CLI output
triggers:
  - json contract
  - output shape
  - null vs array
  - unmarshal
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
