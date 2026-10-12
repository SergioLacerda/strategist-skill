package dojo

import (
	"fmt"
	"strings"
	"time"

	internaldojo "github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
)

// termFor builds the consent term of a judge-layer run: what leaves the machine,
// where to and how much, computed from the same plan the dry run prints.
func termFor(deps Dependencies, cc caseContext, corpus internaldojo.Corpus, cases []calibration.Case, plan Plan, reps int) internaldojo.ConsentTerm {
	cal := *cc.criteria.Calibration
	calls := calibration.Plan(cases, reps)
	if cal.Budget > 0 && cal.Budget < calls {
		calls = cal.Budget
	}
	class := internaldojo.DataReal
	if cal.Corpus.Synthetic {
		class = internaldojo.DataSynthetic
	}
	return internaldojo.NewTerm(internaldojo.ConsentTerm{
		Case: cc.criteria.Scenario, Layer: layerOf(cc.criteria), Provider: providerName(cal), Endpoint: deps.Calibration.Endpoint(cc.strategistRoot), Profile: calibration.ResolveProfile(cal.Profile),
		DataClass: class, Fields: strings.Join(plan.Fields, ", "), Corpus: corpus.Manifest.Digest, Cases: len(cases), Calls: calls,
		Bytes: plan.Bytes * reps, EstimatedTokens: estimateTokens(plan.Bytes*reps, calls),
	})
}

// liveTerm is the consent term of the live smoke check: synthetic text only.
func liveTerm(deps Dependencies, cc caseContext) internaldojo.ConsentTerm {
	calls, bytes := deps.Calibration.LivePlan()
	return internaldojo.NewTerm(internaldojo.ConsentTerm{
		Case: cc.criteria.Scenario, Layer: "live", Provider: providerName(*cc.criteria.Calibration), Endpoint: deps.Calibration.Endpoint(cc.strategistRoot),
		DataClass: internaldojo.DataProbe, Cases: calls, Calls: calls, Bytes: bytes, EstimatedTokens: estimateTokens(bytes, calls),
	})
}

func providerName(cal domainCalibration) string {
	if cal.Provider == "" {
		return "jev"
	}
	return cal.Provider
}

// assent checks the consent flags and returns the record of the assent. Without
// --consent nothing is sent, and a bound term id must match the plan.
func assent(term internaldojo.ConsentTerm, flags calibrateFlags) (internaldojo.ConsentRecord, error) {
	if !flags.consent {
		return internaldojo.ConsentRecord{}, fmt.Errorf("dojo calibrate: this run calls an external provider and needs --consent; run with --dry-run to read the consent term first")
	}
	record, err := internaldojo.Assent(term, flags.consentTerm, time.Now())
	if err != nil {
		return internaldojo.ConsentRecord{}, fmt.Errorf("dojo calibrate: %w", err)
	}
	return record, nil
}
