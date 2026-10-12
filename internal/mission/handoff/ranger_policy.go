package handoff

import (
	"bytes"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RangerPolicyFactsKey identifies the closed policy block carried by the
// normalized Ranger artifact. It is deliberately distinct from the
// Archivist-owned handoff_policy_facts block.
const RangerPolicyFactsKey = "ranger_handoff_policy_facts"

// RangerPolicyFactsSchemaVersion identifies the typed Ranger policy block.
const RangerPolicyFactsSchemaVersion = "strategist-ranger-handoff-policy-facts/v1"

// RangerPolicyFacts contains the explicit facts that decide which
// Ranger-to-Archivist challenge types are required. Every pointer is required
// so absence cannot silently become a low-risk skip.
type RangerPolicyFacts struct {
	SchemaVersion         string `yaml:"schema_version"`
	RequireRecall         *bool  `yaml:"require_recall"`
	RequireBoundary       *bool  `yaml:"require_boundary"`
	RequireClassification *bool  `yaml:"require_classification"`
	RequireVerdict        *bool  `yaml:"require_verdict"`
	InformationalOnly     *bool  `yaml:"informational_only"`
}

// ReadRangerPolicyFacts loads and validates the typed facts from a normalized
// Ranger artifact. Missing, unknown, null, or contradictory facts fail closed.
func ReadRangerPolicyFacts(path, missionID string) (RangerPolicyFacts, error) {
	content, err := readArtifact(filepath.Clean(path))
	if err != nil {
		return RangerPolicyFacts{}, fmt.Errorf("handoff_artifact_invalid: read Ranger artifact: %w", err)
	}
	frontmatter, _, err := parseFrontmatter(content)
	if err != nil {
		return RangerPolicyFacts{}, fmt.Errorf("handoff_artifact_invalid: Ranger artifact: %w", err)
	}
	if err := validateIdentity(frontmatter, missionID, []string{"ranger_pending", "ranger_done"}); err != nil {
		return RangerPolicyFacts{}, fmt.Errorf("handoff_artifact_invalid: Ranger artifact: %w", err)
	}
	raw, ok := frontmatter[RangerPolicyFactsKey]
	if !ok || raw == nil {
		return RangerPolicyFacts{}, fmt.Errorf("ranger_handoff_policy_facts_missing: Ranger artifact frontmatter has no %s block", RangerPolicyFactsKey)
	}
	if err := ValidateRangerArtifact(path, missionID); err != nil {
		return RangerPolicyFacts{}, err
	}
	facts, err := decodeRangerPolicyFacts(raw)
	if err != nil {
		return RangerPolicyFacts{}, err
	}
	return facts, nil
}

func decodeRangerPolicyFacts(raw any) (RangerPolicyFacts, error) {
	encoded, err := yaml.Marshal(raw)
	if err != nil {
		return RangerPolicyFacts{}, fmt.Errorf("ranger_handoff_policy_facts_invalid: encode %s: %w", RangerPolicyFactsKey, err)
	}
	var facts RangerPolicyFacts
	decoder := yaml.NewDecoder(bytes.NewReader(encoded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&facts); err != nil {
		return RangerPolicyFacts{}, fmt.Errorf("ranger_handoff_policy_facts_invalid: %w", err)
	}
	if err := facts.validate(); err != nil {
		return RangerPolicyFacts{}, err
	}
	return facts, nil
}

func (f RangerPolicyFacts) validate() error {
	if f.SchemaVersion != RangerPolicyFactsSchemaVersion {
		return fmt.Errorf("ranger_handoff_policy_facts_invalid: schema_version %q is not %q", f.SchemaVersion, RangerPolicyFactsSchemaVersion)
	}
	if !f.complete() {
		return fmt.Errorf("ranger_handoff_policy_facts_missing: all require_* and informational_only fields are required")
	}
	requiresChallenge := *f.RequireRecall || *f.RequireBoundary || *f.RequireClassification || *f.RequireVerdict
	switch {
	case *f.InformationalOnly && requiresChallenge:
		return fmt.Errorf("ranger_handoff_policy_facts_contradictory: informational_only cannot require a Ranger challenge")
	case !*f.InformationalOnly && !requiresChallenge:
		return fmt.Errorf("ranger_handoff_policy_facts_ambiguous: set informational_only true or require at least one Ranger challenge")
	}
	return nil
}

// complete reports whether every typed fact was declared.
func (f RangerPolicyFacts) complete() bool {
	return f.RequireRecall != nil && f.RequireBoundary != nil && f.RequireClassification != nil && f.RequireVerdict != nil && f.InformationalOnly != nil
}

// RangerToArchivistPolicyForFacts derives the lifecycle policy exclusively from
// the typed Ranger facts. No coarse mission-risk label can enable or disable
// this transition.
func RangerToArchivistPolicyForFacts(facts RangerPolicyFacts) (Policy, error) {
	if err := facts.validate(); err != nil {
		return Policy{}, err
	}
	base := RangerToArchivistPolicy()
	base.RequiredTypes = nil
	if *facts.RequireRecall {
		base.RequiredTypes = append(base.RequiredTypes, ChallengeRecall)
	}
	if *facts.RequireBoundary {
		base.RequiredTypes = append(base.RequiredTypes, ChallengeBoundary)
	}
	if *facts.RequireClassification {
		base.RequiredTypes = append(base.RequiredTypes, ChallengeClassification)
	}
	if *facts.RequireVerdict {
		base.RequiredTypes = append(base.RequiredTypes, ChallengeVerdict)
	}
	base.Enabled = len(base.RequiredTypes) > 0
	return base, nil
}

// RangerPolicyProvenance returns privacy-safe provenance for the policy facts.
// It records only fact names and not the artifact's challenge content.
func RangerPolicyProvenance(facts RangerPolicyFacts) []SignalProvenance {
	const source = "ranger_artifact.frontmatter#" + RangerPolicyFactsKey
	var provenance []SignalProvenance
	for _, item := range []struct {
		name  string
		value bool
	}{
		{"require_recall", facts.RequireRecall != nil && *facts.RequireRecall},
		{"require_boundary", facts.RequireBoundary != nil && *facts.RequireBoundary},
		{"require_classification", facts.RequireClassification != nil && *facts.RequireClassification},
		{"require_verdict", facts.RequireVerdict != nil && *facts.RequireVerdict},
	} {
		if item.value {
			provenance = append(provenance, SignalProvenance{Signal: item.name, Source: source})
		}
	}
	return provenance
}
