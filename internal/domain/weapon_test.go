package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestWeaponManifestValidateAtomic(t *testing.T) {
	t.Parallel()

	weapon := domain.WeaponManifest{
		ID:             "brainstorming",
		Version:        "1.0.0",
		Kind:           domain.WeaponKindAtomic,
		Origin:         domain.WeaponOriginEmbedded,
		Roles:          []string{"ranger"},
		SupportedSlots: []string{"discovery"},
		RiskScore:      "write_analysis",
		Runtime: domain.WeaponRuntime{
			Kind:    domain.WeaponRuntimeHost,
			HostAPI: "strategist-host-skill/v1",
		},
	}

	require.NoError(t, weapon.Validate())
	require.NoError(t, weapon.ValidateActive())
}

func TestWeaponManifestValidateRejectsInvalidAffinityAndRuntime(t *testing.T) {
	t.Parallel()

	weapon := domain.WeaponManifest{
		ID:             "broken",
		Kind:           domain.WeaponKindAtomic,
		Origin:         domain.WeaponOriginEmbedded,
		Roles:          []string{"pathfinder"},
		SupportedSlots: []string{"not-a-slot"},
		Runtime:        domain.WeaponRuntime{Kind: domain.WeaponRuntimeNone},
	}

	require.ErrorContains(t, weapon.Validate(), "not approved")
	require.ErrorContains(t, weapon.Validate(), "slot")
	weapon.Roles = []string{"ranger"}
	weapon.SupportedSlots = []string{"discovery"}
	weapon.Version = "1.0.0"
	require.ErrorContains(t, weapon.ValidateActive(), "runtime")
}

func TestWeaponManifestValidateRequiresCompositionForComposite(t *testing.T) {
	t.Parallel()

	weapon := domain.WeaponManifest{
		ID:             "suite",
		Kind:           domain.WeaponKindComposite,
		Origin:         domain.WeaponOriginCustom,
		Roles:          []string{"ranger"},
		SupportedSlots: []string{"discovery"},
		Runtime: domain.WeaponRuntime{
			Kind:    domain.WeaponRuntimeHost,
			HostAPI: "strategist-host-skill/v1",
		},
	}

	require.ErrorContains(t, weapon.Validate(), "composition")
}

func TestWeaponManifestValidateRequiresCanonicalOrigin(t *testing.T) {
	t.Parallel()

	weapon := domain.WeaponManifest{
		ID: "sample", Version: "1.0.0", Kind: domain.WeaponKindAtomic,
		Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"},
		Runtime: domain.WeaponRuntime{Kind: domain.WeaponRuntimeEmbedded},
	}

	require.ErrorContains(t, weapon.Validate(), "origin")
	weapon.Origin = "skill"
	require.ErrorContains(t, weapon.Validate(), `origin "skill" must be embedded or custom`)
	for _, origin := range []domain.WeaponOrigin{domain.WeaponOriginEmbedded, domain.WeaponOriginCustom} {
		weapon.Origin = origin
		require.NoError(t, weapon.ValidateActive())
	}
}

func TestWeaponRuntimeIsIndependentOfOrigin(t *testing.T) {
	t.Parallel()

	kinds := map[domain.WeaponRuntime]bool{
		{Kind: domain.WeaponRuntimeEmbedded}:                                  true,
		{Kind: domain.WeaponRuntimeHost, HostAPI: "strategist-host-skill/v1"}: true,
		{Kind: domain.WeaponRuntimeExecutable, Entrypoint: "bin/tool"}:        true,
	}
	for runtime := range kinds {
		for _, origin := range []domain.WeaponOrigin{domain.WeaponOriginEmbedded, domain.WeaponOriginCustom} {
			weapon := domain.WeaponManifest{
				ID: "sample", Version: "1.0.0", Kind: domain.WeaponKindAtomic, Origin: origin,
				Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"}, Runtime: runtime,
			}
			require.NoError(t, weapon.ValidateActive(), "origin=%s runtime=%s", origin, runtime.Kind)
		}
	}
}

func TestWeaponRuntimeRejectsLegacyKindsWithMigrationDiagnostic(t *testing.T) {
	t.Parallel()

	for _, legacy := range []string{"embedded_skill", "host_skill"} {
		err := domain.WeaponRuntime{Kind: legacy, HostAPI: "strategist-host-skill/v1"}.Validate()
		require.ErrorContains(t, err, legacy)
		require.ErrorContains(t, err, "regenerate or reinstall")
		require.ErrorContains(t, domain.ValidateActiveRuntimeKind(legacy), "regenerate or reinstall")
	}
	require.ErrorContains(t, domain.ValidateActiveRuntimeKind("none"), "invocable runtime")
	require.ErrorContains(t, domain.ValidateActiveRuntimeKind("teleport"), "unsupported")
}
