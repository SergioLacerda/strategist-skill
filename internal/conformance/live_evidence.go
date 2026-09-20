package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// LiveEvidenceSchemaVersion identifies the provider-neutral artifact contract.
const LiveEvidenceSchemaVersion = "strategist-live-evidence/v1"

// LiveProbeConfig contains authorization metadata, never a secret value.
// AuthorizationRef points at an externally managed secret/configuration
// boundary; the executor deliberately does not resolve or serialize it.
type LiveProbeConfig struct {
	Provider         string
	Runner           string
	RowID            string
	ProbeID          string
	AuthorizationRef string
	ReportLocation   string
	Retention        string
	Timeout          time.Duration
}

// LiveEvidence is the redacted, machine-readable result of one bounded probe.
// Provider payloads and credentials are intentionally absent from this type.
type LiveEvidence struct {
	SchemaVersion  string        `json:"schema_version"`
	Provider       string        `json:"provider"`
	Runner         string        `json:"runner"`
	RowID          string        `json:"row_id"`
	ProbeID        string        `json:"probe_id"`
	ObservedAt     time.Time     `json:"observed_at"`
	Timeout        time.Duration `json:"timeout"`
	Teardown       string        `json:"teardown"`
	State          EvidenceState `json:"state"`
	Reason         string        `json:"reason"`
	ReportLocation string        `json:"report_location"`
	Retention      string        `json:"retention"`
}

// LiveProbe executes a provider-specific probe inside the supplied deadline.
type LiveProbe func(context.Context) (EvidenceState, error)

// LiveTeardown releases provider resources and is required by the executor.
type LiveTeardown func(context.Context) error

// ExecuteLiveProbe validates authorization metadata, bounds invocation, and
// emits only redacted evidence. It never turns a missing or failed probe into
// certified evidence.
func ExecuteLiveProbe(ctx context.Context, config LiveProbeConfig, probe LiveProbe, teardown LiveTeardown, now func() time.Time) (LiveEvidence, error) {
	if err := config.validate(); err != nil {
		return LiveEvidence{}, err
	}
	if probe == nil {
		return LiveEvidence{}, fmt.Errorf("live probe: probe is required")
	}
	if teardown == nil {
		return LiveEvidence{}, fmt.Errorf("live probe: teardown is required")
	}
	if now == nil {
		now = time.Now
	}

	evidence := LiveEvidence{
		SchemaVersion:  LiveEvidenceSchemaVersion,
		Provider:       config.Provider,
		Runner:         config.Runner,
		RowID:          config.RowID,
		ProbeID:        config.ProbeID,
		ObservedAt:     now().UTC(),
		Timeout:        config.Timeout,
		Teardown:       "pending",
		State:          StateUnknown,
		Reason:         "probe_not_started",
		ReportLocation: config.ReportLocation,
		Retention:      config.Retention,
	}

	probeCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	state, probeErr := probe(probeCtx)
	evidence.State, evidence.Reason = classifyProbeResult(probeCtx.Err(), state, probeErr)

	if teardownErr := teardown(ctx); teardownErr != nil {
		evidence.State = StateTeardownFailed
		evidence.Reason = "teardown_failed"
		evidence.Teardown = "failed"
	} else {
		evidence.Teardown = "complete"
	}
	return evidence, nil
}

func classifyProbeResult(contextErr error, state EvidenceState, probeErr error) (EvidenceState, string) {
	switch contextErr {
	case context.DeadlineExceeded:
		return StateTimeout, "probe_timeout"
	case nil:
		switch {
		case !validState(state):
			return StateMalformed, "probe_state_invalid"
		case probeErr != nil:
			return StateFailed, "probe_failed"
		default:
			return state, reasonForState(state)
		}
	default:
		return StateBlocked, "probe_canceled"
	}
}

// MarshalLiveEvidence emits stable JSON and validates the redacted envelope.
func MarshalLiveEvidence(evidence LiveEvidence) ([]byte, error) {
	if err := evidence.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(evidence)
	if err != nil {
		return nil, fmt.Errorf("marshal live evidence: %w", err)
	}
	return data, nil
}

// EvaluateLiveEvidence binds a redacted artifact to its declared matrix row.
// Cleanup failure is always non-passing even if the provider reported success.
func EvaluateLiveEvidence(row Row, evidence LiveEvidence) (RowResult, error) {
	if err := evidence.Validate(); err != nil {
		return RowResult{}, err
	}
	if evidence.RowID != row.ID {
		return RowResult{}, fmt.Errorf("live evidence: row mismatch: expected %q, got %q", row.ID, evidence.RowID)
	}
	if evidence.Teardown != "complete" {
		return RowResult{RowID: row.ID, Client: row.Client, Reason: "live_probe_not_ready"}, nil
	}
	return EvaluateLiveProbe(row, evidence.State)
}

// Validate checks that a live artifact is complete and contains no secret
// material by construction.
func (e LiveEvidence) Validate() error {
	if e.SchemaVersion != LiveEvidenceSchemaVersion || e.Provider == "" || e.Runner == "" || e.RowID == "" || e.ProbeID == "" || e.ReportLocation == "" || e.Retention == "" {
		return fmt.Errorf("live evidence: incomplete envelope")
	}
	if e.ObservedAt.IsZero() || e.Timeout <= 0 || (e.Teardown != "complete" && e.Teardown != "failed") || !validState(e.State) || e.Reason == "" {
		return fmt.Errorf("live evidence: invalid result")
	}
	return nil
}

func (c LiveProbeConfig) validate() error {
	if c.Provider == "" || c.Runner == "" || c.RowID == "" || c.ProbeID == "" || c.AuthorizationRef == "" || c.ReportLocation == "" || c.Retention == "" {
		return fmt.Errorf("live probe: provider, runner, row, probe, authorization, report, and retention are required")
	}
	if c.Timeout <= 0 || c.Timeout > 15*time.Minute {
		return fmt.Errorf("live probe: timeout must be between 1ns and 15m")
	}
	return nil
}

func reasonForState(state EvidenceState) string {
	switch state {
	case StateCertified:
		return "probe_verified"
	case StateUnavailable:
		return "provider_unavailable"
	case StateUnauthorized:
		return "probe_unauthorized"
	case StateStale:
		return "probe_stale"
	case StateUnsupported:
		return "probe_unsupported"
	case StateBlocked:
		return "probe_blocked"
	case StateFailed:
		return "probe_failed"
	case StateUnknown:
		return "probe_not_ready"
	case StateTimeout:
		return "probe_timeout"
	case StateMalformed:
		return "probe_state_invalid"
	case StateTeardownFailed:
		return "teardown_failed"
	}

	return "probe_not_ready"
}
