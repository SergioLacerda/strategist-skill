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
	return append(crossValidationItems(r), criteriaItems(r)...)
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
