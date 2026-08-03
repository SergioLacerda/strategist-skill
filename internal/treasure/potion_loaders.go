package treasure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// --- loading ---

// LoadPotions reads monolithic potions.yaml plus optional potions/*.yaml partitions,
// validates each potion, and groups entries by chest id. Mirrors LoadJewels.
func LoadPotions(root string, governed map[string]GovernedChest) (map[string][]Potion, error) {
	paths, err := PotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	out := make(map[string][]Potion)
	seen := make(map[string]bool)
	for _, path := range paths {
		if err := loadPotionsFromManifest(path, root, governed, out, seen); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func loadPotionsFromManifest(path, root string, governed map[string]GovernedChest, out map[string][]Potion, seen map[string]bool) error {
	m, err := readPotionManifest(path, root)
	if err != nil {
		return err
	}
	for _, p := range m.Potions {
		if err := addPotionFromManifest(p, governed, out, seen); err != nil {
			return fmt.Errorf("%s: %w", potionManifestLabel(root, path), err)
		}
	}
	return nil
}

func readPotionManifest(path, root string) (PotionManifest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: potion manifest path is derived from governed chest metadata
	if err != nil {
		return PotionManifest{}, fmt.Errorf("read %s: %w", potionManifestLabel(root, path), err)
	}
	var m PotionManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return PotionManifest{}, fmt.Errorf("parse %s: %w", potionManifestLabel(root, path), err)
	}
	if m.SchemaVersion != "1" {
		return PotionManifest{}, fmt.Errorf("%s: unsupported schema_version %q (expected \"1\")", potionManifestLabel(root, path), m.SchemaVersion)
	}
	return m, nil
}

func addPotionFromManifest(p Potion, governed map[string]GovernedChest, out map[string][]Potion, seen map[string]bool) error {
	if seen[p.ID] {
		return nil
	}
	if err := ValidatePotionEntry(p, governed); err != nil {
		return err
	}
	out[p.ChestID] = append(out[p.ChestID], p)
	seen[p.ID] = true
	return nil
}

// PotionManifestPaths returns monolithic and partitioned potion manifests in stable read order.
func PotionManifestPaths(root string) ([]string, error) {
	paths, err := monolithicPotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	partitionPaths, err := partitionedPotionManifestPaths(root)
	if err != nil {
		return nil, err
	}
	return append(paths, partitionPaths...), nil
}

func monolithicPotionManifestPaths(root string) ([]string, error) {
	path := filepath.Join(root, "potions.yaml")
	if _, err := os.Stat(path); err == nil {
		return []string{path}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat potions.yaml: %w", err)
	}
	return nil, nil
}

func partitionedPotionManifestPaths(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "potions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read potions/: %w", err)
	}
	var partitionPaths []string
	for _, entry := range entries {
		if isJewelPartitionFile(entry) { // suffix check only, not jewel-specific despite the name
			partitionPaths = append(partitionPaths, filepath.Join(root, "potions", entry.Name()))
		}
	}
	sort.Strings(partitionPaths)
	return partitionPaths, nil
}

func potionManifestLabel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// ValidatePotionEntry validates a single potion against schema and parent trust rules.
func ValidatePotionEntry(p Potion, governed map[string]GovernedChest) error {
	for _, validate := range potionValidators(p, governed) {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func potionValidators(p Potion, governed map[string]GovernedChest) []func() error {
	return []func() error{
		func() error { return validatePotionChest(p) },
		func() error { return validatePotionRunbookRef(p) },
		func() error { return validatePotionSourceRefs(p) },
		func() error { return wrapJewelValidation("status", domain.ValidatePotionStatus(p.ID, p.Status)) },
		func() error { return validatePotionTrust(p, governed) },
	}
}

func validatePotionChest(p Potion) error {
	if p.ChestID == "" {
		return fmt.Errorf("potion %q missing chest_id", p.ID)
	}
	return nil
}

func validatePotionRunbookRef(p Potion) error {
	if p.RunbookRef == "" {
		return fmt.Errorf("potion %q missing runbook_ref", p.ID)
	}
	return nil
}

func validatePotionSourceRefs(p Potion) error {
	if len(p.SourceRefs) == 0 {
		return fmt.Errorf("potion %q missing source_refs", p.ID)
	}
	return nil
}

func validatePotionTrust(p Potion, governed map[string]GovernedChest) error {
	gc, ok := governed[p.ChestID]
	if !ok {
		return nil
	}
	return wrapJewelValidation("trust", domain.ValidatePotionTrust(p.ID, p.Trust, gc.Trust.Tier))
}
