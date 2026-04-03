package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// Schema resource $id values (must match each JSON file's $id).
const (
	URIReinguardConfig     = "https://github.com/k-shibuki/reinguard/schemas/reinguard-config.json"
	URIRulesDocument       = "https://github.com/k-shibuki/reinguard/schemas/rules-document.json"
	URIKnowledgeManifest   = "https://github.com/k-shibuki/reinguard/schemas/knowledge-manifest.json"
	URIObservationDocument = "https://github.com/k-shibuki/reinguard/schemas/observation-document.json"
	URIOperationalContext  = "https://github.com/k-shibuki/reinguard/schemas/operational-context.json"
	URILabelsConfig        = "https://github.com/k-shibuki/reinguard/schemas/labels-config.json"
	URIGateArtifact        = "https://github.com/k-shibuki/reinguard/schemas/gate-artifact.json"
)

// NewCompiler loads all embedded schemas into a jsonschema Compiler.
func NewCompiler() (*jsonschema.Compiler, error) {
	c := jsonschema.NewCompiler()
	if err := addEmbedded(c, ReinguardConfig); err != nil {
		return nil, err
	}
	if err := addEmbedded(c, RulesDocument); err != nil {
		return nil, err
	}
	if err := addEmbedded(c, KnowledgeManifest); err != nil {
		return nil, err
	}
	if err := addEmbedded(c, ObservationDocument); err != nil {
		return nil, err
	}
	if err := addEmbedded(c, OperationalContext); err != nil {
		return nil, err
	}
	if err := addEmbedded(c, LabelsConfig); err != nil {
		return nil, err
	}
	if err := addEmbedded(c, GateArtifact); err != nil {
		return nil, err
	}
	return c, nil
}

func addEmbedded(c *jsonschema.Compiler, name string) error {
	f, err := Files().Open(name)
	if err != nil {
		return fmt.Errorf("schema %s: %w", name, err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("schema %s: read: %w", name, err)
	}
	var doc any
	if jerr := json.Unmarshal(data, &doc); jerr != nil {
		return fmt.Errorf("schema %s: json: %w", name, jerr)
	}
	uri, err := resourceURI(name)
	if err != nil {
		return err
	}
	if err := c.AddResource(uri, doc); err != nil {
		return fmt.Errorf("schema %s: add resource: %w", name, err)
	}
	return nil
}

func resourceURI(name string) (string, error) {
	switch name {
	case ReinguardConfig:
		return URIReinguardConfig, nil
	case RulesDocument:
		return URIRulesDocument, nil
	case KnowledgeManifest:
		return URIKnowledgeManifest, nil
	case ObservationDocument:
		return URIObservationDocument, nil
	case OperationalContext:
		return URIOperationalContext, nil
	case LabelsConfig:
		return URILabelsConfig, nil
	case GateArtifact:
		return URIGateArtifact, nil
	default:
		return "", fmt.Errorf("unknown embedded schema: %s", name)
	}
}

// ListEmbedded returns embedded schema entry names.
func ListEmbedded() ([]string, error) {
	var names []string
	err := fs.WalkDir(Files(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		names = append(names, path)
		return nil
	})
	return names, err
}
