// Package gate records and evaluates runtime gate artifacts under .reinguard/runtime/gates.
package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/k-shibuki/reinguard/internal/gitroot"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

// Gate artifact and derived status values.
const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusSkipped = "skipped"

	StatusMissing = "missing"
	StatusInvalid = "invalid"
	StatusStale   = "stale"
)

var gateIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Check is one recorded verification check in a gate artifact.
type Check struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

// Artifact is the on-disk runtime gate document.
type Artifact struct {
	SchemaVersion string  `json:"schema_version"`
	GateID        string  `json:"gate_id"`
	Status        string  `json:"status"`
	HeadSHA       string  `json:"head_sha"`
	Branch        string  `json:"branch"`
	RecordedAt    string  `json:"recorded_at"`
	Checks        []Check `json:"checks"`
}

// StatusResult is the derived gate status for CLI output and signal injection.
type StatusResult struct {
	GateID     string `json:"gate_id"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	HeadSHA    string `json:"head_sha,omitempty"`
	Branch     string `json:"branch,omitempty"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

// RuntimeDir returns the runtime directory under the resolved config directory.
func RuntimeDir(cfgDir string) string {
	return filepath.Join(cfgDir, "runtime")
}

// GatesDir returns the runtime gate directory under the resolved config directory.
func GatesDir(cfgDir string) string {
	return filepath.Join(RuntimeDir(cfgDir), "gates")
}

// ValidateGateID rejects blank or unsafe gate identifiers.
func ValidateGateID(gateID string) error {
	gateID = strings.TrimSpace(gateID)
	if gateID == "" {
		return fmt.Errorf("gate: empty gate id")
	}
	if !gateIDPattern.MatchString(gateID) {
		return fmt.Errorf("gate: invalid gate id %q", gateID)
	}
	return nil
}

// ArtifactPath returns the on-disk path for one gate artifact.
func ArtifactPath(cfgDir, gateID string) (string, error) {
	if err := ValidateGateID(gateID); err != nil {
		return "", err
	}
	return filepath.Join(GatesDir(cfgDir), gateID+".json"), nil
}

// Record writes one validated gate artifact for the current branch HEAD.
func Record(ctx context.Context, cfgDir, wd, gateID, status string, checks []Check, now time.Time) (Artifact, error) {
	if err := ValidateGateID(gateID); err != nil {
		return Artifact{}, err
	}
	branch, detached, err := gitroot.CurrentBranch(ctx, wd)
	if err != nil {
		return Artifact{}, fmt.Errorf("gate: current branch: %w", err)
	}
	if detached {
		return Artifact{}, fmt.Errorf("gate: cannot record %q on detached HEAD", gateID)
	}
	headSHA, err := currentHeadSHA(ctx, wd)
	if err != nil {
		return Artifact{}, err
	}
	art, err := buildArtifact(gateID, status, checks, branch, headSHA, now)
	if err != nil {
		return Artifact{}, err
	}
	path, err := ArtifactPath(cfgDir, gateID)
	if err != nil {
		return Artifact{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Artifact{}, fmt.Errorf("gate: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := writeArtifactFile(path, art); err != nil {
		return Artifact{}, err
	}
	return art, nil
}

// Show reads and validates one gate artifact.
func Show(cfgDir, gateID string) (Artifact, error) {
	path, err := ArtifactPath(cfgDir, gateID)
	if err != nil {
		return Artifact{}, err
	}
	return readArtifactFile(path)
}

// Status evaluates one gate artifact into pass/fail/missing/invalid/stale.
func Status(ctx context.Context, cfgDir, wd, gateID string) (StatusResult, error) {
	if err := ValidateGateID(gateID); err != nil {
		return StatusResult{}, err
	}
	path, err := ArtifactPath(cfgDir, gateID)
	if err != nil {
		return StatusResult{}, err
	}
	art, err := readArtifactFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusResult{GateID: gateID, Status: StatusMissing, Reason: "artifact missing"}, nil
		}
		return StatusResult{GateID: gateID, Status: StatusInvalid, Reason: err.Error()}, nil
	}
	return deriveStatus(ctx, wd, art), nil
}

// LoadSignals loads all discovered gate statuses into a nested map suitable for signals.Flatten.
func LoadSignals(ctx context.Context, cfgDir, wd string) (map[string]any, error) {
	dir := GatesDir(cfgDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("gate: read %s: %w", dir, err)
	}
	gates := map[string]any{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		gateID := strings.TrimSuffix(entry.Name(), ".json")
		if err := ValidateGateID(gateID); err != nil {
			continue
		}
		res, err := Status(ctx, cfgDir, wd, gateID)
		if err != nil {
			return nil, err
		}
		gates[gateID] = statusMap(res)
	}
	if len(gates) == 0 {
		return map[string]any{}, nil
	}
	return map[string]any{"gates": gates}, nil
}

func buildArtifact(gateID, status string, checks []Check, branch, headSHA string, now time.Time) (Artifact, error) {
	if err := validateArtifactStatus(status); err != nil {
		return Artifact{}, err
	}
	if strings.TrimSpace(branch) == "" {
		return Artifact{}, fmt.Errorf("gate: empty branch")
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(headSHA) {
		return Artifact{}, fmt.Errorf("gate: invalid head_sha %q", headSHA)
	}
	normalized := make([]Check, 0, len(checks))
	for i, chk := range checks {
		if err := validateCheck(chk, i); err != nil {
			return Artifact{}, err
		}
		normalized = append(normalized, Check{
			ID:      strings.TrimSpace(chk.ID),
			Status:  strings.TrimSpace(chk.Status),
			Summary: strings.TrimSpace(chk.Summary),
		})
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	art := Artifact{
		SchemaVersion: schema.CurrentSchemaVersion,
		GateID:        gateID,
		Status:        status,
		HeadSHA:       headSHA,
		Branch:        branch,
		RecordedAt:    now.UTC().Format(time.RFC3339),
		Checks:        normalized,
	}
	if err := validateArtifact(art); err != nil {
		return Artifact{}, err
	}
	return art, nil
}

func validateCheck(chk Check, idx int) error {
	if strings.TrimSpace(chk.ID) == "" {
		return fmt.Errorf("gate: checks[%d].id must be non-empty", idx)
	}
	switch strings.TrimSpace(chk.Status) {
	case StatusPass, StatusFail, StatusSkipped:
		return nil
	default:
		return fmt.Errorf("gate: checks[%d].status %q is invalid", idx, chk.Status)
	}
}

func validateArtifactStatus(status string) error {
	switch strings.TrimSpace(status) {
	case StatusPass, StatusFail:
		return nil
	default:
		return fmt.Errorf("gate: status %q must be %q or %q", status, StatusPass, StatusFail)
	}
}

func writeArtifactFile(path string, art Artifact) error {
	data, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		return fmt.Errorf("gate: marshal artifact: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gate: write %s: %w", path, err)
	}
	return nil
}

func readArtifactFile(path string) (Artifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Artifact{}, err
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return Artifact{}, fmt.Errorf("gate: parse %s: %w", path, err)
	}
	if err := validateArtifactDoc(doc, path); err != nil {
		return Artifact{}, err
	}
	var art Artifact
	if err := json.Unmarshal(data, &art); err != nil {
		return Artifact{}, fmt.Errorf("gate: decode %s: %w", path, err)
	}
	if art.Checks == nil {
		art.Checks = []Check{}
	}
	return art, nil
}

func validateArtifact(art Artifact) error {
	docBytes, err := json.Marshal(art)
	if err != nil {
		return fmt.Errorf("gate: marshal validation doc: %w", err)
	}
	var doc any
	if err := json.Unmarshal(docBytes, &doc); err != nil {
		return fmt.Errorf("gate: decode validation doc: %w", err)
	}
	return validateArtifactDoc(doc, art.GateID)
}

func validateArtifactDoc(doc any, pathHint string) error {
	sch, err := compileGateSchema()
	if err != nil {
		return err
	}
	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("gate: schema validation %s: %w", pathHint, err)
	}
	return nil
}

func compileGateSchema() (*jsonschema.Schema, error) {
	comp, err := schema.NewCompiler()
	if err != nil {
		return nil, fmt.Errorf("gate: new compiler: %w", err)
	}
	sch, err := comp.Compile(schema.URIGateArtifact)
	if err != nil {
		return nil, fmt.Errorf("gate: compile schema: %w", err)
	}
	return sch, nil
}

func deriveStatus(ctx context.Context, wd string, art Artifact) StatusResult {
	res := StatusResult{
		GateID:     art.GateID,
		Status:     art.Status,
		HeadSHA:    art.HeadSHA,
		Branch:     art.Branch,
		RecordedAt: art.RecordedAt,
	}
	branch, detached, err := gitroot.CurrentBranch(ctx, wd)
	if err != nil {
		res.Status = StatusStale
		res.Reason = fmt.Sprintf("cannot determine current branch: %v", err)
		return res
	}
	headSHA, err := currentHeadSHA(ctx, wd)
	if err != nil {
		res.Status = StatusStale
		res.Reason = err.Error()
		return res
	}
	if detached {
		res.Status = StatusStale
		res.Reason = "current HEAD is detached"
		return res
	}
	if art.Branch != branch {
		res.Status = StatusStale
		res.Reason = fmt.Sprintf("artifact branch %q != current branch %q", art.Branch, branch)
		return res
	}
	if art.HeadSHA != headSHA {
		res.Status = StatusStale
		res.Reason = fmt.Sprintf("artifact head_sha %q != current HEAD %q", art.HeadSHA, headSHA)
		return res
	}
	return res
}

func currentHeadSHA(ctx context.Context, wd string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	if wd != "" {
		cmd.Dir = wd
	}
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gate: git rev-parse HEAD: %w: %s", err, strings.TrimSpace(buf.String()))
	}
	return strings.TrimSpace(buf.String()), nil
}

func statusMap(res StatusResult) map[string]any {
	out := map[string]any{"status": res.Status}
	if res.Reason != "" {
		out["reason"] = res.Reason
	}
	if res.HeadSHA != "" {
		out["head_sha"] = res.HeadSHA
	}
	if res.Branch != "" {
		out["branch"] = res.Branch
	}
	if res.RecordedAt != "" {
		out["recorded_at"] = res.RecordedAt
	}
	return out
}
