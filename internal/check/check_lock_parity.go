package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
	"gopkg.in/yaml.v3"
)

// checkPluginLockParity compares active.yaml's slots.<phase> values (the
// values strategist check actually validates via resolveSlotProvider)
// against plugins.lock's own bindings[].installed_instance_id for the same
// slot — the Wizard's other, independent persistence target
// (activateRoleProviderMigration, docs/adr/0037-wizard-role-binding-persistence.md).
// The two are written by different code paths in the same wizard run and are
// not otherwise cross-validated: a workspace can end up with active.yaml
// naming one provider and plugins.lock recording an already-resolved,
// different provider for the same slot, and strategist check silently
// validates only active.yaml's value (see .analysis/refined/
// 20260914-wizard-weapon-options-not-listed/analysis.md KF-11, which found
// this live in this repository's own workspace).
//
// A missing or unreadable plugins.lock is not an error here — a fresh
// install may not have one yet, and no other check in this package requires
// it — this function reports only an actual, readable disagreement.
func checkPluginLockParity(root string, activeSlots map[string]string) []string {
	lockPath := filepath.Join(root, "plugins.lock")
	raw, err := os.ReadFile(lockPath) //nolint:gosec // G304: fixed path under the runtime root
	if err != nil {
		return nil
	}
	var lock domain.PluginLockFile
	if yaml.Unmarshal(raw, &lock) != nil {
		return nil
	}
	var errs []string
	errs = append(errs, lockDigestErrors(lock.Lock)...)
	errs = append(errs, lockParityErrors(lock, activeSlots)...)
	return errs
}

// lockDigestErrors checks the lock graph's internal self-consistency —
// whether graph_digest still matches a fresh digest of its own nodes — when
// a resolved Lock block is present. Only the Custom pipeline's provider
// migration populates Lock (role_provider_migration.go); a zero-value Lock
// (empty schema_version) is a legitimate state for a Ranked-only or
// not-yet-resolved workspace, not corruption, so it is silently skipped
// rather than reported — mirroring checkPluginLockParity's own "no lock is
// not an error" stance for the file as a whole.
func lockDigestErrors(lock domain.PluginLock) []string {
	if lock.SchemaVersion == "" {
		return nil
	}
	if err := resolver.VerifyLockDigest(lock); err != nil {
		return []string{fmt.Sprintf(
			"plugins.lock: %v — the lock graph may have been hand-edited or corrupted; re-run `strategist install` or `strategist compile` to regenerate it",
			err,
		)}
	}
	return nil
}

func lockParityErrors(lock domain.PluginLockFile, activeSlots map[string]string) []string {
	lockedBySlot := make(map[string]domain.SlotBinding, len(lock.Bindings))
	for _, binding := range lock.Bindings {
		if binding.Slot != "" && binding.InstalledInstanceID != "" {
			lockedBySlot[binding.Slot] = binding
		}
	}

	return mismatchedLockBindings(lockedBySlot, activeSlots)
}

func mismatchedLockBindings(lockedBySlot map[string]domain.SlotBinding, activeSlots map[string]string) []string {
	var errs []string
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		locked, hasLockEntry := lockedBySlot[slot]
		activeProvider := activeSlots[slot]
		if !hasLockEntry || activeProvider == "" || domain.WeaponRefMatchesBinding(activeProvider, locked) {
			continue
		}
		errs = append(errs, parityMessage(slot, activeProvider, locked))
	}
	return errs
}

// parityMessage explains one divergence. A package added with `provider add` is a
// custom binding whose instance id carries a version; install and compile cannot
// choose it for the operator, so the fix is to name it in active.yaml.
func parityMessage(slot, activeProvider string, locked domain.SlotBinding) string {
	if locked.EffectiveMode() == domain.SlotBindingModeCustom && strings.Contains(locked.InstalledInstanceID, "@") {
		return fmt.Sprintf(
			"slot %s: active.yaml configures %q but plugins.lock binds the custom package %q for this slot — set slots.%s to %s in active.yaml (see docs/provider-extension.md)",
			slot, activeProvider, locked.InstalledInstanceID, slot, locked.InstalledInstanceID,
		)
	}
	return fmt.Sprintf(
		"slot %s: active.yaml configures %q but plugins.lock already resolved %q for this slot — re-run `strategist install` or `strategist compile` to reconcile (see docs/runbooks/role-invocation-failed.md)",
		slot, activeProvider, locked.InstalledInstanceID,
	)
}
