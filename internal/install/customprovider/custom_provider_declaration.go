package customprovider

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// DeclaredCustomPackage requires a complete Strategist sidecar (strategist.yaml)
// beside a wizard-selected host Weapon and returns its declaration and the
// package version. Roles, slots, risk and version are never inferred from the
// slot or the host directory: a package that does not declare them stops
// onboarding before any state is written.
func DeclaredCustomPackage(evidence connectors.ResolvedProviderPackage, slot string) (plugincatalog.ExternalSkillAdapter, string, error) {
	declaration, err := plugincatalog.LoadExternalSkillAdapter(evidence.Path, evidence.Package.ID)
	if err != nil {
		return plugincatalog.ExternalSkillAdapter{}, "", fmt.Errorf("custom host Weapon %q has no complete Strategist sidecar declaration; run `strategist plugins scaffold-sidecar` for it and select it again: %w", evidence.Package.ID, err)
	}
	if role := plugincatalog.SlotRoleID(domain.SlotName(slot)); !ContainsExact(declaration.Roles, role) {
		return plugincatalog.ExternalSkillAdapter{}, "", fmt.Errorf("custom host Weapon %q declares Roles %v, not %q for slot %s", evidence.Package.ID, declaration.Roles, role, slot)
	}
	if !ContainsExact(declaration.SupportedSlots, slot) {
		return plugincatalog.ExternalSkillAdapter{}, "", fmt.Errorf("custom host Weapon %q declares slots %v, not %q", evidence.Package.ID, declaration.SupportedSlots, slot)
	}
	version, err := CustomPackageVersion(evidence.Package.ID, evidence.Package.Version, declaration.Version)
	if err != nil {
		return plugincatalog.ExternalSkillAdapter{}, "", err
	}
	return declaration, version, nil
}

// CustomPackageVersion takes the version SKILL.md carries, or else the one the
// sidecar declares; two disagreeing declarations or none at all are errors.
func CustomPackageVersion(id, skillVersion, sidecarVersion string) (string, error) {
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

// ContainsExact reports whether values contains want, comparing exact strings.
func ContainsExact(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
