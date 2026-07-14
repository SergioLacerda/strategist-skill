package domain

import (
	"fmt"
	"strings"
)

var validSourceGrades = map[string]bool{"A": true, "B": true, "C": true}

var validReuseValues = map[string]bool{"high": true, "medium": true, "low": true}

var validImplementationStatuses = map[string]bool{
	"not_started": true, "in_progress": true, "implemented": true, "deprecated": true,
}

// ChestGrade holds SQ-002 (Track T-G) source grading fields for a treasure chest.
// All fields are optional and human-reviewed only — no derived or learned values.
type ChestGrade struct {
	SourceGrade          string
	ReuseValue           string
	ImplementationStatus string
}

// ValidateChestGrade returns an error if any set field is not one of its enumerated
// values. Empty fields are valid — grading is optional.
func ValidateChestGrade(chestID string, g ChestGrade) error {
	var errs []string
	if g.SourceGrade != "" && !validSourceGrades[g.SourceGrade] {
		errs = append(errs, fmt.Sprintf("source_grade %q must be one of A, B, C", g.SourceGrade))
	}
	if g.ReuseValue != "" && !validReuseValues[g.ReuseValue] {
		errs = append(errs, fmt.Sprintf("reuse_value %q must be one of high, medium, low", g.ReuseValue))
	}
	if g.ImplementationStatus != "" && !validImplementationStatuses[g.ImplementationStatus] {
		errs = append(errs, fmt.Sprintf(
			"implementation_status %q must be one of not_started, in_progress, implemented, deprecated",
			g.ImplementationStatus))
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("chest %q grade invalid: %s", chestID, strings.Join(errs, "; "))
}
