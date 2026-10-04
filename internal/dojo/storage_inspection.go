package dojo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// StateReport is a read-only compatibility and recovery view for existing dojo
// state. It never moves, deletes, rewrites, or repairs any file.
type StateReport struct {
	Paths            StoragePaths
	CriteriaPresent  bool
	EmitLogPresent   bool
	ResultPresent    bool
	ResultValid      bool
	LessonPresent    bool
	HistoryPresent   bool
	HistoryValid     bool
	Orphaned         bool
	RecoveryRequired bool
	Issues           []string
}

// InspectStorage discovers current state and reports corruption or orphaning
// before any future migration or cleanup decision is made.
func InspectStorage(basePath, scenario string) (StateReport, error) {
	paths, err := NewStoragePaths(basePath, scenario)
	if err != nil {
		return StateReport{}, err
	}
	report := StateReport{Paths: paths, ResultValid: true, HistoryValid: true}
	report.CriteriaPresent = fileExists(filepath.Join(paths.ScenarioDir, "criteria.yaml"))
	report.EmitLogPresent = fileExists(paths.EmitLogPath)
	report.LessonPresent = fileExists(paths.LessonPath)
	report.ResultPresent = fileExists(paths.ResultPath)
	report.HistoryPresent = fileExists(paths.HistoryPath)
	inspectResult(&report)
	inspectHistory(&report)
	markOrphaned(&report)
	return report, nil
}

func inspectResult(report *StateReport) {
	if !report.ResultPresent {
		return
	}
	raw, readErr := os.ReadFile(report.Paths.ResultPath)
	var record ResultRecord
	if readErr == nil && json.Unmarshal(raw, &record) == nil {
		return
	}
	report.ResultValid = false
	report.RecoveryRequired = true
	report.Issues = append(report.Issues, "result.json is unreadable or malformed")
}

func inspectHistory(report *StateReport) {
	if !report.HistoryPresent {
		return
	}
	raw, readErr := os.ReadFile(report.Paths.HistoryPath)
	if readErr == nil && validHistory(raw) {
		return
	}
	report.HistoryValid = false
	report.RecoveryRequired = true
	report.Issues = append(report.Issues, ".history.jsonl contains unreadable or malformed entries")
}

func markOrphaned(report *StateReport) {
	statePresent := report.EmitLogPresent || report.ResultPresent || report.LessonPresent
	if !statePresent || report.CriteriaPresent {
		return
	}
	report.Orphaned = true
	report.RecoveryRequired = true
	report.Issues = append(report.Issues, "persisted scenario state has no criteria.yaml owner")
}

func validHistory(raw []byte) bool {
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry RunRecord
		if json.Unmarshal([]byte(line), &entry) != nil {
			return false
		}
	}
	return true
}
