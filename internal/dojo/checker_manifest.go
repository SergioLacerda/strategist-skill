package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// CheckManifests validates the manifest_checks section of criteria.
// strategistDir is the path to the .strategist/ directory.
func CheckManifests(criteria domain.DojoCriteria, strategistDir string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem

	for _, mc := range criteria.ManifestChecks {
		manifestPath := filepath.Join(strategistDir, "skills", mc.ExpectedProvider, "skill.yaml")
		exists := fileExists(manifestPath)
		items = append(items, newItem(
			fmt.Sprintf("manifest %s/skills/%s/skill.yaml", mc.Slot, mc.ExpectedProvider),
			exists == mc.ManifestExists,
			fmt.Sprintf("manifest_exists=%v but got %v", mc.ManifestExists, exists)))
		if !exists {
			continue
		}

		items = append(items, checkManifestFields(manifestPath, mc)...)
	}
	return items
}

// checkManifestFields parses the provider manifest as structured YAML and checks each
// requested field is present: a plain name (e.g. "canonical_role") matches a key at
// any depth, preserving the old substring-search behavior for existing scenario
// fixtures; a dotted name (e.g. "specialization_taxonomy.canonical_role") is resolved
// as an exact nested path for scenarios that want to assert structure, not just presence.
func checkManifestFields(manifestPath string, mc domain.DojoManifestCheck) []domain.DojoCheckItem {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // G304: dojo reads a manifest path derived from the scenario criteria
	if err != nil {
		return []domain.DojoCheckItem{newItem(fmt.Sprintf("manifest %s read", mc.ExpectedProvider), false, err.Error())}
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return []domain.DojoCheckItem{newItem(fmt.Sprintf("manifest %s parse", mc.ExpectedProvider), false, err.Error())}
	}

	items := make([]domain.DojoCheckItem, 0, len(mc.FieldsPresent))
	for _, field := range mc.FieldsPresent {
		found := manifestHasField(doc, field)
		label := fmt.Sprintf("manifest field %q in skills/%s/skill.yaml", field, mc.ExpectedProvider)
		items = append(items, newItem(label, found, fmt.Sprintf("field %q not found in manifest", field)))
	}
	return items
}

// manifestHasField reports whether field is present in doc. A dot-separated field
// (e.g. "a.b.c") is resolved as an exact nested path. A plain field name is searched
// for as a key at any depth in the document.
func manifestHasField(doc map[string]any, field string) bool {
	if strings.Contains(field, ".") {
		return manifestHasPath(any(doc), splitFieldPath(field))
	}
	return manifestHasKeyAnywhere(doc, field)
}

func manifestHasPath(node any, parts []string) bool {
	current := node
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return false
		}
		v, ok := m[part]
		if !ok {
			return false
		}
		current = v
	}
	return true
}

func manifestHasKeyAnywhere(node any, field string) bool {
	switch v := node.(type) {
	case map[string]any:
		return manifestMapHasKeyAnywhere(v, field)
	case []any:
		return manifestSliceHasKeyAnywhere(v, field)
	}
	return false
}

func manifestMapHasKeyAnywhere(values map[string]any, field string) bool {
	if _, ok := values[field]; ok {
		return true
	}
	for _, value := range values {
		if manifestHasKeyAnywhere(value, field) {
			return true
		}
	}
	return false
}

func manifestSliceHasKeyAnywhere(values []any, field string) bool {
	for _, value := range values {
		if manifestHasKeyAnywhere(value, field) {
			return true
		}
	}
	return false
}

func splitFieldPath(field string) []string {
	var parts []string
	start := 0
	for i, r := range field {
		if r == '.' {
			parts = append(parts, field[start:i])
			start = i + 1
		}
	}
	return append(parts, field[start:])
}
