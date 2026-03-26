package guard

import (
	"strings"
	"testing"
)

func TestRegistry_Register_duplicate(t *testing.T) {
	t.Parallel()
	// Given: a registry already holding merge-readiness
	r := NewRegistry()
	if err := r.Register(mergeReadinessGuard{}); err != nil {
		t.Fatal(err)
	}
	// When: the same id is registered again
	err := r.Register(mergeReadinessGuard{})
	// Then: Register returns an error (no silent overwrite)
	if err == nil {
		t.Fatal("want duplicate registration error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("got %v", err)
	}
}
