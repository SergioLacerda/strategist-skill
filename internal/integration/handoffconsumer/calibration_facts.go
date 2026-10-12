package handoffconsumer

import "github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"

// CalibrationFacts are the content-free facts a display shows about a
// calibration: which judge it is for, how much it stands on and how the
// held-out check went.
type CalibrationFacts struct {
	ID                   string
	Model                string
	Profile              string
	ProfileVersion       string
	Golden               int
	Defects              int
	HeldOutFalseApproval float64
	Threshold            float64
	Date                 string
	Status               calibration.Status
}

// FactsOf extracts the display facts of a record.
func FactsOf(record calibration.Record) CalibrationFacts {
	facts := CalibrationFacts{
		ID: record.ID, Model: record.Model, Profile: record.Profile, ProfileVersion: record.ProfileVersion,
		Golden: record.Counts.Golden, Defects: record.Counts.Defects, Status: record.Status,
	}
	if len(record.CreatedAt) >= len("2006-01-02") {
		facts.Date = record.CreatedAt[:len("2006-01-02")]
	}
	if record.HeldOut != nil {
		facts.HeldOutFalseApproval = record.HeldOut.FalseApproval.Value
	}
	if record.Threshold != nil {
		facts.Threshold = *record.Threshold
	}
	return facts
}
