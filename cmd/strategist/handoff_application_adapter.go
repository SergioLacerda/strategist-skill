package main

import (
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

func telemetryChallengeRecord(record application.HandoffChallengeRecord) telemetry.ChallengeRecord {
	return telemetry.ChallengeRecord{
		MissionID:                record.MissionID,
		Transition:               record.Transition,
		Attempt:                  record.Attempt,
		Timestamp:                record.Timestamp,
		Status:                   record.Status,
		Passed:                   record.Passed,
		MissingRefs:              record.MissingRefs,
		MissingChallenges:        record.MissingChallenges,
		MisclassifiedRefs:        record.MisclassifiedRefs,
		GateMismatch:             record.GateMismatch,
		CounterfactualMismatches: record.CounterfactualMismatches,
		ForbiddenClaimViolations: record.ForbiddenClaimViolations,
		CriticalFailures:         record.CriticalFailures,
	}
}
