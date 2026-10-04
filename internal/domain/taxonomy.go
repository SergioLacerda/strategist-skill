package domain

import (
	"fmt"
	"strings"
)

// CanonicalTaxonomyVersion identifies the versioned seven-family vocabulary.
const CanonicalTaxonomyVersion = "strategist-taxonomy/v1"

// TaxonomyFamily is the canonical public family of a Strategist entity.
// Routes are deliberately not a family: they are compatibility inputs used to
// resolve a governed Stage.
type TaxonomyFamily string

const (
	// TaxonomyRole identifies the Role family.
	TaxonomyRole TaxonomyFamily = "role"
	// TaxonomyWeapon identifies the Weapon family.
	TaxonomyWeapon TaxonomyFamily = "weapon"
	// TaxonomyFeat identifies the Feat family.
	TaxonomyFeat TaxonomyFamily = "feat"
	// TaxonomyTool identifies the Tool family.
	TaxonomyTool TaxonomyFamily = "tool"
	// TaxonomyMechanism identifies the Mechanism family.
	TaxonomyMechanism TaxonomyFamily = "mechanism"
	// TaxonomyStage identifies the Stage family.
	TaxonomyStage TaxonomyFamily = "stage"
	// TaxonomyArtifact identifies the Artifact family.
	TaxonomyArtifact TaxonomyFamily = "artifact"
)

// Validate rejects values outside the canonical seven-family vocabulary.
func (f TaxonomyFamily) Validate() error {
	switch f {
	case TaxonomyRole, TaxonomyWeapon, TaxonomyFeat, TaxonomyTool,
		TaxonomyMechanism, TaxonomyStage, TaxonomyArtifact:
		return nil
	default:
		return fmt.Errorf("taxonomy family %q is not canonical", f)
	}
}

// CanonicalIdentity is the stable family-aware identity of a public entity.
// Version is required for Weapons because a Role binding must never resolve a
// provider by unversioned name or catalog order.
type CanonicalIdentity struct {
	Family  TaxonomyFamily `json:"family" yaml:"family"`
	ID      string         `json:"id" yaml:"id"`
	Version string         `json:"version,omitempty" yaml:"version,omitempty"`
}

// CanonicalIdentity returns the family-aware identity of a Role.
func (r Role) CanonicalIdentity() CanonicalIdentity {
	return CanonicalIdentity{Family: TaxonomyRole, ID: r.ID}
}

// CanonicalIdentity returns the family-aware identity of a Weapon manifest.
func (w WeaponManifest) CanonicalIdentity() CanonicalIdentity {
	return CanonicalIdentity{Family: TaxonomyWeapon, ID: w.ID, Version: w.Version}
}

// CanonicalIdentity returns the family-aware identity of a compiled Weapon.
func (w CompiledWeapon) CanonicalIdentity() CanonicalIdentity {
	return CanonicalIdentity{Family: TaxonomyWeapon, ID: w.ID, Version: w.Version}
}

// Validate checks family-aware identity without inferring family from a path
// or a historical identifier.
func (i CanonicalIdentity) Validate() error {
	if err := i.Family.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(i.ID) == "" {
		return fmt.Errorf("taxonomy identity %s requires id", i.Family)
	}
	if i.Family == TaxonomyWeapon && strings.TrimSpace(i.Version) == "" {
		return fmt.Errorf("Weapon identity %q requires version", i.ID)
	}
	return nil
}
