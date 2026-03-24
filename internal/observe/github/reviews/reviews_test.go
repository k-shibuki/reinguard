package reviews

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_zeroPR(t *testing.T) {
	t.Parallel()
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "https://example.invalid"}
	m, err := Collect(context.Background(), c, "o", "r", 0)
	if err != nil {
		t.Fatal(err)
	}
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_unresolved"].(int) != 0 {
		t.Fatal()
	}
	if rev["pagination_incomplete"].(bool) {
		t.Fatal("expected pagination complete when prNumber is 0")
	}
}

func TestCollect_withPR_graphql(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" || r.Method != http.MethodPost {
			http.Error(w, "want POST /graphql", http.StatusNotFound)
			return
		}
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"pageInfo":{"hasNextPage":false},"nodes":[{"isResolved":true},{"isResolved":false},{"isResolved":false}]}}}}}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	m, err := Collect(context.Background(), c, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_unresolved"].(int) != 2 {
		t.Fatalf("unresolved want 2 got %v", rev["review_threads_unresolved"])
	}
	if rev["pagination_incomplete"].(bool) {
		t.Fatal("expected complete pagination")
	}
}

func TestCollect_paginationIncomplete(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"pageInfo":{"hasNextPage":true},"nodes":[{"isResolved":false}]}}}}}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	m, err := Collect(context.Background(), c, "o", "r", 42)
	if err != nil {
		t.Fatal(err)
	}
	rev := m["reviews"].(map[string]any)
	if !rev["pagination_incomplete"].(bool) {
		t.Fatal("expected pagination_incomplete when hasNextPage")
	}
}
