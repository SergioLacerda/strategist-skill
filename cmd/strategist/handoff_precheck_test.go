package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func runPrecheck(t *testing.T, root string) (string, error) {
	t.Helper()
	cmd := &cobra.Command{}
	var out strings.Builder
	cmd.SetOut(&out)
	err := runHandoffPrecheck(cmd, handoffPrecheckOptions{Root: root, MissionID: "m1"})
	return out.String(), err
}

func confidenceOf(t *testing.T, root string) []telemetry.ConfidenceRecord {
	t.Helper()
	got, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(root))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	return got
}

// providerClaims keeps only the provider readings: the handoff boundary writes
// its own record for the Archivist summary independently.
func providerClaims(t *testing.T, root string) []telemetry.ConfidenceRecord {
	t.Helper()
	var claims []telemetry.ConfidenceRecord
	for _, record := range confidenceOf(t, root) {
		if strings.HasPrefix(record.CorrelationKey, "provider-conformance-") {
			claims = append(claims, record)
		}
	}
	return claims
}

func gateAnswer(t *testing.T, choice string, confidence float64) []byte {
	t.Helper()
	return answerFor(t, handoffconsumer.TransitionArchivistToGate, choice, confidence)
}

func TestPrecheckReadsThePackageBeforeTheGateAndRecordsNoOutcome(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, nil)

	out, err := runPrecheck(t, root)
	require.NoError(t, err)
	require.Contains(t, out, "transition: archivist_to_gate")
	require.Contains(t, out, "delegation: approved provider=jev model=jev-1.13.0 confidence=0.96 threshold=0.80")
	require.Equal(t, 1, sim.Calls())

	_, err = handoff.NewOutcomeStore(root).LatestFor("m1", handoff.TransitionArchivistToSniper)
	require.Error(t, err, "no lifecycle outcome exists before the gate")

	claims := providerClaims(t, root)
	require.Len(t, claims, 1)
	require.Equal(t, telemetry.ConfidenceAgentHandoffChallenge, claims[0].Agent)
	require.Equal(t, 96, claims[0].ConfidencePercent)
	require.Equal(t, "provider-conformance-archivist_to_gate", claims[0].CorrelationKey)
}

func TestTheLaterEvaluationReusesTheGateReadingAndCarriesItsDelegation(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, nil)
	_, err := runPrecheck(t, root)
	require.NoError(t, err)

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Contains(t, out, "delegation: approved")
	require.Contains(t, out, "reused=true")
	require.Equal(t, 1, sim.Calls(), "the provider is not asked twice for one revision")

	outcome, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.NotNil(t, outcome.Delegation)
	require.Equal(t, outcome.PackageDigest, outcome.Delegation.Subject)
	require.NoError(t, outcome.VerifyIntegrity())
	require.Len(t, providerClaims(t, root), 1, "a reuse is not a second observation")
}

func TestEvaluateWithoutAGateReadingAsksAndRecordsTheSniperClaim(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(sniperAnswer(t, "accept", 0.93)))
	enableIntegration(t, root, sim, nil)

	_, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Equal(t, 1, sim.Calls())
	claims := providerClaims(t, root)
	require.Len(t, claims, 1)
	require.Equal(t, "provider-conformance-archivist_to_sniper", claims[0].CorrelationKey)
	require.Equal(t, 93, claims[0].ConfidencePercent)
}

func TestAFailedReadingLeavesAMissingRecordWithTheReason(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Status(503))
	enableIntegration(t, root, sim, nil)

	out, err := runPrecheck(t, root)
	require.NoError(t, err, "the provider never blocks")
	require.Contains(t, out, "delegation: fallback reason=unavailable path=main")
	claims := providerClaims(t, root)
	require.Len(t, claims, 1)
	require.Equal(t, "missing", claims[0].CoverageStatus)
	require.Contains(t, claims[0].MissingReason, "unavailable")
}

func TestPrecheckWithoutAnIntegrationSaysSoAndWritesNothing(t *testing.T) {
	root := informationalMission(t)
	out, err := runPrecheck(t, root)
	require.NoError(t, err)
	require.Contains(t, out, "delegation: disabled", "the absence is stated, never a blank")
	require.Empty(t, providerClaims(t, root))
}

func TestPrecheckRespectsADisabledProvider(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(nil))
	enableIntegration(t, root, sim, func(p *config.Provider) { p.Enabled = false })
	out, err := runPrecheck(t, root)
	require.NoError(t, err)
	require.Contains(t, out, "delegation: disabled")
	require.Zero(t, sim.Calls())
}

func TestPrecheckNeedsAMissionAndAnInstalledRoot(t *testing.T) {
	_, err := runPrecheck(t, filepath.Join(t.TempDir(), "absent"))
	require.Error(t, err)
	cmd := &cobra.Command{}
	require.ErrorContains(t, runHandoffPrecheck(cmd, handoffPrecheckOptions{}), "--mission-id")
}

func TestAShadowProviderIsCalledAndTheHandoffStaysOnTheMainPath(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(gateAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, func(p *config.Provider) { p.Shadow = true })

	out, err := runPrecheck(t, root)
	require.NoError(t, err)
	require.Contains(t, out, "delegation: shadow path=main reading=recorded would_approve=true")
	require.NotContains(t, out, "delegation: approved")
	require.Equal(t, 1, sim.Calls(), "the provider is still called")
	require.Empty(t, providerClaims(t, root), "a shadow reading never reaches the confidence ledger")

	readings, err := handoffconsumer.Ledger{Path: filepath.Join(root, "memory", ledgerFileName)}.Shadow()
	require.NoError(t, err)
	require.Len(t, readings, 1)
	require.True(t, readings[0].Approved)
}
