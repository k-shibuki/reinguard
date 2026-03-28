---
id: testing-setup-error-handling
description: Fail-fast test setup — never discard errors from helpers or registration calls
triggers:
  - test setup error
  - t.Fatal setup
  - discard error test
  - register error test
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Test setup error handling (Go)

Never discard errors from test helper or setup calls:

```go
// BAD — masks root cause
_ = r.Register("a", factory)

// GOOD — fails fast with diagnostic message
if err := r.Register("a", factory); err != nil {
    t.Fatal(err)
}
```

Discarding setup errors masks root causes and makes later assertion failures non-diagnostic.

## Related

- `.reinguard/policy/coding--preflight.md` § Test design confirmation
- [`testing--strategy.md`](testing--strategy.md) — perspectives and table-driven tests
