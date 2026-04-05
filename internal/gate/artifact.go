// Package gate records and evaluates runtime gate artifacts under .reinguard/local/gates.
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
	"slices"
	"strings"
	"sync"
	"time"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/k-shibuki/reinguard/internal/config"
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

var (
	gateIDPattern  = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	headSHAPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

	gateSchemaOnce sync.Once
	gateSchema     *jsonschema.Schema
	gateSchemaErr  error
)

// Check is one recorded verification check in a gate artifact.
type Check struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Summary  string `json:"summary"`
	Evidence string `json:"evidence,omitempty"`
}

// Subject identifies the branch head a proof applies to.
type Subject struct {
	HeadSHA string `json:"head_sha"`
	Branch  string `json:"branch"`
}

// Producer identifies which procedure and tool recorded a gate artifact.
type Producer struct {
	Procedure string `json:"procedure"`
	Tool      string `json:"tool"`
}

// Input is one upstream proof consumed by another gate artifact.
type Input struct {
	GateID     string  `json:"gate_id"`
	Status     string  `json:"status"`
	Subject    Subject `json:"subject"`
	RecordedAt string  `json:"recorded_at"`
}

// Artifact is the on-disk runtime gate document.
type Artifact struct {
	SchemaVersion string   `json:"schema_version"`
	GateID        string   `json:"gate_id"`
	Status        string   `json:"status"`
	RecordedAt    string   `json:"recorded_at"`
	Subject       Subject  `json:"subject"`
	Producer      Producer `json:"producer"`
	Inputs        []Input  `json:"inputs"`
	Checks        []Check  `json:"checks"`
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

// LocalDir returns the local operational state directory under the resolved config directory.
func LocalDir(cfgDir string) string {
	return filepath.Join(cfgDir, "local")
}

// GatesDir returns the gate artifacts directory under the resolved config directory.
func GatesDir(cfgDir string) string {
	return filepath.Join(LocalDir(cfgDir), "gates")
}

// ValidateGateID rejects blank or unsafe gate identifiers.
func ValidateGateID(gateID string) error {
	trimmed := strings.TrimSpace(gateID)
	if trimmed == "" {
		return fmt.Errorf("gate: empty gate id")
	}
	if gateID != trimmed || !gateIDPattern.MatchString(trimmed) {
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
func Record(ctx context.Context, cfgDir, wd, gateID, status string, producer Producer, inputs []Input, checks []Check, now time.Time) (Artifact, error) {
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
	roles, err := config.LoadRuntimeGateRoles(cfgDir)
	if err != nil {
		return Artifact{}, fmt.Errorf("gate: load runtime roles: %w", err)
	}
	art, err := buildArtifact(gateID, status, producer, inputs, checks, branch, headSHA, now, roles)
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
	roles, err := config.LoadRuntimeGateRoles(cfgDir)
	if err != nil {
		return Artifact{}, fmt.Errorf("gate: load runtime roles: %w", err)
	}
	return readArtifactFile(path, gateID, roles)
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
	roles, err := config.LoadRuntimeGateRoles(cfgDir)
	if err != nil {
		return StatusResult{}, fmt.Errorf("gate: load runtime roles: %w", err)
	}
	art, err := readArtifactFile(path, gateID, roles)
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

func buildArtifact(gateID, status string, producer Producer, inputs []Input, checks []Check, branch, headSHA string, now time.Time, roles config.RuntimeGateRolesSpec) (Artifact, error) {
	status = strings.TrimSpace(status)
	if err := validateArtifactStatus(status); err != nil {
		return Artifact{}, err
	}
	subject := Subject{HeadSHA: headSHA, Branch: branch}
	if err := validateSubject(subject, "subject"); err != nil {
		return Artifact{}, err
	}
	if err := validateProducer(producer); err != nil {
		return Artifact{}, err
	}
	normalizedInputs := make([]Input, 0, len(inputs))
	for i, input := range inputs {
		if err := validateInput(input, i); err != nil {
			return Artifact{}, err
		}
		normalizedInputs = append(normalizedInputs, Input{
			GateID:     strings.TrimSpace(input.GateID),
			Status:     strings.TrimSpace(input.Status),
			Subject:    Subject{HeadSHA: input.Subject.HeadSHA, Branch: strings.TrimSpace(input.Subject.Branch)},
			RecordedAt: strings.TrimSpace(input.RecordedAt),
		})
	}
	normalized := make([]Check, 0, len(checks))
	for i, chk := range checks {
		if err := validateCheck(chk, i); err != nil {
			return Artifact{}, err
		}
		normalized = append(normalized, Check{
			ID:       strings.TrimSpace(chk.ID),
			Status:   strings.TrimSpace(chk.Status),
			Summary:  strings.TrimSpace(chk.Summary),
			Evidence: strings.TrimSpace(chk.Evidence),
		})
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	art := Artifact{
		SchemaVersion: schema.CurrentSchemaVersion,
		GateID:        gateID,
		Status:        status,
		RecordedAt:    now.UTC().Format(time.RFC3339),
		Subject:       subject,
		Producer: Producer{
			Procedure: strings.TrimSpace(producer.Procedure),
			Tool:      strings.TrimSpace(producer.Tool),
		},
		Inputs: normalizedInputs,
		Checks: normalized,
	}
	if err := validateGateContract(art, roles); err != nil {
		return Artifact{}, err
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
	if strings.TrimSpace(chk.Summary) == "" {
		return fmt.Errorf("gate: checks[%d].summary must be non-empty", idx)
	}
	switch strings.TrimSpace(chk.Status) {
	case StatusPass, StatusFail, StatusSkipped:
		return nil
	default:
		return fmt.Errorf("gate: checks[%d].status %q is invalid", idx, chk.Status)
	}
}

func validateSubject(subject Subject, field string) error {
	if strings.TrimSpace(subject.Branch) == "" {
		return fmt.Errorf("gate: %s.branch must be non-empty", field)
	}
	if !headSHAPattern.MatchString(subject.HeadSHA) {
		return fmt.Errorf("gate: invalid %s.head_sha %q", field, subject.HeadSHA)
	}
	return nil
}

func validateProducer(producer Producer) error {
	if strings.TrimSpace(producer.Procedure) == "" {
		return fmt.Errorf("gate: producer.procedure must be non-empty")
	}
	if strings.TrimSpace(producer.Tool) == "" {
		return fmt.Errorf("gate: producer.tool must be non-empty")
	}
	return nil
}

func validateInput(input Input, idx int) error {
	if err := ValidateGateID(input.GateID); err != nil {
		return fmt.Errorf("gate: inputs[%d]: %w", idx, err)
	}
	if err := validateArtifactStatus(input.Status); err != nil {
		return fmt.Errorf("gate: inputs[%d]: %w", idx, err)
	}
	if err := validateSubject(input.Subject, fmt.Sprintf("inputs[%d].subject", idx)); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(input.RecordedAt)); err != nil {
		return fmt.Errorf("gate: inputs[%d].recorded_at must be RFC3339: %w", idx, err)
	}
	return nil
}

func validateGateContract(art Artifact, roles config.RuntimeGateRolesSpec) error {
	if len(art.Checks) == 0 {
		return fmt.Errorf("gate: %s requires at least one check entry", art.GateID)
	}
	roleName, role, ok := runtimeGateRoleByGateID(roles, art.GateID)
	if !ok {
		return nil
	}
	if err := validateConfiguredGateRole(roleName, role, art); err != nil {
		return err
	}
	if roleName == config.RuntimeGateRolePRReadiness {
		return validateConfiguredPRReadinessGate(art, roles)
	}
	return nil
}

func validateConfiguredGateRole(roleName string, role config.RuntimeGateRoleSpec, art Artifact) error {
	if len(role.ProducerProcedures) > 0 && !slices.Contains(role.ProducerProcedures, art.Producer.Procedure) {
		return fmt.Errorf("gate: %s producer.procedure %q must be one of %q", art.GateID, art.Producer.Procedure, role.ProducerProcedures)
	}
	if roleName != config.RuntimeGateRolePRReadiness && len(art.Inputs) != 0 {
		return fmt.Errorf("gate: %s must not declare upstream inputs", art.GateID)
	}
	for _, checkID := range role.PassCheckIDs {
		if !hasCheck(art.Checks, checkID) {
			return fmt.Errorf("gate: %s requires checks[].id == %q", art.GateID, checkID)
		}
		if art.Status == StatusPass && !hasPassingCheck(art.Checks, checkID) {
			return fmt.Errorf("gate: %s pass proof requires checks[].id == %q with status %q", art.GateID, checkID, StatusPass)
		}
	}
	return nil
}

func validateConfiguredPRReadinessGate(art Artifact, roles config.RuntimeGateRolesSpec) error {
	if art.Status != StatusPass {
		return nil
	}
	requiredInputs := map[string]bool{}
	for _, roleName := range roles.PRReadiness.PassRequiresRoles {
		role := runtimeGateRoleByName(roles, roleName)
		if strings.TrimSpace(role.GateID) == "" {
			continue
		}
		if !runtimeGateRoleRequiredForPRReadiness(roleName, role) {
			continue
		}
		requiredInputs[role.GateID] = false
	}
	for _, input := range art.Inputs {
		if input.Status != StatusPass {
			continue
		}
		if !subjectsEqual(input.Subject, art.Subject) {
			return fmt.Errorf("gate: %s input %q subject must match artifact subject", art.GateID, input.GateID)
		}
		if _, ok := requiredInputs[input.GateID]; ok {
			requiredInputs[input.GateID] = true
		}
	}
	var missing []string
	for gateID, ok := range requiredInputs {
		if !ok {
			missing = append(missing, gateID)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return fmt.Errorf("gate: %s requires passing input proof for %v on the same subject", art.GateID, missing)
	}
	return nil
}

func runtimeGateRoleRequiredForPRReadiness(roleName string, role config.RuntimeGateRoleSpec) bool {
	if roleName != config.RuntimeGateRolePrePRAIReview {
		return true
	}
	if role.Required == nil {
		return true
	}
	return *role.Required
}

func runtimeGateRoleByGateID(roles config.RuntimeGateRolesSpec, gateID string) (string, config.RuntimeGateRoleSpec, bool) {
	for _, roleName := range []string{
		config.RuntimeGateRoleLocalVerification,
		config.RuntimeGateRolePrePRAIReview,
		config.RuntimeGateRolePRReadiness,
	} {
		role := runtimeGateRoleByName(roles, roleName)
		if role.GateID == gateID {
			return roleName, role, true
		}
	}
	return "", config.RuntimeGateRoleSpec{}, false
}

func runtimeGateRoleByName(roles config.RuntimeGateRolesSpec, roleName string) config.RuntimeGateRoleSpec {
	switch roleName {
	case config.RuntimeGateRoleLocalVerification:
		return roles.LocalVerification
	case config.RuntimeGateRolePrePRAIReview:
		return roles.PrePRAIReview
	case config.RuntimeGateRolePRReadiness:
		return roles.PRReadiness
	default:
		return config.RuntimeGateRoleSpec{}
	}
}

func hasCheck(checks []Check, id string) bool {
	for _, chk := range checks {
		if chk.ID == id {
			return true
		}
	}
	return false
}

func hasPassingCheck(checks []Check, id string) bool {
	for _, chk := range checks {
		if chk.ID == id && chk.Status == StatusPass {
			return true
		}
	}
	return false
}

func subjectsEqual(a, b Subject) bool {
	return a.HeadSHA == b.HeadSHA && strings.TrimSpace(a.Branch) == strings.TrimSpace(b.Branch)
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
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gate-artifact-*.json")
	if err != nil {
		return fmt.Errorf("gate: temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gate: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("gate: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("gate: close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("gate: atomic replace %s: %w", path, err)
	}
	return nil
}

func readArtifactFile(path, expectedGateID string, roles config.RuntimeGateRolesSpec) (Artifact, error) {
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
	if art.GateID != expectedGateID {
		return Artifact{}, fmt.Errorf("gate: artifact %s declares gate_id %q; want %q", path, art.GateID, expectedGateID)
	}
	if art.Inputs == nil {
		art.Inputs = []Input{}
	}
	if art.Checks == nil {
		art.Checks = []Check{}
	}
	if err := validateGateContract(art, roles); err != nil {
		return Artifact{}, err
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
	gateSchemaOnce.Do(func() {
		comp, err := schema.NewCompiler()
		if err != nil {
			gateSchemaErr = fmt.Errorf("gate: new compiler: %w", err)
			return
		}
		gateSchema, gateSchemaErr = comp.Compile(schema.URIGateArtifact)
		if gateSchemaErr != nil {
			gateSchemaErr = fmt.Errorf("gate: compile schema: %w", gateSchemaErr)
		}
	})
	return gateSchema, gateSchemaErr
}

func deriveStatus(ctx context.Context, wd string, art Artifact) StatusResult {
	res := StatusResult{
		GateID:     art.GateID,
		Status:     art.Status,
		HeadSHA:    art.Subject.HeadSHA,
		Branch:     art.Subject.Branch,
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
	if art.Subject.Branch != branch {
		res.Status = StatusStale
		res.Reason = fmt.Sprintf("artifact branch %q != current branch %q", art.Subject.Branch, branch)
		return res
	}
	if art.Subject.HeadSHA != headSHA {
		res.Status = StatusStale
		res.Reason = fmt.Sprintf("artifact head_sha %q != current HEAD %q", art.Subject.HeadSHA, headSHA)
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
