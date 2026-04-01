package issues

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollectSelected_success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/issues/1" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"number":1,"state":"OPEN","title":"Hi","labels":[],"body":""}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	out, err := CollectSelected(context.Background(), c, "o", "r", []int{1})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len %d", len(out))
	}
	m := out[0].(map[string]any)
	if m["number"].(int) != 1 || m["state"] != "open" {
		t.Fatalf("%+v", m)
	}
}

func TestCollectSelected_single404_fatal(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	_, err := CollectSelected(context.Background(), c, "o", "r", []int{404})
	if err == nil || !errors.Is(err, ErrFatalObservation) {
		t.Fatalf("got %v", err)
	}
}

func TestCollectSelected_multiOne404_errorObject(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/issues/1":
			_, _ = w.Write([]byte(`{"number":1,"state":"open","title":"ok","labels":[],"body":""}`))
		case "/repos/o/r/issues/2":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	out, err := CollectSelected(context.Background(), c, "o", "r", []int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len %d", len(out))
	}
	m0 := out[0].(map[string]any)
	if m0["title"] != "ok" {
		t.Fatalf("%+v", m0)
	}
	m1 := out[1].(map[string]any)
	if m1["number"].(int) != 2 {
		t.Fatalf("%+v", m1)
	}
	if _, ok := m1["error"].(string); !ok {
		t.Fatalf("want error string %+v", m1)
	}
}

func TestCollectSelected_apiError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	out, err := CollectSelected(context.Background(), c, "o", "r", []int{1})
	if err != nil {
		t.Fatal(err)
	}
	m := out[0].(map[string]any)
	es, ok := m["error"].(string)
	if !ok || !strings.Contains(es, "502") {
		t.Fatalf("%+v", m)
	}
}

func TestCollectSelected_emptyBodySections(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"number":3,"state":"open","title":"x","labels":[],"body":""}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	out, err := CollectSelected(context.Background(), c, "o", "r", []int{3})
	if err != nil {
		t.Fatal(err)
	}
	m := out[0].(map[string]any)
	if m["has_blockers"].(bool) {
		t.Fatal("want false")
	}
	secs := m["body_sections"].([]string)
	if len(secs) != 0 {
		t.Fatalf("%v", secs)
	}
}
