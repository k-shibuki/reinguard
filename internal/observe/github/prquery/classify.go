package prquery

import "strings"

// StatusClassifier optionally classifies a bot's review status from accumulated
// per-bot signal fields (including enrich plugin output).
type StatusClassifier interface {
	ClassifyStatus(status map[string]any) string
}

// ClassifyBotStatus picks the first enrichment implementing StatusClassifier
// (in enrich name order); otherwise uses classifyBotStatusGeneric.
func ClassifyBotStatus(status map[string]any, enrichNames []string) string {
	enrichmentMu.RLock()
	defer enrichmentMu.RUnlock()
	for _, name := range enrichNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		e, ok := enrichmentByName[name]
		if !ok {
			continue
		}
		if c, ok := e.(StatusClassifier); ok {
			return c.ClassifyStatus(status)
		}
	}
	return classifyBotStatusGeneric(status)
}

func classifyBotStatusGeneric(m map[string]any) string {
	if signalBool(m, "contains_rate_limit") {
		return BotStatusRateLimited
	}
	if signalBool(m, "contains_review_paused") {
		return BotStatusReviewPaused
	}
	if signalBool(m, "contains_review_failed") {
		return BotStatusReviewFailed
	}
	if signalBool(m, "has_review") {
		return BotStatusCompleted
	}
	if strings.TrimSpace(signalString(m, "latest_comment_at")) != "" {
		return BotStatusPending
	}
	return BotStatusNotTriggered
}

func signalBool(m map[string]any, key string) bool {
	v, ok := m[key].(bool)
	return ok && v
}

func signalString(m map[string]any, key string) string {
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}
