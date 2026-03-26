package githubapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient_GetJSON_retry429(t *testing.T) {
	// Given: server that returns 429 once then 200
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
	// When: GetJSON runs
	if err := c.GetJSON(context.Background(), srv.URL+"/ok", &out); err != nil {
		t.Fatal(err)
	}
	// Then: success after retry
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatal("expected retry")
	}
}

func TestClient_GetJSON_retry403RateLimit(t *testing.T) {
	// Given: rate-limit 403 then success
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
	// When: GetJSON runs
	if err := c.GetJSON(context.Background(), srv.URL+"/ok", &out); err != nil {
		t.Fatal(err)
	}
	// Then: retried once
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatal("expected retry on rate-limit 403")
	}
}

func TestClient_GetJSON_retry403RateLimit_usesRateLimitReset(t *testing.T) {
	// Given: 403 with X-RateLimit-Reset then success
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
	// When: GetJSON runs
	if err := c.GetJSON(context.Background(), srv.URL+"/ok", &out); err != nil {
		t.Fatal(err)
	}
	// Then: delayed retry succeeded
	if atomic.LoadInt32(&hits) < 2 {
		t.Fatal("expected retry after X-RateLimit-Reset delay")
	}
}

func TestClient_GetJSON_contextCancelsDuring429Backoff(t *testing.T) {
	// Given: 429 with long Retry-After and short-lived context
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	c := &Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	var out map[string]any
	// When: GetJSON runs
	err := c.GetJSON(ctx, srv.URL+"/x", &out)
	// Then: context deadline exceeded
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
}
