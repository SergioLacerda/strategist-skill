package rolevalidation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

type skillManifest struct {
	RiskScore     string   `yaml:"risk_score"`
	CanonicalRole string   `yaml:"canonical_role"`
	Roles         []string `yaml:"roles"`
}

func validateProviderManifest(root string, slot, role, provider string) []Failure {
	// A native role binding is valid when its role contract is present and maps
	// to the slot. External/embedded providers must additionally expose a valid
	// manifest and explicit role affinity. The manifest is resolved with the
	// catalog as the authority and the generated skills/<id>/skill.yaml view only
	// as a fallback (domain.ResolveWeaponFacts).
	facts, err := domain.ResolveWeaponFacts(root, provider)
	if errors.Is(err, domain.ErrWeaponFactsNotFound) {
		return validateNativeBinding(root, slot, role, provider)
	}
	if err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("skill manifest unreadable: %v", err)}}
	}
	if facts.CompatibilitySource == "native_role" {
		return validateNativeBinding(root, slot, role, provider)
	}
	return validateWeaponFacts(slot, role, provider, facts)
}

func validateSkillManifest(slot, role, provider string, raw []byte) []Failure {
	var manifest skillManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("skill manifest invalid: %v", err)}}
	}
	roles := manifest.Roles
	if len(roles) == 0 && manifest.CanonicalRole != "" {
		roles = []string{manifest.CanonicalRole}
	}
	return validateWeaponFacts(slot, role, provider, domain.WeaponFacts{RiskScore: manifest.RiskScore, Roles: roles})
}

// requiredSlotRisk is the risk_score each configurable slot requires.
var requiredSlotRisk = map[string]string{"discovery": "write_analysis", "refinement": "write_analysis", "execution": "controlled"}

func validateWeaponFacts(slot, role, provider string, facts domain.WeaponFacts) []Failure {
	if required := requiredSlotRisk[slot]; facts.RiskScore != required {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("risk_score=%q, requires %q", facts.RiskScore, required)}}
	}
	if !contains(facts.Roles, role) {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("role affinity %v does not include %q", facts.Roles, role)}}
	}
	return nil
}

func validateNativeBinding(root, slot, role, provider string) []Failure {
	raw, err := os.ReadFile(filepath.Join(root, "roles", provider+".yaml")) //nolint:gosec // provider is read from active.yaml
	if err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: "provider manifest and native role are missing"}}
	}
	var cfg domain.RoleConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("native role invalid: %v", err)}}
	}
	if err := cfg.Validate(); err != nil || cfg.Role != role || cfg.Slot != slot {
		if err != nil {
			return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: "native role invalid: " + err.Error()}}
		}
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("native role declares role=%q slot=%q", cfg.Role, cfg.Slot)}}
	}
	return nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
