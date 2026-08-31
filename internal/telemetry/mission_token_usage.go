package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MissionTokenUsageRelPath is the .strategist-runtime-root-relative path for
// explicit, agent-reported token usage records — the only durable record of
// real (non-self-estimated) tokens_in/tokens_out for a mission_id. See
// SetTokens's doc comment (mission_run.go) for why this file, rather than a
// direct provider-response call site, is the only way real numbers reach
// this binary: strategist is a CLI/contract layer invoked by an LLM agent as
// subprocess commands, and never makes an LLM API call itself.
const MissionTokenUsageRelPath = "memory/mission-token-usage.jsonl"

// MissionUsageSourceAgentReport is the Source value AppendMissionTokenUsage
// records for `strategist mission report-usage` — the invoking agent
// explicitly reporting token counts it read from its own provider response
// (e.g. Claude's usage.input_tokens/usage.output_tokens), as opposed to a
// value the agent merely wrote into chat output text with no persisted
// record behind it.
const MissionUsageSourceAgentReport = "agent_report"

// MissionTokenUsageRecord is the JSONL-persisted record for one explicit,
// after-the-fact token usage report against a mission_id. Unlike
// MissionMetrics (an in-process timing/volume snapshot scoped to a single
// CLI invocation's ambient MissionRun), this is a durable, cross-invocation
// record keyed by the caller-supplied mission_id, written once by
// `strategist mission report-usage` after the mission's own CLI
// invocations have already finished.
type MissionTokenUsageRecord struct {
	MissionID  string `json:"mission_id"`
	TokensIn   int64  `json:"tokens_in"`
	TokensOut  int64  `json:"tokens_out"`
	Source     string `json:"source"`
	ReportedAt string `json:"reported_at"`
}

// ValidateMissionTokenUsage checks required fields and non-negative token
// counts. Cobra's int64 flag parsing already rejects non-numeric
// --tokens-in/--tokens-out input at the flag-parsing layer (see
// cmd/strategist/mission_report_usage.go); this only needs to reject
// negative values and missing identifying fields.
func ValidateMissionTokenUsage(rec MissionTokenUsageRecord) error {
	var errs []error
	if rec.MissionID == "" {
		errs = append(errs, errors.New("mission_id is required"))
	}
	if rec.TokensIn < 0 {
		errs = append(errs, fmt.Errorf("tokens_in must be >= 0, got %d", rec.TokensIn))
	}
	if rec.TokensOut < 0 {
		errs = append(errs, fmt.Errorf("tokens_out must be >= 0, got %d", rec.TokensOut))
	}
	if rec.Source == "" {
		errs = append(errs, errors.New("source is required"))
	}
	if rec.ReportedAt == "" {
		errs = append(errs, errors.New("reported_at is required"))
	}
	return errors.Join(errs...)
}

// MissionTokenUsageHistoryPath returns the default runtime memory path for
// mission token usage reports.
func MissionTokenUsageHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(MissionTokenUsageRelPath))
}

// AppendMissionTokenUsage validates rec and appends it as one JSONL record
// to path, creating the parent directory if needed. Unlike
// outcomes.jsonl/route-decisions.jsonl, mission-token-usage.jsonl has
// exactly one write path (`strategist mission report-usage`) and no
// buffer-flush counterpart, so no idempotency-by-mission_id dedup or file
// locking is needed here — the same simple create-then-append posture as
// AppendSniperMaterialization/AppendHandoffChallenge. A mission_id reported
// twice yields two historical records rather than being silently dropped.
func AppendMissionTokenUsage(path string, rec MissionTokenUsageRecord) (err error) {
	if err = ValidateMissionTokenUsage(rec); err != nil {
		return fmt.Errorf("mission token usage validation failed: %w", err)
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
		return fmt.Errorf("create mission token usage history dir: %w", mkdirErr)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // G304: mission token usage path is owned by the Strategist runtime memory domain
	if err != nil {
		return fmt.Errorf("open mission token usage history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close mission token usage history")

	return writeMissionTokenUsageLine(f, rec)
}

func writeMissionTokenUsageLine(f *os.File, rec MissionTokenUsageRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal mission token usage record: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append mission token usage record: %w", err)
	}
	return nil
}

// ReadMissionTokenUsage reads path's JSONL history. A missing file is not
// an error — it returns a nil slice, so a workspace where no usage has ever
// been reported reads as empty rather than failing. Malformed lines are
// skipped rather than treated as an error, consistent with
// ReadHandoffChallenges/ReadRecentSniperMaterializations.
func ReadMissionTokenUsage(path string) (records []MissionTokenUsageRecord, err error) {
	f, err := os.Open(path) //nolint:gosec // G304: mission token usage path is owned by the Strategist runtime memory domain
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open mission token usage history: %w", err)
	}
	defer closeFileWithContext(f, &err, "close mission token usage history")

	return scanMissionTokenUsage(f)
}

func scanMissionTokenUsage(r io.Reader) ([]MissionTokenUsageRecord, error) {
	var records []MissionTokenUsageRecord
	scanner := newJSONLScanner(r)
	for scanner.Scan() {
		if rec, ok := parseMissionTokenUsageLine(scanner.Bytes()); ok {
			records = append(records, rec)
		}
	}
	return records, jsonlScannerErr(scanner, "scan mission token usage history")
}

func parseMissionTokenUsageLine(line []byte) (rec MissionTokenUsageRecord, ok bool) {
	if len(line) == 0 {
		return rec, false
	}
	if err := json.Unmarshal(line, &rec); err != nil {
		return rec, false
	}
	return rec, true
}
