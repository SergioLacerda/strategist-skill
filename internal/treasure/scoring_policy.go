package treasure

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultScoringPolicy returns the legacy hardcoded score formula as configuration.
func DefaultScoringPolicy() ScoringPolicy {
	return ScoringPolicy{
		ClusterBase:          40,
		ClusterMissionWeight: 10,
		ClusterTagWeight:     5,
		GapBase:              30,
		GapMissionWeight:     15,
		MaxScore:             100,
	}
}

// LoadScoringPolicy reads optional scoring_policy from treasure-chests.yaml.
func LoadScoringPolicy(root string) (ScoringPolicy, error) {
	raw, err := os.ReadFile(filepath.Join(root, "treasure-chests.yaml")) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultScoringPolicy(), nil
		}
		return ScoringPolicy{}, fmt.Errorf("read treasure-chests.yaml: %w", err)
	}
	var m governedManifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return ScoringPolicy{}, fmt.Errorf("parse treasure-chests.yaml: %w", err)
	}
	policy := DefaultScoringPolicy()
	applyScoringPolicyOverrides(&policy, m.ScoringPolicy)
	if err := ValidateScoringPolicy(policy); err != nil {
		return ScoringPolicy{}, fmt.Errorf("treasure-chests.yaml: scoring_policy: %w", err)
	}
	return policy, nil
}

func applyScoringPolicyOverrides(policy *ScoringPolicy, raw rawScoringPolicy) {
	if raw.ClusterBase != nil {
		policy.ClusterBase = *raw.ClusterBase
	}
	if raw.ClusterMissionWeight != nil {
		policy.ClusterMissionWeight = *raw.ClusterMissionWeight
	}
	if raw.ClusterTagWeight != nil {
		policy.ClusterTagWeight = *raw.ClusterTagWeight
	}
	if raw.GapBase != nil {
		policy.GapBase = *raw.GapBase
	}
	if raw.GapMissionWeight != nil {
		policy.GapMissionWeight = *raw.GapMissionWeight
	}
	if raw.MaxScore != nil {
		policy.MaxScore = *raw.MaxScore
	}
}

// ValidateScoringPolicy rejects negative weights and invalid score caps.
func ValidateScoringPolicy(policy ScoringPolicy) error {
	if policy.MaxScore < 1 || policy.MaxScore > 100 {
		return fmt.Errorf("max_score must be between 1 and 100")
	}
	for name, value := range map[string]int{
		"cluster_base":           policy.ClusterBase,
		"cluster_mission_weight": policy.ClusterMissionWeight,
		"cluster_tag_weight":     policy.ClusterTagWeight,
		"gap_base":               policy.GapBase,
		"gap_mission_weight":     policy.GapMissionWeight,
	} {
		if value < 0 {
			return fmt.Errorf("%s must be >= 0", name)
		}
	}
	return nil
}
