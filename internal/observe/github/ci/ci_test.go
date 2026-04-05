package ci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_status(t *testing.T) {
	t.Parallel()
	// Given: git repo with HEAD and API returning commit status success
	dir := t.TempDir()
	gitInit(t, dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"state":"success"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatalf("expected ci map, got %T", m["ci"])
	}
	st, ok := cimap["ci_status"].(string)
	if !ok {
		t.Fatalf("expected ci_status string, got %T", cimap["ci_status"])
	}
	// Then: ci_status success, no warnings
	if st != "success" {
		t.Fatalf("%v", cimap)
	}
}

func TestCollect_http500(t *testing.T) {
	t.Parallel()
	// Given: API returns 500
	dir := t.TempDir()
	gitInit(t, dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	_, _, err := Collect(context.Background(), c, "o", "r", dir, "")
	// Then: error mentions 500
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("got %v", err)
	}
}

func TestCollect_nonGitWorkDir(t *testing.T) {
	t.Parallel()
	// Given: non-git workdir (no HTTP round-trip; head fails first)
	dir := t.TempDir()
	c := &githubapi.Client{HTTP: http.DefaultClient, Token: "t", BaseURL: "http://unused.invalid"}
	// When: Collect runs
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) == 0 {
		t.Fatal("expected warning when workdir is not a git checkout")
	}
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatalf("expected ci map, got %T", m["ci"])
	}
	st, ok := cimap["ci_status"].(string)
	// Then: warning and ci_status unknown
	if !ok || st != "unknown" {
		t.Fatalf("ci_status=%v (%T)", cimap["ci_status"], cimap["ci_status"])
	}
}

func TestCollect_usesHeadSHAOverride(t *testing.T) {
	t.Parallel()
	// Given: git repo with HEAD and an explicit override SHA
	dir := t.TempDir()
	gitInit(t, dir)
	var (
		mu            sync.Mutex
		requestedPath string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestedPath = r.URL.Path
		mu.Unlock()
		_, _ = w.Write([]byte(`{"state":"pending"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	override := "0123456789abcdef0123456789abcdef01234567"
	// When: Collect runs with the override
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, override)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	mu.Lock()
	path := requestedPath
	mu.Unlock()
	if !strings.Contains(path, override) {
		t.Fatalf("path=%q want override %q", path, override)
	}
	// Then: the API request and emitted signal both use the override SHA
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatalf("expected ci map, got %T", m["ci"])
	}
	if cimap["head_sha"] != override {
		t.Fatalf("ci=%+v", cimap)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}
