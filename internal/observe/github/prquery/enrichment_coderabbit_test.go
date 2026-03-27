package prquery

import "testing"

func TestCoderabbitEnrichment_tryAgainMinutesSeconds(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("Rate limit exceeded. Please try again in 5 minutes and 30 seconds")
	if got == nil || got["rate_limit_remaining_seconds"].(int) != 330 {
		t.Fatalf("got %v", got)
	}
}

func TestCoderabbitEnrichment_oneMinute(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("Please try again in 1 minute")
	if got == nil || got["rate_limit_remaining_seconds"].(int) != 60 {
		t.Fatalf("got %v", got)
	}
}

func TestCoderabbitEnrichment_secondsOnly(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("try again in 45 seconds")
	if got == nil || got["rate_limit_remaining_seconds"].(int) != 45 {
		t.Fatalf("got %v", got)
	}
}

func TestCoderabbitEnrichment_noMatch(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	if got := e.Enrich("All good"); got != nil {
		t.Fatalf("got %v", got)
	}
}
