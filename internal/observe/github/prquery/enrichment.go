package prquery

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Enrichment parses a PR comment body and returns extra fact fields (e.g. rate-limit seconds).
type Enrichment interface {
	Name() string
	Enrich(commentBody string) map[string]any
}

var (
	enrichmentMu     sync.RWMutex
	enrichmentByName = map[string]Enrichment{
		"coderabbit": coderabbitEnrichment{},
	}
)

func enrichmentByNameLocked(name string) (Enrichment, bool) {
	enrichmentMu.RLock()
	defer enrichmentMu.RUnlock()
	e, ok := enrichmentByName[name]
	return e, ok
}

// RegisterEnrichment adds an enrichment for validation and Collect. Used by tests.
func RegisterEnrichment(e Enrichment) error {
	if e == nil {
		return fmt.Errorf("prquery: nil enrichment")
	}
	name := strings.TrimSpace(e.Name())
	if name == "" {
		return fmt.Errorf("prquery: empty enrichment name")
	}
	enrichmentMu.Lock()
	defer enrichmentMu.Unlock()
	if _, exists := enrichmentByName[name]; exists {
		return fmt.Errorf("prquery: duplicate enrichment %q", name)
	}
	enrichmentByName[name] = e
	return nil
}

// EnrichmentNames returns registered enrichment names sorted for stable validation messages.
func EnrichmentNames() []string {
	enrichmentMu.RLock()
	defer enrichmentMu.RUnlock()
	return knownEnrichmentNamesLocked()
}

// ValidateEnrichmentNames returns an error if any name is unknown.
func ValidateEnrichmentNames(names []string) error {
	enrichmentMu.RLock()
	defer enrichmentMu.RUnlock()
	knownNames := knownEnrichmentNamesLocked()
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			return fmt.Errorf("prquery: empty enrich name")
		}
		if _, ok := enrichmentByName[n]; !ok {
			return fmt.Errorf("prquery: unknown enrich %q (known: %s)", n, strings.Join(knownNames, ", "))
		}
	}
	return nil
}

func knownEnrichmentNamesLocked() []string {
	out := make([]string, 0, len(enrichmentByName))
	for n := range enrichmentByName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func applyEnrichments(body string, enrichNames []string) map[string]any {
	if len(enrichNames) == 0 {
		return nil
	}
	enrichmentMu.RLock()
	defer enrichmentMu.RUnlock()
	out := make(map[string]any)
	for _, name := range enrichNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		e, ok := enrichmentByName[name]
		if !ok {
			continue
		}
		for k, v := range e.Enrich(body) {
			out[k] = v
		}
	}
	return out
}

// reviewBodyEnricher parses PullRequestReview bodies (distinct from issue comments).
type reviewBodyEnricher interface {
	EnrichReviewBody(reviewBody string) map[string]any
}

// issueCommentTierer lets a plugin define which issue comments should drive status.
type issueCommentTierer interface {
	// CommentMaxTier returns an integer tier for body. Higher values mean the comment is more
	// relevant for ordering or filtering; the numeric scale is defined by each enrichment plugin.
	CommentMaxTier(body string) int
}

// findingConversationCounter counts finding-shaped PR conversation comments for one bot.
type findingConversationCounter interface {
	CountFindingConversationComments(nodes []prCommentNode, login string) int
}

func applyReviewBodyEnrichments(reviewBody string, enrichNames []string) map[string]any {
	if len(enrichNames) == 0 || strings.TrimSpace(reviewBody) == "" {
		return nil
	}
	enrichmentMu.RLock()
	defer enrichmentMu.RUnlock()
	out := make(map[string]any)
	for _, name := range enrichNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		e, ok := enrichmentByName[name]
		if !ok {
			continue
		}
		rbe, ok := e.(reviewBodyEnricher)
		if !ok {
			continue
		}
		for k, v := range rbe.EnrichReviewBody(reviewBody) {
			out[k] = v
		}
	}
	return out
}
