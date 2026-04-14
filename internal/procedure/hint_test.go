package procedure

import "testing"

func TestHintEntry_routeFilter(t *testing.T) {
	t.Parallel()
	entries := []Entry{
		{
			ID:       "p1",
			Path:     "procedure/a.md",
			StateIDs: []string{"Idle"},
			RouteIDs: []string{"next"},
		},
	}
	got := HintEntry(entries, "Idle", true, "next")
	if got == nil || got.ID != "p1" {
		t.Fatalf("got %+v", got)
	}
	if HintEntry(entries, "Idle", true, "wrong") != nil {
		t.Fatal("expected no match when route filter fails")
	}
	if HintEntry(entries, "Idle", false, "") != nil {
		t.Fatal("expected no match when route not resolved")
	}
}

func TestHintEntry_emptyRouteIDs(t *testing.T) {
	t.Parallel()
	entries := []Entry{
		{
			ID:       "p2",
			Path:     "procedure/b.md",
			StateIDs: []string{"ready_for_pr"},
			RouteIDs: nil,
		},
	}
	got := HintEntry(entries, "ready_for_pr", false, "")
	if got == nil || got.ID != "p2" {
		t.Fatalf("got %+v", got)
	}
}

func TestHintEntry_unknownState(t *testing.T) {
	t.Parallel()
	if HintEntry([]Entry{{ID: "x", StateIDs: []string{"A"}}}, "B", true, "r") != nil {
		t.Fatal("expected nil")
	}
}
