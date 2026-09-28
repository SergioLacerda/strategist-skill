package rolevalidation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// BuildRoleInvocationPlan resolves the mission-time Role→Weapon composition
// for slot from the persisted role map and plugins.lock under root. It reuses
// readRoleMap/readLock (role_weapon.go) rather than introducing a second
// role-map/lock-parsing implementation. A Custom binding resolves via
// plugins.lock's digest lookup (domain.NewRoleInvocationPlanFromLock); a
// Ranked binding resolves via the catalog's certification stamp instead
// (docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md).
func BuildRoleInvocationPlan(root, slot string) (domain.RoleInvocationPlan, error) {
	role, lock, err := loadRoleAndLockForSlot(root, slot)
	if err != nil {
		return domain.RoleInvocationPlan{}, err
	}

	binding, err := domain.SingleLockBindingForSlot(lock, slot)
	if err != nil {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: %w", err)
	}
	if !binding.ValidMode() {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: slot %q has invalid binding mode %q", slot, binding.Mode)
	}
	if binding.EffectiveMode() != domain.SlotBindingModeRanked {
		return resolveCustomRoleInvocationPlan(role, slot, lock)
	}
	return resolveRankedRoleInvocationPlan(root, role, slot, binding)
}

// loadRoleAndLockForSlot reads slot's mapped native role and plugins.lock,
// the two pieces of state every resolution path (Custom or Ranked) needs
// before it can branch — split out of BuildRoleInvocationPlan to keep its
// own branching shallow.
func loadRoleAndLockForSlot(root, slot string) (string, domain.PluginLockFile, error) {
	roleMap, err := readRoleMap(root)
	if err != nil {
		return "", domain.PluginLockFile{}, fmt.Errorf("role invocation plan: role slot map: %w", err)
	}
	role := roleMap[slot]
	if role == "" {
		return "", domain.PluginLockFile{}, fmt.Errorf("role invocation plan: native role is not mapped for slot %q", slot)
	}
	lock, err := readLock(root)
	if err != nil {
		return "", domain.PluginLockFile{}, fmt.Errorf("role invocation plan: plugins.lock: %w", err)
	}
	return role, lock, nil
}

func resolveCustomRoleInvocationPlan(role, slot string, lock domain.PluginLockFile) (domain.RoleInvocationPlan, error) {
	plan, err := domain.NewRoleInvocationPlanFromLock(role, slot, lock)
	if err != nil {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: %w", err)
	}
	return plan, nil
}

func resolveRankedRoleInvocationPlan(root, role, slot string, binding domain.SlotBinding) (domain.RoleInvocationPlan, error) {
	stamp, ok, err := readCatalogRankedStamp(root, binding.InstalledInstanceID)
	if err != nil {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: %w", err)
	}
	if !ok {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: ranked binding provider %q not found in catalog", binding.InstalledInstanceID)
	}
	plan, err := domain.NewRankedRoleInvocationPlanFromCatalog(role, slot, binding, stamp)
	if err != nil {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: %w", err)
	}
	if err := validateRankedRuntime(root, plan); err != nil {
		return domain.RoleInvocationPlan{}, fmt.Errorf("role invocation plan: %w", err)
	}
	return plan, nil
}

func validateRankedRuntime(root string, plan domain.RoleInvocationPlan) error {
	if plan.Runtime.Kind == domain.RankedRuntimeNone || plan.Runtime.Kind == domain.RankedRuntimeEmbedded {
		return nil
	}
	state, err := readRankedRuntimeState(root, plan)
	if err != nil {
		return err
	}
	entry, ok := state.Entry(plan.Slot, plan.WeaponID)
	if !ok {
		return fmt.Errorf("ranked provider %q runtime binding is missing for slot %q", plan.WeaponID, plan.Slot)
	}
	if entry.ContractDigest != plan.WeaponDigest {
		return fmt.Errorf("ranked provider %q runtime digest mismatch: expected=%s observed=%s", plan.WeaponID, plan.WeaponDigest, entry.ContractDigest)
	}
	if entry.Role != plan.Role {
		return fmt.Errorf("ranked provider %q runtime was recorded for role %q, not %q; run `strategist install --wizard`", plan.WeaponID, entry.Role, plan.Role)
	}
	return validateRankedRuntimeRoot(root, plan)
}

func readRankedRuntimeState(root string, plan domain.RoleInvocationPlan) (domain.RankedRuntimeState, error) {
	stateRaw, err := os.ReadFile(filepath.Join(root, domain.RankedRuntimeStatePath)) //nolint:gosec // fixed runtime path
	if err != nil {
		return domain.RankedRuntimeState{}, fmt.Errorf("ranked provider %q runtime state is unavailable: %w", plan.WeaponID, err)
	}
	state, err := domain.ParseRankedRuntimeState(stateRaw)
	if err != nil {
		return domain.RankedRuntimeState{}, fmt.Errorf("ranked provider %q runtime state is invalid (run `strategist upgrade`): %w", plan.WeaponID, err)
	}
	return state, nil
}

func validateRankedRuntimeRoot(root string, plan domain.RoleInvocationPlan) error {
	runtimeRoot, err := rankedRuntimeRoot(root, plan.Runtime.Root)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(runtimeRoot, "config.yaml")); err != nil {
		return fmt.Errorf("ranked provider %q runtime root is unavailable: %w", plan.WeaponID, err)
	}
	return nil
}

func rankedRuntimeRoot(root, runtimeRoot string) (string, error) {
	const strategistPrefix = ".strategist/"
	rootSlash := filepath.ToSlash(runtimeRoot)
	if !strings.HasPrefix(rootSlash, strategistPrefix) {
		return "", fmt.Errorf("ranked runtime root %q is outside .strategist", runtimeRoot)
	}
	return filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(rootSlash, strategistPrefix))), nil
}

// readCatalogRankedStamp reads the materialized plugins/catalog.yaml under
// root and returns providerID's certification stamp, the same tolerant-read
// shape as readLock/readRoleMap in this package.
func readCatalogRankedStamp(root, providerID string) (domain.CatalogRankedStamp, bool, error) {
	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml")) //nolint:gosec // G304: fixed runtime path
	if err != nil {
		return domain.CatalogRankedStamp{}, false, fmt.Errorf("read plugins/catalog.yaml: %w", err)
	}
	stamp, ok, err := domain.FindCatalogRankedStamp(raw, providerID)
	if err != nil {
		return domain.CatalogRankedStamp{}, false, fmt.Errorf("find catalog ranked stamp: %w", err)
	}
	return stamp, ok, nil
}
