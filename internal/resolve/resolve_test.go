package resolve

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestResolve_unsupportedRuleType(t *testing.T) {
	t.Parallel()
	// Given: an unsupported rule type "guard"
	// When: Resolve is called
	// Then: error mentions unsupported rule type
	_, err := Resolve(nil, nil, nil, "guard")
	if err == nil || !strings.Contains(err.Error(), `unsupported rule type "guard"`) {
		t.Fatalf("got %v", err)
	}
}

func TestResolve_stateMatchesResolveState(t *testing.T) {
	t.Parallel()
	// Given: two matching state rules with different priorities
	// When: ResolveState and Resolve(..., "state") run
	// Then: both return the same Result
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 20, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "state", ID: "b", Priority: 10, StateID: "B", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	a, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(rules, signals, nil, "state")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Resolve vs ResolveState: %+v vs %+v", a, b)
	}
}

func TestResolve_routeMatchesResolveRoute(t *testing.T) {
	t.Parallel()
	// Given: two matching route rules with different priorities
	// When: ResolveRoute and Resolve(..., "route") run
	// Then: both return the same Result
	rules := []config.Rule{
		{Type: "route", ID: "low", Priority: 5, RouteID: "R5", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "high", Priority: 20, RouteID: "R20", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	a, err := ResolveRoute(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(rules, signals, nil, "route")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Resolve vs ResolveRoute: %+v vs %+v", a, b)
	}
}

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
	// When: ResolveState runs
	res, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: ambiguous outcome with reason
	if res.Kind != OutcomeAmbiguous {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Reason, "same best priority") {
		t.Fatal(res.Reason)
	}
}

func TestResolveState_noMatchDegraded(t *testing.T) {
	t.Parallel()
	// Given: state rules that do not match signals
	// When: ResolveState runs
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 2}},
	}
	res, err := ResolveState(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded with no-match reason
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching state rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_suppressed(t *testing.T) {
	t.Parallel()
	// Given: matching rule with depends_on a degraded provider
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}, DependsOn: []string{"github"}},
	}
	signals := map[string]any{"x": 1}
	deg := map[string]struct{}{"github": {}}
	// When: ResolveState runs
	res, err := ResolveState(rules, signals, deg)
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded due to dependency
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "depends_on") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_missingStateID(t *testing.T) {
	t.Parallel()
	// Given: state rule with empty state_id
	rules := []config.Rule{
		{Type: "state", ID: "bad", Priority: 10, StateID: "", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveState runs
	_, err := ResolveState(rules, map[string]any{"x": 1}, nil)
	// Then: configuration error
	if err == nil || !strings.Contains(err.Error(), `resolve: state rule "bad" missing state_id`) {
		t.Fatalf("%v", err)
	}
}

func TestResolveState_ruleEvalError(t *testing.T) {
	t.Parallel()
	// Given: state rule with invalid when (eval error)
	rules := []config.Rule{
		{Type: "state", ID: "r", Priority: 10, StateID: "S", When: map[string]any{"op": "bogus"}},
	}
	// When: ResolveState runs
	_, err := ResolveState(rules, map[string]any{}, nil)
	// Then: error names rule
	if err == nil || !strings.Contains(err.Error(), "rule r:") {
		t.Fatalf("%v", err)
	}
}

func TestResolveRoute_resolved(t *testing.T) {
	t.Parallel()
	// Given: one matching route rule
	rules := []config.Rule{
		{Type: "route", ID: "r1", Priority: 10, RouteID: "next", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveRoute runs
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: resolved with single candidate
	if res.Kind != OutcomeResolved || res.RouteID != "next" {
		t.Fatalf("%+v", res)
	}
	if len(res.RouteCandidates) != 1 || res.RouteCandidates[0].RouteID != "next" {
		t.Fatalf("route_candidates: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_orderedCandidates(t *testing.T) {
	t.Parallel()
	// Given: two matching routes at different priorities
	rules := []config.Rule{
		{Type: "route", ID: "low", Priority: 5, RouteID: "R5", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "high", Priority: 20, RouteID: "R20", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveRoute runs
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: lower priority wins; candidates ordered by priority
	if res.RouteID != "R5" {
		t.Fatalf("want winner R5 got %+v", res)
	}
	if len(res.RouteCandidates) != 2 || res.RouteCandidates[0].RouteID != "R5" || res.RouteCandidates[1].RouteID != "R20" {
		t.Fatalf("route_candidates order: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_noMatchDegraded(t *testing.T) {
	t.Parallel()
	// Given: route rules that do not match
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "r", Priority: 1, RouteID: "x", When: map[string]any{"op": "eq", "path": "a", "value": 1}},
	}, map[string]any{"a": 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching route rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_allSuppressed(t *testing.T) {
	t.Parallel()
	// Given: matching route suppressed by degraded dependency
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "r", Priority: 1, RouteID: "x", When: map[string]any{"op": "eq", "path": "a", "value": 1}, DependsOn: []string{"git"}},
	}, map[string]any{"a": 1}, map[string]struct{}{"git": {}})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded — all candidates suppressed
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "suppressed") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_missingRouteID(t *testing.T) {
	t.Parallel()
	// Given: route rule with empty route_id
	// When: ResolveRoute runs
	_, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "bad", Priority: 1, RouteID: "", When: map[string]any{"op": "eq", "path": "a", "value": 1}},
	}, map[string]any{"a": 1}, nil)
	// Then: configuration error
	if err == nil || !strings.Contains(err.Error(), `resolve: route rule "bad" missing route_id`) {
		t.Fatalf("%v", err)
	}
}

func TestResolveRoute_tieAmbiguous(t *testing.T) {
	t.Parallel()
	// Given: two matching routes at same priority
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "a", Priority: 5, RouteID: "r1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "b", Priority: 5, RouteID: "r2", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: ambiguous
	if res.Kind != OutcomeAmbiguous {
		t.Fatalf("%+v", res)
	}
}

func TestDuplicatePriorityWarnings(t *testing.T) {
	t.Parallel()
	// Given: two state rules sharing priority
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 1, When: map[string]any{}},
		{Type: "state", ID: "b", Priority: 1, When: map[string]any{}},
	}
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings(rules)
	// Then: one warning
	if len(w) != 1 || !strings.Contains(w[0], "duplicated") {
		t.Fatalf("%v", w)
	}
}

func TestDuplicatePriorityWarnings_singleRuleSilent(t *testing.T) {
	t.Parallel()
	// Given: single rule
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "only", Priority: 1, When: map[string]any{}},
	})
	// Then: no warnings
	if len(w) != 0 {
		t.Fatal(w)
	}
}

func TestDuplicatePriorityWarnings_separateTypes(t *testing.T) {
	t.Parallel()
	// Given: duplicate priorities within state and within route
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "s1", Priority: 1, When: map[string]any{}},
		{Type: "state", ID: "s2", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r1", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r2", Priority: 1, When: map[string]any{}},
	})
	// Then: two within-type duplicate warnings plus one cross-kind warning for the shared priority.
	if len(w) != 3 {
		t.Fatalf("%v", w)
	}
}

func TestDuplicatePriorityWarnings_crossKindOnly(t *testing.T) {
	t.Parallel()
	// Given: one state and one route at same priority (no within-type duplicate)
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "s1", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r1", Priority: 1, When: map[string]any{}},
	})
	// Then: single cross-kind warning
	if len(w) != 1 || !strings.Contains(w[0], "shared across rule kinds") {
		t.Fatalf("%v", w)
	}
}
