// Package schema exposes embedded JSON Schema documents and helpers to load them into a
// jsonschema Compiler (see NewCompiler in compiler.go). Schema versions and $id URIs
// follow ADR-0008; use Files for direct read-only access to the embedded FS.
package schema

import (
	"embed"
	"io/fs"
)

//go:embed *.json
var embedded embed.FS

// Embedded schema file names (stable CLI / export basenames).
const (
	ReinguardConfig      = "reinguard-config.json"
	RulesDocument        = "rules-document.json"
	KnowledgeManifest    = "knowledge-manifest.json"
	ObservationDocument  = "observation-document.json"
	OperationalContext   = "operational-context.json"
	LabelsConfig         = "labels-config.json"
	GateArtifact         = "gate-artifact.json"
	CurrentSchemaVersion = "0.6.0"
)

// Files returns the embedded JSON Schema filesystem (read-only).
func Files() fs.FS {
	return embedded
}
