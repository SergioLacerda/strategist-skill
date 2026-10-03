package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func buildRosterArtifacts(preview RoleProviderMigrationPreview) (domain.WeaponRosterArtifact, []domain.WeaponSelectionArtifact, error) {
	entries := make([]domain.WeaponRosterEntry, 0)
	selections := make([]domain.WeaponSelectionArtifact, 0, len(preview.Entries))
	for _, item := range preview.Entries {
		entries = append(entries, rosterEntries(item)...)
		selection, err := rosterSelection(item)
		if err != nil {
			return domain.WeaponRosterArtifact{}, nil, fmt.Errorf("build Weapon selection artifact: %w", err)
		}
		selections = append(selections, selection)
	}
	roster, err := domain.NewWeaponRosterArtifact(entries)
	if err != nil {
		return domain.WeaponRosterArtifact{}, nil, fmt.Errorf("build Weapon roster artifact: %w", err)
	}
	return roster, selections, nil
}

func rosterEntries(item RoleProviderPreviewEntry) []domain.WeaponRosterEntry {
	selectedID, selectedVersion := resolvedWeaponIdentity(item)
	entries := make([]domain.WeaponRosterEntry, 0, len(item.Candidates))
	for _, candidate := range item.Candidates {
		entries = append(entries, domain.WeaponRosterEntry{
			Role: item.RoleName, Slot: item.Slot, WeaponID: candidate.ID,
			WeaponVersion: candidate.Version, Source: string(candidate.Source),
			Materialization: string(candidate.Materialization), Compatible: true,
			Selected: candidate.ID == selectedID && candidate.Version == selectedVersion,
		})
	}
	return entries
}

func resolvedWeaponIdentity(item RoleProviderPreviewEntry) (string, string) {
	if item.ResolutionError != "" {
		return "", ""
	}
	return item.Resolved.Provider.ID, item.Resolved.Provider.Version
}

func rosterSelection(item RoleProviderPreviewEntry) (domain.WeaponSelectionArtifact, error) {
	selectedID, selectedVersion := resolvedWeaponIdentity(item)
	selection := domain.WeaponSelectionArtifact{
		SchemaVersion: domain.WeaponSelectionArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster,
		Role: item.RoleName, Slot: item.Slot, WeaponID: selectedID, WeaponVersion: selectedVersion,
		Status: "selected", Reason: "resolved by explicit Role/Weapon compatibility",
	}
	if item.ResolutionError != "" {
		selection.Status = "blocked"
		selection.Reason = item.ResolutionError
	}
	if err := selection.Validate(); err != nil {
		return domain.WeaponSelectionArtifact{}, fmt.Errorf("validate Weapon selection artifact: %w", err)
	}
	return selection, nil
}
