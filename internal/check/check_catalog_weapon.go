package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// resolveCatalogWeaponSlot resolves a Weapon the plugin catalog lists (embedded or
// external) without reading any generated skills/<id>/skill.yaml: risk and role
// affinity come from the catalog entry, and readiness from the catalog descriptor
// and the Weapon's payload.
func resolveCatalogWeaponSlot(root, slot, provider string, facts domain.WeaponFacts) (slotResolution, string) {
	required := slotContract[slot]
	if facts.RiskScore != required {
		return slotResolution{}, fmt.Sprintf("slot %s: provider %q has risk_score=%q but slot requires %q — preflight will block", slot, provider, facts.RiskScore, required)
	}
	if errMsg := checkRoleFactsCompatibility(root, slot, provider, facts.RiskScore, facts.Roles); errMsg != "" {
		return slotResolution{}, errMsg
	}
	catalogPath := filepath.Join(root, "plugins", "catalog.yaml")
	return slotResolution{kind: slotResolutionSkillProvider, path: catalogPath, readiness: catalogWeaponReadiness(root, slot, provider, catalogPath, facts)}, ""
}

func catalogWeaponReadiness(root, slot, provider, catalogPath string, facts domain.WeaponFacts) domain.PluginReadinessVector {
	return weaponReadiness(root, slot, provider, readinessFacets{
		descriptor: domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "catalog_entry_valid", Detail: catalogPath},
		source:     domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "catalog_entry_present", Detail: catalogPath},
		entrypoint: catalogEntrypointCheck(root, provider, facts),
		detail:     catalogPath,
		conformance: func(probe connectors.ConnectorResult) domain.ReadinessCheck {
			return customConformanceReadinessFor(root, slot, provider, func() ([]string, domain.ReadinessCheck) {
				if len(facts.Roles) == 0 {
					return nil, conformanceCheck(domain.ReadinessUnknown, "conformance_role_affinity_unknown", "provider does not declare canonical_role or roles")
				}
				return facts.Roles, domain.ReadinessCheck{}
			}, probe)
		},
	})
}

// catalogEntrypointCheck verifies the payload the Weapon's runtime kind needs: the
// host and embedded kinds need their SKILL.md, an openspec_root runtime needs its
// root directory. It is a static presence check, not a live invocation.
func catalogEntrypointCheck(root, provider string, facts domain.WeaponFacts) domain.ReadinessCheck {
	switch facts.RuntimeKind {
	case "host", "embedded":
		return payloadCheck(filepath.Join(root, "skills", provider, "SKILL.md"))
	case "openspec_root":
		return runtimeRootCheck(root, facts.RuntimeRoot)
	default:
		return domain.ReadinessCheck{Status: domain.ReadinessUnknown, ReasonCode: "entrypoint_kind_unsupported", Detail: facts.RuntimeKind}
	}
}

func payloadCheck(path string) domain.ReadinessCheck {
	info, err := os.Stat(path)
	if err != nil {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_payload_missing", Detail: path}
	}
	if info.Size() == 0 {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "entrypoint_payload_empty", Detail: path}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "entrypoint_payload_present", Detail: path}
}

// runtimeRootCheck resolves a catalog runtime root such as ".strategist/openspec"
// against the runtime tree (a path under the .strategist directory) or, for any
// other relative path, against the workspace that holds it.
func runtimeRootCheck(strategistRoot, declared string) domain.ReadinessCheck {
	path := filepath.Join(filepath.Dir(strategistRoot), declared)
	if rest, ok := strings.CutPrefix(filepath.ToSlash(declared), ".strategist/"); ok {
		path = filepath.Join(strategistRoot, filepath.FromSlash(rest))
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "runtime_root_missing", Detail: path}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "runtime_root_present", Detail: path}
}
