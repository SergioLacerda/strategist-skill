package roster_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	domainroster "github.com/SergioLacerda/strategist-skill/internal/domain/roster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeaponRosterArtifactIsSortedAndDigestable(t *testing.T) {
	t.Parallel()

	artifact, err := domainroster.NewWeaponRosterArtifact([]domainroster.WeaponRosterEntry{
		{Role: "sniper", Slot: "execution", WeaponID: "sniper", WeaponVersion: "1.0.0", Source: "native_role", Materialization: "active", Compatible: true, Selected: true},
		{Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", Source: "embedded", Materialization: "installed", Compatible: true},
	})
	require.NoError(t, err)
	require.NoError(t, artifact.Validate())
	assert.Equal(t, domain.CanonicalTaxonomyVersion, artifact.TaxonomyVersion)
	assert.Equal(t, "ranger", artifact.Entries[0].Role)
	assert.NotEmpty(t, artifact.Digest)
}

func TestWeaponSelectionArtifactValidatesTaxonomyVersion(t *testing.T) {
	t.Parallel()

	selection := domainroster.WeaponSelectionArtifact{
		SchemaVersion:   domainroster.WeaponSelectionArtifactSchemaVersion,
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Stage:           domain.StageRoster, Role: "ranger", Slot: "discovery", Status: "selected",
	}
	require.NoError(t, selection.Validate())
	selection.TaxonomyVersion = "strategist-taxonomy/v0"
	assert.ErrorContains(t, selection.Validate(), "unsupported taxonomy version")
}

func TestRosterAndSelectionArtifactValidationCoverage(t *testing.T) {
	entry := domainroster.WeaponRosterEntry{Role: "ranger", Slot: "discovery", WeaponID: "brainstorming"}
	artifact, err := domainroster.NewWeaponRosterArtifact([]domainroster.WeaponRosterEntry{entry})
	require.NoError(t, err)
	require.NoError(t, artifact.Validate())

	for _, candidate := range []domainroster.WeaponRosterArtifact{
		{SchemaVersion: "bad", TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster, Entries: []domainroster.WeaponRosterEntry{entry}},
		{SchemaVersion: domainroster.WeaponRosterArtifactSchemaVersion, TaxonomyVersion: "bad", Stage: domain.StageRoster, Entries: []domainroster.WeaponRosterEntry{entry}},
		{SchemaVersion: domainroster.WeaponRosterArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageFull, Entries: []domainroster.WeaponRosterEntry{entry}},
		{SchemaVersion: domainroster.WeaponRosterArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster},
		{SchemaVersion: domainroster.WeaponRosterArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster, Entries: []domainroster.WeaponRosterEntry{entry, entry}},
		{SchemaVersion: domainroster.WeaponRosterArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster, Entries: []domainroster.WeaponRosterEntry{{Role: "", Slot: "discovery", WeaponID: "brainstorming"}}},
	} {
		require.Error(t, candidate.Validate())
	}
	missingDigest := artifact
	missingDigest.Digest = ""
	require.ErrorContains(t, missingDigest.Validate(), "digest is required")
	tampered := artifact
	tampered.Digest = "sha256:tampered"
	require.ErrorContains(t, tampered.Validate(), "digest mismatch")

	validSelection := domainroster.WeaponSelectionArtifact{SchemaVersion: domainroster.WeaponSelectionArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster, Role: "ranger", Slot: "discovery", Status: "selected"}
	require.NoError(t, validSelection.Validate())
	for _, candidate := range []domainroster.WeaponSelectionArtifact{
		{SchemaVersion: "bad", TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster, Role: "ranger", Slot: "discovery", Status: "selected"},
		{SchemaVersion: domainroster.WeaponSelectionArtifactSchemaVersion, TaxonomyVersion: "bad", Stage: domain.StageRoster, Role: "ranger", Slot: "discovery", Status: "selected"},
		{SchemaVersion: domainroster.WeaponSelectionArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageFull, Role: "ranger", Slot: "discovery", Status: "selected"},
		{SchemaVersion: domainroster.WeaponSelectionArtifactSchemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Stage: domain.StageRoster},
	} {
		assert.Error(t, candidate.Validate())
	}
}

func TestWeaponBindingArtifactRejectsSparseBinding(t *testing.T) {
	t.Parallel()

	_, err := domain.NewWeaponBindingArtifact(domain.SlotBinding{Role: "ranger", Slot: "discovery", InstalledInstanceID: "brainstorming"})
	require.ErrorContains(t, err, "weapon_digest")
}

func TestWeaponBindingArtifactRejectsUnknownBindingTaxonomyVersion(t *testing.T) {
	t.Parallel()

	binding := completeBinding("discovery", "ranger", "brainstorming", "1.0.0")
	binding.TaxonomyVersion = "strategist-taxonomy/v0"
	_, err := domain.NewWeaponBindingArtifact(binding)
	assert.ErrorContains(t, err, "unsupported taxonomy version")
}

func TestWeaponBindingArtifactsAreSortedAndMatchPinnedBindings(t *testing.T) {
	t.Parallel()

	bindings := []domain.SlotBinding{
		completeBinding("execution", "sniper", "sniper", "2.0.0"),
		completeBinding("discovery", "ranger", "brainstorming", "1.0.0"),
	}

	artifacts, err := domain.NewWeaponBindingArtifacts(bindings)
	require.NoError(t, err)
	require.NoError(t, domain.ValidateWeaponBindingArtifacts(bindings, artifacts))
	assert.Equal(t, "discovery", artifacts[0].Slot)
	assert.Equal(t, "ranger", artifacts[0].Role)
}

func TestWeaponBindingArtifactsRejectDrift(t *testing.T) {
	t.Parallel()

	binding := completeBinding("discovery", "ranger", "brainstorming", "1.0.0")
	artifacts, err := domain.NewWeaponBindingArtifacts([]domain.SlotBinding{binding})
	require.NoError(t, err)
	artifacts[0].WeaponDigest = "sha256:tampered"

	err = domain.ValidateWeaponBindingArtifacts([]domain.SlotBinding{binding}, artifacts)
	require.ErrorContains(t, err, "does not match")
}

func completeBinding(slot, role, weapon, version string) domain.SlotBinding {
	return domain.SlotBinding{
		SchemaVersion:       "strategist-plugin-binding/v1",
		Slot:                slot,
		InstalledInstanceID: weapon + "@" + version,
		Role:                role,
		WeaponVersion:       version,
		WeaponDigest:        "sha256:weapon-" + weapon,
		SourceDigest:        "sha256:source-" + weapon,
		BindingDigest:       "sha256:binding-" + weapon,
		RuntimeKind:         "host",
		ConnectorID:         "host",
		Entrypoint:          "host.prompt",
		Origin:              string(domain.WeaponOriginCustom),
		Status:              "active",
		Generation:          1,
	}
}
