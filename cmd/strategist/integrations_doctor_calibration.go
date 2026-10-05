package main

import (
	"fmt"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
)

const notCalibratedText = "não calibrado"

// calibrationState reports, from the pinned record and not from a call, whether
// the handoff would use a calibration now: for the resolved (or pinned) model and
// the question profile of either boundary, within its age limit.
func calibrationState(root string, provider config.Provider, model string, now time.Time) string {
	if provider.CalibrationID == "" {
		return notCalibratedText
	}
	record, err := calibrationResolver(root)(provider.CalibrationID)
	if err != nil {
		return fmt.Sprintf("%s (%s %s)", notCalibratedText, handoffconsumer.ReasonRecordUnavailable, provider.CalibrationID)
	}
	reason := calibration.ReasonProfileChanged
	for _, boundary := range []string{calibration.BoundaryPackage, calibration.BoundaryRanger} {
		if got := record.UsableFor(handoffconsumer.BindingFor(provider.Profile, boundary, model), now, provider.CalibrationMaxAge()); got == calibration.ReasonValid {
			facts := handoffconsumer.FactsOf(record)
			return fmt.Sprintf("calibrado (%s, model %s, profile %s, %d golden cases, held-out false approval %.1f%%, %s, threshold %.2f)",
				record.ID, facts.Model, facts.Profile, facts.Golden, 100*facts.HeldOutFalseApproval, facts.Date, facts.Threshold)
		} else if got != calibration.ReasonProfileChanged {
			reason = got
		}
	}
	return fmt.Sprintf("%s (%s %s)", notCalibratedText, reason, record.ID)
}
