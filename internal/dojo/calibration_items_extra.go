package dojo

import (
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// extraItems are the informational readings of an atomic profile and of a
// cross-validated rule; a failing confirmation is already the status item.
func extraItems(r calibration.Report) []domain.DojoCheckItem {
	return append(append(crossValidationItems(r), criterionItems(r)...), criteriaItems(r)...)
}

func crossValidationItems(r calibration.Report) []domain.DojoCheckItem {
	cv := r.CrossValidation
	if cv == nil {
		return nil
	}
	text := fmt.Sprintf("%d folds, thresholds %v; out-of-fold false approval %d of %d (upper bound %.1f%%%s), false rejection %d of %d (upper bound %.1f%%); %d folds without a feasible threshold",
		cv.Folds, cv.Thresholds, cv.FalseApproval.K, cv.FalseApproval.N, 100*cv.FalseApproval.Upper, effectiveText(cv.FalseApproval),
		cv.FalseRejection.K, cv.FalseRejection.N, 100*cv.FalseRejection.Upper, cv.Infeasible)
	return []domain.DojoCheckItem{item("calibration cross-validation", cv.Confirmed(r.Policy), "%s", text)}
}

// criteriaItems print one row per group of cases: the mean answer of each criterion.
func criteriaItems(r calibration.Report) []domain.DojoCheckItem {
	if r.Criteria == nil {
		return nil
	}
	var items []domain.DojoCheckItem
	for _, row := range r.Criteria.Rows {
		items = append(items, item("calibration criteria "+row.Group, true, "%d runs: %s", row.Runs, meansText(row.Means)))
	}
	return items
}

func meansText(means map[string]float64) string {
	names := make([]string, 0, len(means))
	for name := range means {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = fmt.Sprintf("%s %.2f", name, means[name])
	}
	return strings.Join(parts, ", ")
}

// criterionItems print one row per criterion read on its own; they are informational (the case
// passes or fails on the aggregate), and each says what a hint from that criterion is worth.
func criterionItems(r calibration.Report) []domain.DojoCheckItem {
	var items []domain.DojoCheckItem
	for _, c := range r.CriterionCalibration {
		detail := fmt.Sprintf("%s: auc %.3f over %d golden parents and %d defects of %v; flag below %.2f, out of fold %d of %d golden flagged (upper %.1f%%), %d of %d defects caught (%.0f%%)",
			c.Status, c.AUC, c.Golden, c.Defects, c.Operators, c.Threshold, c.FalseFlag.K, c.FalseFlag.N, 100*c.FalseFlag.Upper, c.Detection.K, c.Detection.N, 100*c.Detection.Value)
		if c.Note != "" {
			detail += " (" + c.Note + ")"
		}
		items = append(items, item("calibration criterion "+c.Criterion, true, "%s", detail))
	}
	return items
}
