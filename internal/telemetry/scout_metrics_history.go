package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RouteDecisionHistoryRelPath and OutcomeHistoryRelPath are relative to the
// .strategist runtime root, per
// .analysis/refined/20260803-scout-metrics-followup/analysis.md DEC-1
// (route-decisions.jsonl matches outcomes.jsonl/handoff-metrics.jsonl
// precedent: everything lives under memory/).
const (
	RouteDecisionHistoryRelPath = "memory/route-decisions.jsonl"
	OutcomeHistoryRelPath       = "memory/outcomes.jsonl"
)

// RouteDecisionHistoryPath returns the default runtime memory path for
// Scout route-decision history.
func RouteDecisionHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(RouteDecisionHistoryRelPath))
}

// OutcomeHistoryPath returns the default runtime memory path for mission
// outcome history.
func OutcomeHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(OutcomeHistoryRelPath))
}

// ReadRouteDecisions reads path's JSONL history into []RouteDecision. A
// missing file is not an error — it returns a nil slice, so a workspace
// where Scout has never appended a route decision (see
// analysis.md's finding that AppendRouteDecisionLine is agent-embodied and
// currently unexercised) reports an empty sample rather than failing.
// Malformed lines are skipped rather than treated as an error, same posture
// as ReadHandoffChallenges/ReadRecentSniperMaterializations.
func ReadRouteDecisions(path string) (decisions []RouteDecision, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: route decision history path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open route decision history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close route decision history")

	return scanRouteDecisions(f)
}

func scanRouteDecisions(r io.Reader) ([]RouteDecision, error) {
	var decisions []RouteDecision
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		if d, ok := parseRouteDecisionLine(scanner.Bytes()); ok {
			decisions = append(decisions, d)
		}
	}
	return decisions, jsonlScannerErr(scanner, "scan route decision history")
}

func parseRouteDecisionLine(line []byte) (d RouteDecision, ok bool) {
	if len(line) == 0 {
		return d, false
	}
	if err := json.Unmarshal(line, &d); err != nil {
		return d, false
	}
	return d, true
}

// ReadOutcomes reads path's JSONL history into []OutcomeEntry. A missing
// file is not an error — it returns a nil slice. Malformed lines are
// skipped rather than treated as an error, since outcomes.jsonl is
// append-only historical data that may include entries from schema
// revisions (same tolerance missionIDExists already applies when scanning
// this same file for AppendOutcomeLine's idempotency check).
func ReadOutcomes(path string) (outcomes []OutcomeEntry, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: outcomes history path is owned by runtime memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open outcomes history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close outcomes history")

	return scanOutcomes(f)
}

func scanOutcomes(r io.Reader) ([]OutcomeEntry, error) {
	var outcomes []OutcomeEntry
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		if o, ok := parseOutcomeLine(scanner.Bytes()); ok {
			outcomes = append(outcomes, o)
		}
	}
	return outcomes, jsonlScannerErr(scanner, "scan outcomes history")
}

func parseOutcomeLine(line []byte) (o OutcomeEntry, ok bool) {
	if len(line) == 0 {
		return o, false
	}
	if err := json.Unmarshal(line, &o); err != nil {
		return o, false
	}
	return o, true
}
