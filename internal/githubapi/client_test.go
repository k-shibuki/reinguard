package githubapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetJSON_success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"a":1}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL+"/x", &out); err != nil {
		t.Fatal(err)
	}
}

func TestClient_APIBase_default(t *testing.T) {
	t.Parallel()
	c := &Client{}
	if c.APIBase() != "https://api.github.com" {
		t.Fatal(c.APIBase())
	}
}
