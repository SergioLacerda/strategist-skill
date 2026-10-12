package gatecriteria

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

const (
	outcomeSource = "missions/handoff"
	metricsSource = "memory/handoff-metrics.jsonl"
)

// handoffRow reports the deterministic handoff outcome of a lifecycle transition.
func handoffRow(in Input, criterion, transition, absent string) Row {
	outcome, err := handoff.NewOutcomeStore(in.Root).LatestFor(in.MissionID, transition)
	switch {
	case err != nil && isMissingOutcome(err):
		return notEvaluated(criterion, outcomeSource, absent)
	case err != nil:
		return unknown(criterion, outcomeSource, "o outcome registrado não pôde ser verificado")
	}
	value := fmt.Sprintf("%s (attempt %d, critical_failures %d)", outcome.Result, outcome.Attempt, outcome.CriticalFailures)
	return evaluated(criterion, value, outcomeSource, "desafio determinístico, "+orDefault(outcome.ChallengeStatus, "sem desafio exigido"))
}

func isMissingOutcome(err error) bool {
	return errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "handoff_outcome_missing")
}

// refinementMetricsRow reports the refinement handoff metrics already recorded.
func refinementMetricsRow(in Input) Row {
	lines, err := telemetry.ReadRefinementHandoffLines(telemetry.HandoffMetricsPath(in.Root))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return unknown("refinement_handoff_metrics", metricsSource, "o ledger de métricas não pôde ser lido")
	}
	latest := latestMetrics(lines, in.MissionID)
	if latest == nil || latest.RefinementReopens == nil {
		return notEvaluated("refinement_handoff_metrics", metricsSource, "reaberturas não medidas")
	}
	return evaluated("refinement_handoff_metrics", fmt.Sprintf("reopens=%d", *latest.RefinementReopens), metricsSource, coverageDetail(latest))
}

func latestMetrics(lines []telemetry.RefinementHandoffLine, missionID string) *telemetry.RefinementHandoffLine {
	var latest *telemetry.RefinementHandoffLine
	for i := range lines {
		if lines[i].MissionID == missionID {
			latest = &lines[i]
		}
	}
	return latest
}

func coverageDetail(line *telemetry.RefinementHandoffLine) string {
	if line.EvidenceCoverageRatio == nil {
		return "evidence_coverage_ratio: não avaliado"
	}
	return fmt.Sprintf("evidence_coverage_ratio: %.2f", *line.EvidenceCoverageRatio)
}
