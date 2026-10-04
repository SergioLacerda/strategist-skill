package installplan_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application/installplan"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPlanInstallProducesValidatedReadOnlyPlan(t *testing.T) {
	t.Parallel()

	plan, err := installplan.PlanInstall(installplan.Input{
		Stage: domain.StageRoster, Mode: "silent", BasePath: "/workspace",
		Slots:     map[string]string{"discovery": "brainstorming"},
		SlotModes: map[string]string{"discovery": domain.SlotBindingModeRanked},
		Lock:      domain.PluginLock{}, Bindings: []domain.SlotBinding{{
			Slot: "discovery", InstalledInstanceID: "brainstorming",
		}},
	})
	require.NoError(t, err)
	require.NoError(t, plan.Validate())
}
