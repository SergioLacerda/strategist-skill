package initiative

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultPolicy returns the built-in INITIATIVE policy used when no
// override is configured.
func DefaultPolicy() Policy {
	return Policy{
		Version: "1",
		Profiles: map[string]Profile{
			"scout":     {RecommendedCapability: "economical", RecommendedEffort: EffortMedium, Diligence: []string{"classify_scope", "surface_uncertainty"}, ConfidenceCeiling: "medium"},
			"ranger":    {RecommendedCapability: "reasoning", RecommendedEffort: EffortHigh, Diligence: []string{"inspect_evidence", "test_alternatives", "record_obligations"}, ConfidenceCeiling: "high"},
			"archivist": {RecommendedCapability: "reasoning", RecommendedEffort: EffortHigh, Diligence: []string{"challenge_handoff", "validate_contracts", "correlate_outcomes"}, ConfidenceCeiling: "high"},
			"sniper":    {RecommendedCapability: "economical", RecommendedEffort: EffortMedium, Diligence: []string{"verify_scope", "verify_gate", "record_result"}, ConfidenceCeiling: "medium"},
		},
		Triggers: []Trigger{TriggerScopeChanged, TriggerSecurityRiskDiscovered, TriggerConflictingEvidence, TriggerHandoffChallenged, TriggerRepeatedFailure, TriggerUserRevisionRequested, TriggerMandatoryObligationBlocked},
	}
}

// Validate checks that the policy's version, profiles, and re-evaluation
// triggers are structurally complete.
func (p Policy) Validate() error {
	if strings.TrimSpace(p.Version) == "" {
		return fmt.Errorf("initiative_policy_invalid: version is required")
	}
	if len(p.Profiles) == 0 {
		return fmt.Errorf("initiative_policy_invalid: profiles are required")
	}
	if err := validateProfiles(p.Profiles); err != nil {
		return err
	}
	return validateReevaluationTriggers(p.Triggers)
}

func validateProfiles(profiles map[string]Profile) error {
	for role, profile := range profiles {
		if err := validateProfile(role, profile); err != nil {
			return err
		}
	}
	return nil
}

func validateProfile(role string, profile Profile) error {
	if normalizeRole(role) == "" || strings.TrimSpace(profile.RecommendedCapability) == "" {
		return fmt.Errorf("initiative_policy_invalid: role %q has no recommended capability", role)
	}
	if !validEffort(profile.RecommendedEffort) {
		return fmt.Errorf("initiative_policy_invalid: role %q has invalid recommended effort %q", role, profile.RecommendedEffort)
	}
	if len(profile.Diligence) == 0 || strings.TrimSpace(profile.ConfidenceCeiling) == "" {
		return fmt.Errorf("initiative_policy_invalid: role %q requires diligence and confidence ceiling", role)
	}
	return nil
}

func validateReevaluationTriggers(triggers []Trigger) error {
	for _, trigger := range triggers {
		if !validTriggers[trigger] || trigger == TriggerInitial {
			return fmt.Errorf("initiative_policy_invalid: invalid re-evaluation trigger %q", trigger)
		}
	}
	return nil
}

// Digest returns a stable hash of the policy's contents, used to detect
// policy drift between an Advice and the policy that produced it.
func (p Policy) Digest() string {
	encoded, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// Profile returns the Profile configured for role, and whether one exists.
func (p Policy) Profile(role string) (Profile, bool) {
	profile, ok := p.Profiles[normalizeRole(role)]
	return profile, ok
}

// Parse decodes the standalone INITIATIVE policy. It deliberately has no
// dependency on LEVELING's parser or policy type.
func Parse(raw []byte) (Policy, error) {
	var policy Policy
	if err := yaml.Unmarshal(raw, &policy); err != nil {
		return Policy{}, fmt.Errorf("initiative_policy_invalid: parse: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func normalizeRole(role string) string { return strings.ToLower(strings.TrimSpace(role)) }

func validEffort(effort EffortTier) bool {
	switch effort {
	case EffortLow, EffortMedium, EffortHigh, EffortXHigh:
		return true
	default:
		return false
	}
}

func effortRank(effort EffortTier) (int, bool) {
	switch effort {
	case EffortLow:
		return 1, true
	case EffortMedium:
		return 2, true
	case EffortHigh:
		return 3, true
	case EffortXHigh:
		return 4, true
	default:
		return 0, false
	}
}
