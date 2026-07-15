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

// jewelScore is the LLM-governed candidate quality signal recorded during `index`. It is
// never a promotion authority — only `mine` (human curation) can change status.
type jewelScore struct {
	Value   int      `yaml:"value"`
	Reasons []string `yaml:"reasons"`
}

// jewelApplicability records when a jewel is and is not a good fit, set during polishing.
type jewelApplicability struct {
	Scope       []string `yaml:"scope"`
	AppliesWhen []string `yaml:"applies_when"`
	AvoidWhen   []string `yaml:"avoid_when"`
}

// jewelVerification holds evidence recorded by `mine --verify`. Empty until a human
// promotes a jewel to status: verified.
type jewelVerification struct {
	EvidenceRefs []string `yaml:"evidence_refs"`
}

type jewelEntry struct {
	ID            string             `yaml:"id"`
	ChestID       string             `yaml:"chest_id"`
	Kind          string             `yaml:"kind"`
	Statement     string             `yaml:"statement"`
	SourceRefs    []string           `yaml:"source_refs"`
	Trust         string             `yaml:"trust"`
	Status        string             `yaml:"status"`
	ReviewedBy    string             `yaml:"reviewed_by"`
	LastReviewed  string             `yaml:"last_reviewed"`
	Score         jewelScore         `yaml:"score"`
	Applicability jewelApplicability `yaml:"applicability"`
	Verification  jewelVerification  `yaml:"verification"`
}

type jewelsManifest struct {
	Jewels []jewelEntry `yaml:"jewels"`
}

// loadJewels reads jewels.yaml, groups entries by chest_id, and validates each jewel's
// status and trust against its parent chest's current trust tier (from governed). Missing
// chest_id, empty source_refs, or an unsupported status (including the removed legacy
// "active" status — see ADR 0012) are load errors, mirroring loadGoverned's corrupt-input
// handling: a stale or drifted jewels.yaml must fail loudly, not silently degrade.
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
		if err := validateJewelEntry(j, governed); err != nil {
			return nil, fmt.Errorf("jewels.yaml: %w", err)
		}
		out[j.ChestID] = append(out[j.ChestID], j)
	}
	return out, nil
}

// validateJewelEntry checks a single jewel entry's required fields, status, and (when its
// parent chest is governed) trust ceiling. Split out of loadJewels to keep the load loop
// simple — each check here fails loudly rather than silently degrading.
func validateJewelEntry(j jewelEntry, governed map[string]govChest) error {
	if j.ChestID == "" {
		return fmt.Errorf("jewel %q missing chest_id", j.ID)
	}
	if len(j.SourceRefs) == 0 {
		return fmt.Errorf("jewel %q missing source_refs", j.ID)
	}
	if err := domain.ValidateJewelStatus(j.ID, j.Status); err != nil {
		return fmt.Errorf("status: %w", err)
	}
	if gc, ok := governed[j.ChestID]; ok {
		if err := domain.ValidateJewelTrust(j.ID, j.Trust, gc.Trust.Tier); err != nil {
			return fmt.Errorf("trust: %w", err)
		}
	}
	return nil
}

// nonDeprecatedJewelCount returns the number of jewels for a chest whose status is not
// deprecated (i.e. proposed, accepted, or verified).
func nonDeprecatedJewelCount(jewels []jewelEntry) int {
	n := 0
	for _, j := range jewels {
		if j.Status != domain.JewelStatusDeprecated {
			n++
		}
	}
	return n
}
