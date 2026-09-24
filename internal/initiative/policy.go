package initiative

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultPolicy returns the built-in INITIATIVE policy used when no
// override is configured.
func DefaultPolicy() Policy {
	return Policy{
		Version: "1",
		Ability: "initiative", DisplayName: "INITIATIVE", Mode: "consultative",
		Authority: PolicyAuthority{Owns: []string{"advice", "diligence", "alignment", "results", "outcome_correlation"}, DoesNotOwn: []string{"model", "provider", "capability", "effort", "level_source", "approval_gate", "implementation_authorization"}},
		Records:   PolicyRecords{Path: ".strategist/memory/initiative-records.jsonl", AppendOnly: true, Correlation: []string{"mission_id", "role", "run_id", "advice_id"}},
		Profiles: map[string]Profile{
			"scout":     {RecommendedCapability: "economical", RecommendedEffort: EffortMedium, Diligence: []string{"classify_scope", "surface_uncertainty"}, ConfidenceCeiling: ConfidenceMedium},
			"ranger":    {RecommendedCapability: "reasoning", RecommendedEffort: EffortHigh, Diligence: []string{"inspect_evidence", "test_alternatives", "record_obligations"}, ConfidenceCeiling: ConfidenceHigh},
			"archivist": {RecommendedCapability: "reasoning", RecommendedEffort: EffortHigh, Diligence: []string{"challenge_handoff", "validate_contracts", "correlate_outcomes"}, ConfidenceCeiling: ConfidenceHigh},
			"sniper":    {RecommendedCapability: "economical", RecommendedEffort: EffortMedium, Diligence: []string{"verify_scope", "verify_gate", "record_result"}, ConfidenceCeiling: ConfidenceMedium},
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
	if len(profile.Diligence) == 0 {
		return fmt.Errorf("initiative_policy_invalid: role %q requires diligence and confidence ceiling", role)
	}
	if err := validateDiligenceChecks(role, profile.Diligence); err != nil {
		return err
	}
	if err := validateConfidenceTier(profile.ConfidenceCeiling); err != nil {
		return fmt.Errorf("initiative_policy_invalid: role %q: %w", role, err)
	}
	return nil
}

func validateDiligenceChecks(role string, diligence []string) error {
	seen := make(map[string]struct{}, len(diligence))
	for _, check := range diligence {
		check = strings.TrimSpace(check)
		if check == "" {
			return fmt.Errorf("initiative_policy_invalid: role %q contains an empty diligence check", role)
		}
		if _, exists := seen[check]; exists {
			return fmt.Errorf("initiative_policy_invalid: role %q contains duplicate diligence check %q", role, check)
		}
		seen[check] = struct{}{}
	}
	return nil
}

func validateReevaluationTriggers(triggers []Trigger) error {
	seen := make(map[Trigger]struct{}, len(triggers))
	for _, trigger := range triggers {
		if !validTriggers[trigger] || trigger == TriggerInitial {
			return fmt.Errorf("initiative_policy_invalid: invalid re-evaluation trigger %q", trigger)
		}
		if _, exists := seen[trigger]; exists {
			return fmt.Errorf("initiative_policy_invalid: duplicate re-evaluation trigger %q", trigger)
		}
		seen[trigger] = struct{}{}
	}
	return nil
}

// AllowsTrigger reports whether trigger is explicitly enabled by the policy.
// Initial advice is always allowed; every re-evaluation trigger must be
// declared so a caller cannot invent a policy transition at runtime.
func (p Policy) AllowsTrigger(trigger Trigger) bool {
	if trigger == TriggerInitial {
		return true
	}
	for _, allowed := range p.Triggers {
		if allowed == trigger {
			return true
		}
	}
	return false
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
	decoder := yaml.NewDecoder(strings.NewReader(string(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return Policy{}, fmt.Errorf("initiative_policy_invalid: parse: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Policy{}, fmt.Errorf("initiative_policy_invalid: multiple YAML documents are not supported")
		}
		return Policy{}, fmt.Errorf("initiative_policy_invalid: parse trailing document: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func normalizeRole(role string) string { return strings.ToLower(strings.TrimSpace(role)) }

func validEffort(effort EffortTier) bool {
	switch effort {
	case EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax:
		return true
	default:
		return false
	}
}

func validateConfidenceTier(tier ConfidenceTier) error {
	switch tier {
	case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
		return nil
	default:
		return fmt.Errorf("initiative_confidence_tier_invalid: %q", tier)
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
	case EffortMax:
		return 5, true
	default:
		return 0, false
	}
}
