// Package preparerunbook prepares a reviewable Runbook candidate from an
// Opportunity Attack. It does not select, accept, or persist a canonical
// Runbook.
package preparerunbook

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
)

// SchemaVersion identifies the prepared Runbook result format.
const SchemaVersion = "strategist-prepare-runbook/v1"

// Status is the local preparation outcome.
type Status string

const (
	// StatusPrepared identifies a complete prepared candidate.
	StatusPrepared Status = "prepared"
	// StatusDuplicate identifies an equivalent existing Runbook.
	StatusDuplicate Status = "duplicate"
	// StatusInsufficientEvidence identifies a candidate missing evidence.
	StatusInsufficientEvidence Status = "insufficient_evidence"
	// StatusNeedsReview identifies a candidate requiring review.
	StatusNeedsReview Status = "needs_review"
)

// ComparisonKind classifies the candidate against the Runbook inventory.
type ComparisonKind string

const (
	// ComparisonNovel identifies a candidate without an overlap.
	ComparisonNovel ComparisonKind = "novel"
	// ComparisonDuplicate identifies an equivalent existing Runbook.
	ComparisonDuplicate ComparisonKind = "duplicate"
	// ComparisonExtension identifies a candidate that may extend an existing Runbook.
	ComparisonExtension ComparisonKind = "extension"
	// ComparisonInsufficient identifies an unreadable or unavailable inventory.
	ComparisonInsufficient ComparisonKind = "insufficient_comparison"
)

// ProviderPath identifies the path selected after advisory provider evaluation.
type ProviderPath string

const (
	// ProviderPathMain identifies the deterministic local path.
	ProviderPathMain ProviderPath = "main"
	// ProviderPathJEV identifies advisory JEV evidence.
	ProviderPathJEV ProviderPath = "jev"
)

// Input is the immutable context of one preparation attempt.
type Input struct {
	MissionID            string
	CandidateID          string
	PackageDigest        string
	RefinedPath          string
	RunbookRoot          string
	SelectedRunbooksHint []string
	MaxInventory         int
	Evaluate             Evaluator
}

// ProviderRequest contains only the bounded, typed projection approved for a
// confidence evaluator. The evaluator must not receive the original files.
type ProviderRequest struct {
	MissionID   string
	CandidateID string
	Subject     string
	Fields      map[string]string
	Criteria    []string
}

// ProviderReading is advisory evidence. Approved is never used as a local
// authorization decision by Prepare.
type ProviderReading struct {
	Path          ProviderPath `yaml:"path" json:"path"`
	Reason        string       `yaml:"reason,omitempty" json:"reason,omitempty"`
	Confidence    float64      `yaml:"confidence,omitempty" json:"confidence,omitempty"`
	Threshold     float64      `yaml:"threshold,omitempty" json:"threshold,omitempty"`
	Approved      bool         `yaml:"approved,omitempty" json:"approved,omitempty"`
	Attempted     bool         `yaml:"attempted,omitempty" json:"attempted,omitempty"`
	BindingDigest string       `yaml:"binding_digest,omitempty" json:"binding_digest,omitempty"`
	Model         string       `yaml:"model,omitempty" json:"model,omitempty"`
}

// Evaluator is the bounded provider evaluation seam.
type Evaluator func(context.Context, ProviderRequest) (ProviderReading, error)

// ComparisonRef identifies an existing Runbook considered during comparison.
type ComparisonRef struct {
	RunbookID string `yaml:"runbook_id" json:"runbook_id"`
	Path      string `yaml:"path" json:"path"`
	Relation  string `yaml:"relation" json:"relation"`
}

// Comparison is the candidate's inventory comparison result.
type Comparison struct {
	Classification ComparisonKind  `yaml:"classification" json:"classification"`
	Compared       []ComparisonRef `yaml:"compared,omitempty" json:"compared,omitempty"`
	SelectedHints  []string        `yaml:"selected_hints,omitempty" json:"selected_hints,omitempty"`
	Reason         string          `yaml:"reason" json:"reason"`
}

// Evidence records the deterministic evidence fields found in the candidate.
type Evidence struct {
	Procedure       bool `yaml:"procedure" json:"procedure"`
	Reproducible    bool `yaml:"reproducible" json:"reproducible"`
	ExpectedOutputs bool `yaml:"expected_outputs" json:"expected_outputs"`
	StopConditions  bool `yaml:"stop_conditions" json:"stop_conditions"`
	Provenance      bool `yaml:"provenance" json:"provenance"`
	Scope           bool `yaml:"scope" json:"scope"`
}

// Missing returns the deterministic evidence fields that are absent.
func (e Evidence) Missing() []string {
	missing := make([]string, 0, 6)
	if !e.Procedure {
		missing = append(missing, "procedure")
	}
	if !e.Reproducible {
		missing = append(missing, "reproducible")
	}
	if !e.ExpectedOutputs {
		missing = append(missing, "expected_outputs")
	}
	if !e.StopConditions {
		missing = append(missing, "stop_conditions")
	}
	if !e.Provenance {
		missing = append(missing, "provenance")
	}
	if !e.Scope {
		missing = append(missing, "scope")
	}
	return missing
}

// Provenance identifies one source artifact used during preparation.
type Provenance struct {
	Path   string `yaml:"path" json:"path"`
	Role   string `yaml:"role" json:"role"`
	Digest string `yaml:"digest" json:"digest"`
}

// Result is the reviewable output of one Runbook preparation attempt.
type Result struct {
	SchemaVersion       string          `yaml:"schema_version" json:"schema_version"`
	Status              Status          `yaml:"status" json:"status"`
	MissionID           string          `yaml:"mission_id" json:"mission_id"`
	CandidateID         string          `yaml:"candidate_id" json:"candidate_id"`
	PackageDigest       string          `yaml:"package_digest" json:"package_digest"`
	SourceDigest        string          `yaml:"source_digest" json:"source_digest"`
	ProposedRunbook     runbook.Runbook `yaml:"proposed_runbook" json:"proposed_runbook"`
	Evidence            Evidence        `yaml:"evidence" json:"evidence"`
	Comparison          Comparison      `yaml:"comparison" json:"comparison"`
	Provider            ProviderReading `yaml:"provider" json:"provider"`
	UnresolvedQuestions []string        `yaml:"unresolved_questions,omitempty" json:"unresolved_questions,omitempty"`
	Provenance          []Provenance    `yaml:"provenance" json:"provenance"`
}
