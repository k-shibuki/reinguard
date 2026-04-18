package observe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
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

func TestGitHubProvider_Collect_fallsBackToOriginRemoteWhenGhRepoViewFails(t *testing.T) {
	// Given: gh auth OK but repo view fails, while git origin still points at GitHub
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

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGitCmd(t, repoDir, "remote", "add", "origin", "git@github.com:octocat/hello-world.git")

	shaBytes, err := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.TrimSpace(string(shaBytes))

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

	p := NewGitHubProvider()
	p.APIBase = srv.URL
	// When: Collect runs against the repo with gh repo view failing but origin still resolvable
	frag, err := p.Collect(context.Background(), Options{WorkDir: repoDir})
	if err != nil {
		t.Fatal(err)
	}
	// Then: repository identity comes from origin and no repo_resolve_failed diagnostic is emitted
	repoSignal, ok := frag.Signals["repository"].(map[string]any)
	if !ok {
		t.Fatalf("repository signal missing: %+v", frag)
	}
	if repoSignal["owner"] != "octocat" || repoSignal["name"] != "hello-world" {
		t.Fatalf("repository signal: %+v", repoSignal)
	}
	if repoSignal["identity_source"] != "local_git" {
		t.Fatalf("identity_source: %+v", repoSignal)
	}
	for _, d := range frag.Diagnostics {
		if d.Code == "repo_resolve_failed" {
			t.Fatalf("unexpected repo_resolve_failed: %+v", frag.Diagnostics)
		}
	}
}

func TestGitHubProvider_Collect_reviewsFacetSkippedWhenPullRequestsFail(t *testing.T) {
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

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGitCmd(t, repoDir, "checkout", "-b", "feature")

	// Given: search/issues succeeds but /pulls fails so pullrequests.Collect errors
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/issues":
			_, _ = w.Write([]byte(`{"total_count": 0}`))
		case strings.HasPrefix(r.URL.Path, "/repos/octocat/hello-world/pulls"):
			http.Error(w, "upstream error", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	p := NewGitHubProvider()
	p.APIBase = srv.URL

	// When: Collect requests reviews facet only
	frag, err := p.Collect(context.Background(), Options{WorkDir: repoDir, GitHubFacet: "reviews"})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded from pull-requests error and no misleading zero review signals
	if !frag.Degraded {
		t.Fatalf("want degraded, got %+v", frag)
	}
	if frag.Signals["reviews"] != nil {
		t.Fatalf("reviews signals must be omitted when PR lookup failed: %v", frag.Signals["reviews"])
	}
	var pullErr bool
	for _, d := range frag.Diagnostics {
		if d.Provider == "github.pull-requests" {
			pullErr = true
			break
		}
	}
	if !pullErr {
		t.Fatalf("want github.pull-requests diagnostic, got %+v", frag.Diagnostics)
	}
}

func TestGitHubProvider_Collect_ghAuthFails(t *testing.T) {
	// Given: gh exits non-zero immediately (auth failure), no resolvable repo identity
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
	// When: Collect runs (empty dir — cannot resolve local git or gh repo)
	frag, err := p.Collect(context.Background(), Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded with repo_resolve_failed (no origin, gh unusable)
	if !frag.Degraded || len(frag.Diagnostics) != 1 || frag.Diagnostics[0].Code != "repo_resolve_failed" {
		t.Fatalf("%+v", frag)
	}
}

func TestGitHubProvider_Collect_ghAuthFails_keepsLocalRepositoryIdentity(t *testing.T) {
	// Given: auth fails but remote.origin.url resolves to GitHub (sandbox-style: token blocked, git readable)
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

	repoDir := t.TempDir()
	runGitCmd(t, repoDir, "init")
	runGitCmd(t, repoDir, "remote", "add", "origin", "git@github.com:octocat/hello-world.git")

	p := NewGitHubProvider()
	// When: Collect runs while auth fails but local origin still identifies the repo
	frag, err := p.Collect(context.Background(), Options{WorkDir: repoDir})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded auth_failed keeps repository identity from local_git
	if !frag.Degraded || len(frag.Diagnostics) != 1 || frag.Diagnostics[0].Code != "auth_failed" {
		t.Fatalf("%+v", frag)
	}
	repoSignal, ok := frag.Signals["repository"].(map[string]any)
	if !ok {
		t.Fatalf("repository signal missing: %+v", frag)
	}
	if repoSignal["owner"] != "octocat" || repoSignal["name"] != "hello-world" {
		t.Fatalf("repository signal: %+v", repoSignal)
	}
	if repoSignal["identity_source"] != "local_git" {
		t.Fatalf("identity_source: %+v", repoSignal)
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

func TestGitHubProviderFactory_botReviewers_ok(t *testing.T) {
	t.Parallel()
	p, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "coderabbit", "login": "coderabbitai[bot]", "required": true, "enrich": []any{"coderabbit"}},
			map[string]any{"id": "codex", "login": "chatgpt-codex-connector[bot]", "required": false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	gp, ok := p.(*GitHubProvider)
	if !ok || len(gp.BotReviewers) != 2 {
		t.Fatalf("got %T %+v", p, gp)
	}
	br0 := gp.BotReviewers[0]
	if br0.ID != "coderabbit" || br0.Login != "coderabbitai[bot]" || !br0.Required || len(br0.Enrich) != 1 || br0.Enrich[0] != "coderabbit" {
		t.Fatalf("%+v", br0)
	}
	br1 := gp.BotReviewers[1]
	if br1.ID != "codex" || br1.Login != "chatgpt-codex-connector[bot]" || br1.Required || len(br1.Enrich) != 0 {
		t.Fatalf("%+v", br1)
	}
}

func TestGitHubProviderFactory_botReviewers_unknownEnrich(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "x", "login": "x", "required": true, "enrich": []any{"no-such-plugin"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown enrich") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_badReviewTriggerRegex(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "x", "login": "x", "required": true, "review_triggers": []any{"("}},
		},
	})
	if err == nil {
		t.Fatal("want error from invalid regexp")
	}
}

func TestGitHubProviderFactory_botReviewers_badShape(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{"bot_reviewers": "nope"})
	if err == nil || !strings.Contains(err.Error(), "must be an array") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_duplicateID(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "same", "login": "a", "required": true},
			map[string]any{"id": "same", "login": "b", "required": false},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_invalidIDPattern(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "Bad-ID", "login": "x", "required": true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "id must match") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_missingRequiredField(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "x", "login": "y"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "required is required") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_requiredMustBeBoolean(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "x", "login": "y", "required": "true"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "required must be a boolean") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_blankLogin(t *testing.T) {
	t.Parallel()
	_, err := GitHubProviderFactory(map[string]any{
		"bot_reviewers": []any{
			map[string]any{"id": "x", "login": "   ", "required": true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "login is required") {
		t.Fatalf("got %v", err)
	}
}

func TestGitHubProviderFactory_botReviewers_blankID(t *testing.T) {
	t.Parallel()
	for name, idVal := range map[string]any{
		"empty":      "",
		"whitespace": "   ",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := GitHubProviderFactory(map[string]any{
				"bot_reviewers": []any{
					map[string]any{"id": idVal, "login": "somebot[bot]", "required": true},
				},
			})
			if err == nil || !strings.Contains(err.Error(), ".id is required") {
				t.Fatalf("got %v", err)
			}
		})
	}
}

//nolint:gocyclo // integration-style stub server + fake gh setup
func TestGitHubProvider_Collect_explicitBranchNoPR_failClosedCI(t *testing.T) {
	// Given: explicit --branch scope with no matching open PR (prNum stays 0) and local HEAD sha present.
	// When:  Collect runs CI facet.
	// Then:  CI must not fall back to the local checkout SHA; signals report unknown + scoped_head_unresolved.
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
	localHead := strings.TrimSpace(string(shaBytes))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/issues":
			_, _ = w.Write([]byte(`{"total_count": 0}`))
		case r.URL.Path == "/repos/octocat/hello-world/pulls":
			_, _ = w.Write([]byte(`[]`))
		case strings.HasPrefix(r.URL.Path, "/repos/octocat/hello-world/commits/") && strings.HasSuffix(r.URL.Path, "/status"):
			t.Errorf("unexpected CI status fetch for local HEAD %q (fail-closed should skip CI)", r.URL.Path)
			http.Error(w, "unexpected", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	p := NewGitHubProvider()
	p.APIBase = srv.URL

	frag, err := p.Collect(context.Background(), Options{
		WorkDir: repoDir,
		Scope:   Scope{Branch: "no-matching-pr-branch"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !frag.Degraded {
		t.Fatalf("expected degraded, got %+v", frag.Diagnostics)
	}
	var foundScoped bool
	for _, d := range frag.Diagnostics {
		if d.Code == "scoped_head_unresolved" {
			foundScoped = true
			break
		}
	}
	if !foundScoped {
		t.Fatalf("missing scoped_head_unresolved: %+v", frag.Diagnostics)
	}
	gitHub := frag.Signals
	if gitHub == nil {
		t.Fatal("nil github signals")
	}
	ci, _ := gitHub["ci"].(map[string]any)
	if ci == nil {
		t.Fatalf("missing ci: %+v", gitHub)
	}
	if got := fmt.Sprint(ci["ci_status"]); got != "unknown" {
		t.Fatalf("ci_status: got %q", got)
	}
	if got := fmt.Sprint(ci["head_sha"]); got != "" {
		t.Fatalf("head_sha: got %q want empty (not local %s)", got, localHead)
	}
}

func TestGitHubProvider_Collect_scopedCISummaryUsesRESTHeadLookupOnly(t *testing.T) {
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

	var graphQLCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":18}]}`))
		case r.URL.Path == "/repos/octocat/hello-world/pulls/18":
			_, _ = w.Write([]byte(`{
				"number":18,
				"state":"open",
				"title":"feat: summary ci scope",
				"draft":false,
				"mergeable":true,
				"mergeable_state":"clean",
				"base":{"ref":"main"},
				"head":{
					"ref":"feature/scoped",
					"sha":"0123456789abcdef0123456789abcdef01234567",
					"repo":{"name":"hello-world","owner":{"login":"octocat"}}
				},
				"labels":[]
			}`))
		case r.URL.Path == "/repos/octocat/hello-world/commits/0123456789abcdef0123456789abcdef01234567/status":
			_, _ = w.Write([]byte(`{"state":"success"}`))
		case strings.Contains(r.URL.Path, "/check-runs"):
			http.Error(w, "unexpected check-runs", http.StatusInternalServerError)
			return
		case r.URL.Path == "/graphql":
			graphQLCalled.Store(true)
			http.Error(w, "unexpected graphql", http.StatusInternalServerError)
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	p := NewGitHubProvider()
	p.APIBase = srv.URL

	frag, err := p.Collect(context.Background(), Options{
		WorkDir:     repoDir,
		GitHubFacet: "ci",
		Scope:       Scope{PRNumber: 18},
		View:        ViewSummary,
	})
	if err != nil {
		t.Fatal(err)
	}
	if frag.Degraded {
		t.Fatalf("unexpected degraded fragment: %+v", frag)
	}
	if graphQLCalled.Load() {
		t.Fatal("summary CI scope must not use GraphQL")
	}
	ciMap, ok := frag.Signals["ci"].(map[string]any)
	if !ok {
		t.Fatalf("missing ci signals: %+v", frag.Signals)
	}
	if ciMap["head_sha"] != "0123456789abcdef0123456789abcdef01234567" || ciMap["ci_status"] != "success" {
		t.Fatalf("ci signals: %+v", ciMap)
	}
}

func TestHeadRepoForCIStatus(t *testing.T) {
	t.Parallel()
	t.Run("fork_head_repo", func(t *testing.T) {
		t.Parallel()
		o, n := headRepoForCIStatus(map[string]any{
			"head_repo_owner": "fork-owner",
			"head_repo_name":  "fork-repo",
		}, "base-owner", "base-repo")
		if o != "fork-owner" || n != "fork-repo" {
			t.Fatalf("got %q %q", o, n)
		}
	})
	t.Run("falls_back_to_base", func(t *testing.T) {
		t.Parallel()
		o, n := headRepoForCIStatus(map[string]any{}, "base-owner", "base-repo")
		if o != "base-owner" || n != "base-repo" {
			t.Fatalf("got %q %q", o, n)
		}
	})
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
