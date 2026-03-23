// Package validate performs stub validation of .reinguard configuration.
package validate

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const requiredFile = "reinguard.yaml"

// Dir checks that the config directory exists and contains a parseable
// reinguard.yaml (MVP stub; full JSON Schema validation comes later).
func Dir(dir string) error {
	st, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("validate: config directory: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("validate: not a directory: %s", dir)
	}
	p := filepath.Join(dir, requiredFile)
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("validate: read %s: %w", requiredFile, err)
	}
	if len(data) == 0 {
		return fmt.Errorf("validate: %s is empty", requiredFile)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("validate: parse %s: %w", requiredFile, err)
	}
	return nil
}
