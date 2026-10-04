// Package roster owns deterministic ROSTER and Weapon selection artifacts.
package roster

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

const (
	// WeaponRosterArtifactSchemaVersion identifies the serialized Weapon roster contract.
	WeaponRosterArtifactSchemaVersion = "strategist-weapon-roster/v1"
	// WeaponSelectionArtifactSchemaVersion identifies the serialized selection contract.
	WeaponSelectionArtifactSchemaVersion = "strategist-weapon-selection/v1"
)

// WeaponRosterEntry is one explicit Role/slot/Weapon compatibility offer.
type WeaponRosterEntry struct {
	Role            string `json:"role" yaml:"role"`
	Slot            string `json:"slot" yaml:"slot"`
	WeaponID        string `json:"weapon_id" yaml:"weapon_id"`
	WeaponVersion   string `json:"weapon_version,omitempty" yaml:"weapon_version,omitempty"`
	Source          string `json:"source" yaml:"source"`
	Materialization string `json:"materialization" yaml:"materialization"`
	Compatible      bool   `json:"compatible" yaml:"compatible"`
	Selected        bool   `json:"selected" yaml:"selected"`
}

// WeaponRosterArtifact is the read-only candidate set produced by ROSTER.
type WeaponRosterArtifact struct {
	SchemaVersion   string              `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string              `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage           domain.Stage        `json:"stage" yaml:"stage"`
	Entries         []WeaponRosterEntry `json:"entries" yaml:"entries"`
	Digest          string              `json:"digest" yaml:"digest"`
}

// NewWeaponRosterArtifact creates a stable, digestable ROSTER artifact.
func NewWeaponRosterArtifact(entries []WeaponRosterEntry) (WeaponRosterArtifact, error) {
	artifact := WeaponRosterArtifact{
		SchemaVersion:   WeaponRosterArtifactSchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Stage:           domain.StageRoster,
		Entries:         append([]WeaponRosterEntry(nil), entries...),
	}
	sort.Slice(artifact.Entries, func(i, j int) bool {
		left, right := artifact.Entries[i], artifact.Entries[j]
		return rosterEntryKey(left) < rosterEntryKey(right)
	})
	if err := artifact.validateShape(); err != nil {
		return WeaponRosterArtifact{}, err
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		return WeaponRosterArtifact{}, fmt.Errorf("weapon roster: encode: %w", err)
	}
	digest := sha256.Sum256(raw)
	artifact.Digest = fmt.Sprintf("sha256:%x", digest)
	return artifact, nil
}

// Validate checks the roster shape and digest integrity.
func (a WeaponRosterArtifact) Validate() error {
	if err := a.validateShape(); err != nil {
		return err
	}
	if strings.TrimSpace(a.Digest) == "" {
		return fmt.Errorf("weapon roster: digest is required")
	}
	unsigned := a
	unsigned.Digest = ""
	raw, err := json.Marshal(unsigned)
	if err != nil {
		return fmt.Errorf("weapon roster: encode: %w", err)
	}
	want := sha256.Sum256(raw)
	if a.Digest != fmt.Sprintf("sha256:%x", want) {
		return fmt.Errorf("weapon roster: digest mismatch")
	}
	return nil
}

func (a WeaponRosterArtifact) validateShape() error {
	if a.SchemaVersion != WeaponRosterArtifactSchemaVersion {
		return fmt.Errorf("weapon roster: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := domain.ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("weapon roster: %w", err)
	}
	if a.Stage != domain.StageRoster {
		return fmt.Errorf("weapon roster: stage must be ROSTER")
	}
	if len(a.Entries) == 0 {
		return fmt.Errorf("weapon roster: entries are required")
	}
	return validateRosterEntries(a.Entries)
}

func validateRosterEntries(entries []WeaponRosterEntry) error {
	seen := map[string]struct{}{}
	for _, entry := range entries {
		if strings.TrimSpace(entry.Role) == "" || strings.TrimSpace(entry.Slot) == "" || strings.TrimSpace(entry.WeaponID) == "" {
			return fmt.Errorf("weapon roster: role, slot, and weapon_id are required")
		}
		key := rosterEntryKey(entry)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("weapon roster: duplicate entry %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func rosterEntryKey(entry WeaponRosterEntry) string {
	return strings.Join([]string{entry.Role, entry.Slot, entry.WeaponID, entry.WeaponVersion}, "\x00")
}

// WeaponSelectionArtifact records the deterministic selection outcome without
// becoming another binding authority.
type WeaponSelectionArtifact struct {
	SchemaVersion   string       `json:"schema_version" yaml:"schema_version"`
	TaxonomyVersion string       `json:"taxonomy_version,omitempty" yaml:"taxonomy_version,omitempty"`
	Stage           domain.Stage `json:"stage" yaml:"stage"`
	Role            string       `json:"role" yaml:"role"`
	Slot            string       `json:"slot" yaml:"slot"`
	WeaponID        string       `json:"weapon_id,omitempty" yaml:"weapon_id,omitempty"`
	WeaponVersion   string       `json:"weapon_version,omitempty" yaml:"weapon_version,omitempty"`
	Status          string       `json:"status" yaml:"status"`
	Reason          string       `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// Validate checks the deterministic selection projection without making it a
// binding authority. Empty taxonomy_version remains valid for legacy records.
func (a WeaponSelectionArtifact) Validate() error {
	if a.SchemaVersion != WeaponSelectionArtifactSchemaVersion {
		return fmt.Errorf("weapon selection: unsupported schema_version %q", a.SchemaVersion)
	}
	if err := domain.ValidateTaxonomyVersion(a.TaxonomyVersion); err != nil {
		return fmt.Errorf("weapon selection: %w", err)
	}
	if a.Stage != domain.StageRoster {
		return fmt.Errorf("weapon selection: stage must be ROSTER")
	}
	if strings.TrimSpace(a.Role) == "" || strings.TrimSpace(a.Slot) == "" || strings.TrimSpace(a.Status) == "" {
		return fmt.Errorf("weapon selection: role, slot, and status are required")
	}
	return nil
}
