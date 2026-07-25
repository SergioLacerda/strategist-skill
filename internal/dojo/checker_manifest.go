package dojo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// CheckManifests validates the manifest_checks section of criteria.
// strategistDir is the path to the .strategist/ directory.
func CheckManifests(criteria domain.DojoCriteria, strategistDir string) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem

	for _, mc := range criteria.ManifestChecks {
		manifestPath := filepath.Join(strategistDir, "skills", mc.ExpectedProvider, "skill.yaml")
		exists := fileExists(manifestPath)
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("manifest %s/skills/%s/skill.yaml", mc.Slot, mc.ExpectedProvider),
			Passed: exists == mc.ManifestExists,
			Detail: ifFail(exists == mc.ManifestExists, fmt.Sprintf("manifest_exists=%v but got %v", mc.ManifestExists, exists)),
		})
		if !exists {
			continue
		}

		items = append(items, checkManifestFields(manifestPath, mc)...)
	}
	return items
}

func checkManifestFields(manifestPath string, mc domain.DojoManifestCheck) []domain.DojoCheckItem {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return []domain.DojoCheckItem{{
			Label:  fmt.Sprintf("manifest %s read", mc.ExpectedProvider),
			Passed: false,
			Detail: err.Error(),
		}}
	}
	text := string(raw)
	items := make([]domain.DojoCheckItem, 0, len(mc.FieldsPresent))
	for _, field := range mc.FieldsPresent {
		found := strings.Contains(text, field+":")
		items = append(items, domain.DojoCheckItem{
			Label:  fmt.Sprintf("manifest field %q in skills/%s/skill.yaml", field, mc.ExpectedProvider),
			Passed: found,
			Detail: ifFail(found, fmt.Sprintf("field %q not found in manifest", field)),
		})
	}
	return items
}
