//go:build live

package live

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
	"github.com/SergioLacerda/strategist-skill/internal/integration/policy"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
	"github.com/SergioLacerda/strategist-skill/internal/integration/transport"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

const liveTimeout = 90 * time.Second

type rig struct {
	provider config.Provider
	secret   credential.Secret
	adapter  *jev.Adapter
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "go.mod not found above the working directory")
		dir = parent
	}
}

// newRig resolves the credential from the workspace .env and builds the real
// transport. The credential is only ever sent to the documented endpoint.
func newRig(t *testing.T) rig {
	t.Helper()
	if os.Getenv("STRATEGIST_LIVE_CONSENT") != "yes" {
		t.Skip("live checks call the real provider: set STRATEGIST_LIVE_CONSENT=yes, or use `strategist dojo calibrate jev-live --consent`")
	}
	require.Equal(t, "https://api.typesafe.ai/v1/systemone", setup.DefaultEndpoint, "the credential must only be sent to the documented endpoint")
	secret, err := credential.Resolve(setup.DefaultCredentialRef, credential.WorkspaceEnv(repoRoot(t)))
	if err != nil {
		t.Skipf("no credential resolves from %s: the live check needs the workspace .env", setup.DefaultCredentialRef)
	}
	provider := config.Provider{
		Enabled: true, Endpoint: setup.DefaultEndpoint, CredentialRef: setup.DefaultCredentialRef, Model: "jev-latest", AllowModelAlias: true,
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: setup.DataCategories(), MaxStateBytes: 32768},
	}
	client, err := transport.New(transport.Config{Endpoint: provider.Endpoint, Timeout: liveTimeout, MaxRetries: -1})
	require.NoError(t, err)
	adapter, err := jev.New(client, func() (credential.Secret, error) { return secret, nil }, jev.Config{Model: provider.Model, AllowAlias: true})
	require.NoError(t, err)
	return rig{provider: provider, secret: secret, adapter: adapter}
}

// leaks reports whether text contains the credential. It deliberately returns a
// bool so a failure never prints the text (and with it the credential).
func (r rig) leaks(text string) bool { return strings.Contains(text, r.secret.Reveal()) }

// diagnose repeats the documented request raw, only to show the status and the
// provider's own error body when a check fails, with the credential redacted.
func (r rig) diagnose(t *testing.T) {
	t.Helper()
	body := `{"state":"` + probe.SyntheticState + `","model":"jev-latest","questions":{"probe":{"type":"noul","instructions":"Is this message a synthetic connectivity check?"}}}`
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, r.provider.Endpoint, strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer "+r.secret.Reveal())
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: liveTimeout}).Do(request)
	if err != nil {
		t.Logf("diagnostic: the raw request failed before a response: %s", strings.ReplaceAll(err.Error(), r.secret.Reveal(), "[redacted]"))
		return
	}
	defer func() { _ = response.Body.Close() }()
	buffer := make([]byte, 2048)
	n, _ := response.Body.Read(buffer)
	t.Logf("diagnostic: status=%d body=%s", response.StatusCode, strings.ReplaceAll(string(buffer[:n]), r.secret.Reveal(), "[redacted]"))
}

func TestLiveProbeAuthenticatesAndReturnsTheDocumentedShape(t *testing.T) {
	r := newRig(t)
	record := probe.Run(context.Background(), r.provider, r.adapter, time.Now)
	if record.State != "ok" {
		r.diagnose(t)
	}
	require.Equal(t, "ok", record.State, "the provider must accept the credential and answer in the documented shape")
	require.Regexp(t, resolvedModel, record.Model, "the provider reports the model that served the call")
	require.Positive(t, record.InputTokens, "usage is reported")
	t.Logf("live probe ok: resolved model=%s input_tokens=%d output_tokens=%d latency_ms=%d", record.Model, record.InputTokens, record.OutputTokens, record.LatencyMS)
}

var wellFormed = handoffconsumer.MapSource(WellFormed)

var poor = handoffconsumer.MapSource(Poor)

func (r rig) consumer(t *testing.T) (*handoffconsumer.Consumer, string) {
	t.Helper()
	env := credential.WorkspaceEnv(repoRoot(t))
	registry, err := integration.NewRegistry(r.adapter)
	require.NoError(t, err)
	ledger := filepath.Join(t.TempDir(), "memory", "integration-calls.jsonl")
	return &handoffconsumer.Consumer{
		Mechanism: mechanism.New(mechanism.Options{
			Config:   config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": r.provider}},
			Registry: registry, Env: env, Breaker: policy.NewBreaker(3, time.Minute, time.Now),
		}),
		Provider: r.provider, Name: "jev", Version: "1", Ledger: handoffconsumer.Ledger{Path: ledger}, Now: time.Now,
	}, ledger
}

func TestLiveHandoffPrecheckWorksEndToEnd(t *testing.T) {
	r := newRig(t)
	consumer, ledger := r.consumer(t)
	request := func(subject string, source handoffconsumer.Source) handoffconsumer.Request {
		return handoffconsumer.Request{MissionID: "live-check", Transition: handoff.TransitionRangerToArchivist, Subject: subject, Source: source}
	}

	good := consumer.Precheck(context.Background(), request("sha256:live-good", wellFormed))
	if good.Status == handoffconsumer.StatusFallback {
		r.diagnose(t)
	}
	require.Equal(t, handoffconsumer.StatusApproved, good.Status,
		"a well-formed handoff must clear the default threshold %.2f with the confidence in acceptance (got %.2f, reason %q)", good.Threshold, good.Confidence, good.Reason)
	require.True(t, good.Attempted)
	require.GreaterOrEqual(t, good.Confidence, 0.0)
	require.LessOrEqual(t, good.Confidence, 1.0)
	if good.Status == handoffconsumer.StatusApproved {
		require.NoError(t, good.Delegation.Validate(), "an approved reading yields a valid delegation")
		require.Equal(t, "sha256:live-good", good.Delegation.Subject)
	}
	t.Logf("well-formed handoff: status=%s confidence=%.2f threshold=%.2f hints=%d", good.Status, good.Confidence, good.Threshold, len(good.Hints))

	bad := consumer.Precheck(context.Background(), request("sha256:live-poor", poor))
	require.Contains(t, []handoffconsumer.Status{handoffconsumer.StatusApproved, handoffconsumer.StatusSignal, handoffconsumer.StatusFallback}, bad.Status)
	require.NotEqual(t, handoffconsumer.StatusApproved, bad.Status, "a vague, unsupported handoff must not be approved above the threshold (got %.2f)", bad.Confidence)
	t.Logf("poor handoff: status=%s confidence=%.2f hints=%d", bad.Status, bad.Confidence, len(bad.Hints))

	count, err := consumer.Ledger.Count("live-check", handoff.TransitionRangerToArchivist)
	require.NoError(t, err)
	require.Equal(t, 2, count, "both real calls are in the call ledger")
	raw, err := os.ReadFile(ledger)
	require.NoError(t, err)
	require.False(t, r.leaks(string(raw)), "the credential appeared in the call ledger")
	require.Contains(t, string(raw), `"answered":true`)
	require.Contains(t, string(raw), `"input_tokens"`, "provider usage is recorded")

	confidence := filepath.Join(t.TempDir(), "memory", "confidence-records.jsonl")
	require.NoError(t, handoffconsumer.RecordConfidence(confidence, request("sha256:live-good", wellFormed), good, "live"))
	records, err := telemetry.ReadConfidenceRecords(confidence)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, telemetry.ConfidenceAgentHandoffChallenge, records[0].Agent)
	require.Contains(t, []string{domain.ClaimKindAssertion, domain.ClaimKindQuestion}, records[0].ClaimKind)
}

func TestLiveAnInvalidCredentialIsATypedAuthenticationFailureThatNeverEchoesIt(t *testing.T) {
	r := newRig(t)
	const fake = "sk-invalid-strategist-live-check-0000"
	client, err := transport.New(transport.Config{Endpoint: r.provider.Endpoint, Timeout: liveTimeout, MaxRetries: -1})
	require.NoError(t, err)
	adapter, err := jev.New(client, func() (credential.Secret, error) {
		return credential.Resolve("env:X", credential.Env{Getenv: func(string) string { return fake }})
	}, jev.Config{Model: "jev-latest", AllowAlias: true})
	require.NoError(t, err)

	record := probe.Run(context.Background(), r.provider, adapter, time.Now)
	require.Contains(t, []string{string(integration.StateAuthenticationFailed), string(integration.StateForbidden)}, record.State,
		"a bad key must map to a typed state, got %q", record.State)
	require.NotContains(t, record.State, fake)
}

func TestLiveRealCredentialNeverReachesTheEnvironmentOfChildren(t *testing.T) {
	r := newRig(t)
	require.Empty(t, os.Getenv("TYPESAFE_API_KEY"), "the .env value is parsed in memory and never exported")
	for _, entry := range os.Environ() {
		require.False(t, r.leaks(entry), "the credential is present in the process environment")
	}
}
