package application

import (
	"errors"
	"fmt"
	"time"
)

// MissionUsageSourceAgentReport identifies usage reported by the invoking agent.
const MissionUsageSourceAgentReport = "agent_report"

// MissionTokenUsageInput is the application-level report supplied by an
// invoking agent after reading usage from its own provider response.
type MissionTokenUsageInput struct {
	MissionID string
	TokensIn  int64
	TokensOut int64
}

// MissionTokenUsageRecord is the adapter-neutral durable usage record.
type MissionTokenUsageRecord struct {
	MissionID  string
	TokensIn   int64
	TokensOut  int64
	Source     string
	ReportedAt string
}

// MissionTokenUsagePorts provides mission lookup and usage persistence.
type MissionTokenUsagePorts struct {
	MissionKnown func(basePath, missionID string) bool
	Append       func(root string, record MissionTokenUsageRecord) error
}

// RecordMissionTokenUsage validates the mission reference, constructs the
// durable record, and delegates persistence through an explicit port.
func RecordMissionTokenUsage(root, basePath string, input MissionTokenUsageInput, now time.Time, ports MissionTokenUsagePorts) (MissionTokenUsageRecord, error) {
	if input.MissionID == "" {
		return MissionTokenUsageRecord{}, errors.New("mission report-usage: mission id is required")
	}
	if input.TokensIn < 0 || input.TokensOut < 0 {
		return MissionTokenUsageRecord{}, fmt.Errorf("mission report-usage: token counts must be non-negative")
	}
	if ports.MissionKnown == nil || ports.Append == nil {
		return MissionTokenUsageRecord{}, errors.New("mission report-usage: usage ports are required")
	}
	if !ports.MissionKnown(basePath, input.MissionID) {
		return MissionTokenUsageRecord{}, fmt.Errorf("mission report-usage: unknown mission_id %q (no pending/refined/archived artifact found under %s)", input.MissionID, basePath)
	}
	record := MissionTokenUsageRecord{
		MissionID: input.MissionID, TokensIn: input.TokensIn, TokensOut: input.TokensOut,
		Source: MissionUsageSourceAgentReport, ReportedAt: now.UTC().Format(time.RFC3339),
	}
	if err := ports.Append(root, record); err != nil {
		return MissionTokenUsageRecord{}, err
	}
	return record, nil
}
