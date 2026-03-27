package githubapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostGraphQL_success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(b, &payload); err != nil {
			t.Errorf("body: %v", err)
		}
		if payload["query"] == nil {
			t.Error("missing query")
		}
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"alice"}}}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	var out struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
	}
	err := c.PostGraphQL(context.Background(), `query { viewer { login } }`, nil, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Viewer.Login != "alice" {
		t.Fatalf("got %+v", out)
	}
}

func TestPostGraphQL_graphqlErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}],"data":null}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	err := c.PostGraphQL(context.Background(), `query { __typename }`, nil, nil)
	if err == nil || err.Error() != "graphql error: boom" {
		t.Fatalf("got %v", err)
	}
}

func TestPostGraphQL_nilClient(t *testing.T) {
	t.Parallel()
	var c *Client
	err := c.PostGraphQL(context.Background(), "q", nil, nil)
	if err == nil || err.Error() != "nil client" {
		t.Fatalf("got %v", err)
	}
}
