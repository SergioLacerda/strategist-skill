package handoffconsumer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	"github.com/SergioLacerda/strategist-skill/internal/integration/transport"
	"github.com/stretchr/testify/require"
)

const syntheticKey = "sk-synthetic-consumer-key"

// answerJSON builds a documented-shape response for the pre-check questions of a
// transition: the aggregate choice plus one noul per criterion.
func answerJSON(t *testing.T, transition, model, choice string, confidence float64, unsatisfied ...string) []byte {
	t.Helper()
	answers := map[string]any{AggregateKey: map[string]any{
		"type": "choice", "choice": choice, "confidence": confidence,
		"probabilities": map[string]float64{OptionAccept: confidence, OptionReject: 1 - confidence},
	}}
	for _, criterion := range Criteria(transition) {
		value := 1.0
		if contains(unsatisfied, criterion.Key) {
			value = 0
		}
		answers[criterion.Key] = map[string]any{"type": "noul", "noul": value}
	}
	raw, err := json.Marshal(map[string]any{"model": model, "answers": answers, "usage": map[string]int{"input_tokens": 120, "output_tokens": 14}})
	require.NoError(t, err)
	return raw
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

type rig struct {
	consumer *Consumer
	sim      *jevtest.Server
	ledger   string
}

func newRig(t *testing.T, handler http.HandlerFunc, mutate func(*config.Provider)) rig {
	t.Helper()
	sim := jevtest.Start(t, handler)
	provider := config.Provider{
		Enabled: true, Endpoint: sim.URL() + "/v1/systemone", CredentialRef: "env:TYPESAFE_API_KEY", Model: "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: Fields(handoff.TransitionRangerToArchivist), MaxStateBytes: 8192},
	}
	if mutate != nil {
		mutate(&provider)
	}
	client, err := transport.New(transport.Config{Endpoint: sim.URL() + "/v1/systemone", RoundTripper: sim.Transport(), MaxRetries: -1, Timeout: 200 * time.Millisecond})
	require.NoError(t, err)
	env := credential.Env{Getenv: func(string) string { return syntheticKey }}
	adapter, err := jev.New(client, func() (credential.Secret, error) { return credential.Resolve(provider.CredentialRef, env) }, jev.Config{Model: "jev-1.13.0"})
	require.NoError(t, err)
	registry, err := integration.NewRegistry(adapter)
	require.NoError(t, err)
	mech := mechanism.New(mechanism.Options{
		Config:   config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": provider}},
		Registry: registry, Breaker: policy.NewBreaker(5, time.Minute, time.Now), Env: env,
	})
	ledger := filepath.Join(t.TempDir(), "memory", "integration-calls.jsonl")
	return rig{sim: sim, ledger: ledger, consumer: &Consumer{
		Mechanism: mech, Provider: provider, Name: "jev", Version: "1", Ledger: Ledger{Path: ledger}, Now: time.Now,
	}}
}

func precheckRequest() Request {
	return Request{MissionID: "m-1", Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:artifact", Source: rangerSource()}
}

func ledgerLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var entry map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &entry))
		lines = append(lines, entry)
	}
	return lines
}

func TestApprovedPrecheckYieldsAValidDelegationBoundToTheSubject(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.95)), nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())

	require.Equal(t, StatusApproved, report.Status)
	require.NotNil(t, report.Delegation)
	require.NoError(t, report.Delegation.Validate())
	require.Equal(t, "sha256:artifact", report.Delegation.Subject)
	require.Equal(t, "jev-1.13.0", report.Delegation.Model)
	require.Equal(t, []string{handoff.CheckInputOutputConformance}, report.Delegation.Checks)
	require.InDelta(t, 0.95, report.Delegation.Confidence, 1e-9)
	require.InDelta(t, 0.90, report.Delegation.Threshold, 1e-9)
	require.Equal(t, 120, report.Delegation.InputTokens)
	require.Equal(t, policy.PathJEV, report.Decision.Effective)

	body := r.sim.LastBody()
	require.Contains(t, body, "Evaluate every refined package.")
	require.NotContains(t, body, "sk-must-never-leave", "AC: a field outside the catalog never reaches the request")
	require.NotContains(t, body, syntheticKey)
}

func TestBelowThresholdSignalsWithRepairHintsAndNoDelegation(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.60, "uncertainties_explicit")), nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())

	require.Equal(t, StatusSignal, report.Status)
	require.Nil(t, report.Delegation)
	require.Equal(t, []string{"uncertainties_explicit"}, hintKeys(report.Hints))
	require.InDelta(t, 0.60, report.Confidence, 1e-9)

	exact := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.90)), nil)
	require.Equal(t, StatusSignal, exact.consumer.Precheck(context.Background(), precheckRequest()).Status, "exactly 0.90 does not approve")

	reject := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionReject, 0.99)), nil)
	require.Equal(t, StatusSignal, reject.consumer.Precheck(context.Background(), precheckRequest()).Status)
}

func TestProviderFailureFallsBackToMainWithTheReason(t *testing.T) {
	slow := func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
	}
	r := newRig(t, slow, nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())

	require.Equal(t, StatusFallback, report.Status)
	require.Equal(t, integration.StateTimeout, report.Reason)
	require.Nil(t, report.Delegation)
	require.Equal(t, 1, r.sim.Calls(), "AC-10: one call and no new handoff attempt")
}

func TestDisabledIntegrationMakesNoCallAndWritesNoLedger(t *testing.T) {
	r := newRig(t, jevtest.Reply(nil), func(p *config.Provider) { p.Enabled = false })
	report := r.consumer.Precheck(context.Background(), precheckRequest())

	require.Equal(t, StatusDisabled, report.Status)
	require.Zero(t, r.sim.Calls())
	require.Empty(t, ledgerLines(t, r.ledger))
}

func TestCallBudgetIsSeparateFromHandoffAttemptsAndPerMissionTransition(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.95)), func(p *config.Provider) { p.CallBudget = 2 })

	require.Equal(t, StatusApproved, r.consumer.Precheck(context.Background(), precheckRequest()).Status)
	require.Equal(t, StatusApproved, r.consumer.Precheck(context.Background(), precheckRequest()).Status)
	third := r.consumer.Precheck(context.Background(), precheckRequest())
	require.Equal(t, StatusFallback, third.Status)
	require.Equal(t, integration.StateRateLimited, third.Reason)
	require.Equal(t, 2, r.sim.Calls(), "the exhausted budget spares the provider")

	other := precheckRequest()
	other.MissionID = "m-2"
	require.Equal(t, StatusApproved, r.consumer.Precheck(context.Background(), other).Status, "another mission has its own budget")
}

func TestDataPolicyDenialMakesNoCallAndDoesNotSpendTheBudget(t *testing.T) {
	r := newRig(t, jevtest.Reply(nil), func(p *config.Provider) {
		p.DataPolicy.AllowedFields = []string{"api_key"}
		p.CallBudget = 1
	})
	report := r.consumer.Precheck(context.Background(), precheckRequest())

	require.Equal(t, StatusFallback, report.Status)
	require.Equal(t, integration.StateDataPolicyDenied, report.Reason)
	require.Zero(t, r.sim.Calls())
	count, err := r.consumer.Ledger.Count("m-1", handoff.TransitionRangerToArchivist)
	require.NoError(t, err)
	require.Zero(t, count, "only attempted calls spend the budget")
}

func TestChangedResolvedModelFallsBackFlaggedSuspect(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-9.9.9", OptionAccept, 0.99)), nil)
	report := r.consumer.Precheck(context.Background(), precheckRequest())

	require.Equal(t, StatusFallback, report.Status)
	require.True(t, report.Decision.Suspect)
	require.Nil(t, report.Delegation, "an answer from a binding that failed integrity is never used")
}

func TestLedgerRecordsPathReasonUsageAndNeverContentOrSecrets(t *testing.T) {
	r := newRig(t, jevtest.Reply(answerJSON(t, handoff.TransitionRangerToArchivist, "jev-1.13.0", OptionAccept, 0.95)), nil)
	r.consumer.Precheck(context.Background(), precheckRequest())

	lines := ledgerLines(t, r.ledger)
	require.Len(t, lines, 1)
	entry := lines[0]
	require.Equal(t, "m-1", entry["mission_id"])
	require.Equal(t, handoff.TransitionRangerToArchivist, entry["transition"])
	require.Equal(t, "jev", entry["preferred_path"])
	require.Equal(t, "jev", entry["effective_path"])
	require.Equal(t, true, entry["approved"])
	require.EqualValues(t, 120, entry["input_tokens"])
	require.NotEmpty(t, entry["binding"])
	raw, err := os.ReadFile(r.ledger)
	require.NoError(t, err)
	require.NotContains(t, string(raw), syntheticKey)
	require.NotContains(t, string(raw), "Evaluate every refined package.", "the ledger records metadata, never artifact content")
}
