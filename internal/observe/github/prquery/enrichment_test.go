package prquery

import (
	"strings"
	"testing"
)

func TestValidateEnrichmentNames_unknownIncludesKnownList(t *testing.T) {
	t.Parallel()
	err := ValidateEnrichmentNames([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown enrich") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "coderabbit") {
		t.Fatalf("expected known enrichment list: %v", err)
	}
}
