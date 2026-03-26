// Package labels defines GitHub label metadata for PR policy and maintainer commands.
package labels

// TypeLabels is the set of PR type label names. Must stay aligned with
// `const TYPE_LABELS` in .github/scripts/pr-policy-check.js.
var TypeLabels = map[string]struct{}{
	"feat": {}, "fix": {}, "refactor": {}, "perf": {}, "docs": {},
	"test": {}, "ci": {}, "build": {}, "chore": {}, "style": {}, "revert": {},
}

// TypeLabelNames returns a sorted copy for tests and iteration (stable order).
func TypeLabelNames() []string {
	return []string{
		"feat", "fix", "refactor", "perf", "docs",
		"test", "ci", "build", "chore", "style", "revert",
	}
}
