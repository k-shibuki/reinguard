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
	// Given: GraphQL returns pullRequest: null for the requested number.
	// When:  Collect processes the response.
	// Then:  pull detail is nil and review counters are zero.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": nil,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
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
	// Given: GraphQL response contains an errors array with no data.
	// When:  Collect processes the error response.
	// Then:  the error is surfaced to the caller.
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
	// Given: a single GraphQL page containing PR detail, latest reviews, and review threads.
	// When:  Collect is executed for a valid PR number.
	// Then:  pull/review fields are normalized and aggregated as expected.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unmarshal request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		include, _ := req.Variables["includeDetail"].(bool)
		if !include {
			t.Errorf("first page must include detail")
			http.Error(w, "first page must include detail", http.StatusBadRequest)
			return
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
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
	// Given: reviewThreads pagination continuously reports hasNextPage=true.
	// When:  Collect paginates with maxReviewThreadPages cap.
	// Then:  pagination_incomplete is true and request count is capped.
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unmarshal request: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
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
	// Given: latestReviews pageInfo reports hasNextPage=true.
	// When:  Collect processes the single page of review decisions.
	// Then:  review_decisions_truncated is true.
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
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

func TestCollect_botReviewer_rateLimitAndEnrich(t *testing.T) {
	t.Parallel()
	// Given: bot reviewer comments contain a CodeRabbit rate-limit message with enrich enabled.
	// When:  Collect computes bot_reviewer_status.
	// Then:  contains_rate_limit=true, rate_limit_remaining_seconds is populated, status=rate_limited.
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["bot_reviewer_status"].([]any)
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
	if m["status"].(string) != BotStatusRateLimited {
		t.Fatalf("status: %+v", m)
	}
	diag := rev["bot_review_diagnostics"].(map[string]any)
	if diag["bot_review_failed"].(bool) != true || diag["bot_review_pending"].(bool) != false {
		t.Fatalf("diag: %+v", diag)
	}
}

func TestCollect_emptyLabelsAndClosingIssuesAreArrays(t *testing.T) {
	t.Parallel()
	// Given: a PR with no labels and no closing issue references.
	// When:  Collect builds pull detail.
	// Then:  labels and closing_issue_numbers are empty slices (not nil/null).
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
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

func TestCollect_botReviewer_noEnrichOmitsRateLimitSeconds(t *testing.T) {
	t.Parallel()
	// Given: bot reviewer with no enrich plugins and rate-limit comments present.
	// When:  Collect computes bot_reviewer_status.
	// Then:  rate_limit_remaining_seconds key is absent; generic classifier still sets rate_limited.
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	bots := []BotReviewer{{ID: "cr", Login: "coderabbitai[bot]", Required: true}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["bot_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("status len: %v", st)
	}
	m := st[0].(map[string]any)
	if _, exists := m["rate_limit_remaining_seconds"]; exists {
		t.Fatalf("unexpected key: %+v", m)
	}
	if m["status"].(string) != BotStatusRateLimited {
		t.Fatalf("status: %+v", m)
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
