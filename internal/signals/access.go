// Package signals reads typed values from nested map[string]any trees using dot-separated paths.
package signals

import "strings"

// GetPath traverses a nested map[string]any along a dot-separated path.
// Empty segments are skipped so "a..b" resolves identically to "a.b".
func GetPath(root map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = root
	for _, p := range parts {
		if p == "" {
			continue
		}
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := mm[p]
		if !ok {
			return nil, false
		}
		cur = v
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
