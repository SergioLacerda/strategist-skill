package treasure

import (
	"fmt"
	"strings"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// validateJewelEvidenceFields checks the additive evidence_class/
// evidence_confidence/valid_until fields: each is independently optional,
// but when set must be a value domain.Evidence's own vocabulary (or, for
// valid_until, RFC3339) already recognizes.
func validateJewelEvidenceFields(j Jewel) error {
	if j.EvidenceClass != "" {
		if _, ok := allowedJewelEvidenceClasses[j.EvidenceClass]; !ok {
			return fmt.Errorf("jewel %q has invalid evidence_class %q", j.ID, j.EvidenceClass)
		}
	}
	if j.EvidenceConfidence != "" {
		if _, ok := allowedJewelEvidenceConfidences[j.EvidenceConfidence]; !ok {
			return fmt.Errorf("jewel %q has invalid evidence_confidence %q", j.ID, j.EvidenceConfidence)
		}
	}
	if j.ValidUntil != "" {
		if _, err := time.Parse(time.RFC3339, j.ValidUntil); err != nil {
			return fmt.Errorf("jewel %q has invalid valid_until %q: not RFC3339", j.ID, j.ValidUntil)
		}
	}
	return nil
}

var allowedJewelEvidenceClasses = stringSet(
	domain.EvidenceClassExplicit,
	domain.EvidenceClassCorroboratedInference,
	domain.EvidenceClassWeakInference,
	domain.EvidenceClassUnknown,
)

var allowedJewelEvidenceConfidences = stringSet(
	domain.ConfidenceLow,
	domain.ConfidenceMedium,
	domain.ConfidenceHigh,
)

// EvidenceQualityReport aggregates the three advisory checks
// (expiration/dedup/conflict) over one chest's jewels. Advisory only —
// nothing in this file ever modifies a jewel.
type EvidenceQualityReport struct {
	ChestID    string
	Expired    []ExpiredJewel
	Duplicates []DedupCandidate
	Conflicts  []ConflictCandidate
}

// HasFindings reports whether the report has anything to show.
func (r EvidenceQualityReport) HasFindings() bool {
	return len(r.Expired) > 0 || len(r.Duplicates) > 0 || len(r.Conflicts) > 0
}

// CheckEvidenceQuality runs all three advisory checks over chestID's
// jewels, per design.md § Detection Flows.
func CheckEvidenceQuality(chestID string, jewels []Jewel, now time.Time) EvidenceQualityReport {
	return EvidenceQualityReport{
		ChestID:    chestID,
		Expired:    ScanExpiredJewels(chestID, jewels, now),
		Duplicates: ScanDuplicateJewels(chestID, jewels),
		Conflicts:  ScanConflictingJewels(chestID, jewels),
	}
}

// ExpiredJewel is one advisory expiration finding: valid_until is set and
// in the past relative to the scan time.
type ExpiredJewel struct {
	ChestID    string
	JewelID    string
	ValidUntil string
}

// ScanExpiredJewels flags jewels whose valid_until has passed. A jewel with
// no valid_until, or a malformed one, is never flagged — schema validity is
// validateJewelEvidenceFields's job, not this advisory check's.
func ScanExpiredJewels(chestID string, jewels []Jewel, now time.Time) []ExpiredJewel {
	var found []ExpiredJewel
	for _, j := range jewels {
		if j.ValidUntil == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, j.ValidUntil)
		if err != nil {
			continue
		}
		if t.Before(now) {
			found = append(found, ExpiredJewel{ChestID: chestID, JewelID: j.ID, ValidUntil: j.ValidUntil})
		}
	}
	return found
}

// DedupCandidate is one advisory dedup finding: two jewels in the same
// chest with the same normalized statement.
type DedupCandidate struct {
	ChestID  string
	JewelIDA string
	JewelIDB string
}

// ScanDuplicateJewels flags jewel pairs whose statements match after
// trimming and case-folding — exact-ish text match only, no semantic
// similarity (design.md's explicit v1 scope, to avoid fuzzy-matcher false
// positives).
func ScanDuplicateJewels(chestID string, jewels []Jewel) []DedupCandidate {
	var found []DedupCandidate
	for i := 0; i < len(jewels); i++ {
		a := normalizeJewelStatement(jewels[i].Statement)
		if a == "" {
			continue
		}
		for k := i + 1; k < len(jewels); k++ {
			if a == normalizeJewelStatement(jewels[k].Statement) {
				found = append(found, DedupCandidate{ChestID: chestID, JewelIDA: jewels[i].ID, JewelIDB: jewels[k].ID})
			}
		}
	}
	return found
}

func normalizeJewelStatement(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ConflictCandidate is one advisory conflict finding: two jewels in the
// same chest whose source_refs overlap while their statement or status
// differs — a human-review trigger, never auto-resolved.
type ConflictCandidate struct {
	ChestID  string
	JewelIDA string
	JewelIDB string
	Reason   string
}

// ScanConflictingJewels flags jewel pairs with overlapping source_refs and
// a differing statement or status, per design.md § Conflict. Heuristic
// only — no semantic understanding of whether the two statements actually
// contradict each other.
func ScanConflictingJewels(chestID string, jewels []Jewel) []ConflictCandidate {
	var found []ConflictCandidate
	for i := 0; i < len(jewels); i++ {
		for k := i + 1; k < len(jewels); k++ {
			a, b := jewels[i], jewels[k]
			if !jewelSourceRefsOverlap(a.SourceRefs, b.SourceRefs) {
				continue
			}
			reason := jewelConflictReason(a, b)
			if reason == "" {
				continue
			}
			found = append(found, ConflictCandidate{ChestID: chestID, JewelIDA: a.ID, JewelIDB: b.ID, Reason: reason})
		}
	}
	return found
}

func jewelSourceRefsOverlap(a, b []string) bool {
	seen := make(map[string]struct{}, len(a))
	for _, ref := range a {
		seen[ref] = struct{}{}
	}
	for _, ref := range b {
		if _, ok := seen[ref]; ok {
			return true
		}
	}
	return false
}

func jewelConflictReason(a, b Jewel) string {
	var parts []string
	if a.Statement != b.Statement {
		parts = append(parts, "differing statement")
	}
	if a.Status != b.Status {
		parts = append(parts, "differing status")
	}
	return strings.Join(parts, ", ")
}
