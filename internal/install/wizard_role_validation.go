package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// validateWizardRoleBindings makes missing role/provider bindings a wizard
// error instead of allowing a partial migration to look like a successful
// installation. Runtime file validation is repeated by strategist check after
// the generated files have been written.
func validateWizardRoleBindings(preview RoleProviderMigrationPreview) error {
	if !preview.FullyResolved() {
		for _, entry := range preview.Entries {
			if entry.ResolutionError != "" {
				return fmt.Errorf("role readiness failed: slot=%s role=%s %s", entry.Slot, entry.RoleName, entry.ResolutionError)
			}
		}
		return fmt.Errorf("role readiness failed: no complete role/provider binding")
	}
	for _, entry := range preview.Entries {
		if entry.Slot != string(domain.SlotDiscovery) && entry.Slot != string(domain.SlotRefinement) {
			continue
		}
		if entry.Resolved.Provider.ID == "" || !entry.Resolved.Compatibility.Compatible {
			return fmt.Errorf("role readiness failed: slot=%s role=%s has no compatible weapon", entry.Slot, entry.RoleName)
		}
	}
	return nil
}

func validatePersistedRoleBindings(lockFile domain.PluginLockFile, preview RoleProviderMigrationPreview) error {
	for _, entry := range preview.Entries {
		if entry.Slot != string(domain.SlotDiscovery) && entry.Slot != string(domain.SlotRefinement) {
			continue
		}
		count := 0
		for _, binding := range lockFile.Bindings {
			if binding.Slot != entry.Slot {
				continue
			}
			count++
			if binding.InstalledInstanceID != entry.Resolved.Provider.ID {
				return fmt.Errorf("role readiness failed: slot=%s role=%s persisted binding points to %q, expected %q", entry.Slot, entry.RoleName, binding.InstalledInstanceID, entry.Resolved.Provider.ID)
			}
		}
		if count != 1 {
			return fmt.Errorf("role readiness failed: slot=%s role=%s persisted binding count=%d, expected 1", entry.Slot, entry.RoleName, count)
		}
	}
	return nil
}
