// Package runbook defines typed domain types for runbook sidecars
// (docs/runbooks/*.runbook.yaml) and the selection/completion logic that
// operates on them. It is additive to the existing markdown-only runbook
// corpus — internal/treasure.ScanRunbookDirectory keeps reading the
// unchanged *.md files directly and never parses a sidecar. See
// .analysis/refined/20260803-runbook-domain-and-cutover/design.md.
package runbook

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Runbook mirrors the *.runbook.yaml sidecar schema documented in design.md
// § Sidecar Schema. Analytical runbooks (investigation/decision) populate
// Analysis/DecisionGates; operational runbooks (execute-and-verify)
// populate Verification instead — RunbookType disambiguates which fields
// apply, per runbook_v2.txt's own analytical/operational split.
type Runbook struct {
	SchemaVersion string `yaml:"schema_version"`
	RunbookID     string `yaml:"runbook_id"`
	RunbookType   string `yaml:"runbook_type"`
	SourceDoc     string `yaml:"source_doc"`

	AppliesWhen []string `yaml:"applies_when"`
	Objective   string   `yaml:"objective"`

	Preconditions []string       `yaml:"preconditions,omitempty"`
	Analysis      []string       `yaml:"analysis,omitempty"`
	DecisionGates []DecisionGate `yaml:"decision_gates,omitempty"`
	Verification  []string       `yaml:"verification,omitempty"`
	Checks        []Check        `yaml:"checks,omitempty"`

	Metadata Metadata `yaml:"metadata"`

	// Trust, EstimatedTokens, and ConflictsWithHigherTrust are not part of
	// the *.runbook.yaml sidecar schema (no yaml tag — ParseSidecar never
	// populates them) and are not part of docs/runbooks/README.md's
	// documented shape. A candidate's trust tier lives at the chest level
	// (treasure-chests.yaml#chests[].trust.tier — see
	// internal/treasure.LoadGoverned), not per-runbook, so the caller
	// assembling Select's candidate list is responsible for setting these
	// three fields from that context before calling Select. Zero values
	// (Trust="", EstimatedTokens=0, ConflictsWithHigherTrust=false) disable
	// the corresponding policy check entirely, so existing callers that
	// never set them keep selecting exactly as before these fields existed.
	Trust                    string `yaml:"-"`
	EstimatedTokens          int    `yaml:"-"`
	ConflictsWithHigherTrust bool   `yaml:"-"`
}

// Runbook type values, per runbook_v2.txt's analytical/operational split.
const (
	RunbookTypeAnalytical  = "analytical"
	RunbookTypeOperational = "operational"
)

var allowedRunbookTypes = stringSet(RunbookTypeAnalytical, RunbookTypeOperational)

// Metadata is the sidecar's bookkeeping block — who authored/owns it and
// when it was last reviewed against its paired markdown.
type Metadata struct {
	Version    int    `yaml:"version"`
	Owner      string `yaml:"owner,omitempty"`
	ReviewedAt string `yaml:"reviewed_at"`
}

// ParseSidecar decodes a *.runbook.yaml sidecar and validates it. It is the
// single authoritative parser for the sidecar schema — docs/runbooks/README.md
// documents the schema in terms of what this function accepts.
func ParseSidecar(data []byte) (Runbook, error) {
	var rb Runbook
	if err := yaml.Unmarshal(data, &rb); err != nil {
		return Runbook{}, fmt.Errorf("runbook_sidecar_invalid: %w", err)
	}
	if err := ValidateRunbook(rb); err != nil {
		return Runbook{}, err
	}
	return rb, nil
}

// ValidateRunbook checks a Runbook against the required fields and allowed
// values documented in design.md § Sidecar Schema, cascading into every
// nested Check and DecisionGate.
func ValidateRunbook(rb Runbook) error {
	var errs []error
	errs = append(errs, validateNamedValue("runbook_invalid", "schema_version", rb.SchemaVersion, nil)...)
	errs = append(errs, validateNamedValue("runbook_invalid", "runbook_id", rb.RunbookID, nil)...)
	errs = append(errs, validateNamedValue("runbook_invalid", "runbook_type", rb.RunbookType, allowedRunbookTypes)...)
	errs = append(errs, validateNamedValue("runbook_invalid", "source_doc", rb.SourceDoc, nil)...)
	errs = append(errs, validateNamedValue("runbook_invalid", "objective", rb.Objective, nil)...)
	if len(rb.AppliesWhen) == 0 {
		errs = append(errs, errors.New("runbook_invalid: applies_when is required and must be non-empty"))
	}

	for _, gate := range rb.DecisionGates {
		if err := ValidateDecisionGate(gate); err != nil {
			errs = append(errs, err)
		}
	}
	for _, check := range rb.Checks {
		if err := ValidateCheck(check); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
