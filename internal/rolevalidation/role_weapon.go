// Package rolevalidation contains the shared static validation used by the
// installer and strategist check for mandatory role/provider bindings.
package rolevalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// Failure is one actionable role/provider validation failure.
type Failure struct {
	Slot     string
	Role     string
	Provider string
	Reason   string
}

func (f Failure) Error() string {
	provider := f.Provider
	if provider == "" {
		provider = "<none>"
	}
	return fmt.Sprintf("role readiness failed: slot=%s role=%s provider=%s %s", f.Slot, f.Role, provider, f.Reason)
}

// ValidateRuntimeBindings validates the persisted binding contract for the
// configurable discovery and refinement roles. It deliberately does not run
// provider probes and does not resolve runtime fallbacks.
func ValidateRuntimeBindings(root string, active domain.ActiveConfig) []Failure {
	roleMap, err := readRoleMap(root)
	if err != nil {
		return []Failure{{Reason: "role slot map: " + err.Error()}}
	}
	lock, err := readLock(root)
	if err != nil {
		return []Failure{{Reason: "plugins.lock: " + err.Error()}}
	}

	var failures []Failure
	for _, slot := range []string{"discovery", "refinement"} {
		role := roleMap[slot]
		provider := active.Slots[slot]
		if role == "" {
			failures = append(failures, Failure{Slot: slot, Provider: provider, Reason: "native role is not mapped"})
			continue
		}
		failures = append(failures, validateSlot(root, lock, slot, role, provider)...)
	}
	return failures
}

type skillManifest struct {
	RiskScore     string   `yaml:"risk_score"`
	CanonicalRole string   `yaml:"canonical_role"`
	Roles         []string `yaml:"roles"`
}

func validateSlot(root string, lock domain.PluginLockFile, slot, role, provider string) []Failure {
	if provider == "" {
		return []Failure{{Slot: slot, Role: role, Reason: "no weapon configured in active.yaml"}}
	}
	failure := persistedSlotBinding(lock, slot, role, provider)
	if failure != nil {
		return failure
	}
	return validateProviderManifest(root, slot, role, provider)
}

func persistedSlotBinding(lock domain.PluginLockFile, slot, role, provider string) []Failure {
	matching := make([]domain.SlotBinding, 0, 1)
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			matching = append(matching, binding)
		}
	}
	if len(matching) == 0 {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: "no persisted weapon binding in plugins.lock"}}
	}
	if len(matching) != 1 {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("plugins.lock has %d bindings for the slot", len(matching))}}
	}
	if matching[0].InstalledInstanceID != provider {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("persisted binding points to %q, not active provider", matching[0].InstalledInstanceID)}}
	}
	return nil
}

func validateProviderManifest(root string, slot, role, provider string) []Failure {
	// A native role binding is valid when its role contract is present and maps
	// to the slot. External/embedded providers must additionally expose a valid
	// manifest and explicit role affinity.
	skillPath := filepath.Join(root, "skills", provider, "skill.yaml")
	raw, err := os.ReadFile(skillPath) //nolint:gosec // path is derived from the runtime root and active provider
	if os.IsNotExist(err) {
		return validateNativeBinding(root, slot, role, provider)
	}
	if err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("skill manifest unreadable: %v", err)}}
	}
	return validateSkillManifest(slot, role, provider, raw)
}

func validateSkillManifest(slot, role, provider string, raw []byte) []Failure {
	var manifest skillManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("skill manifest invalid: %v", err)}}
	}
	requiredRisk := map[string]string{"discovery": "write_analysis", "refinement": "write_analysis"}[slot]
	if manifest.RiskScore != requiredRisk {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("risk_score=%q, requires %q", manifest.RiskScore, requiredRisk)}}
	}
	roles := manifest.Roles
	if len(roles) == 0 && manifest.CanonicalRole != "" {
		roles = []string{manifest.CanonicalRole}
	}
	if !contains(roles, role) {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("role affinity %v does not include %q", roles, role)}}
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

func readRoleMap(root string) (domain.RoleSlotMap, error) {
	raw, err := os.ReadFile(filepath.Join(root, "roles", "default.yaml")) //nolint:gosec // fixed runtime path
	if err != nil {
		return nil, fmt.Errorf("read role slot map: %w", err)
	}
	var m domain.RoleSlotMap
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal role slot map: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate role slot map: %w", err)
	}
	return m, nil
}

func readLock(root string) (domain.PluginLockFile, error) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins.lock")) //nolint:gosec // fixed runtime path
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock domain.PluginLockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("unmarshal plugins.lock: %w", err)
	}
	return lock, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
