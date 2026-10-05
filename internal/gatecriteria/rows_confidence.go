package gatecriteria

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

const (
	routeSource  = "memory/route-decisions.jsonl"
	claimsSource = "memory/confidence-records.jsonl"
)

// claims holds a mission's confidence records, or why they are unreadable.
type claims struct {
	records []telemetry.ConfidenceRecord
	err     error
}

func loadClaims(in Input) claims {
	records, err := telemetry.ReadConfidenceRecords(telemetry.ConfidenceHistoryPath(in.Root))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return claims{err: err}
	}
	var mine []telemetry.ConfidenceRecord
	for _, record := range records {
		if record.MissionID == in.MissionID {
			mine = append(mine, record)
		}
	}
	return claims{records: mine}
}

// claimRow reports the latest record of an agent for the mission.
func claimRow(c claims, criterion, agent string) Row {
	if c.err != nil {
		return unknown(criterion, claimsSource, "o ledger de confiança não pôde ser lido")
	}
	var latest *telemetry.ConfidenceRecord
	for i := range c.records {
		if c.records[i].Agent == agent {
			latest = &c.records[i]
		}
	}
	switch {
	case latest == nil:
		return notEvaluated(criterion, claimsSource, "nenhum registro de "+agent)
	case latest.CoverageStatus == telemetry.ConfidenceCoverageMissing:
		return notEvaluated(criterion, claimsSource, "registro ausente: "+orDefault(latest.MissingReason, "motivo não informado"))
	case latest.ClaimKind == domain.ClaimKindQuestion:
		return evaluated(criterion, "pergunta em aberto (sem percentual)", claimsSource, latest.ClaimID)
	}
	return evaluated(criterion, fmt.Sprintf("%d%% (%s)", latest.ConfidencePercent, latest.ClaimKind), claimsSource, latest.ClaimID)
}

// reviewRow reuses the confidence gate review that already exists.
func reviewRow(in Input) Row {
	review, err := telemetry.LoadConfidenceGateReview(in.Root, in.MissionID)
	switch {
	case err != nil:
		return unknown("confidence_review", claimsSource, "a revisão de confiança não pôde ser carregada")
	case review.Unavailable:
		return notEvaluated("confidence_review", claimsSource, "sem evidência de confiança registrada")
	}
	m := review.Metrics
	value := fmt.Sprintf("review_required=%t; calibration=%s; evidence_coverage=%.2f; unsupported=%.2f; sample=%d",
		review.ReviewRequired, m.CalibrationStatus, m.AssertionEvidenceCoverage, m.UnsupportedAssertionRate, m.SampleSize)
	return evaluated("confidence_review", value, claimsSource, "telemetry.LoadConfidenceGateReview")
}

type routeLine struct {
	MissionID       string  `json:"mission_id"`
	SelectedRoute   string  `json:"selected_route"`
	RouteConfidence float64 `json:"route_confidence"`
}

// routeRow reads the persisted Scout route decision.
func routeRow(in Input) Row {
	file, err := os.Open(in.memory("route-decisions.jsonl")) //nolint:gosec // G304: runtime memory path
	if err != nil {
		return notEvaluated("route_confidence", routeSource, "nenhuma decisão de rota registrada")
	}
	defer closeQuietly(file)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var line routeLine
		if json.Unmarshal(scanner.Bytes(), &line) == nil && line.MissionID == in.MissionID {
			return evaluated("route_confidence", fmt.Sprintf("%d%%", int(math.Round(line.RouteConfidence*100))), routeSource, "rota "+orDefault(line.SelectedRoute, "não informada"))
		}
	}
	return notEvaluated("route_confidence", routeSource, "nenhuma decisão de rota registrada")
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func closeQuietly(file *os.File) {
	if err := file.Close(); err != nil {
		return
	}
}
