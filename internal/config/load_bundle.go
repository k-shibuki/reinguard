package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/k-shibuki/reinguard/internal/configdir"
	"github.com/k-shibuki/reinguard/internal/evaluator"
	"github.com/k-shibuki/reinguard/internal/labels"
	"github.com/k-shibuki/reinguard/internal/procedure"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

// loadSchemaSet holds compiled JSON Schema handles for one Load invocation.
type loadSchemaSet struct {
	root   *jsonschema.Schema
	rules  *jsonschema.Schema
	km     *jsonschema.Schema
	labels *jsonschema.Schema
}

func compileLoadSchemas() (*loadSchemaSet, error) {
	comp, err := schema.NewCompiler()
	if err != nil {
		return nil, err
	}
	rootSch, err := comp.Compile(schema.URIReinguardConfig)
	if err != nil {
		return nil, fmt.Errorf("config: compile root schema: %w", err)
	}
	rulesSch, err := comp.Compile(schema.URIRulesDocument)
	if err != nil {
		return nil, fmt.Errorf("config: compile rules schema: %w", err)
	}
	kmSch, err := comp.Compile(schema.URIKnowledgeManifest)
	if err != nil {
		return nil, fmt.Errorf("config: compile knowledge manifest schema: %w", err)
	}
	labelsSch, err := comp.Compile(schema.URILabelsConfig)
	if err != nil {
		return nil, fmt.Errorf("config: compile labels schema: %w", err)
	}
	return &loadSchemaSet{root: rootSch, rules: rulesSch, km: kmSch, labels: labelsSch}, nil
}

func readAndValidateRoot(dir string, rootSch *jsonschema.Schema) (Root, error) {
	rootPath := filepath.Join(dir, "reinguard.yaml")
	rootData, err := os.ReadFile(rootPath)
	if err != nil {
		return Root{}, fmt.Errorf("config: read %s: %w", rootPath, err)
	}
	var rootMap map[string]any
	if err = yaml.Unmarshal(rootData, &rootMap); err != nil {
		return Root{}, fmt.Errorf("config: parse %s: %w", rootPath, err)
	}
	if err = validateDoc(rootSch, rootMap, rootPath); err != nil {
		return Root{}, err
	}
	var root Root
	if err = yaml.Unmarshal(rootData, &root); err != nil {
		return Root{}, fmt.Errorf("config: decode root: %w", err)
	}
	if err = validateUniqueProviderIDs(&root, rootPath); err != nil {
		return Root{}, err
	}
	if err = validateDeclaredSchemaVersion(root.SchemaVersion, rootPath); err != nil {
		return Root{}, err
	}
	if err = validateRuntimeGateRoles(&root, rootPath); err != nil {
		return Root{}, err
	}
	if err := rejectLegacyRulesDir(dir); err != nil {
		return Root{}, err
	}
	return root, nil
}

func rejectLegacyRulesDir(dir string) error {
	legacyRulesDir := filepath.Join(dir, "rules")
	entries, lerr := os.ReadDir(legacyRulesDir)
	if lerr != nil {
		if os.IsNotExist(lerr) {
			return nil
		}
		return fmt.Errorf("config: read rules dir: %w", lerr)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
			return fmt.Errorf(
				"config: legacy rules/%s detected; migrate files to control/{states,routes,guards}/ with matching type",
				e.Name(),
			)
		}
	}
	return nil
}

func readControlRuleFile(kind, kindDir, name string, rulesSch *jsonschema.Schema) (string, RulesDocument, error) {
	p := filepath.Join(kindDir, name)
	key := kind + "/" + name
	data, readErr := os.ReadFile(p)
	if readErr != nil {
		return "", RulesDocument{}, fmt.Errorf("config: read %s: %w", p, readErr)
	}
	var docMap map[string]any
	if uerr := yaml.Unmarshal(data, &docMap); uerr != nil {
		return "", RulesDocument{}, fmt.Errorf("config: parse %s: %w", p, uerr)
	}
	if err := validateDoc(rulesSch, docMap, p); err != nil {
		return "", RulesDocument{}, err
	}
	var doc RulesDocument
	if uerr := yaml.Unmarshal(data, &doc); uerr != nil {
		return "", RulesDocument{}, fmt.Errorf("config: decode %s: %w", p, uerr)
	}
	if err := validateDeclaredSchemaVersion(doc.SchemaVersion, p); err != nil {
		return "", RulesDocument{}, err
	}
	if err := validateRulesMatchControlKind(kind, doc.Rules, p); err != nil {
		return "", RulesDocument{}, err
	}
	return key, doc, nil
}

func readControlRuleFiles(dir string, rulesSch *jsonschema.Schema) (map[string]RulesDocument, error) {
	ruleFiles := make(map[string]RulesDocument)
	controlKinds := []string{"guards", "routes", "states"}
	for _, kind := range controlKinds {
		kindDir := filepath.Join(dir, "control", kind)
		entries, rerr := os.ReadDir(kindDir)
		if rerr != nil && !os.IsNotExist(rerr) {
			return nil, fmt.Errorf("config: read control/%s dir: %w", kind, rerr)
		}
		if rerr != nil {
			continue
		}
		var yamlNames []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
				yamlNames = append(yamlNames, e.Name())
			}
		}
		sort.Strings(yamlNames)
		for _, name := range yamlNames {
			key, doc, err := readControlRuleFile(kind, kindDir, name, rulesSch)
			if err != nil {
				return nil, err
			}
			ruleFiles[key] = doc
		}
	}
	return ruleFiles, nil
}

func validateEvaluatorReferences(ruleFiles map[string]RulesDocument) error {
	reg := evaluator.DefaultRegistry()
	paths := make([]string, 0, len(ruleFiles))
	for p := range ruleFiles {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, key := range paths {
		doc := ruleFiles[key]
		for i, r := range doc.Rules {
			if err := evaluator.ValidateWhen(r.When, reg); err != nil {
				return fmt.Errorf("config: rule[%d] id %q in control/%s: %w", i, r.ID, key, err)
			}
		}
	}
	return nil
}

func applyOptionalLabels(res *LoadResult, dir string, labelsSch *jsonschema.Schema) error {
	labelsPath := filepath.Join(dir, "labels.yaml")
	labelsData, lerr := os.ReadFile(labelsPath)
	if lerr != nil && !os.IsNotExist(lerr) {
		return fmt.Errorf("config: read %s: %w", labelsPath, lerr)
	}
	if lerr != nil {
		return nil
	}
	var labelsMap map[string]any
	if err := yaml.Unmarshal(labelsData, &labelsMap); err != nil {
		return fmt.Errorf("config: parse %s: %w", labelsPath, err)
	}
	if err := validateDoc(labelsSch, labelsMap, labelsPath); err != nil {
		return err
	}
	var lf labels.Config
	if err := yaml.Unmarshal(labelsData, &lf); err != nil {
		return fmt.Errorf("config: decode %s: %w", labelsPath, err)
	}
	if err := validateDeclaredSchemaVersion(lf.SchemaVersion, labelsPath); err != nil {
		return err
	}
	res.LabelsPresent = true
	res.Labels = &lf
	return nil
}

func applyOptionalKnowledge(res *LoadResult, dir string, kmSch *jsonschema.Schema) error {
	kmPath := filepath.Join(dir, "knowledge", "manifest.json")
	kmData, err := os.ReadFile(kmPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config: read knowledge manifest: %w", err)
	}
	var kmMap map[string]any
	if jerr := json.Unmarshal(kmData, &kmMap); jerr != nil {
		return fmt.Errorf("config: parse knowledge manifest: %w", jerr)
	}
	if err := validateDoc(kmSch, kmMap, kmPath); err != nil {
		return err
	}
	var km KnowledgeManifest
	if jerr := json.Unmarshal(kmData, &km); jerr != nil {
		return fmt.Errorf("config: decode knowledge manifest: %w", jerr)
	}
	if err := validateDeclaredSchemaVersion(km.SchemaVersion, kmPath); err != nil {
		return err
	}
	if err := validateKnowledgeWhenClauses(&km, kmPath); err != nil {
		return err
	}
	res.KnowledgePresent = true
	res.Knowledge = &km
	return nil
}

func validateKnowledgeWhenClauses(km *KnowledgeManifest, pathHint string) error {
	if km == nil {
		return nil
	}
	reg := evaluator.DefaultRegistry()
	for i, e := range km.Entries {
		if e.When == nil {
			continue
		}
		if err := evaluator.ValidateWhen(e.When, reg); err != nil {
			return fmt.Errorf("config: knowledge manifest %s entry[%d] id %q: %w", pathHint, i, e.ID, err)
		}
	}
	return nil
}

func applyOptionalProcedures(res *LoadResult, dir string) error {
	repoRoot := configdir.RepoRoot(dir)
	procDir := filepath.Join(dir, "procedure")
	entries, present, err := procedure.LoadEntries(repoRoot, procDir)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if !present {
		return nil
	}
	declared := declaredStateIDsFromRules(res.Rules())
	if err := procedure.ValidateStateMapping(entries, declared); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	res.ProcedurePresent = true
	res.ProcedureEntries = entries
	return nil
}

func declaredStateIDsFromRules(rules []Rule) map[string]struct{} {
	out := make(map[string]struct{})
	for _, r := range rules {
		if r.Type != "state" {
			continue
		}
		sid := strings.TrimSpace(r.StateID)
		if sid != "" {
			out[sid] = struct{}{}
		}
	}
	return out
}
