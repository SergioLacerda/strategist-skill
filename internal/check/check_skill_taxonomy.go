package check

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// skillTaxonomy accepts canonical_role in either shape a skill.yaml may
// carry it in: top-level (the internal/embed/defaults/skills/<id>/skill.yaml
// authoring convention brainstorming and openspec-explore already use) or
// nested under specialization_taxonomy (the shape a compiled runtime
// instance's .strategist/skills/<id>/skill.yaml may carry — the two are not
// currently guaranteed identical by any generator this check depends on, so
// it reads whichever is present rather than assuming one).
type skillTaxonomy struct {
	CanonicalRole          string                `yaml:"canonical_role"`
	Roles                  []string              `yaml:"roles"`
	WeaponContract         domain.WeaponContract `yaml:"weapon_contract"`
	SpecializationTaxonomy struct {
		CanonicalRole string `yaml:"canonical_role"`
	} `yaml:"specialization_taxonomy"`
}

func (t skillTaxonomy) canonicalRole() string {
	if t.CanonicalRole != "" {
		return t.CanonicalRole
	}
	return t.SpecializationTaxonomy.CanonicalRole
}

func (t skillTaxonomy) roles() []string {
	if len(t.Roles) > 0 {
		return t.Roles
	}
	if role := t.canonicalRole(); role != "" {
		return []string{role}
	}
	return nil
}
