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
	writeProcFile(t, filepath.Join(root, "policy", "required.md"), "# Required\n")
	writeProcFile(t, filepath.Join(pdir, "b.md"), `---
id: procedure-b
purpose: B
applies_to:
  state_ids:
    - merge_ready
  route_ids:
    - user-merge
reads:
  - ../policy/required.md
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
	if len(entries[1].Reads) != 1 || entries[1].Reads[0] != "../policy/required.md" {
		t.Fatalf("reads %+v", entries[1].Reads)
	}
}

func TestLoadEntries_readsReferenceValidation(t *testing.T) {
	t.Parallel()
	// Given/When/Then: each fixture loads a procedure reads entry and expects success or a precise validation error.
	tests := []struct {
		name          string
		read          string
		setup         func(t *testing.T, root string)
		wantErrSubstr string
	}{
		{
			name: "existing_relative_file_succeeds",
			read: "../policy/required.md",
			setup: func(t *testing.T, root string) {
				writeProcFile(t, filepath.Join(root, "policy", "required.md"), "# Required\n")
				writeProcFile(t, filepath.Join(root, "policy", "other.md"), "# Other\n")
			},
		},
		{
			name:          "missing_file_fails",
			read:          "../policy/missing.md",
			wantErrSubstr: "path does not exist",
		},
		{
			name:          "absolute_path_fails",
			read:          "/tmp/required.md",
			wantErrSubstr: "must be relative",
		},
		{
			name:          "escaping_repo_fails",
			read:          "../../../outside.md",
			wantErrSubstr: "escapes repository",
		},
		{
			name: "directory_path_fails",
			read: "../policy",
			setup: func(t *testing.T, root string) {
				if err := os.MkdirAll(filepath.Join(root, "policy"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErrSubstr: "path is a directory",
		},
		{
			name: "symlink_path_fails",
			read: "../policy/link.md",
			setup: func(t *testing.T, root string) {
				writeProcFile(t, filepath.Join(root, "policy", "required.md"), "# Required\n")
				if err := os.Symlink("required.md", filepath.Join(root, "policy", "link.md")); err != nil {
					t.Fatal(err)
				}
			},
			wantErrSubstr: "path is a symlink",
		},
		{
			name: "intermediate_symlink_escape_fails",
			read: "../policy/link/secret.md",
			setup: func(t *testing.T, root string) {
				outside := filepath.Join(t.TempDir(), "outside")
				writeProcFile(t, filepath.Join(outside, "secret.md"), "# Secret\n")
				if err := os.MkdirAll(filepath.Join(root, "policy"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(root, "policy", "link")); err != nil {
					t.Fatal(err)
				}
			},
			wantErrSubstr: "resolves outside repository",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeProcFile(t, filepath.Join(root, "policy", "other.md"), "# Other\n")
			if tt.setup != nil {
				tt.setup(t, root)
			}
			pdir := filepath.Join(root, "procedure")
			writeProcFile(t, filepath.Join(pdir, "p.md"), `---
id: procedure-p
purpose: P
applies_to:
  state_ids: []
  route_ids: []
reads:
  - `+tt.read+`
  - ../policy/other.md
---
`)
			entries, present, err := LoadEntries(root, pdir)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("LoadEntries() err=%v, want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !present || len(entries) != 1 || len(entries[0].Reads) != 2 {
				t.Fatalf("present=%v entries=%+v", present, entries)
			}
		})
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
