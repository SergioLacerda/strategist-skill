package dojo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ResultRecord is the persisted, machine-readable form of one dojo check run,
// written to <base_path>/dojo/.last-run/<scenario>/result.json.
type ResultRecord struct {
	Scenario   string                 `json:"scenario"`
	ScenarioID string                 `json:"scenario_id,omitempty"`
	Passed     bool                   `json:"passed"`
	FailCount  int                    `json:"fail_count"`
	Reasons    []FailureReason        `json:"reasons,omitempty"`
	Items      []domain.DojoCheckItem `json:"items"`
	StartedAt  string                 `json:"started_at"`
	FinishedAt string                 `json:"finished_at"`
}

// RunRecord is the compact form appended, one line per run, to
// <base_path>/dojo/.history.jsonl for trend mining across runs.
type RunRecord struct {
	Scenario   string          `json:"scenario"`
	ScenarioID string          `json:"scenario_id,omitempty"`
	Passed     bool            `json:"passed"`
	FailCount  int             `json:"fail_count"`
	Reasons    []FailureReason `json:"reasons,omitempty"`
	StartedAt  string          `json:"started_at"`
	FinishedAt string          `json:"finished_at"`
}

// PersistResult writes result.json under <base_path>/dojo/.last-run/<scenario>/ and
// appends a compact record to <base_path>/dojo/.history.jsonl. Both writes stay inside
// the dojo storage domain — they never touch source, jewels, or governance files.
func PersistResult(basePath string, result domain.DojoCheckResult, startedAt, finishedAt time.Time) error {
	paths, err := NewStoragePaths(basePath, result.Scenario)
	if err != nil {
		return err
	}
	record := ResultRecord{
		Scenario:   result.Scenario,
		ScenarioID: paths.ScenarioID,
		Passed:     result.Passed(),
		FailCount:  result.FailCount(),
		Reasons:    ClassifyFailures(result.Items),
		Items:      result.Items,
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
		FinishedAt: finishedAt.UTC().Format(time.RFC3339),
	}
	entry := RunRecord{
		Scenario:   record.Scenario,
		ScenarioID: record.ScenarioID,
		Passed:     record.Passed,
		FailCount:  record.FailCount,
		Reasons:    record.Reasons,
		StartedAt:  record.StartedAt,
		FinishedAt: record.FinishedAt,
	}
	return withStorageLock(paths, func() error {
		if err := writeResultJSON(paths, record); err != nil {
			return err
		}
		return appendHistory(paths, entry)
	})
}

func writeResultJSON(paths StoragePaths, record ResultRecord) error {
	if err := os.MkdirAll(paths.LastRunDir, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain, not source
		return fmt.Errorf("dojo: create %s: %w", paths.LastRunDir, err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("dojo: marshal result: %w", err)
	}
	if err := atomicWriteFile(paths.ResultPath, data, 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
		return fmt.Errorf("dojo: write %s: %w", paths.ResultPath, err)
	}
	return nil
}

func appendHistory(paths StoragePaths, entry RunRecord) error {
	if err := os.MkdirAll(paths.DojoRoot, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain, not source
		return fmt.Errorf("dojo: create %s: %w", paths.DojoRoot, err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("dojo: marshal history entry: %w", err)
	}
	f, err := os.OpenFile(paths.HistoryPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // G302/G304: dojo storage domain
	if err != nil {
		return fmt.Errorf("dojo: open %s: %w", paths.HistoryPath, err)
	}
	writeErr := writeHistoryRecord(f, append(data, '\n'))
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("dojo: append %s: %w", paths.HistoryPath, err)
	}
	return nil
}

func writeHistoryRecord(f *os.File, data []byte) error {
	if n, err := f.Write(data); err != nil {
		return fmt.Errorf("write history record: %w", err)
	} else if n != len(data) {
		return errors.New("short history write")
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync history record: %w", err)
	}
	return nil
}
