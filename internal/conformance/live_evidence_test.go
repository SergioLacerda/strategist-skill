package conformance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func validLiveProbeConfig() LiveProbeConfig {
	return LiveProbeConfig{
		Provider:         "provider-under-test",
		Runner:           "authorized-runner",
		RowID:            "codex-live-probe",
		ProbeID:          "ready-v1",
		AuthorizationRef: "secret://provider/token",
		ReportLocation:   "artifacts/live-evidence.json",
		Retention:        "30d",
		Timeout:          time.Second,
	}
}

func TestExecuteLiveProbeProducesRedactedCertifiedEvidence(t *testing.T) {
	config := validLiveProbeConfig()
	evidence, err := ExecuteLiveProbe(context.Background(), config, func(context.Context) (EvidenceState, error) {
		return StateCertified, nil
	}, func(context.Context) error { return nil }, func() time.Time { return time.Unix(10, 0) })
	require.NoError(t, err)
	require.NoError(t, evidence.Validate())
	require.Equal(t, StateCertified, evidence.State)
	require.Equal(t, "complete", evidence.Teardown)
	require.Equal(t, "probe_verified", evidence.Reason)
	row := Row{ID: config.RowID, Client: "codex", EvidenceTier: EvidenceLive, ExpectedState: StateCertified}
	result, err := EvaluateLiveEvidence(row, evidence)
	require.NoError(t, err)
	require.True(t, result.Passed)

	data, err := MarshalLiveEvidence(evidence)
	require.NoError(t, err)
	require.NotContains(t, string(data), config.AuthorizationRef)
}

func TestExecuteLiveProbeClassifiesTimeoutAndTeardownFailure(t *testing.T) {
	config := validLiveProbeConfig()
	config.Timeout = time.Millisecond
	evidence, err := ExecuteLiveProbe(context.Background(), config, func(ctx context.Context) (EvidenceState, error) {
		<-ctx.Done()
		return StateCertified, nil
	}, func(context.Context) error { return nil }, time.Now)
	require.NoError(t, err)
	require.Equal(t, StateTimeout, evidence.State)
	require.Equal(t, "probe_timeout", evidence.Reason)
	require.Equal(t, "complete", evidence.Teardown)

	config.Timeout = time.Second
	evidence, err = ExecuteLiveProbe(context.Background(), config, func(context.Context) (EvidenceState, error) {
		return StateCertified, nil
	}, func(context.Context) error { return errors.New("cleanup failed") }, time.Now)
	require.NoError(t, err)
	require.Equal(t, StateTeardownFailed, evidence.State)
	require.Equal(t, "teardown_failed", evidence.Reason)
	require.Equal(t, "failed", evidence.Teardown)
	row := Row{ID: config.RowID, Client: "codex", EvidenceTier: EvidenceLive, ExpectedState: StateCertified}
	result, err := EvaluateLiveEvidence(row, evidence)
	require.NoError(t, err)
	require.False(t, result.Passed)
}

func TestExecuteLiveProbeRejectsMissingAuthorizationAndInvalidState(t *testing.T) {
	config := validLiveProbeConfig()
	config.AuthorizationRef = ""
	_, err := ExecuteLiveProbe(context.Background(), config, func(context.Context) (EvidenceState, error) {
		return StateCertified, nil
	}, func(context.Context) error { return nil }, time.Now)
	require.ErrorContains(t, err, "authorization")

	config = validLiveProbeConfig()
	evidence, err := ExecuteLiveProbe(context.Background(), config, func(context.Context) (EvidenceState, error) {
		return EvidenceState("provider-secret-payload"), nil
	}, func(context.Context) error { return nil }, time.Now)
	require.NoError(t, err)
	require.Equal(t, StateMalformed, evidence.State)
	data, err := MarshalLiveEvidence(evidence)
	require.NoError(t, err)
	require.NotContains(t, string(data), "provider-secret-payload")
	require.Contains(t, string(data), "probe_state_invalid")
}
