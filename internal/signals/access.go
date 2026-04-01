// Package signals reads typed values from nested map[string]any trees using dot-separated paths.
package signals

import (
	"strconv"
	"strings"
)

// GetPath traverses a nested map[string]any along a dot-separated path.
// Empty segments are skipped so "a..b" resolves identically to "a.b".
// Numeric segments index into []any slices (e.g. selected_issues.0.state).
func GetPath(root map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = root
	for _, p := range parts {
		if p == "" {
			continue
		}
		if mm, ok := cur.(map[string]any); ok {
			v, ok := mm[p]
			if !ok {
				return nil, false
			}
			cur = v
			continue
		}
		if idx, err := strconv.Atoi(p); err == nil && idx >= 0 {
			if arr, ok := cur.([]any); ok && idx < len(arr) {
				cur = arr[idx]
				continue
			}
		}
		return nil, false
	}
	return cur, true
}

// GetString returns the string at the given dot path, or ("", false).
func GetString(root map[string]any, path string) (string, bool) {
	v, ok := GetPath(root, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetBool returns the bool at the given dot path, or (false, false).
func GetBool(root map[string]any, path string) (bool, bool) {
	v, ok := GetPath(root, path)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// GetInt returns the integer at the given dot path. JSON-decoded float64
// values are truncated to int. Returns (0, false) on missing path or
// non-numeric type.
func GetInt(root map[string]any, path string) (int, bool) {
	v, ok := GetPath(root, path)
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return x, true
	case float64:
		return int(x), true
	case int64:
		return int(x), true
	default:
		return 0, false
	}
}
