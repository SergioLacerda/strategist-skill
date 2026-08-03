package domain

import "errors"

// Evidence is a graded, sourced claim backing one or more Decisions —
// critique_skill.txt item 3's "finding → source → snippet/hash →
// classification → confidence" chain, refined by the evidence_classes
// vocabulary proposed in .analysis/todo/v2/pathfinder.txt (never promote an
// inference to a fact; cite every historical claim). See
// .analysis/done/20260803-critique-skill-affinity-review/design.md §
// "Consolidated Decision/Evidence Model".
type Evidence struct {
	ID         string `yaml:"id"`
	SourceRef  string `yaml:"source_ref"`
	Class      string `yaml:"class"`
	Confidence string `yaml:"confidence"`
	// ValidUntil is an optional RFC3339 timestamp string. Empty means no
	// expiry is declared. Interpreting/comparing it against "now" is a
	// caller concern (this package stays free of a time dependency);
	// EvaluateMissionQuality only checks that the field is well-formed
	// enough to be non-empty when the caller declares it.
	ValidUntil string `yaml:"valid_until,omitempty"`
}

// Evidence classification, per .analysis/todo/v2/pathfinder.txt's
// evidence_classes: explicit | corroborated_inference | weak_inference |
// unknown. This is fact_inference_separation's vocabulary (mission_quality.go)
// — an Evidence record with an empty or unrecognized Class cannot be
// distinguished from an unsupported claim dressed up as a citation.
const (
	EvidenceClassExplicit              = "explicit"
	EvidenceClassCorroboratedInference = "corroborated_inference"
	EvidenceClassWeakInference         = "weak_inference"
	EvidenceClassUnknown               = "unknown"
)

var allowedEvidenceClasses = stringSet(
	EvidenceClassExplicit,
	EvidenceClassCorroboratedInference,
	EvidenceClassWeakInference,
	EvidenceClassUnknown,
)

// ValidateEvidence checks an Evidence record against the required fields
// and allowed values documented in schemas/evidence.schema.yaml.
func ValidateEvidence(e Evidence) error {
	errs := validateNamedValue("evidence_invalid", "id", e.ID, nil)
	if e.SourceRef == "" {
		errs = append(errs, errors.New("evidence_invalid: source_ref is required"))
	}
	errs = append(errs, validateNamedValue("evidence_invalid", "class", e.Class, allowedEvidenceClasses)...)
	errs = append(errs, validateNamedValue("evidence_invalid", "confidence", e.Confidence, allowedConfidenceLevels)...)
	return errors.Join(errs...)
}
