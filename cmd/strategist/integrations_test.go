package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const doctorKey = "sk-doctor-synthetic-key"
const probeBody = `{"model":"jev-1.13.0","answers":{"probe":{"type":"noul","noul":1.0}},"usage":{"input_tokens":12,"output_tokens":3}}`

func bareRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	testutil.MinimalRoot(t, root)
	return root
}

func runIntegrations(t *testing.T, fn func(*cobra.Command, integrationsOptions) error, opts integrationsOptions) (string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := fn(cmd, opts)
	return out.String(), err
}

func writeDotenv(t *testing.T, root string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), ".env"), []byte("TYPESAFE_API_KEY="+doctorKey+"\n"), mode))
}

func TestDoctorWithoutAnyDecisionReportsNotDeclaredAndUnknown(t *testing.T) {
	root := bareRoot(t)
	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, out, "declared: false")
	require.Contains(t, out, "enabled: false")
	require.Contains(t, out, "available: unknown")
	require.NotContains(t, out, "available: available")
}

func TestDoctorNeverReportsAvailableWithoutARecordedObservation(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Reply([]byte(probeBody)))
	enableIntegration(t, root, sim, nil)
	t.Setenv("TYPESAFE_API_KEY", doctorKey)

	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, out, "declared: true")
	require.Contains(t, out, "enabled: true")
	require.Contains(t, out, "bound: true")
	require.Contains(t, out, "compatible: true")
	require.Contains(t, out, "available: unknown", "a configured and credentialed provider is still unknown until probed")
	require.Zero(t, sim.Calls(), "doctor never calls the provider")
}

func TestProbeThenDoctorReportsAvailabilityWithObservationAndValidity(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Reply([]byte(probeBody)))
	enableIntegration(t, root, sim, nil)
	t.Setenv("TYPESAFE_API_KEY", doctorKey)

	out, err := runIntegrations(t, runIntegrationsProbe, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, out, "probe: ok model=jev-1.13.0")
	require.NotContains(t, sim.LastBody(), "objective", "AC: a probe sends no mission content")
	require.Contains(t, sim.LastBody(), probe.SyntheticState)

	doctor, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, doctor, "available: available")
	require.Contains(t, doctor, "resolved_model: jev-1.13.0")
	require.Contains(t, doctor, "observed_at: ")
	require.Contains(t, doctor, "valid_until: ")
}

func TestAConfigurationChangeInvalidatesThePreviousObservation(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Reply([]byte(probeBody)))
	enableIntegration(t, root, sim, nil)
	t.Setenv("TYPESAFE_API_KEY", doctorKey)
	_, err := runIntegrations(t, runIntegrationsProbe, integrationsOptions{Root: root})
	require.NoError(t, err)

	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	changed := file.Providers["jev"]
	changed.DataPolicy.MaxStateBytes = 999
	file.Providers["jev"] = changed
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), file))

	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, out, "available: unknown", "a data-policy change invalidates the observation")
}

func TestAFailedProbeIsRecordedAndReportedUnavailable(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Status(401))
	enableIntegration(t, root, sim, nil)
	t.Setenv("TYPESAFE_API_KEY", doctorKey)

	out, err := runIntegrations(t, runIntegrationsProbe, integrationsOptions{Root: root})
	require.Error(t, err)
	require.Contains(t, out, "probe: failed state=authentication_failed")
	doctor, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, doctor, "available: unavailable")
}

func TestProbeRefusesADisabledOrCredentiallessProvider(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Reply([]byte(probeBody)))
	enableIntegration(t, root, sim, nil)
	t.Setenv("TYPESAFE_API_KEY", "")

	_, err := runIntegrations(t, runIntegrationsProbe, integrationsOptions{Root: root})
	require.ErrorContains(t, err, "credential_missing", "no credential, no call")
	require.Zero(t, sim.Calls())

	enableIntegration(t, root, sim, func(p *config.Provider) { p.Enabled = false })
	t.Setenv("TYPESAFE_API_KEY", doctorKey)
	_, err = runIntegrations(t, runIntegrationsProbe, integrationsOptions{Root: root})
	require.ErrorContains(t, err, "disabled")
	require.Zero(t, sim.Calls())
}

func TestDoctorReportsDotenvHygieneAndNeverTheSecret(t *testing.T) {
	root := bareRoot(t)
	sim := jevtest.Start(t, jevtest.Reply([]byte(probeBody)))
	enableIntegration(t, root, sim, func(p *config.Provider) { p.CredentialRef = "dotenv:.env#TYPESAFE_API_KEY" })
	t.Setenv("TYPESAFE_API_KEY", "")
	writeDotenv(t, root, 0o644)

	out, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root})
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		require.Contains(t, out, "dotenv_loose_permissions")
	}
	require.NotContains(t, out, doctorKey)
	require.Contains(t, out, "credential: resolves")

	asJSON, err := runIntegrations(t, runIntegrationsDoctor, integrationsOptions{Root: root, JSON: true})
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(asJSON), &parsed))
	require.Equal(t, "unknown", parsed["available"])
	require.NotContains(t, asJSON, doctorKey)
}

func TestEnableRecordsADecisionAndWarnsWhenTheCredentialIsPending(t *testing.T) {
	root := bareRoot(t)
	t.Setenv("TYPESAFE_API_KEY", "")
	out, err := runIntegrations(t, runIntegrationsEnable, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, out, "integration: enabled")
	require.Contains(t, out, "credential: pending", "WIZ-07: the missing credential is saved as a pending intent")
	require.Contains(t, out, "endpoint: https://api.typesafe.ai/v1/systemone")
	require.Contains(t, out, "data: ")

	file, found, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, file.Providers["jev"].Enabled)
}

func TestDisableBlocksNewCallsButPreservesOutcomesAndTheProviderSettings(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(sniperAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, nil)

	_, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Equal(t, 1, sim.Calls())
	before, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.NotNil(t, before.Delegation)

	out, err := runIntegrations(t, runIntegrationsDisable, integrationsOptions{Root: root})
	require.NoError(t, err)
	require.Contains(t, out, "integration: disabled")

	after, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.NoError(t, after.VerifyIntegrity(), "WIZ-08: a recorded outcome stays valid")
	require.Equal(t, before.Integrity, after.Integrity)
	file, _, err := config.Load(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	require.False(t, file.Providers["jev"].Enabled)
	require.Equal(t, "jev-1.13.0", file.Providers["jev"].Model, "settings are preserved")

	// A new handoff under the same (now disabled) decision makes no call.
	next := informationalMission(t)
	raw, err := os.ReadFile(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(next, config.FileName), raw, 0o600))
	out, err = runEvaluate(t, handoffEvaluateOptions{Root: next, MissionID: "m1"})
	require.NoError(t, err)
	require.NotContains(t, out, "delegation:")
	require.Equal(t, 1, sim.Calls(), "a disabled provider is never called again")
}
