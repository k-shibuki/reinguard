package pullrequests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_withGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
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
	commit, _ := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	run(t, "git", dir, "checkout", string(commit[:len(commit)-1]))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
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
