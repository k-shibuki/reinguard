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
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "https://example.invalid"}
	m, err := Collect(context.Background(), c, "o", "r", 0)
	if err != nil {
		t.Fatal(err)
	}
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_unresolved"].(int) != 0 {
		t.Fatal()
	}
}

func TestCollect_withPR(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{},{}]`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	m, err := Collect(context.Background(), c, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	rev := m["reviews"].(map[string]any)
	if rev["review_threads_unresolved"].(int) != 2 {
		t.Fatal()
	}
}
