// Package schema exposes embedded JSON Schema assets for reinguard.
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
	CurrentSchemaVersion = "0.5.0"
)

// Files returns the embedded JSON Schema filesystem (read-only).
func Files() fs.FS {
	return embedded
}
