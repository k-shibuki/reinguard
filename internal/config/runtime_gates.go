package config

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

// Names of workflow.runtime_gate_roles keys and pr_readiness.pass_requires_roles entries.
const (
	RuntimeGateRoleLocalVerification = "local_verification"
	RuntimeGateRolePrePRAIReview     = "pre_pr_ai_review"
	RuntimeGateRolePRReadiness       = "pr_readiness"
)

var validPRReadinessPassRequiresRoles = []string{
	RuntimeGateRoleLocalVerification,
	RuntimeGateRolePrePRAIReview,
}

// DefaultRuntimeGateRoles returns the repo-default runtime gate role contract.
func DefaultRuntimeGateRoles() RuntimeGateRolesSpec {
	required := true
	return RuntimeGateRolesSpec{
		LocalVerification: RuntimeGateRoleSpec{
			GateID:             "local-verification",
			ProducerProcedures: []string{"implement", "change-inspect"},
		},
		PrePRAIReview: RuntimeGateRoleSpec{
			GateID:             "local-coderabbit",
			ProducerProcedures: []string{"change-inspect"},
			PassCheckIDs:       []string{"local-coderabbit-cli"},
			Required:           &required,
		},
		PRReadiness: RuntimeGateRoleSpec{
			GateID:             "pr-readiness",
			ProducerProcedures: []string{"change-inspect"},
			PassCheckIDs:       []string{"review-closure"},
			PassRequiresRoles:  prReadinessDefaultPassRequiresRoles(),
		},
	}
}

// EffectiveRuntimeGateRoles merges repo config with the default runtime gate contract.
func (r *Root) EffectiveRuntimeGateRoles() RuntimeGateRolesSpec {
	base := DefaultRuntimeGateRoles()
	if r == nil {
		return base
	}
	base.LocalVerification = mergeRuntimeGateRole(base.LocalVerification, r.Workflow.RuntimeGateRoles.LocalVerification)
	base.PrePRAIReview = mergeRuntimeGateRole(base.PrePRAIReview, r.Workflow.RuntimeGateRoles.PrePRAIReview)
	base.PRReadiness = mergeRuntimeGateRole(base.PRReadiness, r.Workflow.RuntimeGateRoles.PRReadiness)
	return base
}

func mergeRuntimeGateRole(base, override RuntimeGateRoleSpec) RuntimeGateRoleSpec {
	if strings.TrimSpace(override.GateID) != "" {
		base.GateID = strings.TrimSpace(override.GateID)
	}
	if trimmed := cloneTrimmedNonEmpty(override.ProducerProcedures); len(trimmed) > 0 {
		base.ProducerProcedures = trimmed
	}
	if trimmed := cloneTrimmedNonEmpty(override.PassCheckIDs); len(trimmed) > 0 {
		base.PassCheckIDs = trimmed
	}
	if override.PassRequiresRoles != nil {
		trimmed := cloneTrimmedNonEmpty(*override.PassRequiresRoles)
		base.PassRequiresRoles = &trimmed
	}
	if override.Required != nil {
		base.Required = boolPtr(*override.Required)
	}
	return base
}

func prReadinessDefaultPassRequiresRoles() *[]string {
	s := []string{RuntimeGateRoleLocalVerification, RuntimeGateRolePrePRAIReview}
	return &s
}

// DerefPassRequiresRoles returns the slice for a pass_requires_roles pointer field.
func DerefPassRequiresRoles(p *[]string) []string {
	if p == nil {
		return nil
	}
	return *p
}

func boolPtr(v bool) *bool {
	return &v
}

func cloneTrimmedNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// LoadRuntimeGateRoles returns the effective runtime gate contract for one config dir.
// When reinguard.yaml is absent, default roles are returned so isolated gate-only tests
// and legacy callers remain compatible.
func LoadRuntimeGateRoles(dir string) (RuntimeGateRolesSpec, error) {
	root, err := LoadRoot(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return DefaultRuntimeGateRoles(), nil
		}
		return RuntimeGateRolesSpec{}, err
	}
	return root.EffectiveRuntimeGateRoles(), nil
}

func validateRuntimeGateRoles(root *Root, pathHint string) error {
	roles := root.EffectiveRuntimeGateRoles()
	roleMap := map[string]RuntimeGateRoleSpec{
		RuntimeGateRoleLocalVerification: roles.LocalVerification,
		RuntimeGateRolePrePRAIReview:     roles.PrePRAIReview,
		RuntimeGateRolePRReadiness:       roles.PRReadiness,
	}

	seenGateIDs := map[string]string{}
	roleOrder := []string{
		RuntimeGateRoleLocalVerification,
		RuntimeGateRolePrePRAIReview,
		RuntimeGateRolePRReadiness,
	}
	for _, roleName := range roleOrder {
		role := roleMap[roleName]
		if strings.TrimSpace(role.GateID) == "" {
			return fmt.Errorf("config: workflow.runtime_gate_roles.%s.gate_id must be non-empty in %s", roleName, pathHint)
		}
		if prev, ok := seenGateIDs[role.GateID]; ok {
			return fmt.Errorf("config: workflow.runtime_gate_roles.%s.gate_id %q duplicates %s in %s", roleName, role.GateID, prev, pathHint)
		}
		seenGateIDs[role.GateID] = roleName
		for i, proc := range role.ProducerProcedures {
			if strings.TrimSpace(proc) == "" {
				return fmt.Errorf("config: workflow.runtime_gate_roles.%s.producer_procedures[%d] must be non-empty in %s", roleName, i, pathHint)
			}
		}
		for i, checkID := range role.PassCheckIDs {
			if strings.TrimSpace(checkID) == "" {
				return fmt.Errorf("config: workflow.runtime_gate_roles.%s.pass_check_ids[%d] must be non-empty in %s", roleName, i, pathHint)
			}
		}
	}

	for i, roleName := range DerefPassRequiresRoles(roles.PRReadiness.PassRequiresRoles) {
		if !slices.Contains(validPRReadinessPassRequiresRoles, roleName) {
			return fmt.Errorf("config: workflow.runtime_gate_roles.pr_readiness.pass_requires_roles[%d] %q is invalid in %s", i, roleName, pathHint)
		}
	}
	return nil
}
