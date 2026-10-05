package dojo

import (
	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

type domainCalibration = domain.DojoCalibration

func summaryZero() calibration.Summary { return calibration.Summary{} }

// whatIf is what an offline recompute changes about the reading of stored observations.
type whatIf struct {
	exclude, local, criteria []string
	gateRule                 string
	judgeOnly                bool
}

func (f calibrateFlags) whatIf() whatIf {
	return whatIf{exclude: f.exclude, local: f.whatIfLocal, criteria: f.excludeCriteria, gateRule: f.gateRule, judgeOnly: f.judgeOnly}
}

// exploratory reports whether the reading is a what-if: it persists nothing and yields no record.
func (w whatIf) exploratory() bool {
	return len(w.exclude) > 0 || len(w.local) > 0 || len(w.criteria) > 0 || w.judgeOnly
}
