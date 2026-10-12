package calibration

// chooser is the threshold selection rule of a gate rule: the lowest false
// rejection on the edge of the feasible set, or, under cross-validation, the
// middle of its widest band.
func (cfg Config) chooser() func([]SweepRow) (float64, bool) {
	if cfg.Limits.CVFolds > 0 {
		return RecommendCentered
	}
	return Recommend
}

// decideStatus applies, in order: the fitness gates, the sample-size rule, the
// sweep, and the confirmation. A failing step names itself and gives no
// recommendation. sel is the set the gates and the sweep read: the selection part
// under a held-out split, the whole corpus under cross-validation.
func decideStatus(report *Report, sel, held []CaseStat, cfg Config) {
	report.Notes = SampleDeficits(sel, cfg.Policy)
	if open := FirstOpenGate(report.Gates); open != "" {
		report.Status, report.FailingGate = StatusNotFit, open
		return
	}
	if len(report.Notes) > 0 {
		report.Status = StatusInsufficientSample
		return
	}
	threshold, ok := cfg.chooser()(report.Sweep)
	if !ok {
		report.Status = StatusNoFeasibleThreshold
		return
	}
	if cfg.Limits.CVFolds > 0 {
		confirmCrossValidated(report, sel, threshold, cfg)
		return
	}
	confirmHeldOut(report, sel, held, threshold, cfg)
}

func confirmHeldOut(report *Report, dev, held []CaseStat, threshold float64, cfg Config) {
	selection, heldOut := setResult(dev, threshold, cfg.Policy.ClusterRho), setResult(held, threshold, cfg.Policy.ClusterRho)
	report.Selection, report.HeldOut = &selection, &heldOut
	switch {
	case heldOut.Positives == 0 || heldOut.Defects == 0:
		report.Status = StatusInsufficientSample
		report.Notes = append(report.Notes, "held-out set has no golden positives or no defect negatives")
	case heldOut.FalseApproval.Value > cfg.Policy.MaxFalseApproval || heldOut.FalseRejection.Value > cfg.Policy.MaxFalseRejection:
		report.Status, report.FailingGate = StatusNotFit, "held_out"
	default:
		report.Status, report.Threshold = StatusRecommended, &threshold
	}
}

// confirmCrossValidated reads the threshold band and then the out-of-fold bounds.
func confirmCrossValidated(report *Report, sel []CaseStat, threshold float64, cfg Config) {
	selection := setResult(sel, threshold, cfg.Policy.ClusterRho)
	report.Selection = &selection
	band := bandGate(report.Sweep, sel, cfg.Limits)
	report.Gates = append(report.Gates, band)
	if band.Status != GatePassed {
		report.Status, report.FailingGate = StatusNotFit, band.Name
		return
	}
	cv := CrossValidate(sel, cfg.Policy, cfg.Limits.CVFolds, cfg.chooser())
	report.CrossValidation = &cv
	if !cv.Confirmed(cfg.Policy) {
		report.Status, report.FailingGate = StatusNotFit, "cross_validation"
		report.Notes = append(report.Notes, cv.Note(cfg.Policy))
		return
	}
	report.Status, report.Threshold = StatusRecommended, &threshold
}
