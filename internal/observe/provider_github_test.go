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

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	shaBytes, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(shaBytes))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count": 0}`))
		case "/repos/octocat/hello-world/pulls":
			_, _ = w.Write([]byte(`[]`))
		case "/repos/octocat/hello-world/commits/" + sha + "/status":
			_, _ = w.Write([]byte(`{"state":"success"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	p := NewGitHubProvider()
	p.APIBase = srv.URL

	frag, err := p.Collect(context.Background(), Options{WorkDir: repoDir})
	if err != nil {
		t.Fatal(err)
	}
	if frag.Signals["issues"] == nil {
		t.Fatalf("%v", frag.Signals)
	}
}

func TestGitHubProvider_Collect_ghRepoViewFails(t *testing.T) {
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
	frag, err := p.Collect(context.Background(), Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !frag.Degraded || len(frag.Diagnostics) != 1 || frag.Diagnostics[0].Code != "repo_resolve_failed" {
		t.Fatalf("%+v", frag)
	}
}

func TestGitHubProvider_Collect_ghAuthFails(t *testing.T) {
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
	frag, err := p.Collect(context.Background(), Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if !frag.Degraded || len(frag.Diagnostics) != 1 || frag.Diagnostics[0].Code != "auth_failed" {
		t.Fatalf("%+v", frag)
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
