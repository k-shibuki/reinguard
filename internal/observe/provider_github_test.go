package observe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGitHubProvider_Collect_fakeGH(t *testing.T) {
	// Given: fake gh on PATH (auth + repo view succeed)
	if runtime.GOOS == "windows" {
		t.Skip("fake gh executable is a Unix #!/bin/sh script")
	}
	tmp := t.TempDir()
	ghBin := filepath.Join(tmp, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "token" ]; then
  echo testtoken
  exit 0
fi
if [ "$1" = "repo" ] && [ "$2" = "view" ]; then
  echo "octocat/hello-world"
  exit 0
fi
exit 1
`
	if err := os.WriteFile(ghBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Given: a git repo with at least one commit (HEAD sha used by status stub)
	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	shaBytes, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(shaBytes))

	// Given: httptest server returning minimal GitHub REST payloads for this repo/sha
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/issues":
			_, _ = w.Write([]byte(`{"total_count": 0}`))
		case r.URL.Path == "/repos/octocat/hello-world/pulls":
			_, _ = w.Write([]byte(`[]`))
		case r.URL.Path == "/repos/octocat/hello-world/commits/"+sha+"/status":
			_, _ = w.Write([]byte(`{"state":"success"}`))
		case strings.HasSuffix(r.URL.Path, "/comments"):
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	// Given: provider targeting the test server as API base
	p := NewGitHubProvider()
	p.APIBase = srv.URL

	// When: Collect runs against the repo
	frag, err := p.Collect(context.Background(), Options{WorkDir: repoDir})
	if err != nil {
		t.Fatal(err)
	}
	// Then: issues facet is present under signals
	if frag.Signals["issues"] == nil {
		t.Fatalf("%v", frag.Signals)
	}
}

func TestGitHubProvider_Collect_ghRepoViewFails(t *testing.T) {
	// Given: gh auth OK but repo view fails
	if runtime.GOOS == "windows" {
		t.Skip("fake gh executable is a Unix #!/bin/sh script")
	}
	tmp := t.TempDir()
	ghBin := filepath.Join(tmp, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "token" ]; then
  echo testtoken
  exit 0
fi
exit 1
`
	if err := os.WriteFile(ghBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))
	p := NewGitHubProvider()
	// When: Collect runs
	frag, err := p.Collect(context.Background(), Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded with repo_resolve_failed diagnostic
	if !frag.Degraded || len(frag.Diagnostics) != 1 || frag.Diagnostics[0].Code != "repo_resolve_failed" {
		t.Fatalf("%+v", frag)
	}
}

func TestGitHubProvider_Collect_ghAuthFails(t *testing.T) {
	// Given: gh exits non-zero immediately (auth failure)
	if runtime.GOOS == "windows" {
		t.Skip("fake gh executable is a Unix #!/bin/sh script")
	}
	tmp := t.TempDir()
	ghBin := filepath.Join(tmp, "gh")
	script := `#!/bin/sh
exit 1
`
	if err := os.WriteFile(ghBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))
	p := NewGitHubProvider()
	// When: Collect runs
	frag, err := p.Collect(context.Background(), Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded with auth_failed diagnostic
	if !frag.Degraded || len(frag.Diagnostics) != 1 || frag.Diagnostics[0].Code != "auth_failed" {
		t.Fatalf("%+v", frag)
	}
}

func TestGitHubProviderFactory_apiBase_valid(t *testing.T) {
	t.Parallel()
	want := "https://api.example.test/"
	p, err := GitHubProviderFactory(map[string]any{"api_base": want})
	if err != nil {
		t.Fatal(err)
	}
	gp, ok := p.(*GitHubProvider)
	if !ok {
		t.Fatalf("got %T", p)
	}
	if gp.APIBase != want {
		t.Fatalf("APIBase=%q want %q", gp.APIBase, want)
	}
}

func TestGitHubProviderFactory_apiBase_wrongType(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{"api_base": 123})
	if err == nil || !strings.Contains(err.Error(), "api_base must be a string") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_apiBase_emptyWhenSet(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{"api_base": "  "})
	if err == nil || !strings.Contains(err.Error(), "non-empty") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_apiBase_notAbsoluteHTTPURL(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"api.example.com",
		"://bad",
		"/relative-only",
		"ftp://api.example.com/",
	} {
		_, err := GitHubProviderFactory(map[string]any{"api_base": raw})
		if err == nil {
			t.Fatalf("api_base=%q: want error", raw)
		}
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
