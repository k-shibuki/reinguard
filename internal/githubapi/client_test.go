package githubapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetJSON_success(t *testing.T) {
	t.Parallel()
	// Given: test server returning JSON object
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"a":1}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	var out map[string]any
	// When: GetJSON decodes response
	if err := c.GetJSON(context.Background(), srv.URL+"/x", &out); err != nil {
		t.Fatal(err)
	}
	v, ok := out["a"].(float64)
	// Then: map decodes numeric field
	if !ok || v != 1 {
		t.Fatalf("unexpected decoded payload: %#v", out)
	}
}

func TestClient_APIBase_default(t *testing.T) {
	t.Parallel()
	// Given: zero-value Client
	// When/Then: APIBase is GitHub default
	c := &Client{}
	if c.APIBase() != "https://api.github.com" {
		t.Fatal(c.APIBase())
	}
}
