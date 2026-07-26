package dojo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ResultRecord is the persisted, machine-readable form of one dojo check run,
// written to <base_path>/dojo/.last-run/<scenario>/result.json.
type ResultRecord struct {
	Scenario   string                 `json:"scenario"`
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
	record := ResultRecord{
		Scenario:   result.Scenario,
		Passed:     result.Passed(),
		FailCount:  result.FailCount(),
		Reasons:    ClassifyFailures(result.Items),
		Items:      result.Items,
		StartedAt:  startedAt.UTC().Format(time.RFC3339),
		FinishedAt: finishedAt.UTC().Format(time.RFC3339),
	}
	if err := writeResultJSON(basePath, record); err != nil {
		return err
	}
	return appendHistory(basePath, RunRecord{
		Scenario:   record.Scenario,
		Passed:     record.Passed,
		FailCount:  record.FailCount,
		Reasons:    record.Reasons,
		StartedAt:  record.StartedAt,
		FinishedAt: record.FinishedAt,
	})
}

func writeResultJSON(basePath string, record ResultRecord) error {
	dir := filepath.Join(basePath, "dojo", ".last-run", record.Scenario)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain, not source
		return fmt.Errorf("dojo: create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("dojo: marshal result: %w", err)
	}
	path := filepath.Join(dir, "result.json")
	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // G306: dojo storage domain
		return fmt.Errorf("dojo: write %s: %w", path, err)
	}
	return nil
}

func appendHistory(basePath string, entry RunRecord) error {
	dir := filepath.Join(basePath, "dojo")
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: dojo storage domain, not source
		return fmt.Errorf("dojo: create %s: %w", dir, err)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("dojo: marshal history entry: %w", err)
	}
	path := filepath.Join(dir, ".history.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // G302/G304: dojo storage domain
	if err != nil {
		return fmt.Errorf("dojo: open %s: %w", path, err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		if closeErr := f.Close(); closeErr != nil {
			return fmt.Errorf("dojo: append %s: %w", path, errors.Join(err, closeErr))
		}
		return fmt.Errorf("dojo: append %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("dojo: close %s: %w", path, err)
	}
	return nil
}
