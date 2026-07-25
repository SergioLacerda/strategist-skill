package main

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
)

func renderWarningsSection(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("  WARNINGS")
	for _, warn := range warnings {
		fmt.Println("  " + warn)
	}
}

func collectWarnings(rows []treasure.StatusRow, govErr, idxErr, compErr error, compiledAt int64) []string {
	var w []string

	w = append(w, loadWarnings(govErr, idxErr)...)
	w = append(w, compiledIndexWarnings(compErr, compiledAt)...)

	var driftIDs []string
	var historicalMissing []string
	for _, r := range rows {
		driftIDs = appendDriftID(driftIDs, r)
		historicalMissing = appendHistoricalMissing(historicalMissing, r)
	}
	return appendTreasureWarnings(w, driftIDs, historicalMissing, compiledAt)
}

func loadWarnings(govErr, idxErr error) []string {
	var warnings []string
	if govErr != nil {
		warnings = append(warnings, "⚠ treasure-chests.yaml unavailable: "+govErr.Error())
	}
	if idxErr != nil {
		warnings = append(warnings, "⚠ knowledge.index.yaml unavailable: "+idxErr.Error())
	}
	return warnings
}

func compiledIndexWarnings(compErr error, compiledAt int64) []string {
	if compErr != nil {
		return []string{"⚠ .compiled/.index.gz corrupt — run: strategist treasure-chest --index"}
	}
	if compiledAt == 0 {
		return []string{"⚠ compiled index absent — run: strategist treasure-chest --index"}
	}
	return nil
}

func appendDriftID(ids []string, r treasure.StatusRow) []string {
	if len(r.Drift) == 0 {
		return ids
	}
	return append(ids, r.ID+"("+strings.Join(r.Drift, ",")+")")
}

func appendHistoricalMissing(ids []string, r treasure.StatusRow) []string {
	if (r.TrustTier == "T2" || r.TrustTier == "T3") && r.LastReviewed == "" {
		return append(ids, r.ID)
	}
	return ids
}

func appendTreasureWarnings(warnings, driftIDs, historicalMissing []string, compiledAt int64) []string {
	if len(driftIDs) > 0 {
		warnings = append(warnings, "⚠ drift detected: "+strings.Join(driftIDs, " "))
		if compiledAt != 0 {
			warnings = append(warnings, "  → run: strategist treasure-chest --index to refresh")
		}
	}
	if len(historicalMissing) > 0 {
		warnings = append(warnings, "⚠ historical sources missing last_reviewed (freshness=unknown): "+strings.Join(historicalMissing, ", "))
	}
	return warnings
}
