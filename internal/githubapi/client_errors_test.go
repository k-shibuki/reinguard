package githubapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetJSON_http404IncludesBody(t *testing.T) {
	t.Parallel()
	// Given: server returns 404 with body
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such resource", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: GetJSON is called
	err := c.GetJSON(context.Background(), srv.URL+"/missing", &map[string]any{})
	// Then: error mentions status and response text
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "no such resource") {
		t.Fatalf("%v", err)
	}
}
