package roster_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domainroster "github.com/SergioLacerda/strategist-skill/internal/domain/roster"
	"github.com/SergioLacerda/strategist-skill/internal/roster"
	"github.com/stretchr/testify/require"
)

func TestPlanIsReadOnlyAndValidatesSelectionArtifacts(t *testing.T) {
	t.Parallel()

	entries := []domainroster.WeaponRosterEntry{{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming",
		WeaponVersion: "1.0.0", Source: "embedded", Materialization: "embedded",
		Compatible: true, Selected: true,
	}}
	selections := []domainroster.WeaponSelectionArtifact{{
		SchemaVersion:   domainroster.WeaponSelectionArtifactSchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster,
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming",
		WeaponVersion: "1.0.0", Status: "selected",
	}}

	got, gotSelections, err := roster.Plan(entries, selections)
	require.NoError(t, err)
	require.NoError(t, got.Validate())
	require.Equal(t, selections, gotSelections)
	entries[0].WeaponID = "changed-after-plan"
	require.Equal(t, "brainstorming", got.Entries[0].WeaponID)
}

func TestPlanRejectsInvalidSelection(t *testing.T) {
	t.Parallel()

	_, _, err := roster.Plan([]domainroster.WeaponRosterEntry{{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming",
	}}, []domainroster.WeaponSelectionArtifact{{Stage: domain.StageRoster}})
	require.ErrorContains(t, err, "roster: validate selection")
}

func TestPlanRejectsAnInvalidRosterArtifact(t *testing.T) {
	t.Parallel()

	_, _, err := roster.Plan(nil, nil)

	require.ErrorContains(t, err, "roster: build artifact")
}
