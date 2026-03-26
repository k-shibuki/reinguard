package signals

import (
	"math"
	"testing"
)

func TestGetPath(t *testing.T) {
	t.Parallel()
	// Given: table cases covering path traversal, missing keys, and malformed structure
	// When: each case calls GetPath
	// Then: ok and value match expectations
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
	// Given: table cases for string extraction and type errors
	// When: each case calls GetString
	// Then: ok and string match expectations
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
	// Given: table cases for bool extraction and type errors
	// When: each case calls GetBool
	// Then: ok and bool match expectations
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
	nan := math.NaN()
	pinf := math.Inf(1)
	ninf := math.Inf(-1)
	hugeF := 1e100
	maxIntF := float64(math.MaxInt64)
	// Given: table cases for integer extraction, JSON numeric shapes, and float edge cases
	// When: each case calls GetInt
	// Then: ok and int match expectations (float64 uses Go's int(x) conversion rules)
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
			name:   "fractional float64 truncated toward zero",
			root:   map[string]any{"count": 3.7},
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
		{
			name:   "NaN float64 coerced via int()",
			root:   map[string]any{"count": nan},
			path:   "count",
			want:   int(nan),
			wantOK: true,
		},
		{
			name:   "positive Inf float64 coerced via int()",
			root:   map[string]any{"count": pinf},
			path:   "count",
			want:   int(pinf),
			wantOK: true,
		},
		{
			name:   "negative Inf float64 coerced via int()",
			root:   map[string]any{"count": ninf},
			path:   "count",
			want:   int(ninf),
			wantOK: true,
		},
		{
			name:   "float64 magnitude beyond int range",
			root:   map[string]any{"count": hugeF},
			path:   "count",
			want:   int(hugeF),
			wantOK: true,
		},
		{
			name:   "float64 max int64 bit pattern",
			root:   map[string]any{"count": maxIntF},
			path:   "count",
			want:   int(maxIntF),
			wantOK: true,
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
