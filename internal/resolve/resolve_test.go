package resolve

import (
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestResolveState_priorityWins(t *testing.T) {
	t.Parallel()
	// Given: two matching state rules with different priorities
	// When: ResolveState runs
	// Then: lower priority number wins
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 20, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "state", ID: "b", Priority: 10, StateID: "B", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	res, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeResolved || res.StateID != "B" {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_tieAmbiguous(t *testing.T) {
	t.Parallel()
	// Given: two matches at same best priority
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "state", ID: "b", Priority: 10, StateID: "B", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	res, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeAmbiguous {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Reason, "same best priority") {
		t.Fatal(res.Reason)
	}
}

func TestResolveState_noMatchDegraded(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 2}},
	}
	res, err := ResolveState(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching state rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_suppressed(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}, DependsOn: []string{"github"}},
	}
	signals := map[string]any{"x": 1}
	deg := map[string]struct{}{"github": {}}
	res, err := ResolveState(rules, signals, deg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "depends_on") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_missingStateID(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "state", ID: "bad", Priority: 10, StateID: "", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	_, err := ResolveState(rules, map[string]any{"x": 1}, nil)
	if err == nil || !strings.Contains(err.Error(), `resolve: state rule "bad" missing state_id`) {
		t.Fatalf("%v", err)
	}
}

func TestResolveState_ruleEvalError(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "state", ID: "r", Priority: 10, StateID: "S", When: map[string]any{"op": "bogus"}},
	}
	_, err := ResolveState(rules, map[string]any{}, nil)
	if err == nil || !strings.Contains(err.Error(), "rule r:") {
		t.Fatalf("%v", err)
	}
}

func TestResolveRoute_resolved(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "route", ID: "r1", Priority: 10, RouteID: "next", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeResolved || res.RouteID != "next" {
		t.Fatalf("%+v", res)
	}
	if len(res.RouteCandidates) != 1 || res.RouteCandidates[0].RouteID != "next" {
		t.Fatalf("route_candidates: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_orderedCandidates(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "route", ID: "low", Priority: 5, RouteID: "R5", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "high", Priority: 20, RouteID: "R20", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.RouteID != "R5" {
		t.Fatalf("want winner R5 got %+v", res)
	}
	if len(res.RouteCandidates) != 2 || res.RouteCandidates[0].RouteID != "R5" || res.RouteCandidates[1].RouteID != "R20" {
		t.Fatalf("route_candidates order: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_noMatchDegraded(t *testing.T) {
	t.Parallel()
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "r", Priority: 1, RouteID: "x", When: map[string]any{"op": "eq", "path": "a", "value": 1}},
	}, map[string]any{"a": 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching route rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_allSuppressed(t *testing.T) {
	t.Parallel()
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "r", Priority: 1, RouteID: "x", When: map[string]any{"op": "eq", "path": "a", "value": 1}, DependsOn: []string{"git"}},
	}, map[string]any{"a": 1}, map[string]struct{}{"git": {}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "suppressed") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_missingRouteID(t *testing.T) {
	t.Parallel()
	_, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "bad", Priority: 1, RouteID: "", When: map[string]any{"op": "eq", "path": "a", "value": 1}},
	}, map[string]any{"a": 1}, nil)
	if err == nil || !strings.Contains(err.Error(), `resolve: route rule "bad" missing route_id`) {
		t.Fatalf("%v", err)
	}
}

func TestResolveRoute_tieAmbiguous(t *testing.T) {
	t.Parallel()
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "a", Priority: 5, RouteID: "r1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "b", Priority: 5, RouteID: "r2", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeAmbiguous {
		t.Fatalf("%+v", res)
	}
}

func TestDuplicatePriorityWarnings(t *testing.T) {
	t.Parallel()
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 1, When: map[string]any{}},
		{Type: "state", ID: "b", Priority: 1, When: map[string]any{}},
	}
	w := DuplicatePriorityWarnings(rules)
	if len(w) != 1 || !strings.Contains(w[0], "duplicated") {
		t.Fatalf("%v", w)
	}
}

func TestDuplicatePriorityWarnings_singleRuleSilent(t *testing.T) {
	t.Parallel()
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "only", Priority: 1, When: map[string]any{}},
	})
	if len(w) != 0 {
		t.Fatal(w)
	}
}

func TestDuplicatePriorityWarnings_separateTypes(t *testing.T) {
	t.Parallel()
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "s1", Priority: 1, When: map[string]any{}},
		{Type: "state", ID: "s2", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r1", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r2", Priority: 1, When: map[string]any{}},
	})
	if len(w) != 2 {
		t.Fatalf("%v", w)
	}
}
