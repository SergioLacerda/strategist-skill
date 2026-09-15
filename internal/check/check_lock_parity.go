package check

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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

	lockedBySlot := make(map[string]string, len(lock.Bindings))
	for _, binding := range lock.Bindings {
		if binding.Slot != "" && binding.InstalledInstanceID != "" {
			lockedBySlot[binding.Slot] = binding.InstalledInstanceID
		}
	}

	var errs []string
	for _, slot := range []string{"discovery", "refinement", "execution"} {
		lockedProvider, hasLockEntry := lockedBySlot[slot]
		activeProvider := activeSlots[slot]
		if !hasLockEntry || activeProvider == "" || lockedProvider == activeProvider {
			continue
		}
		errs = append(errs, fmt.Sprintf(
			"slot %s: active.yaml configures %q but plugins.lock already resolved %q for this slot — re-run `strategist install` or `strategist compile` to reconcile (see docs/runbooks/role-invocation-failed.md)",
			slot, activeProvider, lockedProvider,
		))
	}
	return errs
}
