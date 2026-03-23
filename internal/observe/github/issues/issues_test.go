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
	inner := m["issues"].(map[string]any)
	if inner["open_count"].(int) != 3 {
		t.Fatalf("%v", m)
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
	if err == nil || !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "server boom") {
		t.Fatalf("%v", err)
	}
}
