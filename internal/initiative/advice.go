package initiative

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Observation is the raw execution-level state reported for a role, as
// resolved by LEVELING.
type Observation struct {
	State       ObservationState `json:"state" yaml:"state"`
	Model       string           `json:"model,omitempty" yaml:"model,omitempty"`
	Provider    string           `json:"provider,omitempty" yaml:"provider,omitempty"`
	Effort      EffortTier       `json:"effort,omitempty" yaml:"effort,omitempty"`
	Capability  string           `json:"capability,omitempty" yaml:"capability,omitempty"`
	LevelSource string           `json:"level_source,omitempty" yaml:"level_source,omitempty"`
}

// Recommendation is the capability and effort a Policy profile advises for
// a role.
type Recommendation struct {
	RecommendedCapability string     `json:"recommended_capability" yaml:"recommended_capability"`
	RecommendedEffort     EffortTier `json:"recommended_effort" yaml:"recommended_effort"`
	Rationale             []string   `json:"rationale" yaml:"rationale"`
}

// DiligenceProfile lists the checks a role must perform and the confidence
// ceiling it is bound by.
type DiligenceProfile struct {
	Checks            []string `json:"checks" yaml:"checks"`
	ConfidenceCeiling string   `json:"confidence_ceiling" yaml:"confidence_ceiling"`
}

// AdviceInput carries the context needed to produce an Advice.
type AdviceInput struct {
	MissionID  string
	Role       string
	RunID      string
	Trigger    Trigger
	Sequence   int
	Observed   Observation
	Leveling   *LevelingResolution
	Supersedes string
}

// Advice is the consultative recommendation produced by an Advisor for a
// role. It never resolves or changes the execution level selected by
// LEVELING.
type Advice struct {
	AdviceID       string              `json:"advice_id" yaml:"advice_id"`
	MissionID      string              `json:"mission_id" yaml:"mission_id"`
	Role           string              `json:"role" yaml:"role"`
	RunID          string              `json:"run_id" yaml:"run_id"`
	PolicyVersion  string              `json:"policy_version" yaml:"policy_version"`
	PolicyDigest   string              `json:"policy_digest" yaml:"policy_digest"`
	Trigger        Trigger             `json:"trigger" yaml:"trigger"`
	Supersedes     string              `json:"supersedes,omitempty" yaml:"supersedes,omitempty"`
	Leveling       *LevelingResolution `json:"leveling,omitempty" yaml:"leveling,omitempty"`
	Observed       Observation         `json:"observed" yaml:"observed"`
	Recommendation Recommendation      `json:"recommendation" yaml:"recommendation"`
	Alignment      AlignmentState      `json:"alignment" yaml:"alignment"`
	Diligence      DiligenceProfile    `json:"diligence" yaml:"diligence"`
}

// Advisor produces Advice from an AdviceInput according to its Policy.
type Advisor struct{ Policy Policy }

// Advise validates input against the advisor's policy and returns the
// resulting Advice.
func (a Advisor) Advise(input AdviceInput) (Advice, error) {
	if err := a.Policy.Validate(); err != nil {
		return Advice{}, err
	}
	if !validTriggers[input.Trigger] {
		return Advice{}, fmt.Errorf("initiative_advice_invalid: unknown trigger %q", input.Trigger)
	}
	if !a.Policy.AllowsTrigger(input.Trigger) {
		return Advice{}, fmt.Errorf("initiative_advice_invalid: trigger %q is not enabled by policy", input.Trigger)
	}
	role, profile, sequence, err := a.resolveAdviceInput(input)
	if err != nil {
		return Advice{}, err
	}
	advice := buildAdvice(input, role, profile, a.Policy, sequence)
	if err := advice.Validate(); err != nil {
		return Advice{}, err
	}
	return advice, nil
}

func (a Advisor) resolveAdviceInput(input AdviceInput) (string, Profile, int, error) {
	role := normalizeRole(input.Role)
	if err := validateAdviceIdentity(input, role); err != nil {
		return "", Profile{}, 0, err
	}
	if input.Leveling != nil {
		if err := input.Leveling.ValidateFor(role); err != nil {
			return "", Profile{}, 0, err
		}
	}
	profile, ok := a.Policy.Profile(role)
	if !ok {
		return "", Profile{}, 0, fmt.Errorf("initiative_advice_invalid: role %q is not in policy", role)
	}
	if input.Sequence < 0 {
		return "", Profile{}, 0, fmt.Errorf("initiative_advice_invalid: sequence cannot be negative")
	}
	sequence := input.Sequence
	if sequence == 0 {
		sequence = 1
	}
	return role, profile, sequence, nil
}

func validateAdviceIdentity(input AdviceInput, role string) error {
	if strings.TrimSpace(input.MissionID) == "" || role == "" || strings.TrimSpace(input.RunID) == "" {
		return fmt.Errorf("initiative_advice_invalid: mission_id, role, and run_id are required")
	}
	if !validTriggers[input.Trigger] {
		return fmt.Errorf("initiative_advice_invalid: unknown trigger %q", input.Trigger)
	}
	if input.Trigger != TriggerInitial && strings.TrimSpace(input.Supersedes) == "" {
		return fmt.Errorf("initiative_advice_invalid: trigger %q requires supersedes", input.Trigger)
	}
	return nil
}

func buildAdvice(input AdviceInput, role string, profile Profile, policy Policy, sequence int) Advice {
	observed := normalizeObservation(input.Observed)
	var resolution *LevelingResolution
	if input.Leveling != nil {
		copyResolution := *input.Leveling
		resolution = &copyResolution
		observed = normalizeObservation(copyResolution.Observation())
	}
	return Advice{
		AdviceID:      adviceID(input, policy.Digest(), sequence),
		MissionID:     input.MissionID,
		Role:          role,
		RunID:         input.RunID,
		PolicyVersion: policy.Version,
		PolicyDigest:  policy.Digest(),
		Trigger:       input.Trigger,
		Supersedes:    input.Supersedes,
		Leveling:      resolution,
		Observed:      observed,
		Recommendation: Recommendation{
			RecommendedCapability: profile.RecommendedCapability,
			RecommendedEffort:     profile.RecommendedEffort,
			Rationale:             append([]string(nil), profile.Diligence...),
		},
		Alignment: AlignmentFor(observed, profile.RecommendedEffort),
		Diligence: DiligenceProfile{Checks: append([]string(nil), profile.Diligence...), ConfidenceCeiling: profile.ConfidenceCeiling},
	}
}

func adviceID(input AdviceInput, digest string, sequence int) string {
	material := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%d\x00%s", input.MissionID, normalizeRole(input.Role), input.RunID, input.Trigger, input.Supersedes, sequence, digest)
	sum := sha256.Sum256([]byte(material))
	return "adv-" + hex.EncodeToString(sum[:])[:24]
}
