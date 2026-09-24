package initiative

import "fmt"

// LevelingResolution is the immutable value handed from LEVELING to
// INITIATIVE. INITIATIVE may observe these fields, but it has no operation
// that can update the resolution or write to LEVELING's ledger.
type LevelingResolution struct {
	EventID      string           `json:"event_id" yaml:"event_id"`
	Role         string           `json:"role" yaml:"role"`
	State        ObservationState `json:"state" yaml:"state"`
	Model        string           `json:"model,omitempty" yaml:"model,omitempty"`
	Provider     string           `json:"provider,omitempty" yaml:"provider,omitempty"`
	Effort       EffortTier       `json:"effort,omitempty" yaml:"effort,omitempty"`
	Capability   string           `json:"capability,omitempty" yaml:"capability,omitempty"`
	LevelSource  string           `json:"level_source,omitempty" yaml:"level_source,omitempty"`
	PolicyDigest string           `json:"policy_digest,omitempty" yaml:"policy_digest,omitempty"`
}

// ValidateFor checks that a resolution belongs to the role receiving it.
func (r LevelingResolution) ValidateFor(role string) error {
	if normalizeRole(r.Role) != normalizeRole(role) || normalizeRole(r.Role) == "" {
		return fmt.Errorf("initiative_leveling_invalid: resolution role does not match %q", role)
	}
	if !validObservationState(r.State) {
		return fmt.Errorf("initiative_leveling_invalid: invalid resolution state %q", r.State)
	}
	return nil
}

// ValidateForRole is the strict role-boundary validation used for new
// consultations and re-evaluations.
func (r LevelingResolution) ValidateForRole(role string) error {
	if r.EventID == "" {
		return fmt.Errorf("initiative_leveling_invalid: resolution event_id is required")
	}
	return r.ValidateFor(role)
}

// Observation returns the advisory projection of the resolution. The
// returned value is a copy, so INITIATIVE cannot mutate the LEVELING input.
func (r LevelingResolution) Observation() Observation {
	return Observation{State: r.State, Model: r.Model, Provider: r.Provider, Effort: r.Effort, Capability: r.Capability, LevelSource: r.LevelSource}
}

func validObservationState(state ObservationState) bool {
	switch state {
	case ObservationKnown, ObservationUnknown, ObservationUnavailable, ObservationNotComparable:
		return true
	default:
		return false
	}
}
