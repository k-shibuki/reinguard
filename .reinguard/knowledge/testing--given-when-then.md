---
id: testing-given-when-then
description: Given / When / Then comment format for Go tests
triggers:
  - Given When Then
  - test comment format
  - GWT
  - Go testing
when:
  or:
    - op: exists
      path: git.branch
    - op: exists
      path: github.repository.owner
---

# Given / When / Then Comment Format (Go)

**Non-trivial** test cases — those with multi-step setup or multiple
assertions — **must** include the following comment format. Trivial
single-assertion tests (e.g. one-line error check) may omit GWT.

```text
// Given: Preconditions
// When:  Action under test
// Then:  Expected outcome
```

## Example

```go
func TestLoadConfig_validFixture(t *testing.T) {
	t.Helper()
	// Given: A directory with valid reinguard.yaml and no control/ rules
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), minimalValidYAML)

	// When: Load is called
	cfg, err := config.Load(dir)

	// Then: No error and schema version is set
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SchemaVersion == "" {
		t.Fatal("expected non-empty schema_version")
	}
}
```

## Table-driven tests

When using a `[]struct{ ... }` table with `t.Run(tc.name, ...)`:

- Put **one summary GWT block at the function start** (or a single-line
  `// Given/When/Then:`) describing what the table exercises as a whole.
- **Do not** repeat `// Given:` / `// When:` / `// Then:` inside the loop body —
  the case `name` and struct fields document each row.
- For **non-table** tests with multiple steps, keep full GWT comments as in the
  example above.

Standalone `t.Run` subtests **without** a table (each subtest is a distinct
scenario) may still use per-subtest GWT when the setup is non-trivial.

## When creating or editing tests

When creating or substantially editing non-trivial tests:

- **must** add a concise **function-level** Given/When/Then summary
- for **table-driven** tests, do not repeat GWT inside each table row or loop
  iteration (see Table-driven tests above)
- keep test intent readable in code review
