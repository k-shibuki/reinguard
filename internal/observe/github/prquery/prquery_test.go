package prquery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
						"headRefName":      "feat/scoped",
						"headRefOid":       "abc123",
						"headRepository": map[string]any{
							"name":  "reinguard",
							"owner": map[string]any{"login": "forkowner"},
						},
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
								{
									"id":         "THREAD_1",
									"isResolved": true,
									"isOutdated": false,
									"comments": map[string]any{
										"nodes": []map[string]any{
											{
												"databaseId": 101,
												"body":       "resolved",
												"path":       "a.go",
												"line":       1,
											},
										},
									},
								},
								{
									"id":         "THREAD_2",
									"isResolved": false,
									"isOutdated": true,
									"comments": map[string]any{
										"nodes": []map[string]any{
											{
												"databaseId":   202,
												"body":         "please update scope",
												"path":         "internal/rgdcli/rgdcli.go",
												"line":         42,
												"originalLine": 40,
												"author":       map[string]any{"login": "coderabbitai[bot]"},
												"commit":       map[string]any{"oid": "0123456789abcdef0123456789abcdef01234567"},
												"originalCommit": map[string]any{
													"oid": "89abcdef0123456789abcdef0123456789abcdef",
												},
											},
										},
									},
								},
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
	if pull["head_ref"].(string) != "feat/scoped" {
		t.Fatalf("pull: %+v", pull)
	}
	if pull["head_repo_owner"].(string) != "forkowner" || pull["head_repo_name"].(string) != "reinguard" {
		t.Fatalf("head repository: %+v", pull)
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
	inbox := rev["review_inbox"].([]any)
	if len(inbox) != 1 {
		t.Fatalf("review_inbox: %+v", rev)
	}
	thread := inbox[0].(map[string]any)
	if thread["thread_id"] != "THREAD_2" {
		t.Fatalf("thread: %+v", thread)
	}
	switch id := thread["root_comment_id"].(type) {
	case int:
		if id != 202 {
			t.Fatalf("want root_comment_id 202, got %d", id)
		}
	case float64:
		if int(id) != 202 {
			t.Fatalf("want root_comment_id 202, got %v", id)
		}
	default:
		t.Fatalf("thread: %+v", thread)
	}
	if thread["is_outdated"] != true {
		t.Fatalf("is_outdated: %+v", thread)
	}
	if thread["path"] != "internal/rgdcli/rgdcli.go" {
		t.Fatalf("path: %+v", thread)
	}
	if thread["body"] != "please update scope" {
		t.Fatalf("body: %+v", thread)
	}
	if thread["author"] != "coderabbitai[bot]" {
		t.Fatalf("author: %+v", thread)
	}
	wantSHA := "0123456789abcdef0123456789abcdef01234567"
	origSHA := "89abcdef0123456789abcdef0123456789abcdef"
	if got := thread["commit_sha"]; got != wantSHA {
		t.Fatalf("commit_sha: got %v want %q", got, wantSHA)
	}
	if got := thread["original_commit_sha"]; got != origSHA {
		t.Fatalf("original_commit_sha: got %v want %q", got, origSHA)
	}
	switch ln := thread["line"].(type) {
	case int:
		if ln != 42 {
			t.Fatalf("line: %d", ln)
		}
	case float64:
		if int(ln) != 42 {
			t.Fatalf("line: %v", ln)
		}
	default:
		t.Fatalf("line type: %T", thread["line"])
	}
	switch oln := thread["original_line"].(type) {
	case int:
		if oln != 40 {
			t.Fatalf("original_line: %d", oln)
		}
	case float64:
		if int(oln) != 40 {
			t.Fatalf("original_line: %v", oln)
		}
	default:
		t.Fatalf("original_line type: %T", thread["original_line"])
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
	// When:  Collect computes bot_reviewer_status at the same instant as the status comment updatedAt.
	// Then:  contains_rate_limit=true, rate_limit_remaining_seconds equals parsed body duration, status=rate_limited.
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
	obsAt := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)
	_, rev, err := CollectWithOptions(context.Background(), c, "o", "r", 1, bots, &CollectOptions{Now: &obsAt})
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

func TestCollect_botReviewer_rateLimitAgeAdjustsRemaining(t *testing.T) {
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	// 60s after status comment: 125 - 60 = 65
	obsAt := time.Date(2026, 3, 27, 12, 1, 0, 0, time.UTC)
	_, rev, err := CollectWithOptions(context.Background(), c, "o", "r", 1, bots, &CollectOptions{Now: &obsAt})
	if err != nil {
		t.Fatal(err)
	}
	m := rev["bot_reviewer_status"].([]any)[0].(map[string]any)
	if m["rate_limit_remaining_seconds"].(int) != 65 {
		t.Fatalf("want 65 remaining, got %+v", m)
	}
}

func TestCollect_botReviewer_rateLimitAgeClampsToZero(t *testing.T) {
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
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	obsAt := time.Date(2026, 3, 27, 12, 10, 0, 0, time.UTC)
	_, rev, err := CollectWithOptions(context.Background(), c, "o", "r", 1, bots, &CollectOptions{Now: &obsAt})
	if err != nil {
		t.Fatal(err)
	}
	m := rev["bot_reviewer_status"].([]any)[0].(map[string]any)
	if m["rate_limit_remaining_seconds"].(int) != 0 {
		t.Fatalf("want 0 remaining, got %+v", m)
	}
}

func TestCollect_botReviewer_rateLimitWinsOverNewerAckComment(t *testing.T) {
	t.Parallel()
	// Given: an older issue comment with a rate-limit Review Status body and a newer short ack without markers.
	// When: Collect computes bot_reviewer_status.
	// Then: substring flags and enrichment use the status-bearing comment; latest_comment_at stays newest.
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
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "Sure! I'll kick off a new review to verify the fixes.",
									"updatedAt": "2026-03-27T12:05:00Z",
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
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	m := rev["bot_reviewer_status"].([]any)[0].(map[string]any)
	if !m["contains_rate_limit"].(bool) {
		t.Fatalf("want rate limit from status comment, got %+v", m)
	}
	if m["latest_comment_at"].(string) != "2026-03-27T12:05:00Z" {
		t.Fatalf("latest_comment_at: %+v", m)
	}
	if m["status_comment_at"].(string) != "2026-03-27T12:00:00Z" {
		t.Fatalf("status_comment_at: %+v", m)
	}
	if m["status_comment_source"].(string) != "status_marker" {
		t.Fatalf("status_comment_source: %+v", m)
	}
	if m["status"].(string) != BotStatusRateLimited {
		t.Fatalf("status: %+v", m)
	}
}

func TestCollect_botReviewer_terminalCleanSupersedesOlderRateLimitComment(t *testing.T) {
	t.Parallel()
	// Given: an older rate-limit comment and a newer terminal-clean issue comment (higher tier).
	// When: Collect runs with coderabbit enrichment.
	// Then: status reflects the newer completion, not the stale rate-limit text.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "head-sha",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes": []any{
								map[string]any{
									"state":  "COMMENTED",
									"author": map[string]any{"login": "coderabbitai"},
									"commit": map[string]any{"oid": "head-sha"},
								},
							},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "Rate limit exceeded. Please try again in 2 minutes and 5 seconds",
									"updatedAt": "2026-03-27T12:00:00Z",
								},
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "No issues found.",
									"updatedAt": "2026-03-27T12:10:00Z",
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
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	m := rev["bot_reviewer_status"].([]any)[0].(map[string]any)
	if m["contains_rate_limit"].(bool) {
		t.Fatalf("want rate limit cleared by newer terminal-clean comment, got %+v", m)
	}
	if m["status_comment_at"].(string) != "2026-03-27T12:10:00Z" {
		t.Fatalf("status_comment_at: %+v", m)
	}
	if m["status"].(string) != BotStatusCompletedClean {
		t.Fatalf("status: %+v", m)
	}
}

func TestCollect_botReviewer_newerRateLimitSupersedesOlderCleanComment(t *testing.T) {
	t.Parallel()
	// Given: an older terminal-clean comment and a newer rate-limit issue comment (same tier).
	// When: Collect runs with coderabbit enrichment.
	// Then: status reflects the newer rate limit, not the stale clean bill.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "head-sha",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes": []any{
								map[string]any{
									"state":  "COMMENTED",
									"author": map[string]any{"login": "coderabbitai"},
									"commit": map[string]any{"oid": "head-sha"},
								},
							},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "No issues found.",
									"updatedAt": "2026-03-27T12:00:00Z",
								},
								{
									"author":    map[string]any{"login": "coderabbitai[bot]"},
									"body":      "Rate limit exceeded. Please try again in 2 minutes and 5 seconds",
									"updatedAt": "2026-03-27T12:10:00Z",
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
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	m := rev["bot_reviewer_status"].([]any)[0].(map[string]any)
	if !m["contains_rate_limit"].(bool) {
		t.Fatalf("want rate limit from newer comment, got %+v", m)
	}
	if m["status_comment_at"].(string) != "2026-03-27T12:10:00Z" {
		t.Fatalf("status_comment_at: %+v", m)
	}
	if m["status"].(string) != BotStatusRateLimited {
		t.Fatalf("status: %+v", m)
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

func TestCollect_botReviewer_completedClean(t *testing.T) {
	t.Parallel()
	// Given: CodeRabbit leaves a matching review and a clean completion comment.
	// When:  Collect computes bot_reviewer_status with coderabbit enrichment.
	// Then:  status is completed_clean and aggregate diagnostics are terminal success.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "head-sha",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes": []any{
								map[string]any{
									"state":  "COMMENTED",
									"author": map[string]any{"login": "coderabbitai"},
									"commit": map[string]any{"oid": "head-sha"},
								},
							},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai"},
									"body":      "No issues found.",
									"updatedAt": "2026-03-28T12:00:00Z",
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
	bots := []BotReviewer{{ID: "cr", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["bot_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("status len: %v", st)
	}
	m := st[0].(map[string]any)
	if m["status"].(string) != BotStatusCompletedClean {
		t.Fatalf("status: %+v", m)
	}
	diag := rev["bot_review_diagnostics"].(map[string]any)
	if !diag["bot_review_completed"].(bool) {
		t.Fatalf("want bot_review_completed=true, got %+v", diag)
	}
	if diag["bot_review_pending"].(bool) {
		t.Fatalf("want bot_review_pending=false, got %+v", diag)
	}
	if !diag["bot_review_terminal"].(bool) {
		t.Fatalf("want bot_review_terminal=true, got %+v", diag)
	}
	if diag["bot_review_failed"].(bool) {
		t.Fatalf("want bot_review_failed=false, got %+v", diag)
	}
	if diag["bot_review_stale"].(bool) {
		t.Fatalf("want bot_review_stale=false, got %+v", diag)
	}
	if diag["duplicate_findings_detected"].(bool) {
		t.Fatalf("want duplicate_findings_detected=false, got %+v", diag)
	}
}

func TestCollect_botReviewer_commentOnlyNoReviewClean(t *testing.T) {
	t.Parallel()
	// Given: CodeRabbit posts an issue comment with "no actionable comments" and a commit range,
	// but does not create a GitHub Review (latestReviews empty).
	// When: Collect runs with coderabbit enrichment.
	// Then: status is completed_clean, diagnostics succeed, and review_commit_sha falls back to parsed HEAD.
	head := "abc1234567890abcdef1234567890abcdef1234"
	commentBody := "No actionable comments were generated in the recent review.\n\n" +
		"Reviewing files that changed from the base of the PR and between [1111111111111111111111111111111111111111](https://example.com/a) " +
		"and [" + head + "](https://example.com/b).\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": head,
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes":    []any{},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai"},
									"body":      commentBody,
									"updatedAt": "2026-03-28T12:00:00Z",
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
	bots := []BotReviewer{{ID: "cr", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["bot_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("status len: %v", st)
	}
	m := st[0].(map[string]any)
	if hr, ok := m["has_review"].(bool); ok && hr {
		t.Fatalf("want has_review=false, got %+v", m)
	}
	if m["status"].(string) != BotStatusCompletedClean {
		t.Fatalf("status: %+v", m)
	}
	if got := m["review_commit_sha"].(string); got != head {
		t.Fatalf("review_commit_sha: want %q, got %q", head, got)
	}
	diag := rev["bot_review_diagnostics"].(map[string]any)
	if !diag["bot_review_completed"].(bool) || diag["bot_review_pending"].(bool) {
		t.Fatalf("diag: %+v", diag)
	}
	if diag["bot_review_stale"].(bool) {
		t.Fatalf("want bot_review_stale=false, got %+v", diag)
	}
}

func TestCollect_reviewBodyDuplicateFindings(t *testing.T) {
	t.Parallel()
	// Given: CodeRabbit latest review body contains a Duplicate comments (N) summary line.
	// When:  Collect runs with coderabbit enrichment.
	// Then:  duplicate count is set on both normalized and legacy keys, and aggregate diagnostics fire.
	dupBody := "**Actionable comments posted: 4**\n\n<details>\n<summary>♻️ Duplicate comments (2)</summary><blockquote>\n</blockquote></details>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequest": map[string]any{
						"state": "OPEN", "isDraft": false, "title": "t",
						"mergeable": "UNKNOWN", "mergeStateStatus": "UNKNOWN",
						"baseRefName": "main", "headRefOid": "head-sha",
						"labels":                  map[string]any{"nodes": []any{}},
						"closingIssuesReferences": map[string]any{"nodes": []any{}},
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes": []any{
								map[string]any{
									"state":  "COMMENTED",
									"body":   dupBody,
									"author": map[string]any{"login": "coderabbitai"},
									"commit": map[string]any{"oid": "head-sha"},
								},
							},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai"},
									"body":      "Review completed.",
									"updatedAt": "2026-03-28T12:00:00Z",
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
	bots := []BotReviewer{{ID: "cr", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	st := rev["bot_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("bot_reviewer_status len: %d, want 1", len(st))
	}
	m := st[0].(map[string]any)
	if m["duplicate_findings_count"].(int) != 2 {
		t.Fatalf("duplicate_findings_count: %+v", m)
	}
	if m["cr_duplicate_findings_count"].(int) != 2 {
		t.Fatalf("cr_duplicate_findings_count: %+v", m)
	}
	diag := rev["bot_review_diagnostics"].(map[string]any)
	if !diag["duplicate_findings_detected"].(bool) {
		t.Fatalf("want duplicate_findings_detected=true, got %+v", diag)
	}
}

func TestCollect_commentPagination_fetchesOlderBotHeadMovedMessage(t *testing.T) {
	t.Parallel()
	// Given: newest comment page has no bot; older page has CodeRabbit head-moved notice.
	// When:  Collect merges comment pages for configured bots.
	// Then:  contains_review_failed is true and status is review_failed (second GraphQL round-trip).
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "PRCommentPage") {
			resp := map[string]any{
				"data": map[string]any{
					"repository": map[string]any{
						"pullRequest": map[string]any{
							"comments": map[string]any{
								"pageInfo": map[string]any{"hasPreviousPage": false, "startCursor": ""},
								"nodes": []map[string]any{
									{
										"author":    map[string]any{"login": "coderabbitai[bot]"},
										"body":      "The head commit changed during the review.",
										"updatedAt": "2026-03-28T11:00:00Z",
									},
								},
							},
						},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("encode: %v", err)
			}
			return
		}
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("unmarshal: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		include, _ := req.Variables["includeDetail"].(bool)
		if !include {
			t.Errorf("this test expects a single detail page only")
		}
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
							"pageInfo": map[string]any{"hasPreviousPage": true, "startCursor": "cur-old"},
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "someone"},
									"body":      "thread noise",
									"updatedAt": "2026-03-28T12:00:00Z",
								},
							},
						},
						"reviewThreads": map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []any{}},
					},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	bots := []BotReviewer{{ID: "coderabbit", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	if got := int(calls.Load()); got != 2 {
		t.Fatalf("want 2 graphql calls, got %d", got)
	}
	st := rev["bot_reviewer_status"].([]any)
	if len(st) != 1 {
		t.Fatalf("status len: %v", st)
	}
	m := st[0].(map[string]any)
	if !m["contains_review_failed"].(bool) {
		t.Fatalf("contains_review_failed: %+v", m)
	}
	if m["status"].(string) != BotStatusReviewFailed {
		t.Fatalf("status: %+v", m)
	}
}

func TestNormalizeGitHubActorLogin(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a, b  string
		equal bool
	}{
		{"coderabbitai[bot]", "coderabbitai", true},
		{"CoderabbitAI", "coderabbitai[bot]", true},
		{"foo", "bar", false},
	} {
		if got := normalizeGitHubActorLogin(tt.a) == normalizeGitHubActorLogin(tt.b); got != tt.equal {
			t.Fatalf("%q vs %q: got %v want %v", tt.a, tt.b, got, tt.equal)
		}
	}
}

func TestCollect_configLoginBotSuffix_graphQLLoginPlain(t *testing.T) {
	t.Parallel()
	// Given: GraphQL uses plain app login on comments/reviews; config uses "[bot]" suffix.
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
						"latestReviews": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false},
							"nodes": []any{
								map[string]any{"state": "COMMENTED", "author": map[string]any{"login": "coderabbitai"}},
							},
						},
						"comments": map[string]any{
							"nodes": []map[string]any{
								{
									"author":    map[string]any{"login": "coderabbitai"},
									"body":      "hello",
									"updatedAt": "2026-03-28T12:00:00Z",
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
	bots := []BotReviewer{{ID: "cr", Login: "coderabbitai[bot]", Required: true, Enrich: []string{"coderabbit"}}}
	_, rev, err := Collect(context.Background(), c, "o", "r", 1, bots)
	if err != nil {
		t.Fatal(err)
	}
	m := rev["bot_reviewer_status"].([]any)[0].(map[string]any)
	if !m["has_review"].(bool) || m["review_state"].(string) != "COMMENTED" {
		t.Fatalf("review: %+v", m)
	}
	if m["latest_comment_at"].(string) != "2026-03-28T12:00:00Z" {
		t.Fatalf("comment: %+v", m)
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
