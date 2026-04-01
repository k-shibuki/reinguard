package issues

import (
	"regexp"
	"strings"
)

var blockedByPattern = regexp.MustCompile(`Blocked by #(\d+)`)

// ParseSections returns H2 header titles from issue body (lines matching "## " at line start).
// "###" and deeper headings are not treated as section roots.
func ParseSections(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		t := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(t, "## ") {
			continue
		}
		rest := strings.TrimPrefix(t, "## ")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			continue
		}
		if strings.HasPrefix(rest, "#") {
			continue
		}
		out = append(out, rest)
	}
	return out
}

// HasBlockers reports whether the body contains a "Blocked by #N" reference.
func HasBlockers(body string) bool {
	return blockedByPattern.MatchString(body)
}
