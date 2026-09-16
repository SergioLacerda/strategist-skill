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
		if err := firstResolutionError(preview.Entries); err != nil {
			return err
		}
		return fmt.Errorf("role readiness failed: no complete role/provider binding")
	}
	return validateResolvedRoleEntries(preview.Entries)
}

func firstResolutionError(entries []RoleProviderPreviewEntry) error {
	for _, entry := range entries {
		if entry.ResolutionError != "" {
			return fmt.Errorf("role readiness failed: slot=%s role=%s %s", entry.Slot, entry.RoleName, entry.ResolutionError)
		}
	}
	return nil
}

func validateResolvedRoleEntries(entries []RoleProviderPreviewEntry) error {
	for _, entry := range entries {
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
		if err := validatePersistedRoleEntry(lockFile, entry); err != nil {
			return err
		}
	}
	return nil
}

func validatePersistedRoleEntry(lockFile domain.PluginLockFile, entry RoleProviderPreviewEntry) error {
	if entry.Slot != string(domain.SlotDiscovery) && entry.Slot != string(domain.SlotRefinement) {
		return nil
	}
	count, err := countPersistedRoleBindings(lockFile, entry)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("role readiness failed: slot=%s role=%s persisted binding count=%d, expected 1", entry.Slot, entry.RoleName, count)
	}
	return nil
}

func countPersistedRoleBindings(lockFile domain.PluginLockFile, entry RoleProviderPreviewEntry) (int, error) {
	count := 0
	for _, binding := range lockFile.Bindings {
		if binding.Slot != entry.Slot {
			continue
		}
		count++
		if binding.InstalledInstanceID != entry.Resolved.Provider.ID {
			return count, fmt.Errorf("role readiness failed: slot=%s role=%s persisted binding points to %q, expected %q", entry.Slot, entry.RoleName, binding.InstalledInstanceID, entry.Resolved.Provider.ID)
		}
	}
	return count, nil
}
