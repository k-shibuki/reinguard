package prquery

import (
	"strings"
	"testing"
)

func TestValidateEnrichmentNames(t *testing.T) {
	t.Parallel()
	// Given: a set of valid, empty, nil, unknown, and blank enrichment name inputs
	// When:  ValidateEnrichmentNames is called
	// Then:  valid/empty/nil return nil; unknown and blank return descriptive errors
	tests := []struct {
		name    string
		wantErr string
		input   []string
	}{
		{name: "valid_known", input: []string{"coderabbit"}, wantErr: ""},
		{name: "empty_slice", input: []string{}, wantErr: ""},
		{name: "nil_slice", input: nil, wantErr: ""},
		{name: "unknown_includes_known", input: []string{"unknown"}, wantErr: "unknown enrich"},
		{name: "empty_name", input: []string{""}, wantErr: "empty enrich"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateEnrichmentNames(tc.input)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestValidateEnrichmentNames_unknownIncludesKnownList(t *testing.T) {
	t.Parallel()
	err := ValidateEnrichmentNames([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "coderabbit") {
		t.Fatalf("expected known enrichment list: %v", err)
	}
}
