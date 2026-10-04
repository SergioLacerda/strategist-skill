package probe

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev"
	"github.com/SergioLacerda/strategist-skill/internal/integration/jev/jevtest"
	"github.com/SergioLacerda/strategist-skill/internal/integration/transport"
	"github.com/stretchr/testify/require"
)

var clock = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func provider() config.Provider {
	return config.Provider{
		Enabled: true, Endpoint: "https://api.example.test/v1", CredentialRef: "env:K", Model: "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: []string{"objective"}, MaxStateBytes: 1000},
	}
}

const probeAnswer = `{"model":"jev-1.13.0","answers":{"probe":{"type":"noul","noul":1.0}},"usage":{"input_tokens":12,"output_tokens":3}}`

func adapterFor(t *testing.T, sim *jevtest.Server) integration.Adapter {
	t.Helper()
	client, err := transport.New(transport.Config{Endpoint: sim.URL() + "/v1/systemone", RoundTripper: sim.Transport(), MaxRetries: -1, Timeout: 200 * time.Millisecond})
	require.NoError(t, err)
	env := credential.Env{Getenv: func(string) string { return "sk-probe-synthetic" }}
	adapter, err := jev.New(client, func() (credential.Secret, error) { return credential.Resolve("env:K", env) }, jev.Config{Model: "jev-1.13.0"})
	require.NoError(t, err)
	return adapter
}

func TestFingerprintChangesWithEndpointModelPolicyAndCapabilitiesButNotTheCredential(t *testing.T) {
	base := Fingerprint(provider())
	require.Equal(t, base, Fingerprint(provider()))

	for name, mutate := range map[string]func(*config.Provider){
		"endpoint":    func(p *config.Provider) { p.Endpoint = "https://other.example.test/v1" },
		"model":       func(p *config.Provider) { p.Model = "jev-2.0.0" },
		"data fields": func(p *config.Provider) { p.DataPolicy.AllowedFields = []string{"objective", "known_facts"} },
		"size limit":  func(p *config.Provider) { p.DataPolicy.MaxStateBytes = 2000 },
		"capabilities": func(p *config.Provider) {
			p.AllowedCapabilities = append(p.AllowedCapabilities, integration.CapConfidenceEvaluate)
		},
	} {
		changed := provider()
		mutate(&changed)
		require.NotEqual(t, base, Fingerprint(changed), name)
	}

	rotated := provider()
	rotated.CredentialRef = "env:ROTATED"
	require.Equal(t, base, Fingerprint(rotated), "AC-14: rotating the credential reference does not change the identity")
}

func TestRunSendsOnlySyntheticTextAndRecordsTheObservation(t *testing.T) {
	sim := jevtest.Start(t, jevtest.Reply([]byte(probeAnswer)))
	record := Run(context.Background(), provider(), adapterFor(t, sim), func() time.Time { return clock })

	require.Equal(t, "ok", record.State)
	require.Equal(t, "jev-1.13.0", record.Model)
	require.Equal(t, 12, record.InputTokens)
	require.Equal(t, clock, record.At)
	require.Equal(t, Fingerprint(provider()), record.Fingerprint)
	body := sim.LastBody()
	require.Contains(t, body, SyntheticState)
	require.NotContains(t, body, "objective", "no mission content ever rides a probe")
}

func TestRunRecordsTheTypedFailureWithoutTheProviderBody(t *testing.T) {
	sim := jevtest.Start(t, jevtest.Status(http.StatusUnauthorized))
	record := Run(context.Background(), provider(), adapterFor(t, sim), func() time.Time { return clock })

	require.Equal(t, string(integration.StateAuthenticationFailed), record.State)
	require.Empty(t, record.Model)
}

func TestAvailabilityIsUnknownWithoutAValidRecordedObservation(t *testing.T) {
	good := Record{Provider: "jev", Fingerprint: Fingerprint(provider()), At: clock, State: "ok", Model: "jev-1.13.0"}

	state, _ := Evaluate(Record{}, false, provider(), clock)
	require.Equal(t, Unknown, state, "no recorded probe means unknown, never available")

	state, validity := Evaluate(good, true, provider(), clock.Add(time.Hour))
	require.Equal(t, Available, state)
	require.Equal(t, clock.Add(TTL), validity)

	state, _ = Evaluate(good, true, provider(), clock.Add(TTL+time.Second))
	require.Equal(t, Unknown, state, "an expired observation proves nothing")

	changed := provider()
	changed.Model = "jev-2.0.0"
	state, _ = Evaluate(good, true, changed, clock)
	require.Equal(t, Unknown, state, "a configuration change invalidates the observation")

	failed := good
	failed.State = string(integration.StateTimeout)
	state, _ = Evaluate(failed, true, provider(), clock)
	require.Equal(t, Unavailable, state)
}

func TestStoreRoundTripsAndMissingIsNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory", "integration-probe.json")
	_, found, err := Load(path)
	require.NoError(t, err)
	require.False(t, found)

	want := Record{Provider: "jev", Fingerprint: "sha256:x", At: clock, State: "ok", Model: "jev-1.13.0", InputTokens: 12, OutputTokens: 3}
	require.NoError(t, Save(path, want))
	got, found, err := Load(path)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, want, got)
}
