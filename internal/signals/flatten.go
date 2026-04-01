package signals

import "strconv"

// Flatten expands namespaced signal maps into dotted keys for rule matching.
// For each top-level namespace, intermediate maps are stored under both their
// dotted path and as nested values at the parent key (preserving structure).
// []any values are indexed as prefix.0, prefix.1, … for path resolution.
func Flatten(signals map[string]any) map[string]any {
	out := map[string]any{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		if m, ok := v.(map[string]any); ok {
			if prefix != "" {
				out[prefix] = v
			}
			for k, vv := range m {
				p := k
				if prefix != "" {
					p = prefix + "." + k
				}
				out[p] = vv
				walk(p, vv)
			}
			return
		}
		if arr, ok := v.([]any); ok {
			if prefix != "" {
				out[prefix] = v
			}
			for i, vv := range arr {
				p := prefix + "." + strconv.Itoa(i)
				out[p] = vv
				walk(p, vv)
			}
			return
		}
		if prefix != "" {
			out[prefix] = v
		}
	}
	for ns, v := range signals {
		out[ns] = v
		walk(ns, v)
	}
	return out
}
