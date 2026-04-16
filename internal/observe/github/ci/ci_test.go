package ci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func ciTestHandlerEmptyCheckRuns(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "/check-runs") {
		_, _ = w.Write([]byte(`{"check_runs":[]}`))
		return
	}
	_, _ = w.Write([]byte(`{"state":"success"}`))
}

func TestCollect_status(t *testing.T) {
	t.Parallel()
	// Given: git repo with HEAD and API returning commit status success
	dir := t.TempDir()
	gitInit(t, dir)

	srv := httptest.NewServer(http.HandlerFunc(ciTestHandlerEmptyCheckRuns))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "", ViewFull)
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
	if _, ok := cimap["check_runs"].([]any); !ok {
		t.Fatalf("expected check_runs: %+v", cimap)
	}
}

func TestCollect_checkRunsMapping(t *testing.T) {
	t.Parallel()
	// Given: git repo with HEAD and API returning one completed check run
	dir := t.TempDir()
	gitInit(t, dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/check-runs") {
			_, _ = w.Write([]byte(`{"check_runs":[{"name":"ci","status":"completed","conclusion":"success"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"state":"success"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "", ViewFull)
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
	runs, ok := cimap["check_runs"].([]any)
	if !ok {
		t.Fatalf("expected check_runs slice, got %T", cimap["check_runs"])
	}
	if len(runs) != 1 {
		t.Fatalf("runs=%+v", runs)
	}
	rm, ok := runs[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map for run[0], got %T", runs[0])
	}
	// Then: check run fields are preserved
	if rm["name"] != "ci" || rm["status"] != "completed" || rm["conclusion"] != "success" {
		t.Fatalf("%+v", rm)
	}
}

func TestCollect_summaryOmitsCheckRuns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitInit(t, dir)
	var checkRunsRequested atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/check-runs") {
			checkRunsRequested.Store(true)
			http.Error(w, "unexpected check-runs", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"state":"success"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}

	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "", ViewSummary)
	if err != nil {
		t.Fatal(err)
	}
	if checkRunsRequested.Load() {
		t.Fatal("summary view must not fetch check-runs")
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatalf("expected ci map, got %T", m["ci"])
	}
	if _, exists := cimap["check_runs"]; exists {
		t.Fatalf("summary view must omit check_runs: %+v", cimap)
	}
}

func TestCollect_checkRunsFailureFallsBackToWarning(t *testing.T) {
	t.Parallel()
	// Given: check-runs endpoint errors but commit status succeeds
	dir := t.TempDir()
	gitInit(t, dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/check-runs") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"state":"success"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "", ViewFull)
	// Then: warning is recorded and check_runs is empty
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) == 0 {
		t.Fatal("expected warning when check-runs fetch fails")
	}
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatalf("expected ci map, got %T", m["ci"])
	}
	runs, ok := cimap["check_runs"].([]any)
	if !ok {
		t.Fatalf("expected check_runs slice, got %T", cimap["check_runs"])
	}
	if len(runs) != 0 {
		t.Fatalf("want empty check_runs, got %+v", runs)
	}
}

func TestCollect_checkRunsTruncationWarning(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitInit(t, dir)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/check-runs") {
			_, _ = w.Write([]byte(`{"total_count":2,"check_runs":[{"name":"ci","status":"completed","conclusion":"success"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"state":"success"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "", ViewFull)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "truncated") {
		t.Fatalf("want truncation warning, got warns=%v", warns)
	}
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatal("expected ci map")
	}
	runs := cimap["check_runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("runs=%+v", runs)
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
	_, _, err := Collect(context.Background(), c, "o", "r", dir, "", ViewFull)
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
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "", ViewFull)
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

func TestCollect_statusEndpointUsesOwnerRepoArguments(t *testing.T) {
	t.Parallel()
	// Given: a fork PR scenario — CI statuses live on the head repository
	dir := t.TempDir()
	gitInit(t, dir)
	var sawPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPaths = append(sawPaths, r.URL.Path)
		ciTestHandlerEmptyCheckRuns(w, r)
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	head := "abcd0123456789abcdef0123456789abcdef01"
	// When: Collect is called with head repo owner/name (not the base repo)
	m, warns, err := Collect(context.Background(), c, "fork-owner", "fork-repo", dir, head, ViewFull)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	// Then: status and check-runs requests target the head fork repo
	var sawStatus, sawCheckRuns bool
	for _, p := range sawPaths {
		if strings.Contains(p, "/repos/fork-owner/fork-repo/commits/"+head+"/status") {
			sawStatus = true
		}
		if strings.Contains(p, "/repos/fork-owner/fork-repo/commits/"+head+"/check-runs") {
			sawCheckRuns = true
		}
	}
	if !sawStatus || !sawCheckRuns {
		t.Fatalf("paths=%v", sawPaths)
	}
	cimap, ok := m["ci"].(map[string]any)
	if !ok {
		t.Fatalf("expected ci map, got %T", m["ci"])
	}
	if cimap["head_sha"] != head {
		t.Fatalf("ci=%+v", cimap)
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
		if strings.Contains(r.URL.Path, "/check-runs") {
			_, _ = w.Write([]byte(`{"check_runs":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"state":"pending"}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	override := "0123456789abcdef0123456789abcdef01234567"
	// When: Collect runs with the override
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, override, ViewFull)
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

func TestCollect_whitespaceHeadSHAOverrideFallsBackToHEAD(t *testing.T) {
	t.Parallel()
	// Given: a git repo with HEAD and a whitespace-only override (treated as empty)
	dir := t.TempDir()
	gitInit(t, dir)
	srv := httptest.NewServer(http.HandlerFunc(ciTestHandlerEmptyCheckRuns))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "t", BaseURL: srv.URL}
	// When: Collect runs with whitespace-only override
	m, warns, err := Collect(context.Background(), c, "o", "r", dir, "   ", ViewFull)
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
	head, err := headSHA(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if cimap["head_sha"] != head {
		t.Fatalf("want head_sha %q from git HEAD, got %+v", head, cimap)
	}
	// Then: CI signals use the local HEAD SHA, not a remote override
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
