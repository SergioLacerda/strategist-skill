//go:build spec

package spec_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"
)

// Task 4 (G03, .analysis/refined/20260830-skill-gaps-triage): every
// enforced_by tag anywhere in the shipped contract corpus must be one of the
// three tiers canonically defined at the top of contracts/machine/errors.yaml
// — machine_enforced, machine_observed, or agent_only. This guards against a
// typo, a stray old binary|agent value, or an ad-hoc fourth value drifting in
// as the vocabulary is adopted by more contract files over time.

var validEnforcedByTiers = map[string]bool{
	"machine_enforced": true,
	"machine_observed": true,
	"agent_only":       true,
}

// enforcedByYAMLFiles are the machine/*.yaml contract files known to carry
// structured `enforced_by: <value>` tags as of 2026-08-30. Extend this list
// as more files adopt the vocabulary — TestErrorCatalogEnforcedByIsAccurate
// (error_catalog_test.go) additionally checks errors.yaml's own tokens for
// reachability, not just vocabulary validity.
var enforcedByYAMLFiles = []string{
	"contracts/machine/errors.yaml",
	"contracts/machine/preflight.yaml",
	"contracts/machine/approval-gate.yaml",
}

// enforcedByMarkdownFiles are the narrative/*.md contract files known to
// carry inline `enforced_by: <value>` tags (backtick-delimited) as of
// 2026-08-30.
var enforcedByMarkdownFiles = []string{
	"contracts/narrative/00-routing.md",
	"contracts/narrative/05-approval-gate.md",
}

func TestEnforcedByTagsUseUnifiedVocabulary(t *testing.T) {
	t.Parallel()
	root := filepath.Join(repoRoot(t), "internal", "embed", "defaults")

	for _, rel := range enforcedByYAMLFiles {
		path := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(path) //nolint:gosec // G304: rel is a fixed literal from enforcedByYAMLFiles
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		values := collectYAMLEnforcedByValues(t, rel, data)
		if len(values) == 0 {
			t.Errorf("%s carries no enforced_by tags — remove it from enforcedByYAMLFiles or add tags", rel)
		}
		for _, v := range values {
			if !validEnforcedByTiers[v] {
				t.Errorf("%s: enforced_by=%q is not one of machine_enforced|machine_observed|agent_only", rel, v)
			}
		}
	}

	inlineRe := regexp.MustCompile("`enforced_by:\\s*([a-zA-Z_]+)`")
	for _, rel := range enforcedByMarkdownFiles {
		path := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(path) //nolint:gosec // G304: rel is a fixed literal from enforcedByMarkdownFiles
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		matches := inlineRe.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			t.Errorf("%s carries no inline `enforced_by: <value>` tags — remove it from enforcedByMarkdownFiles or add tags", rel)
		}
		for _, m := range matches {
			if !validEnforcedByTiers[m[1]] {
				t.Errorf("%s: enforced_by=%q is not one of machine_enforced|machine_observed|agent_only", rel, m[1])
			}
		}
	}
}

// collectYAMLEnforcedByValues walks a YAML document's node tree and returns
// every scalar value assigned to a mapping key literally named "enforced_by",
// at any nesting depth. Walking the generic node tree — rather than a bespoke
// Go struct per file — means one function handles errors.yaml's flat list of
// error maps, preflight.yaml's error_conditions list, and approval-gate.yaml's
// invariants list without caring which shape it is. Comment lines are never
// visited: yaml.Node parses structure only, so historical prose mentions of
// "enforced_by: binary" inside # comments (see errors.yaml's migration notes)
// are correctly invisible to this walk.
func collectYAMLEnforcedByValues(t *testing.T, rel string, data []byte) []string {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	var values []string
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil {
			return
		}
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				keyNode, valNode := n.Content[i], n.Content[i+1]
				if keyNode.Value == "enforced_by" && valNode.Kind == yaml.ScalarNode {
					values = append(values, valNode.Value)
				}
				walk(valNode)
			}
			return
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(&doc)
	return values
}
