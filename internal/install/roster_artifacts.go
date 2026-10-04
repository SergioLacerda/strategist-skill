package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domainroster "github.com/SergioLacerda/strategist-skill/internal/domain/roster"
	"github.com/SergioLacerda/strategist-skill/internal/roster"
)

func buildRosterArtifacts(preview RoleProviderMigrationPreview) (domainroster.WeaponRosterArtifact, []domainroster.WeaponSelectionArtifact, error) {
	entries := make([]domainroster.WeaponRosterEntry, 0)
	selections := make([]domainroster.WeaponSelectionArtifact, 0, len(preview.Entries))
	for _, item := range preview.Entries {
		entries = append(entries, rosterEntries(item)...)
		selection, err := rosterSelection(item)
		if err != nil {
			return domainroster.WeaponRosterArtifact{}, nil, fmt.Errorf("build Weapon selection artifact: %w", err)
		}
		selections = append(selections, selection)
	}
	rosterArtifact, plannedSelections, err := roster.Plan(entries, selections)
	if err != nil {
		return domainroster.WeaponRosterArtifact{}, nil, fmt.Errorf("build Weapon roster artifact: %w", err)
	}
	return rosterArtifact, plannedSelections, nil
}

func rosterEntries(item RoleProviderPreviewEntry) []domainroster.WeaponRosterEntry {
	selectedID, selectedVersion := resolvedWeaponIdentity(item)
	entries := make([]domainroster.WeaponRosterEntry, 0, len(item.Candidates))
	for _, candidate := range item.Candidates {
		entries = append(entries, domainroster.WeaponRosterEntry{
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

func rosterSelection(item RoleProviderPreviewEntry) (domainroster.WeaponSelectionArtifact, error) {
	selectedID, selectedVersion := resolvedWeaponIdentity(item)
	selection := domainroster.WeaponSelectionArtifact{
		SchemaVersion: domainroster.WeaponSelectionArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster,
		Role: item.RoleName, Slot: item.Slot, WeaponID: selectedID, WeaponVersion: selectedVersion,
		Status: "selected", Reason: "resolved by explicit Role/Weapon compatibility",
	}
	if item.ResolutionError != "" {
		selection.Status = "blocked"
		selection.Reason = item.ResolutionError
	}
	if err := selection.Validate(); err != nil {
		return domainroster.WeaponSelectionArtifact{}, fmt.Errorf("validate Weapon selection artifact: %w", err)
	}
	return selection, nil
}
