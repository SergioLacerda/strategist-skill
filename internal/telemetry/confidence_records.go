package telemetry

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// NormalizeConfidenceClaim converts a domain claim into the shared telemetry envelope.
func NormalizeConfidenceClaim(agent, missionID, timestamp string, claim domain.ConfidenceClaim, evidence []domain.Evidence) (ConfidenceRecord, error) {
	if err := domain.ValidateConfidenceClaim(claim, evidence); err != nil {
		return ConfidenceRecord{}, fmt.Errorf("normalize confidence claim: %w", err)
	}
	level, err := domain.ConfidenceLevelForPercent(claim.ConfidencePercent)
	if err != nil {
		return ConfidenceRecord{}, fmt.Errorf("normalize confidence claim: %w", err)
	}
	record := confidenceRecordFromClaim(agent, missionID, timestamp, claim, level)
	record.EvidenceClass = preferredEvidenceClass(claim.EvidenceIDs, evidence)
	record.EvidenceClasses = evidenceClasses(record.EvidenceIDs, evidence, record.EvidenceClasses)
	record.EventID = ConfidenceEventID(record)
	return record, ValidateConfidenceRecord(record)
}

func confidenceRecordFromClaim(agent, missionID, timestamp string, claim domain.ConfidenceClaim, level string) ConfidenceRecord {
	return ConfidenceRecord{
		MissionID:          missionID,
		ClaimID:            claim.ID,
		Agent:              agent,
		CorrelationKey:     claim.CorrelationKey,
		ClaimKind:          claim.ClaimKind,
		ConfidenceLevel:    level,
		ConfidencePercent:  claim.ConfidencePercent,
		EvidenceIDs:        append([]string(nil), claim.EvidenceIDs...),
		EvidenceClasses:    append([]string(nil), claim.EvidenceClasses...),
		EvidenceProvided:   len(claim.EvidenceIDs) > 0,
		QuestionPreserved:  claim.ClaimKind == domain.ClaimKindQuestion,
		GroundTruthRef:     claim.GroundTruthRef,
		GroundTruthKind:    claim.GroundTruthKind,
		GroundTruthOutcome: claim.GroundTruthOutcome,
		CalibrationStatus:  claim.CalibrationStatus,
		SampleSize:         claim.SampleSize,
		Timestamp:          timestamp,
		CoverageStatus:     ConfidenceCoverageReported,
	}
}

func preferredEvidenceClass(ids []string, evidence []domain.Evidence) string {
	class := ""
	for _, item := range evidence {
		if !containsString(ids, item.ID) {
			continue
		}
		if class == "" || item.Class == domain.EvidenceClassExplicit {
			class = item.Class
		}
	}
	return class
}

// ConfidenceEventID is stable for replayed observations and excludes the timestamp.
func ConfidenceEventID(record ConfidenceRecord) string {
	ids := append([]string(nil), record.EvidenceIDs...)
	sort.Strings(ids)
	parts := []string{record.MissionID}
	if record.Run != "" {
		parts = append(parts, record.Run)
	}
	parts = append(parts,
		record.Agent, record.ClaimID, record.CorrelationKey,
		record.ClaimKind, fmt.Sprint(record.ConfidencePercent), strings.Join(ids, ","),
		record.GroundTruthRef, record.GroundTruthKind, record.GroundTruthOutcome,
	)
	seed := strings.Join(parts, "\x00")
	return fmt.Sprintf("ce-%x", sha256.Sum256([]byte(seed)))
}

func evidenceClasses(ids []string, evidence []domain.Evidence, declared []string) []string {
	if len(declared) > 0 {
		return append([]string(nil), declared...)
	}
	byID := make(map[string]string, len(evidence))
	for _, item := range evidence {
		byID[item.ID] = item.Class
	}
	classes := make([]string, 0, len(ids))
	for _, id := range ids {
		classes = append(classes, byID[id])
	}
	return classes
}
