package signals

import "testing"

func TestGetPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wantVal any
		root    map[string]any
		name    string
		path    string
		wantOK  bool
	}{
		{
			name:    "simple key",
			root:    map[string]any{"a": 1},
			path:    "a",
			wantVal: 1,
			wantOK:  true,
		},
		{
			name:    "nested key",
			root:    map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}},
			path:    "a.b.c",
			wantVal: "deep",
			wantOK:  true,
		},
		{
			name:   "missing key",
			root:   map[string]any{"a": 1},
			path:   "b",
			wantOK: false,
		},
		{
			name:   "missing nested key",
			root:   map[string]any{"a": map[string]any{"b": 1}},
			path:   "a.c",
			wantOK: false,
		},
		{
			name:    "double dots ignored",
			root:    map[string]any{"a": map[string]any{"b": 42}},
			path:    "a..b",
			wantVal: 42,
			wantOK:  true,
		},
		{
			name:   "nil root",
			root:   nil,
			path:   "a",
			wantOK: false,
		},
		{
			name:   "non-map intermediate",
			root:   map[string]any{"a": "string"},
			path:   "a.b",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := GetPath(tc.root, tc.path)
			if ok != tc.wantOK {
				t.Fatalf("GetPath ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.wantVal {
				t.Fatalf("GetPath = %v, want %v", got, tc.wantVal)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		root   map[string]any
		path   string
		want   string
		wantOK bool
	}{
		{
			name:   "present string",
			root:   map[string]any{"ci": map[string]any{"status": "success"}},
			path:   "ci.status",
			want:   "success",
			wantOK: true,
		},
		{
			name:   "non-string type",
			root:   map[string]any{"ci": map[string]any{"status": 42}},
			path:   "ci.status",
			wantOK: false,
		},
		{
			name:   "missing path",
			root:   map[string]any{},
			path:   "ci.status",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := GetString(tc.root, tc.path)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		root   map[string]any
		path   string
		want   bool
		wantOK bool
	}{
		{
			name:   "present true",
			root:   map[string]any{"git": map[string]any{"clean": true}},
			path:   "git.clean",
			want:   true,
			wantOK: true,
		},
		{
			name:   "present false",
			root:   map[string]any{"git": map[string]any{"clean": false}},
			path:   "git.clean",
			want:   false,
			wantOK: true,
		},
		{
			name:   "non-bool type",
			root:   map[string]any{"git": map[string]any{"clean": "yes"}},
			path:   "git.clean",
			wantOK: false,
		},
		{
			name:   "missing path returns false false",
			root:   map[string]any{},
			path:   "git.clean",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := GetBool(tc.root, tc.path)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		root   map[string]any
		path   string
		want   int
		wantOK bool
	}{
		{
			name:   "int value",
			root:   map[string]any{"count": 5},
			path:   "count",
			want:   5,
			wantOK: true,
		},
		{
			name:   "float64 value truncated",
			root:   map[string]any{"count": float64(3)},
			path:   "count",
			want:   3,
			wantOK: true,
		},
		{
			name:   "int64 value",
			root:   map[string]any{"count": int64(7)},
			path:   "count",
			want:   7,
			wantOK: true,
		},
		{
			name:   "non-numeric type",
			root:   map[string]any{"count": "five"},
			path:   "count",
			wantOK: false,
		},
		{
			name:   "missing path",
			root:   map[string]any{},
			path:   "count",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := GetInt(tc.root, tc.path)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
