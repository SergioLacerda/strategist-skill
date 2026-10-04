package handoffconsumer

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/stretchr/testify/require"
)

func rangerSource() MapSource {
	return MapSource{
		"objective":                    "Evaluate every refined package.",
		"known_facts":                  "Six packages were present.",
		"uncertainties":                "Whether four close together.",
		"recommended_refinement_focus": "Preserve per-package verdicts.",
		"api_key":                      "sk-must-never-leave",
		"unrelated_section":            "internal notes",
	}
}

func TestProjectIncludesOnlyAllowedCatalogFields(t *testing.T) {
	state, err := Project(handoff.TransitionRangerToArchivist, rangerSource(), []string{"objective", "known_facts", "api_key"}, 4096)
	require.NoError(t, err)

	require.Contains(t, state, "Evaluate every refined package.")
	require.Contains(t, state, "Six packages were present.")
	require.NotContains(t, state, "Whether four close together.", "a catalog field the policy does not allow never leaves")
	require.NotContains(t, state, "sk-must-never-leave", "a field outside the catalog never leaves even when the policy lists it")
	require.NotContains(t, state, "internal notes")
}

func TestProjectNeverEmitsAFieldOutsideTheCatalogOrPolicy(t *testing.T) {
	for _, transition := range transitions {
		allowed := Fields(transition)
		source := MapSource{"leaked": "LEAK"}
		for _, field := range allowed {
			source[field] = "value of " + field
		}
		state, err := Project(transition, source, append(allowed, "leaked"), 1<<20)
		require.NoError(t, err)
		require.NotContains(t, state, "LEAK")
		for _, field := range allowed {
			require.Contains(t, state, "value of "+field)
		}
	}
}

func TestProjectDeniesWhenNothingMayBeSentOrTheStateIsTooLarge(t *testing.T) {
	_, err := Project(handoff.TransitionRangerToArchivist, rangerSource(), []string{"api_key"}, 4096)
	state, ok := integration.StateOf(err)
	require.True(t, ok)
	require.Equal(t, integration.StateDataPolicyDenied, state)

	big := MapSource{"objective": strings.Repeat("x", 5000)}
	_, err = Project(handoff.TransitionRangerToArchivist, big, []string{"objective"}, 1000)
	state, _ = integration.StateOf(err)
	require.Equal(t, integration.StateDataPolicyDenied, state, "an oversized state is denied, never silently truncated")
}
