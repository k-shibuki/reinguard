package githubapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPostGraphQL_success(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var handlerErr error
	setHandlerErr := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if handlerErr == nil {
			handlerErr = err
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(b, &payload); err != nil {
			setHandlerErr(err)
			return
		}
		if payload["query"] == nil {
			setHandlerErr(errors.New("missing query"))
			return
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
	if handlerErr != nil {
		t.Fatal(handlerErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	if out.Viewer.Login != "alice" {
		t.Fatalf("got %+v", out)
	}
}

func TestClient_graphQLEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		base string
		want string
	}{
		{"", "https://api.github.com/graphql"},
		{"https://api.github.com", "https://api.github.com/graphql"},
		{"https://ghe.example.com/api/v3", "https://ghe.example.com/api/graphql"},
		{"https://ghe.example.com/api/v3/", "https://ghe.example.com/api/graphql"},
	}
	for _, tc := range tests {
		c := &Client{BaseURL: tc.base}
		if got := c.graphQLEndpoint(); got != tc.want {
			t.Fatalf("BaseURL=%q: got %q want %q", tc.base, got, tc.want)
		}
	}
}

func TestPostGraphQL_restApiV3BaseHitsApiGraphqlPath(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/graphql" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"data":{"__typename":"Query"}}`))
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL + "/api/v3"}
	err := c.PostGraphQL(context.Background(), `query { __typename }`, nil, nil)
	if err != nil {
		t.Fatal(err)
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
