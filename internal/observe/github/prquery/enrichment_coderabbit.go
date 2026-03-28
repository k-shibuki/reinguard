package prquery

import (
	"regexp"
	"strconv"
	"strings"
)

type coderabbitEnrichment struct{}

func (coderabbitEnrichment) Name() string { return "coderabbit" }

// Enrich extracts rate_limit_remaining_seconds from CodeRabbit-style messages.
func (coderabbitEnrichment) Enrich(commentBody string) map[string]any {
	sec := parseCoderabbitRateLimitSeconds(commentBody)
	if sec <= 0 {
		return nil
	}
	return map[string]any{"rate_limit_remaining_seconds": sec}
}

// Matches: "try again in 5 minutes and 30 seconds", "try again in 1 minute", "in 2 minutes and 15 seconds"
var (
	reTryAgainMinutesSeconds = regexp.MustCompile(`(?i)try\s+again\s+in\s+(\d+)\s*minutes?(?:\s+and\s+(\d+)\s*seconds?)?`)
	reInMinutesSeconds       = regexp.MustCompile(`(?i)in\s+(\d+)\s*minutes?(?:\s+and\s+(\d+)\s*seconds?)?`)
	reSecondsOnly            = regexp.MustCompile(`(?i)(?:try\s+again\s+)?in\s+(\d+)\s*seconds?`)
)

func parseCoderabbitRateLimitSeconds(body string) int {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0
	}
	if m := reTryAgainMinutesSeconds.FindStringSubmatch(body); len(m) >= 2 {
		return minutesSecondsToTotal(m[1], subOrEmpty(m, 2))
	}
	if m := reInMinutesSeconds.FindStringSubmatch(body); len(m) >= 2 {
		return minutesSecondsToTotal(m[1], subOrEmpty(m, 2))
	}
	if m := reSecondsOnly.FindStringSubmatch(body); len(m) >= 2 {
		if s, err := strconv.Atoi(m[1]); err == nil && s >= 0 {
			return s
		}
	}
	return 0
}

func subOrEmpty(m []string, i int) string {
	if i < len(m) && m[i] != "" {
		return m[i]
	}
	return ""
}

func minutesSecondsToTotal(minStr, secStr string) int {
	minutes, err := strconv.Atoi(minStr)
	if err != nil || minutes < 0 {
		return 0
	}
	total := minutes * 60
	if secStr != "" {
		s, err := strconv.Atoi(secStr)
		if err == nil && s >= 0 {
			total += s
		}
	}
	return total
}
