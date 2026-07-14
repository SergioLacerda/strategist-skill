package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// --- SQ-009 (Track T-J): jewels ---
//
// Jewels are children of a specific chest, generated directly by the agent analyzing that
// chest (no pre-approval gate) — see .analysis/refined/bau-tesouro-sq003-004-007/design.md.
// The Go-side responsibility here is read-only: load, validate the trust-tier ceiling, and
// surface counts in `strategist treasure-chest`. Generation itself is contract/LLM-level
// behavior (internal_skills/*/skill.yaml, machine/context-enrichment.yaml), not Go code.

type jewelEntry struct {
	ID           string   `yaml:"id"`
	ChestID      string   `yaml:"chest_id"`
	Statement    string   `yaml:"statement"`
	SourceRefs   []string `yaml:"source_refs"`
	Trust        string   `yaml:"trust"`
	Status       string   `yaml:"status"`
	ReviewedBy   string   `yaml:"reviewed_by"`
	LastReviewed string   `yaml:"last_reviewed"`
}

type jewelsManifest struct {
	Jewels []jewelEntry `yaml:"jewels"`
}

// loadJewels reads jewels.yaml, groups entries by chest_id, and validates each jewel's
// trust against its parent chest's current trust tier (from governed). Missing chest_id
// or empty source_refs are load errors, mirroring loadGoverned's corrupt-input handling.
func loadJewels(root string, governed map[string]govChest) (map[string][]jewelEntry, error) {
	raw, err := os.ReadFile(filepath.Join(root, "jewels.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read jewels.yaml: %w", err)
	}
	var m jewelsManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse jewels.yaml: %w", err)
	}

	out := make(map[string][]jewelEntry, len(m.Jewels))
	for _, j := range m.Jewels {
		if j.ChestID == "" {
			return nil, fmt.Errorf("jewels.yaml: jewel %q missing chest_id", j.ID)
		}
		if len(j.SourceRefs) == 0 {
			return nil, fmt.Errorf("jewels.yaml: jewel %q missing source_refs", j.ID)
		}
		if gc, ok := governed[j.ChestID]; ok {
			if err := domain.ValidateJewelTrust(j.ID, j.Trust, gc.Trust.Tier); err != nil {
				return nil, fmt.Errorf("jewels.yaml: %w", err)
			}
		}
		out[j.ChestID] = append(out[j.ChestID], j)
	}
	return out, nil
}

// activeJewelCount returns the number of non-deprecated jewels for a chest.
func activeJewelCount(jewels []jewelEntry) int {
	n := 0
	for _, j := range jewels {
		if j.Status != "deprecated" {
			n++
		}
	}
	return n
}
