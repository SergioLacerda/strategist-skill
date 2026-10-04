package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/stretchr/testify/require"
)

const delegationKey = "sk-synthetic-cmd-key"

// sniperAnswer is a documented-shape provider answer for the Archivist-to-Sniper pre-check.
func sniperAnswer(t *testing.T, choice string, confidence float64) []byte {
	t.Helper()
	answers := map[string]any{handoffconsumer.AggregateKey: map[string]any{
		"type": "choice", "choice": choice, "confidence": confidence,
		"probabilities": map[string]float64{"accept": confidence, "reject": 1 - confidence},
	}}
	for _, criterion := range handoffconsumer.Criteria(handoff.TransitionArchivistToSniper) {
		answers[criterion.Key] = map[string]any{"type": "noul", "noul": 1.0}
	}
	raw, err := json.Marshal(map[string]any{"model": "jev-1.13.0", "answers": answers, "usage": map[string]int{"input_tokens": 80, "output_tokens": 9}})
	require.NoError(t, err)
	return raw
}

// enableIntegration points the operator file at a simulated provider.
func enableIntegration(t *testing.T, root string, sim *jevtest.Server, mutate func(*config.Provider)) {
	t.Helper()
	provider := config.Provider{
		Enabled: true, Endpoint: sim.URL() + "/v1/systemone", CredentialRef: "env:TYPESAFE_API_KEY", Model: "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: handoffconsumer.Fields(handoff.TransitionArchivistToSniper), MaxStateBytes: 16384},
	}
	if mutate != nil {
		mutate(&provider)
	}
	require.NoError(t, config.Save(filepath.Join(root, config.FileName), config.File{
		SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": provider},
	}))
	t.Setenv("TYPESAFE_API_KEY", delegationKey)
	delegationRoundTripper = sim.Transport()
	t.Cleanup(func() { delegationRoundTripper = nil })
}

func informationalMission(t *testing.T) string {
	t.Helper()
	return evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
}

func TestEvaluateWithoutIntegrationIsUnchanged(t *testing.T) {
	root := informationalMission(t)
	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.NotContains(t, out, "delegation:", "no integration configured prints nothing extra")

	outcome, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.Nil(t, outcome.Delegation)
}

func TestEvaluateRecordsAnApprovedDelegationAndTellsTheAgent(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(sniperAnswer(t, "accept", 0.96)))
	enableIntegration(t, root, sim, nil)

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Contains(t, out, "delegation: approved provider=jev model=jev-1.13.0 confidence=0.96 threshold=0.90")

	outcome, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.NotNil(t, outcome.Delegation)
	require.Equal(t, outcome.PackageDigest, outcome.Delegation.Subject)
	require.NoError(t, outcome.VerifyIntegrity())
	require.Equal(t, 1, sim.Calls())
	require.Equal(t, "Bearer "+delegationKey, sim.LastAuthorization())

	ledger, err := os.ReadFile(filepath.Join(root, "memory", "integration-calls.jsonl"))
	require.NoError(t, err)
	require.NotContains(t, string(ledger), delegationKey)
}

func TestEvaluateSignalsBelowThresholdAndRecordsNoDelegation(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(sniperAnswer(t, "accept", 0.70)))
	enableIntegration(t, root, sim, nil)

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err, "a low confidence never blocks the handoff")
	require.Contains(t, out, "delegation: signal confidence=0.70 threshold=0.90 path=main")

	outcome, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.Nil(t, outcome.Delegation)
}

func TestEvaluateFallsBackToMainWhenTheProviderFails(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Status(http.StatusServiceUnavailable))
	enableIntegration(t, root, sim, nil)

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err, "AC-10: the handoff proceeds without a new attempt")
	require.Contains(t, out, "delegation: fallback reason=unavailable path=main")
	require.Contains(t, out, "outcome: ")
	outcome, err := handoff.NewOutcomeStore(root).Latest("m1")
	require.NoError(t, err)
	require.Equal(t, 1, outcome.Attempt)
}

func TestEvaluateWithADisabledIntegrationMakesNoCall(t *testing.T) {
	root := informationalMission(t)
	sim := jevtest.Start(t, jevtest.Reply(nil))
	enableIntegration(t, root, sim, func(p *config.Provider) { p.Enabled = false })

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.NotContains(t, out, "delegation:")
	require.Zero(t, sim.Calls())
}

func TestAnUnreadableIntegrationFileFallsBackAndNeverBlocks(t *testing.T) {
	root := informationalMission(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, config.FileName), []byte("schema_version: bogus\nproviders: {}\n"), 0o600))

	out, err := runEvaluate(t, handoffEvaluateOptions{Root: root, MissionID: "m1"})
	require.NoError(t, err)
	require.Contains(t, out, "delegation: fallback reason=incompatible_contract path=main")
}

func TestRangerDelegateCallbackPreChecksTheArtifact(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	basePath := filepath.Join(filepath.Dir(root), ".analysis")
	pending := filepath.Join(basePath, "pending")
	require.NoError(t, os.MkdirAll(pending, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pending, "m1-analysis.md"), []byte("---\nmission_id: m1\n---\n\n## mission_objective\nEvaluate.\n## known_facts\n- one\n"), 0o644))
	rangerAnswers := map[string]any{handoffconsumer.AggregateKey: map[string]any{
		"type": "choice", "choice": "accept", "confidence": 0.97, "probabilities": map[string]float64{"accept": 0.97, "reject": 0.03},
	}}
	for _, criterion := range handoffconsumer.Criteria(handoff.TransitionRangerToArchivist) {
		rangerAnswers[criterion.Key] = map[string]any{"type": "noul", "noul": 1.0}
	}
	raw, err := json.Marshal(map[string]any{"model": "jev-1.13.0", "answers": rangerAnswers, "usage": map[string]int{"input_tokens": 5, "output_tokens": 1}})
	require.NoError(t, err)
	sim := jevtest.Start(t, jevtest.Reply(raw))
	enableIntegration(t, root, sim, func(p *config.Provider) {
		p.DataPolicy.AllowedFields = handoffconsumer.Fields(handoff.TransitionRangerToArchivist)
	})

	delegation := rangerDelegate(context.Background(), root, basePath, "m1")("ignored")
	require.NotNil(t, delegation)
	digest, err := handoff.RangerArtifactDigest(filepath.Join(pending, "m1-analysis.md"))
	require.NoError(t, err)
	require.Equal(t, digest, delegation.Subject, "the delegation is bound to the artifact actually read")
}
