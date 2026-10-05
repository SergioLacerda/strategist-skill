package dojo

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// PairCheck is the offline evaluation of a pair case.
type PairCheck struct {
	Result domain.DojoCheckResult
	Report calibration.PairReport
	Model  string
}

// CheckPairs evaluates the stored pair observations with no call. Every dimension
// is an item: it passes only when the dimension is fit. Families are informational.
func CheckPairs(criteria domain.DojoCriteria, paths CasePaths, session string) (PairCheck, error) {
	observed, err := ReadPairObservations(paths.PairObservationsPath)
	if err != nil {
		return PairCheck{}, err
	}
	if session == "latest" {
		session = latestPairSession(observed)
	}
	observed = pairSession(observed, session)
	check := PairCheck{Report: calibration.EvaluatePairs(observed, calibration.DefaultPairLimits())}
	for _, o := range observed {
		if o.Model != "" {
			check.Model = o.Model
		}
	}
	check.Result = domain.DojoCheckResult{Scenario: criteria.Scenario, Items: pairItems(check.Report, len(observed), paths)}
	return check, nil
}

func pairItems(r calibration.PairReport, observed int, paths CasePaths) []domain.DojoCheckItem {
	if observed == 0 {
		return []domain.DojoCheckItem{item("calibration observations", false, "no pair observations at %s: run `strategist dojo calibrate <case>`", paths.PairObservationsPath)}
	}
	var items []domain.DojoCheckItem
	for _, d := range r.Dimensions {
		items = append(items, item("calibration dimension "+d.Dimension, d.Status == calibration.DimensionFit, "%s%s", d.Describe(), failingGateText(d)))
	}
	for _, f := range r.Families {
		items = append(items, item("calibration family "+f.Family, true, "%d pairs, responded %d (%.0f%%), mean delta %+.3f%s", f.Pairs, f.Responded.K, 100*f.Responded.Value, f.MeanDelta, dimensionText(f)))
	}
	return items
}

func failingGateText(d calibration.DimensionReport) string {
	if d.FailingGate == "" {
		return ""
	}
	return "; failing gate " + d.FailingGate
}

func dimensionText(f calibration.FamilyReport) string {
	if f.Dimension == "" {
		return " (control: responded means unchanged)"
	}
	return " (breaks " + f.Dimension + ")"
}

// ReadPairObservations reads the pair ledger; a missing file is no observations.
func ReadPairObservations(path string) ([]calibration.PairObservation, error) {
	var out []calibration.PairObservation
	bad := 0
	if err := readJSONL(path, func(raw []byte) bool {
		o, ok := decodePair(raw)
		if ok {
			out = append(out, o)
		}
		return ok
	}, &bad); err != nil {
		return nil, err
	}
	if bad > 0 {
		return out, fmt.Errorf("dojo: %d malformed lines in %s", bad, path)
	}
	return out, nil
}

func latestPairSession(observed []calibration.PairObservation) string {
	for i := len(observed) - 1; i >= 0; i-- {
		if observed[i].Session != "" {
			return observed[i].Session
		}
	}
	return ""
}

func pairSession(observed []calibration.PairObservation, session string) []calibration.PairObservation {
	if session == "" {
		return observed
	}
	var out []calibration.PairObservation
	for _, o := range observed {
		if o.Session == session {
			out = append(out, o)
		}
	}
	return out
}
