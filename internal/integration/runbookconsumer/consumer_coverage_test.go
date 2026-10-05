package runbookconsumer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	preparerunbook "github.com/SergioLacerda/strategist-skill/internal/tools/prepare_runbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type localAdapter struct {
	result integration.Result
	err    error
}

func (a localAdapter) Identity() integration.Identity {
	return integration.Identity{
		Provider: "jev", Version: "1", Contract: integration.ContractVersion,
		Capabilities: []integration.Capability{integration.CapConfidenceEvaluate},
	}
}

func (a localAdapter) Call(context.Context, integration.Request) (integration.Result, error) {
	return a.result, a.err
}

func localProvider() config.Provider {
	return config.Provider{
		Enabled: true, CredentialRef: "env:RUNBOOK_KEY", Model: "local-model",
		AllowedCapabilities: []integration.Capability{integration.CapConfidenceEvaluate},
		DataPolicy:          config.DataPolicy{AllowedFields: []string{"objective", "verification", "comparison"}, MaxStateBytes: 4096},
		ApprovalThreshold:   0.80, CallBudget: 2,
	}
}

func localConsumer(t *testing.T, ledgerPath string, provider config.Provider) *Consumer {
	t.Helper()
	adapter, err := integration.NewRegistry(localAdapter{result: integration.Result{
		Model: "local-model",
		Answers: map[string]integration.Answer{AggregateKey: {
			Kind: integration.KindChoice, Choice: OptionSupport,
			HasConfidence: true, Confidence: 0.95,
		}},
	}})
	require.NoError(t, err)
	mech := mechanism.New(mechanism.Options{
		Config:   config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": provider}},
		Registry: adapter, Env: credential.Env{Getenv: func(string) string { return "secret" }},
		Breaker: policy.NewBreaker(3, time.Minute, time.Now),
	})
	return &Consumer{Mechanism: mech, Provider: provider, Ledger: handoffconsumer.Ledger{Path: ledgerPath}}
}

func providerRequest() preparerunbook.ProviderRequest {
	return preparerunbook.ProviderRequest{
		MissionID: "mission-1", Subject: "sha256:subject",
		Fields: map[string]string{"objective": "recover", "verification": "green", "secret": "never"},
	}
}

func TestEvaluateCoversDisabledProjectionAndSuccessfulLedgerPaths(t *testing.T) {
	request := providerRequest()
	disabled := &Consumer{Provider: config.Provider{Enabled: false}}
	reading, err := disabled.Evaluate(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, string(integration.StateDisabled), reading.Reason)

	projectDenied := localProvider()
	projectDenied.DataPolicy.AllowedFields = nil
	reading, err = localConsumer(t, "", projectDenied).Evaluate(context.Background(), request)
	require.NoError(t, err)
	assert.Empty(t, reading.Reason)

	ledger := filepath.Join(t.TempDir(), "memory", "calls.jsonl")
	consumer := localConsumer(t, ledger, localProvider())
	reading, err = consumer.Evaluate(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, preparerunbook.ProviderPathJEV, reading.Path)
	assert.True(t, reading.Approved)
	require.FileExists(t, ledger)
	count, err := consumer.Ledger.Count(request.MissionID, Transition)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRateLimitedAndRecordOutcomeFailureBranches(t *testing.T) {
	request := providerRequest()
	assert.Nil(t, (&Consumer{}).rateLimited(request.MissionID))

	broken := &Consumer{Provider: localProvider(), Ledger: handoffconsumer.Ledger{Path: t.TempDir()}}
	assert.Equal(t, "ledger_unavailable", broken.rateLimited(request.MissionID).Reason)

	ledgerPath := filepath.Join(t.TempDir(), "calls.jsonl")
	ledger := handoffconsumer.Ledger{Path: ledgerPath}
	require.NoError(t, ledger.Append(handoffconsumer.Record{MissionID: request.MissionID, Transition: Transition, Attempted: true}))
	limited := &Consumer{Provider: config.Provider{CallBudget: 1}, Ledger: ledger}
	assert.Equal(t, string(integration.StateRateLimited), limited.rateLimited(request.MissionID).Reason)

	below := &Consumer{Provider: config.Provider{CallBudget: 2}, Ledger: ledger}
	assert.Nil(t, below.rateLimited(request.MissionID))
	(&Consumer{}).recordOutcome(request, mechanism.Outcome{}, preparerunbook.ProviderReading{})
	(&Consumer{Ledger: handoffconsumer.Ledger{Path: t.TempDir()}}).recordOutcome(request, mechanism.Outcome{}, preparerunbook.ProviderReading{})
}

func TestProjectionAndReadingHelpersCoverFallbackShapes(t *testing.T) {
	request := providerRequest()
	state, err := project(request, []string{"objective", "verification"}, 4096)
	require.NoError(t, err)
	assert.Contains(t, state, "## objective")
	assert.NotContains(t, state, "secret")
	_, err = project(request, nil, 4096)
	require.Error(t, err)
	_, err = project(request, []string{"objective"}, 1)
	require.Error(t, err)

	base := preparerunbook.ProviderReading{Threshold: 0.8}
	assert.Equal(t, string(integration.StateUnavailable), unavailableReading(base, mechanism.Outcome{}).Reason)
	assert.Equal(t, "timeout", unavailableReading(base, mechanism.Outcome{Decision: policy.Decision{Reason: integration.StateTimeout}}).Reason)
	assert.Equal(t, string(integration.StateInvalidResponse), invalidReading(base).Reason)

	for _, tc := range []struct {
		name       string
		answer     integration.Answer
		confidence float64
	}{
		{"probability", integration.Answer{Probabilities: map[string]float64{OptionSupport: 0.7}, HasConfidence: true}, 0.7},
		{"choice confidence", integration.Answer{Choice: OptionSupport, Confidence: 0.7, HasConfidence: true}, 0.7},
		{"reversed confidence", integration.Answer{Choice: OptionReview, Confidence: 0.7, HasConfidence: true}, 0.3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.confidence, answerConfidence(tc.answer), 1e-9)
		})
	}

	reading := readingFromOutcome(mechanism.Outcome{Decision: policy.Decision{Binding: "digest"}}, 0.8)
	assert.Equal(t, string(integration.StateUnavailable), reading.Reason)
	reading = readingFromOutcome(mechanism.Outcome{Result: &integration.Result{Answers: map[string]integration.Answer{AggregateKey: {Kind: integration.KindNoul}}}}, 0.8)
	assert.Equal(t, string(integration.StateInvalidResponse), reading.Reason)
	reading = readingFromOutcome(mechanism.Outcome{Result: &integration.Result{Answers: map[string]integration.Answer{AggregateKey: {Kind: integration.KindChoice, Choice: OptionReject, HasConfidence: true, Confidence: 0.9}}}}, 0.8)
	assert.False(t, reading.Approved)
	assert.Equal(t, "local_threshold_not_cleared", reading.Reason)

	assert.Equal(t, integration.State(""), readingReason(preparerunbook.ProviderReading{}))
	assert.Equal(t, integration.StateTimeout, readingReason(preparerunbook.ProviderReading{Reason: string(integration.StateTimeout)}))
	assert.True(t, contains([]string{"objective"}, "objective"))
	assert.False(t, contains(nil, "objective"))
	assert.Equal(t, "fallback", fallback("fallback").Reason)
}

func TestBindingDefaultsAndRecordOutcomeErrorsAreSafe(t *testing.T) {
	binding := (&Consumer{Provider: config.Provider{Model: "model"}}).binding()
	assert.Equal(t, "jev", binding.Provider)
	assert.Equal(t, "1", binding.Version)
	custom := (&Consumer{Name: "custom", Version: "2", Provider: config.Provider{Model: "model"}}).binding()
	assert.Equal(t, "custom", custom.Provider)
	assert.Equal(t, "2", custom.Version)

	path := filepath.Join(t.TempDir(), "ledger")
	require.NoError(t, os.Mkdir(path, 0o755))
	consumer := &Consumer{Ledger: handoffconsumer.Ledger{Path: path}}
	consumer.recordOutcome(providerRequest(), mechanism.Outcome{Result: &integration.Result{Model: "model"}}, preparerunbook.ProviderReading{})
}
