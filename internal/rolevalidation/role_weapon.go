// Package rolevalidation contains the shared static validation used by the
// installer and strategist check for mandatory role/provider bindings.
package rolevalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/catalog"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
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

// Provider manifest validation (skillManifest, validateProviderManifest,
// validateSkillManifest, validateNativeBinding) lives in
// role_weapon_manifest.go, split out to keep this file under the repo's
// file-size budget. readRoleMap/readLock live in role_weapon_io.go.

func validateSlot(root string, lock domain.PluginLockFile, slot, role, provider string) []Failure {
	if provider == "" {
		return []Failure{{Slot: slot, Role: role, Reason: "no weapon configured in active.yaml"}}
	}
	failure := persistedSlotBinding(root, lock, slot, role, provider)
	if failure != nil {
		return failure
	}
	if bindingModeForSlot(lock, slot) == domain.SlotBindingModeRanked {
		// Ranked was already validated against its certification stamp
		// inside persistedSlotBinding — the manifest/risk_score/role-affinity
		// checks validateProviderManifest runs below are Custom-specific
		// (docs/adr/0041's pipeline-scope decision) and never apply to a
		// Ranked binding's build-time-certified provider.
		return nil
	}
	return validateProviderManifest(root, slot, role, provider)
}

func bindingModeForSlot(lock domain.PluginLockFile, slot string) string {
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			return binding.EffectiveMode()
		}
	}
	return domain.SlotBindingModeCustom
}

func persistedSlotBinding(root string, lock domain.PluginLockFile, slot, role, provider string) []Failure {
	matching := bindingsForSlot(lock, slot)
	if len(matching) == 0 {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: "no persisted weapon binding in plugins.lock"}}
	}
	if len(matching) != 1 {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("plugins.lock has %d bindings for the slot", len(matching))}}
	}
	return validateSingleSlotBinding(root, slot, role, provider, matching[0])
}

func bindingsForSlot(lock domain.PluginLockFile, slot string) []domain.SlotBinding {
	matching := make([]domain.SlotBinding, 0, 1)
	for _, binding := range lock.Bindings {
		if binding.Slot == slot {
			matching = append(matching, binding)
		}
	}
	return matching
}

// validateSingleSlotBinding validates the slot's single persisted binding —
// split out of persistedSlotBinding, which only handles the zero-or-many
// cases, to keep each function's branching shallow.
func validateSingleSlotBinding(root string, slot, role, provider string, binding domain.SlotBinding) []Failure {
	if !domain.WeaponRefMatchesBinding(provider, binding) {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("persisted binding points to %q, not active provider", binding.Ref())}}
	}
	if !binding.ValidMode() {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("persisted binding has invalid mode %q", binding.Mode)}}
	}
	if mode := binding.EffectiveMode(); mode == domain.SlotBindingModeRanked {
		return validateRankedSlotBinding(root, slot, role, provider, binding)
	}
	return nil
}

// validateRankedSlotBinding validates a Ranked binding against the catalog's
// certification stamp instead of Custom's manifest/risk_score checks (see
// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md DEC-003).
func validateRankedSlotBinding(root, slot, role, provider string, binding domain.SlotBinding) []Failure {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("plugins/catalog.yaml unreadable: %v", err)}}
	}
	if strings.Contains(string(raw), "ranked_bindings:") {
		return validateCompiledRankedSlotBinding(slot, role, binding.InstalledInstanceID, binding, raw)
	}
	stamp, ok, err := catalog.FindRankedStamp(raw, binding.InstalledInstanceID)
	if err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("plugins/catalog.yaml invalid: %v", err)}}
	}
	if !ok {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: "provider not found in catalog"}}
	}
	if !stamp.Certified() {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: "provider is not a certified ranked candidate"}}
	}
	if !stamp.HasRole(role) {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("certified provider role affinity does not include %q", role)}}
	}
	return nil
}

func validateCompiledRankedSlotBinding(slot, role, provider string, lockBinding domain.SlotBinding, raw []byte) []Failure {
	registry, err := catalog.ParseCompiledRegistryCatalog(raw)
	if err != nil {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: fmt.Sprintf("compiled role/weapon registry invalid: %v", err)}}
	}
	fail := func(reason string) []Failure {
		return []Failure{{Slot: slot, Role: role, Provider: provider, Reason: reason}}
	}
	if lockBinding.WeaponVersion == "" {
		return fail(fmt.Sprintf("plugins.lock Ranked binding has no weapon_version; re-run install to migrate the lock (certified: %s)", registry.RankedOfferNames(role, slot)))
	}
	compiled, ok := registry.RankedBinding(role, slot, provider, lockBinding.WeaponVersion)
	if !ok {
		return fail(fmt.Sprintf("compiled Ranked binding for %s/%s does not certify %s (certified: %s)", role, slot, domain.WeaponIdentity(provider, lockBinding.WeaponVersion), registry.RankedOfferNames(role, slot)))
	}
	if lockBinding.Role != "" && lockBinding.Role != role {
		return fail("plugins.lock Ranked binding Role does not match the role map")
	}
	if !lockMatchesCompiled(lockBinding, compiled) {
		return fail("plugins.lock Ranked binding does not match the compiled registry")
	}
	return nil
}

// lockMatchesCompiled reports whether every resolved field the lock persists
// equals the compiled Ranked binding.
func lockMatchesCompiled(lock domain.SlotBinding, compiled domain.CompiledRankedBinding) bool {
	return lock.WeaponDigest == compiled.WeaponDigest && lock.SourceDigest == compiled.SourceDigest &&
		lock.BindingDigest == compiled.BindingDigest && lock.ExecutionMode == compiled.ExecutionMode &&
		lock.RuntimeKind == compiled.Runtime.Kind && lock.ConnectorID == compiled.ConnectorID &&
		lock.Entrypoint == compiled.Entrypoint && lock.CertificationDigest == compiled.CertificationDigest
}
