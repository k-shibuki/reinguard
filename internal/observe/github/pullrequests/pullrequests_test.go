package pullrequests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_prForBranch_whenSearchOmitsHeadMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "feature")

	// GitHub search often returns PR items without populated head.ref; the
	// head:<branch> qualifier still constrains matches.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if strings.Contains(q, "head:feature") {
			_, _ = w.Write([]byte(`{"total_count":1,"incomplete_results":false,"items":[{"number":99}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"total_count":0,"incomplete_results":false,"items":[]}`))
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	m, warns, err := Collect(context.Background(), c, "o", "r", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	pr := m["pull_requests"].(map[string]any)
	if !pr["pr_exists_for_branch"].(bool) {
		t.Fatalf("want pr for branch, got %v", pr)
	}
	if pr["pr_number_for_branch"].(int) != 99 {
		t.Fatalf("want pr number 99, got %v", pr["pr_number_for_branch"])
	}
}

func TestCollect_withGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	emptySearch := `{"total_count":0,"incomplete_results":false,"items":[]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if strings.Contains(q, "head:main") {
			_, _ = w.Write([]byte(emptySearch))
			return
		}
		_, _ = w.Write([]byte(emptySearch))
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	m, warns, err := Collect(context.Background(), c, "o", "r", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	pr := m["pull_requests"].(map[string]any)
	if pr["current_branch"].(string) != "main" {
		t.Fatalf("%v", pr)
	}
	if oc, ok := pr["open_count"].(int); !ok || oc != 0 {
		t.Fatalf("open_count want 0 got %v (%T)", pr["open_count"], pr["open_count"])
	}
}

func run(t *testing.T, name, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v %s", name, args, err, string(out))
	}
}

func TestCollect_detachedHeadWarning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	commit, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	run(t, "git", dir, "checkout", strings.TrimSpace(string(commit)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total_count":0,"incomplete_results":false,"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	_, warns, err := Collect(context.Background(), c, "o", "r", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) == 0 {
		t.Fatal("expected warning")
	}
}
