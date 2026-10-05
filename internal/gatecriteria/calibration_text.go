package gatecriteria

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
)

const notCalibrated = "não calibrado"

// appliedRecordID is the calibration id only when that calibration decided the
// threshold of the recorded reading.
func appliedRecordID(record handoffconsumer.Record) string {
	if record.Calibrated {
		return record.CalibrationID
	}
	return ""
}

// calibrationText says, from evidence, whether the reading was calibrated: the
// numbers of the record that decided its threshold, or `não calibrado` with the
// reason a pinned calibration did not apply. It never invents a calibration.
func calibrationText(in Input, appliedID, reason string) string {
	if appliedID == "" {
		return unappliedText(reason)
	}
	record, err := dojo.LoadCalibration(in.BasePath, appliedID)
	if err != nil {
		return fmt.Sprintf("calibrado (%s)", appliedID)
	}
	facts := handoffconsumer.FactsOf(record)
	return fmt.Sprintf("calibrado (modelo %s, perfil %s, %d casos golden, falsa aprovação held-out %.1f%%, %s)", facts.Model, facts.Profile, facts.Golden, 100*facts.HeldOutFalseApproval, facts.Date)
}

func unappliedText(reason string) string {
	if reason == "" || reason == handoffconsumer.ReasonNonePinned {
		return notCalibrated
	}
	return fmt.Sprintf("%s (%s)", notCalibrated, reason)
}
