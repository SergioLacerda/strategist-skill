package handoff

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// PolicyFactsKey is the analysis.md frontmatter key that carries the typed
// facts a require/skip predicate cannot be derived from anywhere else.
const PolicyFactsKey = "handoff_policy_facts"

// PolicyFactsSchemaVersion identifies the typed facts block.
const PolicyFactsSchemaVersion = "strategist-handoff-policy-facts/v1"

// PolicyFacts are the explicit, typed classifications Archivist declares for a
// refined package. Every field is required: a missing one makes a predicate
// unknowable, and unknowable facts are rejected rather than inferred from prose.
// Implementation_handoff presence is not declared here; it is derived from the
// task classifications in tasks.md.
type PolicyFacts struct {
	SchemaVersion                string   `yaml:"schema_version"`
	MandatoryConstraints         []string `yaml:"mandatory_constraints"`
	UnresolvedQuestions          []string `yaml:"unresolved_questions"`
	ForbiddenScope               []string `yaml:"forbidden_scope"`
	DestructiveOperationPossible *bool    `yaml:"destructive_operation_possible"`
	SecuritySensitiveTask        *bool    `yaml:"security_sensitive_task"`
	InformationalOnly            *bool    `yaml:"informational_only"`
}

// ParsePolicyFacts decodes and validates the typed facts block of an
// analysis.md frontmatter. It rejects an absent block, unknown keys, missing or
// null fields and a wrong schema version.
func ParsePolicyFacts(frontmatter map[string]any) (PolicyFacts, error) {
	raw, ok := frontmatter[PolicyFactsKey]
	if !ok || raw == nil {
		return PolicyFacts{}, fmt.Errorf("handoff_policy_facts_missing: analysis.md frontmatter has no %s block; Archivist must declare it", PolicyFactsKey)
	}
	encoded, err := yaml.Marshal(raw)
	if err != nil {
		return PolicyFacts{}, fmt.Errorf("handoff_policy_facts_invalid: encode %s: %w", PolicyFactsKey, err)
	}
	var facts PolicyFacts
	decoder := yaml.NewDecoder(bytes.NewReader(encoded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&facts); err != nil {
		return PolicyFacts{}, fmt.Errorf("handoff_policy_facts_invalid: %w", err)
	}
	if err := facts.validate(); err != nil {
		return PolicyFacts{}, err
	}
	return facts, nil
}

func (f PolicyFacts) validate() error {
	if f.SchemaVersion != PolicyFactsSchemaVersion {
		return fmt.Errorf("handoff_policy_facts_invalid: schema_version %q is not %q", f.SchemaVersion, PolicyFactsSchemaVersion)
	}
	lists := map[string]bool{
		"mandatory_constraints": f.MandatoryConstraints != nil, "unresolved_questions": f.UnresolvedQuestions != nil,
		"forbidden_scope": f.ForbiddenScope != nil,
	}
	flags := map[string]bool{
		"destructive_operation_possible": f.DestructiveOperationPossible != nil,
		"security_sensitive_task":        f.SecuritySensitiveTask != nil, "informational_only": f.InformationalOnly != nil,
	}
	for _, group := range []map[string]bool{lists, flags} {
		for name, present := range group {
			if !present {
				return fmt.Errorf("handoff_policy_facts_missing: %s.%s is required (use an empty list or false, never omit it)", PolicyFactsKey, name)
			}
		}
	}
	return nil
}
