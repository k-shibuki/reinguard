package schema

import "testing"

// TestNewCompiler compiles every embedded schema URI so JSON Schema assets stay loadable.
func TestNewCompiler(t *testing.T) {
	t.Parallel()
	// Given/When: compiler loads embedded schemas
	c, err := NewCompiler()
	if err != nil {
		t.Fatal(err)
	}
	// Then: every embedded resource URI compiles (table-driven; no schema left untested)
	tests := []struct {
		name string
		uri  string
	}{
		{name: "reinguard config", uri: URIReinguardConfig},
		{name: "rules document", uri: URIRulesDocument},
		{name: "knowledge manifest", uri: URIKnowledgeManifest},
		{name: "observation document", uri: URIObservationDocument},
		{name: "operational context", uri: URIOperationalContext},
		{name: "labels config", uri: URILabelsConfig},
		{name: "gate artifact", uri: URIGateArtifact},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if _, err := c.Compile(tc.uri); err != nil {
				t.Fatalf("compile %s (%s): %v", tc.name, tc.uri, err)
			}
		})
	}
}

func TestOperationalContextSchema_procedureHintResolvedOnly(t *testing.T) {
	t.Parallel()
	c, err := NewCompiler()
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Compile(URIOperationalContext)
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]any{
		"schema_version": CurrentSchemaVersion,
		"state": map[string]any{
			"kind":     "resolved",
			"state_id": "working_no_pr",
			"procedure_hint": map[string]any{
				"procedure_id": "procedure-implement",
				"path":         ".reinguard/procedure/implement.md",
				"derived_from": "state_id",
			},
		},
	}
	if err := s.Validate(valid); err != nil {
		t.Fatalf("valid context rejected: %v", err)
	}
	invalid := map[string]any{
		"schema_version": CurrentSchemaVersion,
		"state": map[string]any{
			"kind":     "ambiguous",
			"state_id": "working_no_pr",
			"procedure_hint": map[string]any{
				"procedure_id": "procedure-implement",
				"path":         ".reinguard/procedure/implement.md",
				"derived_from": "state_id",
			},
		},
	}
	if err := s.Validate(invalid); err == nil {
		t.Fatal("expected schema validation error for procedure_hint on non-resolved state")
	}
}
