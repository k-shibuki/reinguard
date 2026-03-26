package observe

import "testing"

func TestGitProviderID(t *testing.T) {
	t.Parallel()
	p := &GitProvider{}
	if id := p.ID(); id != "git" {
		t.Fatalf("got %q, want %q", id, "git")
	}
}

func TestGitHubProviderID(t *testing.T) {
	t.Parallel()
	p := &GitHubProvider{}
	if id := p.ID(); id != "github" {
		t.Fatalf("got %q, want %q", id, "github")
	}
}

func TestIntFromMap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want int
	}{
		{"int", map[string]any{"n": 42}, "n", 42},
		{"int64", map[string]any{"n": int64(99)}, "n", 99},
		{"float64", map[string]any{"n": 3.0}, "n", 3},
		{"missing", map[string]any{}, "n", 0},
		{"string_fallback", map[string]any{"n": "nope"}, "n", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := intFromMap(tc.m, tc.key); got != tc.want {
				t.Fatalf("intFromMap(%v, %q) = %d, want %d", tc.m, tc.key, got, tc.want)
			}
		})
	}
}
