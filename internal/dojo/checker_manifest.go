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
// strategistDir is the path to the .strategist/ directory. A Weapon's manifest is
// its plugins/catalog.yaml entry; a provider the catalog does not list is read from
// the transitional skills/<id>/skill.yaml compat view.
func CheckManifests(criteria domain.DojoCriteria, strategistDir string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem

	for _, mc := range criteria.ManifestChecks {
		doc, found, err := loadManifestDoc(strategistDir, mc.ExpectedProvider)
		if err != nil {
			items = append(items, newItem(fmt.Sprintf("manifest %s/%s read", mc.Slot, mc.ExpectedProvider), false, err.Error()))
			continue
		}
		items = append(items, newItem(
			fmt.Sprintf("manifest %s/%s", mc.Slot, mc.ExpectedProvider),
			found == mc.ManifestExists,
			fmt.Sprintf("manifest_exists=%v but got %v", mc.ManifestExists, found)))
		if !found {
			continue
		}

		items = append(items, checkManifestFields(doc, mc)...)
	}
	return items
}

// loadManifestDoc returns the provider's manifest as a generic document: its catalog
// entry, or the compat view when the catalog does not list it.
func loadManifestDoc(strategistDir, provider string) (map[string]any, bool, error) {
	if doc, found, err := catalogEntryDoc(strategistDir, provider); err != nil || found {
		return doc, found, err
	}
	raw, err := os.ReadFile(filepath.Join(strategistDir, "skills", provider, "skill.yaml")) //nolint:gosec // G304: path derived from the scenario criteria
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read compat view for %s: %w", provider, err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, fmt.Errorf("parse compat view for %s: %w", provider, err)
	}
	return doc, true, nil
}

func catalogEntryDoc(strategistDir, provider string) (map[string]any, bool, error) {
	raw, err := os.ReadFile(filepath.Join(strategistDir, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed path under the runtime root
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read plugin catalog: %w", err)
	}
	var file struct {
		Providers []map[string]any `yaml:"providers"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, false, fmt.Errorf("parse plugin catalog: %w", err)
	}
	for _, entry := range file.Providers {
		if id, ok := entry["id"].(string); ok && id == provider {
			return entry, true, nil
		}
	}
	return nil, false, nil
}

// checkManifestFields checks each requested field is present in the provider
// manifest: a plain name (e.g. "canonical_role") matches a key at any depth,
// preserving the old substring-search behavior for existing scenario fixtures; a
// dotted name (e.g. "runtime.kind") is resolved as an exact nested path for
// scenarios that want to assert structure, not just presence.
func checkManifestFields(doc map[string]any, mc domain.DojoManifestCheck) []domain.DojoCheckItem {
	items := make([]domain.DojoCheckItem, 0, len(mc.FieldsPresent))
	for _, field := range mc.FieldsPresent {
		found := manifestHasField(doc, field)
		label := fmt.Sprintf("manifest field %q of %s", field, mc.ExpectedProvider)
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
