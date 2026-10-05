package handoffconsumer

import (
	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
)

// CalibrationResolver loads a published calibration record by id. The CLI
// supplies it from the DOJO storage domain, so this package never reads files
// outside its own concerns.
type CalibrationResolver func(id string) (calibration.Record, error)

// Reasons a pinned calibration is not applied, besides the calibration.Reason ones.
const (
	ReasonNonePinned        = "no_calibration_pinned"
	ReasonRecordUnavailable = "record_unavailable"
	// ReasonProbabilityUnavailable: the answer carried no probability of accept.
	ReasonProbabilityUnavailable = "probability_unavailable"
)

// CalibrationUse is how a call's threshold was decided: by a valid pinned
// calibration or by the configured default. The default is never silently
// replaced: a calibration applies only while it is valid for the resolved model
// and the question profile, and is otherwise reported with the reason.
type CalibrationUse struct {
	ID        string
	Applied   bool
	Reason    string
	Threshold float64
	Samples   int
}

// BoundaryOf maps a consuming transition to its calibration boundary.
func BoundaryOf(transition string) string {
	if transition == handoff.TransitionRangerToArchivist {
		return calibration.BoundaryRanger
	}
	return calibration.BoundaryPackage
}

// calibrationFor resolves the pinned calibration for one call, after the
// provider answered and the resolved model is known.
func (c *Consumer) calibrationFor(transition, model string) CalibrationUse {
	id := c.Provider.CalibrationID
	switch {
	case id == "":
		return CalibrationUse{Reason: ReasonNonePinned}
	case c.Calibrations == nil:
		return CalibrationUse{ID: id, Reason: ReasonRecordUnavailable}
	}
	record, err := c.Calibrations(id)
	if err != nil {
		return CalibrationUse{ID: id, Reason: ReasonRecordUnavailable}
	}
	if reason := record.UsableFor(BindingFor(c.Provider.Profile, BoundaryOf(transition), model), c.now(), c.Provider.CalibrationMaxAge()); reason != calibration.ReasonValid {
		return CalibrationUse{ID: id, Reason: string(reason)}
	}
	return CalibrationUse{ID: id, Applied: true, Threshold: *record.Threshold, Samples: record.Counts.Golden + record.Counts.Defects}
}

// threshold is the threshold in force: a valid calibration's, else the configured one.
func (c *Consumer) threshold(use CalibrationUse) float64 {
	if use.Applied {
		return use.Threshold
	}
	return c.Provider.Threshold()
}
