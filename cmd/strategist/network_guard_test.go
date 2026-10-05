package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
	"github.com/stretchr/testify/require"
)

func TestThePackageStartsWithoutAnyProviderCredentialInTheEnvironment(t *testing.T) {
	for _, name := range []string{"TYPESAFE_API_KEY", "JEV_API_KEY"} {
		require.Empty(t, os.Getenv(name), "%s from the developer shell must never reach a test", name)
	}
}

// A test that enables the integration but forgets to inject a simulated provider
// would have called the real endpoint. The default transport refuses instead, the
// handoff still proceeds on main, and the refusal is counted.
func TestAForgottenSimulatorCannotReachTheRealProvider(t *testing.T) {
	root := informationalMission(t)
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), config.File{
		SchemaVersion: config.SchemaVersion,
		Providers: map[string]config.Provider{"jev": {
			Enabled: true, Endpoint: setup.DefaultEndpoint, CredentialRef: "env:TYPESAFE_API_KEY", Model: setup.DefaultModel,
			AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
			DataPolicy:          config.DataPolicy{AllowedFields: setup.DataCategories(), MaxStateBytes: 16384},
		}},
	}))
	t.Setenv("TYPESAFE_API_KEY", "sk-synthetic-never-sent")
	before := networkGuard.Attempts()

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err, "a refused call never blocks the handoff")
	require.Contains(t, out, "delegation: fallback reason=unavailable path=main")
	require.Greater(t, networkGuard.Attempts(), before, "the attempt reached the guard, not the network")
	require.NotContains(t, out, "sk-synthetic-never-sent")
}

func TestEveryProviderEnabledTestRunsAgainstTheGuardOrASimulator(t *testing.T) {
	require.NotNil(t, delegationRoundTripper, "the production default (nil, a real transport) must never be active in tests")
}
