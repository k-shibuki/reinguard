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

Every non-trivial test case should use the following comment format above
the test body or at the start of a subtest:

```text
// Given: Preconditions
// When:  Action under test
// Then:  Expected outcome
```

## Example

```go
func TestLoadConfig_validFixture(t *testing.T) {
	t.Helper()
	// Given: A directory with valid reinguard.yaml and empty rules/
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
