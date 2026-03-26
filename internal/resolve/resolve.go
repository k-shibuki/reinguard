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
	TargetID        string           `json:"target_id,omitempty"` // unified winning ID (mirrors state_id or route_id)
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

type resolveProfile struct {
	ruleType            string
	noMatchReason       string
	allSuppressedReason string
	ambiguousReason     string
	routeStyle          bool
}

func profileForRuleType(ruleType string) (resolveProfile, error) {
	switch ruleType {
	case "state":
		return resolveProfile{
			ruleType:            "state",
			noMatchReason:       "no matching state rule",
			allSuppressedReason: "all matches suppressed by depends_on",
			ambiguousReason:     "multiple rules at same best priority",
			routeStyle:          false,
		}, nil
	case "route":
		return resolveProfile{
			ruleType:            "route",
			noMatchReason:       "no matching route rule",
			allSuppressedReason: "all route matches suppressed",
			ambiguousReason:     "multiple route rules at same best priority",
			routeStyle:          true,
		}, nil
	default:
		return resolveProfile{}, fmt.Errorf("resolve: unsupported rule type %q", ruleType)
	}
}

// Resolve applies ADR-0004 selection to rules of a single type ("state" or "route").
func Resolve(rules []config.Rule, signals map[string]any, degraded map[string]struct{}, ruleType string) (Result, error) {
	p, err := profileForRuleType(ruleType)
	if err != nil {
		return Result{}, err
	}
	candidates, err := matchingRules(filterType(rules, p.ruleType), signals)
	if err != nil {
		return Result{}, err
	}
	if len(candidates) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: p.noMatchReason}, nil
	}
	active := suppressDependsOn(candidates, degraded)
	if len(active) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: p.allSuppressedReason}, nil
	}

	ordered := active
	var routeCandidates []RouteCandidate
	if p.routeStyle {
		ordered = append([]config.Rule(nil), active...)
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].Priority != ordered[j].Priority {
				return ordered[i].Priority < ordered[j].Priority
			}
			return ordered[i].ID < ordered[j].ID
		})
		for _, c := range ordered {
			if c.RouteID == "" {
				continue
			}
			routeCandidates = append(routeCandidates, RouteCandidate{
				RuleID:   c.ID,
				RouteID:  c.RouteID,
				Priority: c.Priority,
			})
		}
	}

	best := minPriority(ordered)
	var atBest []config.Rule
	for _, c := range ordered {
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
			Reason:          p.ambiguousReason,
		}, nil
	}
	r := atBest[0]
	if p.routeStyle {
		if r.RouteID == "" {
			return Result{}, fmt.Errorf("resolve: route rule %q missing route_id", r.ID)
		}
		return Result{
			Kind:            OutcomeResolved,
			RouteID:         r.RouteID,
			TargetID:        r.RouteID,
			RuleID:          r.ID,
			Priority:        r.Priority,
			RouteCandidates: routeCandidates,
		}, nil
	}
	if r.StateID == "" {
		return Result{}, fmt.Errorf("resolve: state rule %q missing state_id", r.ID)
	}
	return Result{
		Kind:     OutcomeResolved,
		StateID:  r.StateID,
		TargetID: r.StateID,
		RuleID:   r.ID,
		Priority: r.Priority,
	}, nil
}

// minPriority returns the minimum priority in rules. rules must be non-empty;
// Resolve only calls this after len(active) > 0.
func minPriority(rules []config.Rule) float64 {
	best := rules[0].Priority
	for _, c := range rules[1:] {
		if c.Priority < best {
			best = c.Priority
		}
	}
	return best
}

// ResolveState evaluates state rules against signals and degraded sources.
func ResolveState(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	return Resolve(rules, signals, degraded, "state")
}

// ResolveRoute selects route rules (same priority semantics as state).
func ResolveRoute(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	return Resolve(rules, signals, degraded, "route")
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

// DuplicatePriorityWarnings returns validation warnings for rules that share the same
// numeric priority (ADR-0004: one shared priority space across rule kinds).
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

	// Cross-kind collisions: same numeric priority on different rule types (not caught above
	// when each type has at most one rule at that priority).
	global := map[string][]config.Rule{} // priorityKey -> rules
	for _, r := range rules {
		key := strconv.FormatFloat(r.Priority, 'f', -1, 64)
		global[key] = append(global[key], r)
	}
	for priKey, group := range global {
		if len(group) < 2 {
			continue
		}
		typeSeen := map[string]struct{}{}
		for _, r := range group {
			typeSeen[r.Type] = struct{}{}
		}
		if len(typeSeen) < 2 {
			continue
		}
		ids := make([]string, len(group))
		for i := range group {
			ids[i] = group[i].ID
		}
		msgs = append(msgs, fmt.Sprintf("priority %s shared across rule kinds by rules %v", priKey, ids))
	}
	sort.Strings(msgs)
	return msgs
}
