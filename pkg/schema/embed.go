// Package schema exposes embedded JSON Schema assets for reinguard.
package schema

import (
	"embed"
	"io/fs"
)

//go:embed operational-context.placeholder.json
var embedded embed.FS

// Files returns the embedded JSON Schema filesystem (read-only).
func Files() fs.FS {
	return embedded
}

// OperationalContextPlaceholder is the embedded filename for the MVP placeholder.
const OperationalContextPlaceholder = "operational-context.placeholder.json"
