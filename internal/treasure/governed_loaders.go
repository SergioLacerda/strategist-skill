package treasure

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadGoverned reads treasure-chests.yaml and returns governed chests by id.
func LoadGoverned(root string) (map[string]GovernedChest, error) {
	raw, err := os.ReadFile(filepath.Join(root, "treasure-chests.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read treasure-chests.yaml: %w", err)
	}
	var m governedManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse treasure-chests.yaml: %w", err)
	}
	out := make(map[string]GovernedChest, len(m.Chests))
	for _, c := range m.Chests {
		if err := domain.ValidateChestGrade(c.ID, domain.ChestGrade{
			SourceGrade:          c.Grade.SourceGrade,
			ReuseValue:           c.Grade.ReuseValue,
			ImplementationStatus: c.Grade.ImplementationStatus,
		}); err != nil {
			return nil, fmt.Errorf("treasure-chests.yaml: %w", err)
		}
		out[c.ID] = c
	}
	return out, nil
}
