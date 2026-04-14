package procedure

import "testing"

func TestHintEntry(t *testing.T) {
	t.Parallel()
	stateIDs := []string{"Idle", "ready_for_pr", "B"}
	routeIDs := []string{"", "next", "wrong", "r"}
	tests := map[string]struct {
		entries       []Entry
		stateIDIndex  int
		routeIDIndex  int
		wantIndex     int
		routeResolved bool
	}{
		"route filter match": {
			entries: []Entry{{
				ID:       "p1",
				Path:     "procedure/a.md",
				StateIDs: []string{"Idle"},
				RouteIDs: []string{"next"},
			}},
			stateIDIndex:  0,
			routeResolved: true,
			routeIDIndex:  1,
			wantIndex:     0,
		},
		"route filter mismatch": {
			entries: []Entry{{
				ID:       "p1",
				Path:     "procedure/a.md",
				StateIDs: []string{"Idle"},
				RouteIDs: []string{"next"},
			}},
			stateIDIndex:  0,
			routeResolved: true,
			routeIDIndex:  2,
			wantIndex:     -1,
		},
		"route unresolved blocks filtered match": {
			entries: []Entry{{
				ID:       "p1",
				Path:     "procedure/a.md",
				StateIDs: []string{"Idle"},
				RouteIDs: []string{"next"},
			}},
			stateIDIndex:  0,
			routeResolved: false,
			routeIDIndex:  0,
			wantIndex:     -1,
		},
		"empty route scope still matches state": {
			entries: []Entry{{
				ID:       "p2",
				Path:     "procedure/b.md",
				StateIDs: []string{"ready_for_pr"},
				RouteIDs: nil,
			}},
			stateIDIndex:  1,
			routeResolved: false,
			routeIDIndex:  0,
			wantIndex:     0,
		},
		"unknown state": {
			entries:       []Entry{{ID: "x", StateIDs: []string{"A"}}},
			stateIDIndex:  2,
			routeResolved: true,
			routeIDIndex:  3,
			wantIndex:     -1,
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := HintEntry(tc.entries, stateIDs[tc.stateIDIndex], tc.routeResolved, routeIDs[tc.routeIDIndex])
			if tc.wantIndex < 0 {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			wantID := tc.entries[tc.wantIndex].ID
			if got == nil || got.ID != wantID {
				t.Fatalf("got %+v, want id %q", got, wantID)
			}
		})
	}
}
