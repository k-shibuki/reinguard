package prquery

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

func TestCollect_zeroPR(t *testing.T) {
	t.Parallel()
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "https://example.invalid"}
	pull, rev, err := Collect(context.Background(), c, "o", "r", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pull != nil {
		t.Fatalf("want nil pull detail, got %v", pull)
	}
	assertReviewsZeros(t, rev, false)
}

func TestCollect_pullRequestNull(t *testing.T) {
	t.Parallel()
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
	pull, rev, err := Collect(context.Background(), c, "o", "r", 99, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pull != nil {
		t.Fatal("expected nil pull detail")
	}
	assertReviewsZeros(t, rev, false)
}

func TestCollect_graphqlErrorsPropagates(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"upstream failure"}],"data":null}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	_, _, err := Collect(context.Background(), c, "o", "r", 1, nil)
	if err == nil || err.Error() != "graphql error: upstream failure" {
		t.Fatalf("got %v", err)
	}
}

//nolint:gocyclo // table-style GraphQL stub
func TestCollect_onePage_threadsAndDetail(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		_ = json.Unmarshal(body, &req)
		include, _ := req.Variables["includeDetail"].(bool)
		if !include {
			t.Fatal("first page must include detail")
		}
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state":            "OPEN",
						"isDraft":          false,
						"title":            "feat: x",
						"mergeable":        "MERGEABLE",
						"mergeStateStatus": "CLEAN",
						"baseRefName":      "main",
						"headRefOid":       "abc123",
						"labels": map[string]any{
							"nodes": []map[string]any{{"name": "feat"}},
						},
						"closingIssuesReferences": map[string]any{
							"nodes": []map[string]any{{"number": 70}},
						},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes": []map[string]any{
								{"state": "APPROVED", "author": map[string]any{"login": "alice"}},
								{"state": "CHANGES_REQUESTED", "author": map[string]any{"login": "bob"}},
							},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{},
						},
						"reviewThreads": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
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
	pull, rev, err := Collect(context.Background(), c, "o", "r", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pull["state"].(string) != "open" || pull["title"].(string) != "feat: x" {
		t.Fatalf("pull: %+v", pull)
	}
	if pull["mergeable"].(string) != "mergeable" {
		t.Fatalf("mergeable: %+v", pull)
	}
	labels := pull["labels"].([]any)
	if len(labels) != 1 || labels[0].(string) != "feat" {
		t.Fatalf("labels: %+v", labels)
	}
	issues := pull["closing_issue_numbers"].([]any)
	if len(issues) != 1 {
		t.Fatalf("closing issues: %+v", issues)
	}
	switch n := issues[0].(type) {
	case int:
		if n != 70 {
			t.Fatalf("want 70, got %d", n)
		}
	case float64:
		if int(n) != 70 {
			t.Fatalf("want 70, got %v", n)
		}
	default:
		t.Fatalf("unexpected type %T", issues[0])
	}
	if rev["review_threads_total"].(int) != 2 || rev["review_threads_unresolved"].(int) != 1 {
		t.Fatalf("threads: %+v", rev)
	}
	if rev["review_decisions_total"].(int) != 2 || rev["review_decisions_approved"].(int) != 1 || rev["review_decisions_changes_requested"].(int) != 1 {
		t.Fatalf("decisions: %+v", rev)
	}
}

func TestCollect_reviewThreadsPaginationIncomplete(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		_ = json.Unmarshal(body, &req)
		include, _ := req.Variables["includeDetail"].(bool)
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{},
				},
			},
		}
		pr := resp["data"].(map[string]any)["repository"].(map[string]any)["pullRequest"].(map[string]any)
		if include {
			pr["state"] = "OPEN"
			pr["isDraft"] = false
			pr["title"] = "t"
			pr["mergeable"] = "UNKNOWN"
			pr["mergeStateStatus"] = "UNKNOWN"
			pr["baseRefName"] = "main"
			pr["headRefOid"] = "x"
			pr["labels"] = map[string]any{"nodes": []any{}}
			pr["closingIssuesReferences"] = map[string]any{"nodes": []any{}}
			pr["latestReviews"] = map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}}
			pr["comments"] = map[string]any{"nodes": []any{}}
		}
		pr["reviewThreads"] = map[string]any{
			"pageInfo": map[string]any{"hasNextPage": true, "endCursor": "next"},
			"nodes":    []map[string]any{{"isResolved": false}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rev["pagination_incomplete"].(bool) {
		t.Fatalf("want incomplete: %+v", rev)
	}
	if got := int(calls.Load()); got != maxReviewThreadPages {
		t.Fatalf("want %d calls, got %d", maxReviewThreadPages, got)
	}
}

func TestCollect_latestReviewsTruncated(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "x",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"comments":                map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": true},
							"nodes":    []map[string]any{{"state": "COMMENTED", "author": map[string]any{"login": "u"}}},
						},
						"reviewThreads": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes":    []any{},
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
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rev["review_decisions_truncated"].(bool) {
		t.Fatalf("want truncated: %+v", rev)
	}
}

func TestCollect_trackedReviewer_rateLimitAndEnrich(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "x",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes":    []any{},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "Rate limit exceeded. Please try again in 2 minutes and 5 seconds",
									"updatedAt": "2026-03-27T12:00:00Z",
								},
							},
						},
						"reviewThreads": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes":    []any{},
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
	tracked := []TrackedReviewer{{Login: "coderabbitai[bot]", Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, tracked)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["tracked_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("status len: %v", st)
	}
	m := st[0].(map[string]any)
	if !m["contains_rate_limit"].(bool) {
		t.Fatalf("rate limit: %+v", m)
	}
	if m["rate_limit_remaining_seconds"].(int) != 125 {
		t.Fatalf("seconds: %+v", m)
	}
}

func TestCollect_emptyLabelsAndClosingIssuesAreArrays(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "x",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews":           map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}},
						"comments":                map[string]any{"nodes": []any{}},
						"reviewThreads":           map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	pull, _, err := Collect(context.Background(), c, "o", "r", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	labels, ok := pull["labels"].([]any)
	if !ok || len(labels) != 0 {
		t.Fatalf("labels: %+v", pull["labels"])
	}
	issues, ok := pull["closing_issue_numbers"].([]any)
	if !ok || len(issues) != 0 {
		t.Fatalf("issues: %+v", pull["closing_issue_numbers"])
	}
}

func TestCollect_trackedReviewer_noEnrichOmitsRateLimitSeconds(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "x",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews":           map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "Rate limit exceeded. Please try again in 2 minutes and 5 seconds",
									"updatedAt": "2026-03-27T12:00:00Z",
								},
							},
						},
						"reviewThreads": map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	tracked := []TrackedReviewer{{Login: "coderabbitai[bot]"}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, tracked)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["tracked_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("status len: %v", st)
	}
	m := st[0].(map[string]any)
	if _, exists := m["rate_limit_remaining_seconds"]; exists {
		t.Fatalf("unexpected key: %+v", m)
	}
}

func assertReviewsZeros(t *testing.T, rev map[string]any, incomplete bool) {
	t.Helper()
	if rev["review_threads_total"].(int) != 0 || rev["review_threads_unresolved"].(int) != 0 {
		t.Fatalf("%+v", rev)
	}
	if rev["pagination_incomplete"].(bool) != incomplete {
		t.Fatalf("%+v", rev)
	}
}
