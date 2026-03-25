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
	m, err := Collect(context.Background(), c, "o", "r")
	if err != nil {
		t.Fatal(err)
	}
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
	_, err := Collect(context.Background(), c, "  o ", " r ")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCollect_validationBoundaries(t *testing.T) {
	t.Parallel()
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
			_, err := Collect(context.Background(), tc.cli, tc.owner, tc.repo)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCollect_http500(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	_, err := Collect(context.Background(), c, "o", "r")
	if err == nil || !strings.Contains(err.Error(), "fetch open issues") {
		t.Fatalf("expected wrap: %v", err)
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "server boom") {
		t.Fatalf("%v", err)
	}
}
