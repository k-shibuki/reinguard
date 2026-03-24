package githubapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_GetJSON_retry429(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL+"/ok", &out); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatal("expected retry")
	}
}

func TestClient_GetJSON_retry403RateLimit(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"You have exceeded a secondary rate limit"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL+"/ok", &out); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatal("expected retry on rate-limit 403")
	}
}

func TestClient_GetJSON_retry403RateLimit_usesRateLimitReset(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 2 {
			// GitHub uses whole-second epoch timestamps; wait until the next second boundary + 1s.
			reset := time.Now().Truncate(time.Second).Add(2 * time.Second).Unix()
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	var out map[string]any
	if err := c.GetJSON(context.Background(), srv.URL+"/ok", &out); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatal("expected retry after X-RateLimit-Reset delay")
	}
}
