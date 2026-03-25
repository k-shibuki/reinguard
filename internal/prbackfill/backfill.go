// Package prbackfill updates open GitHub PR bodies and labels to satisfy pr-policy.yaml.
// It shells out to the gh CLI.
package prbackfill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/k-shibuki/reinguard/internal/labels"
)

type pull struct {
	Body   *string     `json:"body"`
	Title  string      `json:"title"`
	Labels []pullLabel `json:"labels"`
	Number int         `json:"number"`
}

type pullLabel struct {
	Name string `json:"name"`
}

var (
	titleTypeRe  = regexp.MustCompile(`^(feat|fix|refactor|perf|test|docs|build|ci|chore|style|revert)(\([^)]+\))?!?:`)
	closesLineRe = regexp.MustCompile(`(?i)(closes|fixes|resolves)\s+#\d+`)
)

// Run fetches open PRs via gh, patches bodies and labels as needed. Writes progress to w.
func Run(w io.Writer) error {
	raw, err := ghOutput("api", "repos/{owner}/{repo}/pulls",
		"-f", "state=open", "-f", "per_page=100", "--paginate")
	if err != nil {
		return err
	}
	pulls, err := parseOpenPullPages(raw)
	if err != nil {
		return err
	}

	for _, pr := range pulls {
		body := ""
		if pr.Body != nil {
			body = *pr.Body
		}
		labelNames := make(map[string]struct{}, len(pr.Labels))
		for _, lb := range pr.Labels {
			labelNames[lb.Name] = struct{}{}
		}

		newBody := ensureSections(body)
		if newBody != body {
			if err := patchPRBody(pr.Number, newBody); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "PR #%d: updated body (policy sections)\n", pr.Number)
		}

		if err := syncTypeLabelFromTitle(w, pr.Number, pr.Title, labelNames); err != nil {
			return err
		}
	}
	return nil
}

// syncTypeLabelFromTitle sets the single policy type label from the PR title when inferrable.
// It replaces all type labels with the inferred one and preserves non-type labels.
// Multiple type labels with no inferrable title only logs a warning.
func syncTypeLabelFromTitle(w io.Writer, num int, title string, present map[string]struct{}) error {
	typeHits := typeLabelNames(present)
	want := prTypeFromTitle(title)

	if want == "" {
		if len(typeHits) > 1 {
			slices.Sort(typeHits)
			_, _ = fmt.Fprintf(w, "PR #%d: multiple type labels %v; title has no inferrable type — manual fix required\n",
				num, typeHits)
		}
		return nil
	}
	if _, ok := labels.TypeLabels[want]; !ok {
		return nil
	}

	desired := desiredLabelsWithInferredType(present, want)
	if mapStringSetEqual(present, desired) {
		return nil
	}
	names := sortedStringSetKeys(desired)
	if err := putIssueLabels(num, names); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "PR #%d: normalized type labels -> %s\n", num, want)
	return nil
}

func typeLabelNames(present map[string]struct{}) []string {
	var hits []string
	for name := range present {
		if _, ok := labels.TypeLabels[name]; ok {
			hits = append(hits, name)
		}
	}
	return hits
}

// desiredLabelsWithInferredType keeps all non-type labels and exactly one type label (want).
func desiredLabelsWithInferredType(present map[string]struct{}, want string) map[string]struct{} {
	out := make(map[string]struct{})
	for name := range present {
		if _, ok := labels.TypeLabels[name]; ok {
			continue
		}
		out[name] = struct{}{}
	}
	out[want] = struct{}{}
	return out
}

func mapStringSetEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func sortedStringSetKeys(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	slices.Sort(s)
	return s
}

func ghOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh %v: %w\n%s", args, err, stderr.String())
	}
	return out, nil
}

func parseOpenPullPages(raw []byte) ([]pull, error) {
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimSpace(raw)))
	var merged []pull
	for {
		var batch []pull
		err := dec.Decode(&batch)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode pulls page: %w", err)
		}
		merged = append(merged, batch...)
	}
	return merged, nil
}

func hasHeading(body, title string) bool {
	esc := regexp.QuoteMeta(title)
	re := regexp.MustCompile(`(?im)^##\s+` + esc + `\s*$`)
	return re.MatchString(body)
}

func patchPRBody(num int, newBody string) error {
	payload, err := json.Marshal(map[string]string{"body": newBody})
	if err != nil {
		return err
	}
	cmd := exec.Command("gh", "api", "-X", "PATCH",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d", num), "--input", "-")
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh patch PR #%d: %w\n%s", num, err, stderr.String())
	}
	return nil
}

// putIssueLabels replaces all labels on the issue (GitHub PUT semantics).
func putIssueLabels(num int, names []string) error {
	payload, err := json.Marshal(map[string][]string{"labels": names})
	if err != nil {
		return err
	}
	cmd := exec.Command("gh", "api", "-X", "PUT",
		fmt.Sprintf("repos/{owner}/{repo}/issues/%d/labels", num), "--input", "-")
	cmd.Stdin = bytes.NewReader(payload)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh put labels PR #%d: %w\n%s", num, err, stderr.String())
	}
	return nil
}

func prTypeFromTitle(title string) string {
	m := titleTypeRe.FindStringSubmatch(title)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func extractClosesLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if closesLineRe.MatchString(strings.TrimSpace(line)) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func ensureSections(body string) string {
	b := strings.ReplaceAll(body, "\r\n", "\n")
	b = strings.TrimRightFunc(b, unicode.IsSpace)
	var additions []string

	if !hasHeading(b, "Traceability") {
		if close := extractClosesLine(b); close != "" {
			additions = append(additions, "## Traceability\n\n"+close+"\n")
		} else {
			additions = append(additions,
				"## Traceability\n\nCloses # (fill in — see Linked issues above if present)\n")
		}
	}

	if !hasHeading(b, "Risk / Impact") {
		additions = append(additions,
			"## Risk / Impact\n\n"+
				"- Affected area: see Summary.\n"+
				"- Breaking change: no (update if yes).\n")
	}

	if !hasHeading(b, "Rollback Plan") {
		additions = append(additions, "## Rollback Plan\n\nRevert this PR on main.\n")
	}

	if !hasHeading(b, "Definition of Done") && !hasHeading(b, "Acceptance Criteria") {
		additions = append(additions,
			"## Definition of Done\n\n"+
				"- [ ] (fill in verifiable criteria; see Issue if linked)\n")
	}

	if len(additions) == 0 {
		return body
	}
	sep := "\n\n"
	if b == "" {
		sep = ""
	}
	return b + sep + strings.Join(additions, "\n\n")
}
