package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// declaredCustomPackage requires a complete Strategist sidecar (strategist.yaml)
// beside a wizard-selected host Weapon and returns its declaration and the
// package version. Roles, slots, risk and version are never inferred from the
// slot or the host directory: a package that does not declare them stops
// onboarding before any state is written.
func declaredCustomPackage(evidence connectors.ResolvedProviderPackage, slot string) (externalSkillAdapter, string, error) {
	declaration, err := loadExternalSkillAdapter(evidence.Path, evidence.Package.ID)
	if err != nil {
		return externalSkillAdapter{}, "", fmt.Errorf("custom host Weapon %q has no complete Strategist sidecar declaration; run `strategist plugins scaffold-sidecar` for it and select it again: %w", evidence.Package.ID, err)
	}
	if role := slotRoleID(domain.SlotName(slot)); !containsExact(declaration.Roles, role) {
		return externalSkillAdapter{}, "", fmt.Errorf("custom host Weapon %q declares Roles %v, not %q for slot %s", evidence.Package.ID, declaration.Roles, role, slot)
	}
	if !containsExact(declaration.SupportedSlots, slot) {
		return externalSkillAdapter{}, "", fmt.Errorf("custom host Weapon %q declares slots %v, not %q", evidence.Package.ID, declaration.SupportedSlots, slot)
	}
	version, err := customPackageVersion(evidence.Package.ID, evidence.Package.Version, declaration.Version)
	if err != nil {
		return externalSkillAdapter{}, "", err
	}
	return declaration, version, nil
}

// customPackageVersion takes the version SKILL.md carries, or else the one the
// sidecar declares; two disagreeing declarations or none at all are errors.
func customPackageVersion(id, skillVersion, sidecarVersion string) (string, error) {
	skillVersion, sidecarVersion = strings.TrimSpace(skillVersion), strings.TrimSpace(sidecarVersion)
	switch {
	case skillVersion != "" && sidecarVersion != "" && skillVersion != sidecarVersion:
		return "", fmt.Errorf("custom host Weapon %q declares version %q in SKILL.md and %q in its sidecar", id, skillVersion, sidecarVersion)
	case skillVersion != "":
		return skillVersion, nil
	case sidecarVersion != "":
		return sidecarVersion, nil
	}
	return "", fmt.Errorf("custom host Weapon %q declares no version in SKILL.md metadata or its sidecar", id)
}

func containsExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
