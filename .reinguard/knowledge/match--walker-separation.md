---
id: match-walker-separation
description: "Keep match-time evaluation and validate-time walkers aligned on optional YAML/JSON keys"
triggers:
  - match walker
  - validate walker
  - eval key
  - op key conflict
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Match-time vs validate-time walkers

When decoded YAML/JSON maps carry optional keys (for example `eval` beside
`op` / `and` / `or` / `not`), keep **runtime evaluation** and **config
validation** walkers aligned:

- If the key is **present**, require the expected type and non-empty values
  where applicable — do not fall through to another branch on type mismatch.
- When forbidding combinations (e.g. `eval` with `op`), treat **`op` present**
  as a conflict even if `op` is not a string, so malformed configs error
  instead of executing as a scalar op.

## Related

- `.reinguard/knowledge/implementation--defensive-config-validation.md` — typed option validation
