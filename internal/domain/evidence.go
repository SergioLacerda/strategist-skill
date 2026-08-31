package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

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
	// Hash completes critique_skill.txt item 3's "snippet/hash" link: a
	// sha256 hex digest (see HashExcerpt) of the cited excerpt text, or of
	// the whole SourceRef file when only file-level (not excerpt-level)
	// content was available to the caller at construction time. Empty when
	// no hashable content was available — e.g. a hand-authored citation with
	// no excerpt captured. Document at each call site which of the two
	// (excerpt vs. whole file) was hashed.
	Hash string `yaml:"hash,omitempty"`
	// ExcerptAnchor identifies exactly which part of SourceRef was cited —
	// e.g. a line range ("L12-L34"), a heading/section anchor
	// ("#consolidated-decision-evidence-model"), or a YAML path. Empty means
	// the citation is to the whole source rather than a specific excerpt.
	ExcerptAnchor string `yaml:"excerpt_anchor,omitempty"`
	// Commit is the git commit SHA the citation was verified against, when
	// SourceRef lives in a git-tracked file. Empty for non-git sources (a
	// URL, a human conversation, a generated artifact) or when the caller
	// did not resolve a commit at construction time.
	Commit string `yaml:"commit,omitempty"`
}

// NewEvidence builds an Evidence record and derives Hash from excerpt (a
// sha256 hex digest via HashExcerpt) whenever excerpt is non-empty — the
// "finding → source → snippet/hash" chain's snippet half. Pass an empty
// excerpt when only file-level, not excerpt-level, content is practical to
// hash at the call site (Hash then stays empty; hashing a whole source
// file's bytes is the caller's job, since this package has no filesystem
// dependency). excerptAnchor and commit are stored verbatim — see the
// Evidence field docs for what each means and when it's expected to be
// empty.
func NewEvidence(id, sourceRef, class, confidence, excerpt, excerptAnchor, commit string) Evidence {
	e := Evidence{
		ID:            id,
		SourceRef:     sourceRef,
		Class:         class,
		Confidence:    confidence,
		ExcerptAnchor: excerptAnchor,
		Commit:        commit,
	}
	if excerpt != "" {
		e.Hash = HashExcerpt(excerpt)
	}
	return e
}

// HashExcerpt returns the sha256 hex digest of excerpt text — the reusable
// primitive behind Evidence.Hash, exported so callers that build an Evidence
// value field-by-field (e.g. via YAML unmarshal, then filling Hash in after
// the fact) can compute the same digest NewEvidence would.
func HashExcerpt(excerpt string) string {
	sum := sha256.Sum256([]byte(excerpt))
	return hex.EncodeToString(sum[:])
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
