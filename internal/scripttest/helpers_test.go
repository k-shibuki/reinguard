package scripttest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func scriptPath(t *testing.T, script string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), ".reinguard", "scripts", script)
}

func mustMikefarahYq(t *testing.T, root string) {
	t.Helper()
	binDir := filepath.Join(root, ".reinguard", "scripts", ".bin")
	cached := filepath.Join(binDir, "yq")
	if fi, err := os.Stat(cached); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	p, err := exec.LookPath("yq")
	if err != nil || p == "" {
		t.Skip("yq not in PATH (install mikefarah/yq v4 or run .reinguard/scripts/sync-issue-templates.sh once; see CONTRIBUTING.md)")
	}
	out, err := exec.Command("yq", "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "mikefarah") {
		t.Skip("mikefarah yq required for script integration tests (see CONTRIBUTING.md)")
	}
}

func writeTempFile(t *testing.T, dir, pattern, contents string) string {
	t.Helper()
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if _, err := f.WriteString(contents); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func writeExecutable(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func runBashScript(t *testing.T, dir, script string, extraEnv []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
