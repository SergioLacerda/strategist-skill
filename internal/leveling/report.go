package leveling

import (
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
func ReadRecords(path string) ([]Record, error) {
	var records []Record
	err := scanLedger(path, func(_ string, record Record) {
		records = append(records, record)
	})
	return records, err
}

// Summarize aggregates records by role and level. Roles are sorted by name so
// the output is stable; unknown levels are counted, not listed as a level.
func Summarize(records []Record) Report {
	report := Report{Records: len(records)}
	missions := map[string]bool{}
	byRole := map[string]*RoleSummary{}
	for _, record := range records {
		missions[record.MissionID] = true
		addToSummary(&report, roleSummaryFor(byRole, record.Role), record)
	}
	report.Missions = len(missions)
	for _, summary := range byRole {
		report.Roles = append(report.Roles, *summary)
	}
	sort.Slice(report.Roles, func(i, j int) bool { return report.Roles[i].Role < report.Roles[j].Role })
	return report
}

func roleSummaryFor(byRole map[string]*RoleSummary, role string) *RoleSummary {
	summary := byRole[role]
	if summary == nil {
		summary = &RoleSummary{Role: role, Levels: map[string]int{}, Sources: map[string]int{}}
		byRole[role] = summary
	}
	return summary
}

func addToSummary(report *Report, summary *RoleSummary, record Record) {
	summary.Records++
	if record.Reason == escalatedReason {
		summary.Escalations++
		report.Escalations++
	}
	if record.Unknown() {
		summary.Unknown++
		report.Unknown++
		return
	}
	summary.Levels[record.Label()]++
	if record.Source != "" {
		summary.Sources[record.Source]++
	}
}
