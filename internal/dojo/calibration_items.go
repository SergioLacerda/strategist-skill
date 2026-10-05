package dojo

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func calibrationItems(in CheckInput, out CheckOutcome, observed, malformed int) []domain.DojoCheckItem {
	if observed == 0 {
		return []domain.DojoCheckItem{item("calibration observations", false, "no observations at %s: run `strategist dojo calibrate %s`", in.Paths.ObservationsPath, in.Criteria.Scenario)}
	}
	r := out.Report
	items := append(exploratoryItems(r), backfillItems(out)...)
	items = append(items, item("calibration status", r.Status == calibration.StatusRecommended, "status=%s failing_gate=%s notes=%v malformed_lines=%d", r.Status, r.FailingGate, r.Notes, malformed))
	for _, gate := range r.Gates {
		items = append(items, item("calibration gate "+gate.Name, gate.Status == calibration.GatePassed, "%s value=%.3f limit=%.3f %s", gate.Status, gate.Value, gate.Limit, gate.Detail))
	}
	items = append(append(items, extraItems(r)...), riskItems(r)...)
	return append(items, staleItems(in, out, r)...)
}

func riskItems(r calibration.Report) []domain.DojoCheckItem {
	at, p := r.AtCurrent, r.Policy
	items := []domain.DojoCheckItem{
		item(fmt.Sprintf("calibration false_approval@%.2f", at.Threshold), at.Defects > 0 && at.FalseApproval.Value <= p.MaxFalseApproval,
			"%d of %d defects approved (%.1f%%, 95%% interval %.1f-%.1f%%, upper bound %.1f%%%s), limit %.1f%%", at.FalseApproval.K, at.Defects, 100*at.FalseApproval.Value,
			100*at.FalseApproval.Low, 100*at.FalseApproval.High, 100*at.FalseApproval.Upper, effectiveText(at.FalseApproval), 100*p.MaxFalseApproval),
		item(fmt.Sprintf("calibration false_rejection@%.2f", at.Threshold), at.Positives > 0 && at.FalseRejection.Value <= p.MaxFalseRejection,
			"%d of %d golden parents rejected (%.1f%%, 95%% interval %.1f-%.1f%%), limit %.1f%%", at.FalseRejection.K, at.Positives, 100*at.FalseRejection.Value,
			100*at.FalseRejection.Low, 100*at.FalseRejection.High, 100*p.MaxFalseRejection),
		refusalItem(r.Refusals),
	}
	items = append(items, policyItems(r)...)
	for _, op := range r.Operators {
		if op.Insensitive() {
			items = append(items, item("calibration insensitive "+op.Operator, false, "responded in %d of %d pairs", op.Sensitivity.K, op.Sensitivity.N))
		}
	}
	return items
}

func effectiveText(rate calibration.Rate) string {
	if rate.Effective == 0 {
		return ""
	}
	return fmt.Sprintf(", on an effective %d of %d", rate.Effective, rate.N)
}

// refusalItem is informational: it says why golden parents were refused, run by run.
func refusalItem(f calibration.Refusals) domain.DojoCheckItem {
	return item(fmt.Sprintf("calibration refusals@%.2f", f.Threshold), true, "%d runs of golden parents: %d approved, %d choice not accept, %d accept at or below the threshold, %d without a usable answer",
		f.Runs, f.Approved, f.ChoiceNotAccept, f.BelowThreshold, f.Invalid)
}

// policyItems measure the whole decision (judge and local checks) when the local checks were evaluated.
func policyItems(r calibration.Report) []domain.DojoCheckItem {
	level := r.PolicyLevel
	if level == nil {
		return nil
	}
	p := r.Policy
	return []domain.DojoCheckItem{
		item(fmt.Sprintf("calibration policy false_approval@%.2f", level.Threshold), level.FalseApproval.Value <= p.MaxFalseApproval,
			"the judge and the local checks together approved %d of %d defects (upper bound %.1f%%%s), limit %.1f%%", level.FalseApproval.K, level.FalseApproval.N,
			100*level.FalseApproval.Upper, effectiveText(level.FalseApproval), 100*p.MaxFalseApproval),
		item(fmt.Sprintf("calibration policy false_rejection@%.2f", level.Threshold), level.FalseRejection.Value <= p.MaxFalseRejection,
			"%d of %d golden parents not approved by the policy (%d flagged by a local check), limit %.1f%%", level.FalseRejection.K, level.FalseRejection.N, level.GoldenFlagged.K, 100*p.MaxFalseRejection),
	}
}

func staleItems(in CheckInput, out CheckOutcome, _ calibration.Report) []domain.DojoCheckItem {
	if in.ProfileVersion == "" || out.Version == "" || in.ProfileVersion == out.Version {
		return nil
	}
	return []domain.DojoCheckItem{item("calibration stale profile", false, "observations were taken with profile version %s, the boundary is now %s", out.Version, in.ProfileVersion)}
}

func toSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

// exploratoryItems label a what-if so it is never read as a calibration.
func exploratoryItems(r calibration.Report) []domain.DojoCheckItem {
	if len(r.Exploratory) == 0 {
		return nil
	}
	return []domain.DojoCheckItem{item("calibration exploratory what-if", true, "operators left out %v: this is not a calibration and none is published", r.Exploratory)}
}

// backfill sets the local flags of observations that were made without them. It
// changes the in-memory copy only and returns how many it changed.
func backfill(observations []calibration.Observation, flags map[string][]string) int {
	n := 0
	for i := range observations {
		if observations[i].Local {
			continue
		}
		if found, ok := flags[observations[i].CaseID]; ok {
			observations[i].Local, observations[i].LocalFlags = true, found
			n++
		}
	}
	return n
}

func backfillItems(out CheckOutcome) []domain.DojoCheckItem {
	if out.Backfilled == 0 {
		return nil
	}
	return []domain.DojoCheckItem{item("calibration local checks backfilled", true, "%d observations had no recorded local flags; they were derived now from the regenerated cases with the current local checks (the ledger is not rewritten)", out.Backfilled)}
}
