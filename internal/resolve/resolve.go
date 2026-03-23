// Package resolve applies ADR-0004 priority selection and ADR-0007 outcomes.
package resolve

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/match"
)

const priorityEpsilon = 1e-9

// OutcomeKind classifies resolution result.
type OutcomeKind string

const (
	OutcomeResolved  OutcomeKind = "resolved"
	OutcomeAmbiguous OutcomeKind = "ambiguous"
	OutcomeDegraded  OutcomeKind = "degraded"
)

// Result is the outcome of resolving among matching rules of one type.
type Result struct {
	Kind            OutcomeKind      `json:"kind"`
	StateID         string           `json:"state_id,omitempty"`
	RouteID         string           `json:"route_id,omitempty"`
	RuleID          string           `json:"rule_id,omitempty"`
	Reason          string           `json:"reason,omitempty"`
	Candidates      []string         `json:"candidates,omitempty"`
	RouteCandidates []RouteCandidate `json:"route_candidates,omitempty"`
	Priority        float64          `json:"priority,omitempty"`
}

// RouteCandidate is one matching route rule after depends_on suppression, ordered for output.
type RouteCandidate struct {
	RuleID   string  `json:"rule_id"`
	RouteID  string  `json:"route_id"`
	Priority float64 `json:"priority"`
}

// StateRules filters rules to type state.
func StateRules(rules []config.Rule) []config.Rule {
	return filterType(rules, "state")
}

// RouteRules filters rules to type route.
func RouteRules(rules []config.Rule) []config.Rule {
	return filterType(rules, "route")
}

func filterType(rules []config.Rule, typ string) []config.Rule {
	var out []config.Rule
	for _, r := range rules {
		if r.Type == typ {
			out = append(out, r)
		}
	}
	return out
}

// ResolveState evaluates state rules against signals and degraded sources.
func ResolveState(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	candidates, err := matchingRules(StateRules(rules), signals)
	if err != nil {
		return Result{}, err
	}
	if len(candidates) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: "no matching state rule"}, nil
	}
	active := suppressDependsOn(candidates, degraded)
	if len(active) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: "all matches suppressed by depends_on"}, nil
	}
	best := active[0].Priority
	for _, c := range active[1:] {
		if c.Priority < best {
			best = c.Priority
		}
	}
	var atBest []config.Rule
	for _, c := range active {
		if nearlyEqual(c.Priority, best) {
			atBest = append(atBest, c)
		}
	}
	if len(atBest) > 1 {
		ids := make([]string, len(atBest))
		for i := range atBest {
			ids[i] = atBest[i].ID
		}
		return Result{
			Kind:       OutcomeAmbiguous,
			Priority:   best,
			Candidates: ids,
			Reason:     "multiple rules at same best priority",
		}, nil
	}
	r := atBest[0]
	if r.StateID == "" {
		return Result{}, fmt.Errorf("resolve: state rule %q missing state_id", r.ID)
	}
	return Result{
		Kind:     OutcomeResolved,
		StateID:  r.StateID,
		RuleID:   r.ID,
		Priority: r.Priority,
	}, nil
}

// ResolveRoute selects route rules (same priority semantics as state).
func ResolveRoute(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	candidates, err := matchingRules(RouteRules(rules), signals)
	if err != nil {
		return Result{}, err
	}
	if len(candidates) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: "no matching route rule"}, nil
	}
	active := suppressDependsOn(candidates, degraded)
	if len(active) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: "all route matches suppressed"}, nil
	}

	sorted := append([]config.Rule(nil), active...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})
	var routeCandidates []RouteCandidate
	for _, c := range sorted {
		if c.RouteID == "" {
			continue
		}
		routeCandidates = append(routeCandidates, RouteCandidate{
			RuleID:   c.ID,
			RouteID:  c.RouteID,
			Priority: c.Priority,
		})
	}

	best := sorted[0].Priority
	for _, c := range sorted[1:] {
		if c.Priority < best {
			best = c.Priority
		}
	}
	var atBest []config.Rule
	for _, c := range sorted {
		if nearlyEqual(c.Priority, best) {
			atBest = append(atBest, c)
		}
	}
	if len(atBest) > 1 {
		ids := make([]string, len(atBest))
		for i := range atBest {
			ids[i] = atBest[i].ID
		}
		return Result{
			Kind:            OutcomeAmbiguous,
			Priority:        best,
			Candidates:      ids,
			RouteCandidates: routeCandidates,
			Reason:          "multiple route rules at same best priority",
		}, nil
	}
	r := atBest[0]
	if r.RouteID == "" {
		return Result{}, fmt.Errorf("resolve: route rule %q missing route_id", r.ID)
	}
	return Result{
		Kind:            OutcomeResolved,
		RouteID:         r.RouteID,
		RuleID:          r.ID,
		Priority:        r.Priority,
		RouteCandidates: routeCandidates,
	}, nil
}

func matchingRules(rules []config.Rule, signals map[string]any) ([]config.Rule, error) {
	var out []config.Rule
	for _, r := range rules {
		ok, err := match.Eval(r.When, signals)
		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", r.ID, err)
		}
		if ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func suppressDependsOn(rules []config.Rule, degraded map[string]struct{}) []config.Rule {
	if len(degraded) == 0 {
		return rules
	}
	var out []config.Rule
	for _, r := range rules {
		skip := false
		for _, d := range r.DependsOn {
			if _, ok := degraded[d]; ok {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, r)
		}
	}
	return out
}

// NearlyEqual reports floating-point equality for duplicate-priority warnings (ADR-0004).
func NearlyEqual(a, b float64) bool {
	return nearlyEqual(a, b)
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= priorityEpsilon
}

// DuplicatePriorityWarnings returns rule id pairs that share the same priority (same type).
func DuplicatePriorityWarnings(rules []config.Rule) []string {
	by := map[string]map[string][]string{} // type -> priorityKey -> ids
	for _, r := range rules {
		if _, ok := by[r.Type]; !ok {
			by[r.Type] = map[string][]string{}
		}
		key := strconv.FormatFloat(r.Priority, 'f', -1, 64)
		by[r.Type][key] = append(by[r.Type][key], r.ID)
	}
	var msgs []string
	for typ, m := range by {
		for priKey, ids := range m {
			if len(ids) < 2 {
				continue
			}
			msgs = append(msgs, fmt.Sprintf("type %s priority %s duplicated by rules %v", typ, priKey, ids))
		}
	}
	return msgs
}
