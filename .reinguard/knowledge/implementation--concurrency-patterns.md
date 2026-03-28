---
id: implementation-concurrency-patterns
description: "RWMutex re-entrancy trap and lock-splitting patterns in Go"
triggers:
  - RWMutex
  - re-entrancy
  - deadlock
  - sync mutex
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Concurrency patterns (Go)

## RWMutex re-entrancy trap

`sync.RWMutex` is **not re-entrant**. Avoid calling a helper that acquires the
same lock from a section that already holds `RLock`/`Lock` on that mutex.

Preferred pattern:

- keep a `...Locked` helper that assumes the caller already holds the lock
- call that helper from both exported lock-taking functions and validation paths

Example shape:

```go
func Names() []string {
    mu.RLock()
    defer mu.RUnlock()
    return namesLocked()
}

func Validate(input []string) error {
    mu.RLock()
    defer mu.RUnlock()
    known := namesLocked()
    // ... validate against known
    return nil
}
```

## Related

- `.reinguard/knowledge/implementation--defensive-config-validation.md` — nil guards, typed options
