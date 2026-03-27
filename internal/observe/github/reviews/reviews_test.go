package reviews

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// TestCollect_zeroPR ensures PR number ≤0 returns zero thread counts without calling the network.
func TestCollect_zeroPR(t *testing.T) {
	t.Parallel()
	// Given: PR number 0 — no HTTP needed
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "https://example.invalid"}
	// When: Collect runs
	m, err := Collect(context.Background(), c, "o", "r", 0)
	if err != nil {
		t.Fatal(err)
	}
	// Then: zeros and no pagination gap
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_total"].(int) != 0 || rev["review_threads_unresolved"].(int) != 0 || rev["pagination_incomplete"].(bool) {
		t.Fatalf("%+v", rev)
	}
}

// TestCollect_graphqlOnePage exercises one GraphQL page with mixed resolved threads and complete pagination.
func TestCollect_graphqlOnePage(t *testing.T) {
	t.Parallel()
	// Given: GraphQL returns two threads, one unresolved
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(body, &req)
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"reviewThreads": map[string]any{
							"pageInfo": map[string]any{
								"hasNextPage": false,
								"endCursor":   "",
							},
							"nodes": []map[string]any{
								{"isResolved": true},
								{"isResolved": false},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs with non-zero PR
	m, err := Collect(context.Background(), c, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Then: total 2, unresolved 1
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_total"].(int) != 2 {
		t.Fatalf("total: %+v", rev)
	}
	if rev["review_threads_unresolved"].(int) != 1 {
		t.Fatalf("unresolved: %+v", rev)
	}
	if rev["pagination_incomplete"].(bool) {
		t.Fatal("expected complete pagination")
	}
}

// TestCollect_graphqlPaginationIncomplete simulates endless hasNextPage until the page cap sets pagination_incomplete.
func TestCollect_graphqlPaginationIncomplete(t *testing.T) {
	t.Parallel()
	// Given: every page reports hasNextPage true (simulated infinite pages → cap)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"reviewThreads": map[string]any{
							"pageInfo": map[string]any{
								"hasNextPage": true,
								"endCursor":   "next",
							},
							"nodes": []map[string]any{
								{"isResolved": false},
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When
	m, err := Collect(context.Background(), c, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	// Then: incomplete true; fetched max pages worth of nodes
	rev := m["reviews"].(map[string]any)
	if !rev["pagination_incomplete"].(bool) {
		t.Fatalf("expected incomplete: %+v calls=%d", rev, calls.Load())
	}
	if got := int(calls.Load()); got != maxReviewThreadPages {
		t.Fatalf("expected %d graphql calls, got %d", maxReviewThreadPages, got)
	}
	if rev["review_threads_total"].(int) != maxReviewThreadPages {
		t.Fatalf("total: %+v", rev)
	}
}

// TestCollect_nullPullRequest covers pullRequest: null (missing PR) as zero threads and complete pagination.
func TestCollect_nullPullRequest(t *testing.T) {
	t.Parallel()
	// Given: GraphQL returns pullRequest: null (e.g. wrong PR number)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": nil,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	m, err := Collect(context.Background(), c, "o", "r", 99)
	if err != nil {
		t.Fatal(err)
	}
	// Then: no threads, pagination complete
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_total"].(int) != 0 || rev["review_threads_unresolved"].(int) != 0 || rev["pagination_incomplete"].(bool) {
		t.Fatalf("%+v", rev)
	}
}

// TestCollect_graphqlErrorsPropagates asserts Collect returns an error when GraphQL responds with an errors envelope (facet reliability).
func TestCollect_graphqlErrorsPropagates(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"upstream failure"}],"data":null}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	_, err := Collect(context.Background(), c, "o", "r", 1)
	if err == nil || err.Error() != "graphql error: upstream failure" {
		t.Fatalf("got %v", err)
	}
}
