package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func readRefinementHandoffMetrics(root, missionID string) ([]telemetry.RefinementHandoffLine, error) {
	all, err := telemetry.ReadRefinementHandoffLines(telemetry.HandoffMetricsPath(root))
	if err != nil {
		return nil, fmt.Errorf("read handoff metrics: %w", err)
	}
	applicationLines := make([]application.HandoffMetricsLine, 0, len(all))
	for _, line := range all {
		applicationLines = append(applicationLines, application.HandoffMetricsLine{
			MissionID: line.MissionID, DiscoveryTokens: line.DiscoveryTokens, BriefTokens: line.BriefTokens,
			BriefCompressionRatio: line.BriefCompressionRatio, RefinementReopens: line.RefinementReopens,
			Revision: line.Revision, EvidenceCoverageRatio: line.EvidenceCoverageRatio,
			Model: dereferenceString(line.Model), Effort: dereferenceString(line.Effort), LevelSource: dereferenceString(line.LevelSource),
		})
	}
	filtered := application.FilterHandoffMetrics(applicationLines, missionID)
	result := make([]telemetry.RefinementHandoffLine, 0, len(filtered))
	for _, line := range filtered {
		result = append(result, telemetry.RefinementHandoffLine{
			MissionID: line.MissionID, DiscoveryTokens: line.DiscoveryTokens, BriefTokens: line.BriefTokens,
			BriefCompressionRatio: line.BriefCompressionRatio, RefinementReopens: line.RefinementReopens,
			Revision: line.Revision, EvidenceCoverageRatio: line.EvidenceCoverageRatio,
			Model: optionalString(line.Model), Effort: optionalString(line.Effort), LevelSource: optionalString(line.LevelSource),
		})
	}
	return result, nil
}

func readMissionTokenUsage(root, missionID string) ([]telemetry.MissionTokenUsageRecord, error) {
	all, err := telemetry.ReadMissionTokenUsage(telemetry.MissionTokenUsageHistoryPath(root))
	if err != nil {
		return nil, fmt.Errorf("read mission token usage: %w", err)
	}
	applicationRecords := make([]application.MissionTokenUsageRecord, 0, len(all))
	for _, rec := range all {
		applicationRecords = append(applicationRecords, application.MissionTokenUsageRecord{
			MissionID: rec.MissionID, TokensIn: rec.TokensIn, TokensOut: rec.TokensOut,
			Source: rec.Source, ReportedAt: rec.ReportedAt,
		})
	}
	filtered := application.FilterMissionTokenUsage(applicationRecords, missionID)
	result := make([]telemetry.MissionTokenUsageRecord, 0, len(filtered))
	for _, rec := range filtered {
		result = append(result, telemetry.MissionTokenUsageRecord{
			MissionID: rec.MissionID, TokensIn: rec.TokensIn, TokensOut: rec.TokensOut,
			Source: rec.Source, ReportedAt: rec.ReportedAt,
		})
	}
	return result, nil
}

func dereferenceString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
