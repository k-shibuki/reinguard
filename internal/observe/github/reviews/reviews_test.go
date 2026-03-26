package reviews

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_zeroPR(t *testing.T) {
	t.Parallel()
	// Given: PR number 0 (no open PR) — no HTTP needed
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "https://example.invalid"}
	// When: Collect runs
	m, err := Collect(context.Background(), c, "o", "r", 0)
	if err != nil {
		t.Fatal(err)
	}
	// Then: zero unresolved threads
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_unresolved"].(int) != 0 {
		t.Fatal()
	}
}

func TestCollect_withPR(t *testing.T) {
	t.Parallel()
	// Given: API returns two review thread objects
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{},{}]`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs with non-zero PR
	m, err := Collect(context.Background(), c, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Then: unresolved count matches array length
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_unresolved"].(int) != 2 {
		t.Fatal()
	}
}
