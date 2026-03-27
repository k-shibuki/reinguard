package resolve

import "testing"

func TestNearlyEqual(t *testing.T) {
	t.Parallel()
	// Given/When/Then: each subtest compares two floats against priorityEpsilon semantics
	tests := []struct {
		name string
		a, b float64
		want bool
	}{
		{"identical", 1.0, 1.0, true},
		{"within_epsilon", 1.0, 1.0 + priorityEpsilon/2, true},
		{"beyond_epsilon", 1.0, 1.0 + priorityEpsilon*100, false},
		{"negative", -5.0, -5.0, true},
		{"zero_and_small", 0.0, priorityEpsilon / 10, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NearlyEqual(tc.a, tc.b); got != tc.want {
				t.Fatalf("NearlyEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
