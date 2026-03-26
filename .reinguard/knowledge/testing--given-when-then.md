---
id: testing-given-when-then
description: Given / When / Then comment format for Go tests
triggers:
  - Given When Then
  - test comment format
  - GWT
  - Go testing
---

# Given / When / Then Comment Format (Go)

Use the following comment format on **non-trivial** test cases — those
with multi-step setup or multiple assertions. Trivial single-assertion
tests (e.g. one-line error check) may omit GWT comments.

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

Use `t.Run` for multiple scenarios; each subtest may repeat Given/When/Then
for clarity.
