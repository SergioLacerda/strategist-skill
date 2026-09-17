package lifecycle

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ProbeOutcome is the typed evidence consumed by lifecycle activation.
// Unknown and unsupported outcomes are never treated as successful probes.
type ProbeOutcome struct {
	Status     domain.ReadinessStatus
	ReasonCode string
	Detail     string
}

// Probe records a non-mutating probe result. Passing probes can activate.
func (s *Store) Probe(txID string, passed bool) error {
	status := domain.ReadinessBlocked
	if passed {
		status = domain.ReadinessReady
	}
	return s.ProbeResult(txID, ProbeOutcome{Status: status, ReasonCode: probeReason(passed)})
}

// ProbeResult records typed connector evidence. Only an explicit ready result
// may advance a candidate to the activation state.
func (s *Store) ProbeResult(txID string, result ProbeOutcome) error {
	tx, err := s.transaction(txID)
	if err != nil {
		return err
	}
	passed := result.Status == domain.ReadinessReady
	if tx.State == StateProbed && tx.ProbePassed == passed {
		return nil
	}
	if tx.State != StateStaged {
		return fmt.Errorf("probe_invalid_state: %s", tx.State)
	}
	tx.ProbePassed = passed
	if !passed {
		code := result.ReasonCode
		if code == "" {
			code = "probe_failed"
		}
		return s.transitionCandidate(tx, StateFailed, code)
	}
	return s.transitionCandidate(tx, StateProbed, result.ReasonCode)
}

func probeReason(passed bool) string {
	if passed {
		return "probe_passed"
	}
	return "probe_failed"
}
