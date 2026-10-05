package handoffconsumer

import (
	"sort"
)

// Shadow reads the intact readings a shadow provider recorded, oldest first. A reuse
// marker is not a reading.
func (l Ledger) Shadow() ([]Record, error) {
	var readings []Record
	err := l.scanIntact(func(record Record) {
		if record.Shadow && !record.Reused {
			readings = append(readings, record)
		}
	})
	return readings, err
}

// ShadowRow summarizes the shadow readings of one transition. It carries counts and
// numbers only, never artifact content.
type ShadowRow struct {
	Transition    string
	Readings      int
	Answered      int
	WouldApprove  int
	Calibrated    int
	Missions      int
	MeanConfident float64
	MinConfident  float64
	MaxConfident  float64
	// Hints counts how often each criterion was the one the provider found unmet.
	Hints map[string]int
}

type shadowTally struct {
	row      ShadowRow
	missions map[string]bool
	sum      float64
}

func (t *shadowTally) add(record Record) {
	t.row.Readings++
	t.missions[record.MissionID] = true
	if !record.Answered {
		return
	}
	t.row.Answered++
	t.sum += record.Confidence
	if t.row.Answered == 1 || record.Confidence < t.row.MinConfident {
		t.row.MinConfident = record.Confidence
	}
	t.row.MaxConfident = max(t.row.MaxConfident, record.Confidence)
	t.count(record)
}

func (t *shadowTally) count(record Record) {
	if record.Approved {
		t.row.WouldApprove++
	}
	if record.Calibrated {
		t.row.Calibrated++
	}
	for _, hint := range record.Hints {
		t.row.Hints[hint]++
	}
}

// SummarizeShadow groups shadow readings by transition, in transition order.
func SummarizeShadow(records []Record) []ShadowRow {
	tallies := map[string]*shadowTally{}
	for _, record := range records {
		tally := tallies[record.Transition]
		if tally == nil {
			tally = &shadowTally{row: ShadowRow{Transition: record.Transition, Hints: map[string]int{}}, missions: map[string]bool{}}
			tallies[record.Transition] = tally
		}
		tally.add(record)
	}
	rows := make([]ShadowRow, 0, len(tallies))
	for _, tally := range tallies {
		tally.row.Missions = len(tally.missions)
		if tally.row.Answered > 0 {
			tally.row.MeanConfident = tally.sum / float64(tally.row.Answered)
		}
		rows = append(rows, tally.row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Transition < rows[j].Transition })
	return rows
}

// shadowed turns an answered reading of a shadow provider into a shadow report: the
// decision it would have taken is kept for the report, and the delegation and the
// hints are dropped so that nothing downstream uses them.
func (c *Consumer) shadowed(report Report) Report {
	if !c.Provider.Shadow || (report.Status != StatusApproved && report.Status != StatusSignal) {
		return report
	}
	report.WouldApprove = report.Status == StatusApproved
	report.Status, report.Delegation, report.Hints = StatusShadow, nil, nil
	return report
}
