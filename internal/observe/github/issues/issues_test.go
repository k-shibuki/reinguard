package issues

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_openCount(t *testing.T) {
	t.Parallel()
	// Given: search API returns total_count 3 for repo query
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query().Get("q")
		wantQ := "repo:o/r is:issue is:open"
		if q != wantQ {
			t.Errorf("search q=%q, want %q", q, wantQ)
		}
		_, _ = w.Write([]byte(`{"total_count": 3}`))
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{
		HTTP:    srv.Client(),
		Token:   "test-token",
		BaseURL: srv.URL,
	}
	// When: Collect runs
	m, err := Collect(context.Background(), c, "o", "r", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: open_count is 3
	inner, ok := m["issues"].(map[string]any)
	if !ok {
		t.Fatalf("expected issues map, got %T", m["issues"])
	}
	oc, ok := inner["open_count"].(int)
	if !ok {
		t.Fatalf("expected open_count int, got %T", inner["open_count"])
	}
	if oc != 3 {
		t.Fatalf("open_count=%d want 3", oc)
	}
}

func TestCollect_trimmedOwnerRepo_query(t *testing.T) {
	t.Parallel()
	// Given: owner/repo args with surrounding whitespace
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("q") != "repo:o/r is:issue is:open" {
			t.Errorf("unexpected q=%q", r.URL.Query().Get("q"))
		}
		_, _ = w.Write([]byte(`{"total_count": 0}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	_, err := Collect(context.Background(), c, "  o ", " r ", nil)
	// Then: no error (handler asserts trimmed query)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCollect_validationBoundaries(t *testing.T) {
	t.Parallel()
	// Given/When/Then: table of invalid client or owner/repo inputs
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "http://unused"}
	tests := []struct {
		name  string
		cli   *githubapi.Client
		owner string
		repo  string
		want  string
	}{
		{name: "nil client", cli: nil, owner: "o", repo: "r", want: "nil client"},
		{name: "empty owner", cli: c, owner: "", repo: "r", want: "non-empty"},
		{name: "empty repo", cli: c, owner: "o", repo: "", want: "non-empty"},
		{name: "whitespace repo", cli: c, owner: "o", repo: "   ", want: "non-empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Collect(context.Background(), tc.cli, tc.owner, tc.repo, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCollect_http500(t *testing.T) {
	t.Parallel()
	// Given: search endpoint returns 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	_, err := Collect(context.Background(), c, "o", "r", nil)
	// Then: wrapped error mentions fetch and status/body
	if err == nil || !strings.Contains(err.Error(), "fetch open issues") {
		t.Fatalf("expected wrap: %v", err)
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "server boom") {
		t.Fatalf("%v", err)
	}
}

func TestCollect_selectedIssues_merged(t *testing.T) {
	t.Parallel()
	// Given: search succeeds and GET issue returns a body with H2 and blocker line
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count": 1}`))
		case "/repos/o/r/issues/9":
			_, _ = w.Write([]byte(`{
  "number": 9,
  "state": "open",
  "title": "T",
  "labels": [{"name": "feat"}, {"name": "epic"}],
  "milestone": {"title": "v1"},
  "assignees": [{"login": "a1"}],
  "body": "## Context\n\nBlocked by #42\n\n### Not a section root\n"
}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect with issue numbers
	m, err := Collect(context.Background(), c, "o", "r", []int{9})
	if err != nil {
		t.Fatal(err)
	}
	inner := m["issues"].(map[string]any)
	if inner["open_count"].(int) != 1 {
		t.Fatalf("open_count %v", inner["open_count"])
	}
	sel := inner["selected_issues"].([]any)
	if len(sel) != 1 {
		t.Fatalf("selected_issues len %d", len(sel))
	}
	first := sel[0].(map[string]any)
	if n, ok := first["number"].(int); !ok || n != 9 {
		t.Fatalf("number got %v", first["number"])
	}
	if first["state"] != "open" {
		t.Fatalf("state %v", first["state"])
	}
	if !first["has_blockers"].(bool) || !first["is_epic"].(bool) {
		t.Fatalf("flags %+v", first)
	}
	secs := first["body_sections"].([]string)
	if len(secs) != 1 || secs[0] != "Context" {
		t.Fatalf("body_sections %v", secs)
	}
}
