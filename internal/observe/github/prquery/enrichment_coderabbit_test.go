package prquery

import "testing"

func TestCoderabbitEnrichment_tryAgainMinutesSeconds(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("Rate limit exceeded. Please try again in 5 minutes and 30 seconds")
	assertSeconds(t, got, 330)
}

func TestCoderabbitEnrichment_oneMinute(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("Please try again in 1 minute")
	assertSeconds(t, got, 60)
}

func TestCoderabbitEnrichment_secondsOnly(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("try again in 45 seconds")
	assertSeconds(t, got, 45)
}

func TestCoderabbitEnrichment_caseInsensitive(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("TRY AGAIN IN 5 MINUTES")
	assertSeconds(t, got, 300)
}

func TestCoderabbitEnrichment_zeroSecondsNoSignal(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	if got := e.Enrich("try again in 0 seconds"); got != nil {
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

func assertSeconds(t *testing.T, got map[string]any, want int) {
	t.Helper()
	if got == nil {
		t.Fatal("got nil")
	}
	sec, ok := got["rate_limit_remaining_seconds"].(int)
	if !ok || sec != want {
		t.Fatalf("got %v", got)
	}
}
