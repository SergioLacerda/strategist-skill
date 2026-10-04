//go:build spec

package spec_test

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application/installplan"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

func TestTaxonomyBaselineCharacterizesCommandAndInstallSurfaces(t *testing.T) {
	t.Parallel()

	commandTree := readFile(t, repoRoot(t)+"/docs/generated/command-tree.md")
	for _, command := range []string{"- `start` —", "- `status` —", "- `route` —", "- `record` —", "- `prepare-embedded` —"} {
		require.True(t, strings.Contains(commandTree, command), "command tree must retain %q", command)
	}

	input := installplan.Input{
		Stage:     domain.StageRoster,
		Mode:      "wizard",
		BasePath:  "/workspace",
		Slots:     map[string]string{"discovery": "brainstorming"},
		SlotModes: map[string]string{"discovery": domain.SlotBindingModeCustom},
		Bindings:  []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming"}},
	}
	wizardPlan, err := installplan.PlanInstall(input)
	require.NoError(t, err)
	input.Mode = "headless"
	headlessPlan, err := installplan.PlanInstall(input)
	require.NoError(t, err)
	require.NotEqual(t, wizardPlan.PlanDigest, headlessPlan.PlanDigest)
	require.NoError(t, wizardPlan.Validate())
	require.NoError(t, headlessPlan.Validate())
}

func TestTaxonomyBaselineCharacterizesRouteBindingArtifactsAndTelemetry(t *testing.T) {
	t.Parallel()

	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{Route: domain.MissionRouteFullPipeline, Role: "scout"})
	require.NoError(t, err)
	stageArtifact, err := domain.NewStageResolutionArtifact(resolution, "scout_request")
	require.NoError(t, err)
	event := telemetry.NewStageResolutionEvent("mission-1", stageArtifact)
	require.NoError(t, event.Validate())
	require.Equal(t, domain.StageFull, stageArtifact.Stage)
	require.Equal(t, domain.StageResolutionArtifactSchemaVersion, event.Attributes[telemetry.AttrSchemaVersion])

	bindingArtifact, err := domain.NewWeaponBindingArtifact(domain.SlotBinding{
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Slot:            "discovery", Role: "ranger", InstalledInstanceID: "brainstorming", WeaponVersion: "1.0.0",
		WeaponDigest: "sha256:weapon", SourceDigest: "sha256:source", BindingDigest: "sha256:binding",
		RuntimeKind: "embedded_skill", ConnectorID: "embedded", Entrypoint: "internal_skills/brainstorming/SKILL.md",
		Origin: string(domain.WeaponOriginEmbedded),
	})
	require.NoError(t, err)
	require.NoError(t, bindingArtifact.Validate())
	require.Equal(t, domain.StageRoster, bindingArtifact.Stage)
}
