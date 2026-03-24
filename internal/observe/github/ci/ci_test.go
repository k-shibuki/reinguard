package ci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_status(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitInit(t, dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"state":"success"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	m, warns, err := Collect(context.Background(), c, "o", "r", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	cimap := m["ci"].(map[string]any)
	if cimap["ci_status"].(string) != "success" {
		t.Fatalf("%v", cimap)
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
