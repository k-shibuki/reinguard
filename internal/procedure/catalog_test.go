package procedure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProcFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEntries_missingDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entries, present, err := LoadEntries(root, filepath.Join(root, "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if present || entries != nil {
		t.Fatalf("present=%v entries=%v", present, entries)
	}
}

func TestLoadEntries_okAndSorted(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	pdir := filepath.Join(root, "procedure")
	writeProcFile(t, filepath.Join(pdir, "b.md"), `---
id: procedure-b
purpose: B
applies_to:
  state_ids:
    - merge_ready
  route_ids:
    - user-merge
---
`)
	writeProcFile(t, filepath.Join(pdir, "a.md"), `---
id: procedure-a
purpose: A
applies_to:
  state_ids:
    - working_no_pr
  route_ids: []
---
`)
	entries, present, err := LoadEntries(root, pdir)
	if err != nil {
		t.Fatal(err)
	}
	if !present || len(entries) != 2 {
		t.Fatalf("present=%v %+v", present, entries)
	}
	if entries[0].ID != "procedure-a" || entries[1].ID != "procedure-b" {
		t.Fatalf("%+v", entries)
	}
	if entries[0].Path != "procedure/a.md" {
		t.Fatalf("path %q", entries[0].Path)
	}
}

func TestLoadEntries_duplicateProcedureID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	pdir := filepath.Join(root, "procedure")
	body := `---
id: same
purpose: x
applies_to:
  state_ids: []
  route_ids: []
---
`
	writeProcFile(t, filepath.Join(pdir, "a.md"), body)
	writeProcFile(t, filepath.Join(pdir, "b.md"), body)
	_, _, err := LoadEntries(root, pdir)
	if err == nil || !strings.Contains(err.Error(), "duplicate procedure id") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStateMapping_duplicateAcrossProcedures(t *testing.T) {
	t.Parallel()
	declared := map[string]struct{}{"s1": {}}
	err := ValidateStateMapping([]Entry{
		{ID: "p1", StateIDs: []string{"s1"}},
		{ID: "p2", StateIDs: []string{"s1"}},
	}, declared)
	if err == nil || !strings.Contains(err.Error(), "state_id \"s1\"") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStateMapping_orphanState(t *testing.T) {
	t.Parallel()
	err := ValidateStateMapping([]Entry{
		{ID: "p1", StateIDs: []string{"unknown_state"}},
	}, map[string]struct{}{"working_no_pr": {}})
	if err == nil || !strings.Contains(err.Error(), "no matching control state rule") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStateMapping_ok(t *testing.T) {
	t.Parallel()
	declared := map[string]struct{}{
		"working_no_pr": {},
		"merge_ready":   {},
	}
	err := ValidateStateMapping([]Entry{
		{ID: "p1", StateIDs: []string{"working_no_pr"}},
		{ID: "p2", StateIDs: []string{"merge_ready"}},
	}, declared)
	if err != nil {
		t.Fatal(err)
	}
}
