---
id: implementation-defensive-config-validation
description: Defensive coding patterns for config-driven Go code — nil guards, typed option validation, blank/duplicate ID rejection
triggers:
  - defensive config
  - nil guard
  - silent ignore
  - blank id
  - duplicate id
  - typed options
  - config validation pattern
---

# Defensive config validation (Go)

Patterns for config-driven code that prevent silent failures. Referenced
by `.reinguard/policy/coding--preflight.md` § Defensive implementation
checks.

## Nil pointer guard

Every exported function that accepts a pointer or interface parameter
should check for `nil` before dereferencing:

```go
func NewEngineFromConfig(root *config.Root) (*Engine, error) {
    if root == nil {
        return nil, fmt.Errorf("observe: nil config root")
    }
    // ...
}
```

Place the guard at the function entry — before any field access.

## No silent ignore on typed options

When a configuration key is present, validate its type. Return an error
instead of silently falling through on type mismatch or empty value:

```go
if raw, exists := opts["api_base"]; exists {
    v, ok := raw.(string)
    if !ok {
        return nil, fmt.Errorf("provider: options.api_base must be a string")
    }
    s := strings.TrimSpace(v)
    if s == "" {
        return nil, fmt.Errorf("provider: options.api_base must be non-empty when set")
    }
    p.APIBase = s
}
```

The anti-pattern to avoid:

```go
// BAD: silently ignores wrong type or empty value
if v, ok := opts["api_base"].(string); ok {
    if s := strings.TrimSpace(v); s != "" {
        p.APIBase = s
    }
}
```

## Blank and duplicate ID rejection

When iterating enabled collection entries (e.g. provider specs), treat
blank or duplicate IDs as errors — not silent skips:

```go
for i, spec := range specs {
    if !spec.Enabled {
        continue
    }
    id := strings.TrimSpace(spec.ID)
    if id == "" {
        return nil, fmt.Errorf("providers[%d]: empty id", i)
    }
    if _, exists := out[id]; exists {
        return nil, fmt.Errorf("duplicate provider id %q", id)
    }
    // ... build provider
}
```

## Test setup error handling

Never discard errors from test helper or setup calls:

```go
// BAD
_ = r.Register("a", factory)

// GOOD
if err := r.Register("a", factory); err != nil {
    t.Fatal(err)
}
```

Discarding setup errors masks root causes and makes later assertion
failures non-diagnostic.

## Related

- `.reinguard/policy/coding--preflight.md` — verification obligations
- `.reinguard/knowledge/testing--strategy.md` — test perspectives
