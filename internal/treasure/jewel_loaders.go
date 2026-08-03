package treasure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadJewels reads monolithic jewels.yaml plus optional jewels/*.yaml partitions,
// validates each jewel, and groups entries by chest id. Duplicate jewel ids are loaded
// from the first file encountered so mixed-layout workspaces do not double-count jewels.
func LoadJewels(root string, governed map[string]GovernedChest) (map[string][]Jewel, error) {
	paths, err := JewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	out := make(map[string][]Jewel)
	seen := make(map[string]bool)
	for _, path := range paths {
		if err := loadJewelsFromManifest(path, root, governed, out, seen); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func loadJewelsFromManifest(path, root string, governed map[string]GovernedChest, out map[string][]Jewel, seen map[string]bool) error {
	m, err := readJewelManifest(path, root)
	if err != nil {
		return err
	}
	for _, j := range m.Jewels {
		if err := addJewelFromManifest(j, governed, out, seen); err != nil {
			return fmt.Errorf("%s: %w", jewelManifestLabel(root, path), err)
		}
	}
	return nil
}

func readJewelManifest(path, root string) (Manifest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: jewel manifest path is derived from governed chest metadata
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", jewelManifestLabel(root, path), err)
	}
	var m Manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", jewelManifestLabel(root, path), err)
	}
	if m.SchemaVersion != "1" {
		return Manifest{}, fmt.Errorf("%s: unsupported schema_version %q (expected \"1\")", jewelManifestLabel(root, path), m.SchemaVersion)
	}
	return m, nil
}

func addJewelFromManifest(j Jewel, governed map[string]GovernedChest, out map[string][]Jewel, seen map[string]bool) error {
	if seen[j.ID] {
		return nil
	}
	if err := ValidateJewelEntry(j, governed); err != nil {
		return err
	}
	out[j.ChestID] = append(out[j.ChestID], j)
	seen[j.ID] = true
	return nil
}

// JewelManifestPaths returns monolithic and partitioned jewel manifests in stable read order.
func JewelManifestPaths(root string) ([]string, error) {
	paths, err := monolithicJewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	partitionPaths, err := partitionedJewelManifestPaths(root)
	if err != nil {
		return nil, err
	}
	return append(paths, partitionPaths...), nil
}

func monolithicJewelManifestPaths(root string) ([]string, error) {
	path := filepath.Join(root, "jewels.yaml")
	if _, err := os.Stat(path); err == nil {
		return []string{path}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat jewels.yaml: %w", err)
	}
	return nil, nil
}

func partitionedJewelManifestPaths(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "jewels"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read jewels/: %w", err)
	}
	var partitionPaths []string
	for _, entry := range entries {
		if isJewelPartitionFile(entry) {
			partitionPaths = append(partitionPaths, filepath.Join(root, "jewels", entry.Name()))
		}
	}
	sort.Strings(partitionPaths)
	return partitionPaths, nil
}

func isJewelPartitionFile(entry os.DirEntry) bool {
	name := entry.Name()
	return !entry.IsDir() && (strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml"))
}

func jewelManifestLabel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

// ValidateJewelEntry validates a single jewel against schema and parent trust rules.
func ValidateJewelEntry(j Jewel, governed map[string]GovernedChest) error {
	for _, validate := range jewelValidators(j, governed) {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func jewelValidators(j Jewel, governed map[string]GovernedChest) []func() error {
	return []func() error{
		func() error { return validateJewelChest(j) },
		func() error { return validateJewelSourceRefs(j) },
		func() error { return wrapJewelValidation("status", domain.ValidateJewelStatus(j.ID, j.Status)) },
		func() error { return wrapJewelValidation("kind", domain.ValidateJewelKind(j.ID, j.Kind)) },
		func() error { return wrapJewelValidation("score", domain.ValidateJewelScore(j.ID, j.Score.Value)) },
		func() error { return validateJewelTrust(j, governed) },
		func() error { return wrapJewelValidation("challenge_template", validateJewelChallengeTemplate(j)) },
		func() error { return wrapJewelValidation("evidence_quality", validateJewelEvidenceFields(j)) },
	}
}

func validateJewelChest(j Jewel) error {
	if j.ChestID == "" {
		return fmt.Errorf("jewel %q missing chest_id", j.ID)
	}
	return nil
}

func validateJewelSourceRefs(j Jewel) error {
	if len(j.SourceRefs) == 0 {
		return fmt.Errorf("jewel %q missing source_refs", j.ID)
	}
	return nil
}

func validateJewelTrust(j Jewel, governed map[string]GovernedChest) error {
	gc, ok := governed[j.ChestID]
	if !ok {
		return nil
	}
	return wrapJewelValidation("trust", domain.ValidateJewelTrust(j.ID, j.Trust, gc.Trust.Tier))
}

func wrapJewelValidation(label string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}

// NonDeprecatedJewelCount counts jewels that are still active in the curation lifecycle.
func NonDeprecatedJewelCount(jewels []Jewel) int {
	n := 0
	for _, j := range jewels {
		if j.Status != domain.JewelStatusDeprecated {
			n++
		}
	}
	return n
}
