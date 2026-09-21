package leveling

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

// escalatedReason is the Reason recorded when a role's level was raised.
const escalatedReason = "escalated"

// RoleSummary aggregates the recorded levels of one role.
type RoleSummary struct {
	Role        string         `json:"role"`
	Records     int            `json:"records"`
	Unknown     int            `json:"unknown"`
	Escalations int            `json:"escalations"`
	Levels      map[string]int `json:"levels"`
	Sources     map[string]int `json:"sources"`
}

// Report is the aggregate view of the role-level ledger.
type Report struct {
	Records     int           `json:"records"`
	Missions    int           `json:"missions"`
	Unknown     int           `json:"unknown"`
	Escalations int           `json:"escalations"`
	Roles       []RoleSummary `json:"roles"`
}

// ReadRecords reads every well-formed record of the ledger in order. A missing
// ledger yields no records; malformed lines are skipped so a damaged history
// never blocks a report or a mission.
func ReadRecords(path string) (records []Record, err error) {
	f, err := os.Open(path) //nolint:gosec // path is resolved below .strategist/memory
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("leveling: open role level ledger: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			records, err = nil, fmt.Errorf("leveling: close role level ledger: %w", closeErr)
		}
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record Record
		if json.Unmarshal(scanner.Bytes(), &record) == nil {
			records = append(records, record)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("leveling: read role level ledger: %w", scanErr)
	}
	return records, nil
}

// Summarize aggregates records by role and level. Roles are sorted by name so
// the output is stable; unknown levels are counted, not listed as a level.
func Summarize(records []Record) Report {
	report := Report{Records: len(records)}
	missions := map[string]bool{}
	byRole := map[string]*RoleSummary{}
	for _, record := range records {
		missions[record.MissionID] = true
		summary := byRole[record.Role]
		if summary == nil {
			summary = &RoleSummary{Role: record.Role, Levels: map[string]int{}, Sources: map[string]int{}}
			byRole[record.Role] = summary
		}
		summary.Records++
		if record.Reason == escalatedReason {
			summary.Escalations++
			report.Escalations++
		}
		if record.Unknown() {
			summary.Unknown++
			report.Unknown++
			continue
		}
		summary.Levels[record.Label()]++
		if record.Source != "" {
			summary.Sources[record.Source]++
		}
	}
	report.Missions = len(missions)
	for _, summary := range byRole {
		report.Roles = append(report.Roles, *summary)
	}
	sort.Slice(report.Roles, func(i, j int) bool { return report.Roles[i].Role < report.Roles[j].Role })
	return report
}
