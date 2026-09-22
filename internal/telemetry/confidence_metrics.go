package telemetry

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// ConfidenceRecord is the common durable envelope used to compare agent claims.
type ConfidenceRecord struct {
	EventID            string   `json:"event_id,omitempty"`
	MissionID          string   `json:"mission_id,omitempty"`
	Run                string   `json:"run,omitempty"`
	ClaimID            string   `json:"claim_id"`
	Agent              string   `json:"agent"`
	CorrelationKey     string   `json:"correlation_key,omitempty"`
	ClaimKind          string   `json:"claim_kind"`
	ConfidenceLevel    string   `json:"confidence_level"`
	ConfidencePercent  int      `json:"confidence_percent"`
	EvidenceIDs        []string `json:"evidence_ids,omitempty"`
	EvidenceClasses    []string `json:"evidence_classes,omitempty"`
	EvidenceClass      string   `json:"evidence_class,omitempty"`
	EvidenceProvided   bool     `json:"evidence_provided"`
	QuestionPreserved  bool     `json:"question_preserved,omitempty"`
	Corrected          bool     `json:"corrected,omitempty"`
	CoverageStatus     string   `json:"coverage_status,omitempty"`
	MissingReason      string   `json:"missing_reason,omitempty"`
	Violation          string   `json:"violation,omitempty"`
	GroundTruthRef     string   `json:"ground_truth_ref,omitempty"`
	GroundTruthKind    string   `json:"ground_truth_kind,omitempty"`
	GroundTruthOutcome string   `json:"ground_truth_outcome,omitempty"`
	Reviewed           bool     `json:"reviewed,omitempty"`
	CalibrationStatus  string   `json:"calibration_status,omitempty"`
	SampleSize         int      `json:"sample_size,omitempty"`
	Timestamp          string   `json:"timestamp,omitempty"`
}

const (
	// ConfidenceCoverageReported marks a successfully normalized claim.
	ConfidenceCoverageReported = "reported"
	// ConfidenceCoverageMissing marks an expected producer record that was not supplied.
	ConfidenceCoverageMissing = "missing"
	// ConfidenceCoverageRejected marks a claim observed but rejected by validation.
	ConfidenceCoverageRejected = "rejected"
)

// ConfidenceMetrics contains comparable claim-quality measures.
type ConfidenceMetrics struct {
	PolicyVersion                    string                            `json:"policy_version"`
	Distribution                     map[string]int                    `json:"distribution"`
	ClaimKinds                       map[string]int                    `json:"claim_kinds"`
	AssertionEvidenceCoverage        float64                           `json:"assertion_evidence_coverage"`
	UnsupportedAssertionRate         float64                           `json:"unsupported_assertion_rate"`
	QuestionPreservationRate         float64                           `json:"question_preservation_rate"`
	CorrectedHighConfidenceClaimRate float64                           `json:"corrected_high_confidence_claim_rate"`
	SampleSize                       int                               `json:"sample_size"`
	GroundTruthSampleSize            int                               `json:"ground_truth_sample_size"`
	CalibrationStatus                string                            `json:"calibration_status"`
	ByAgent                          map[string]int                    `json:"by_agent"`
	AgentMetrics                     map[string]ConfidenceAgentMetrics `json:"agent_metrics"`
	MissingRecords                   int                               `json:"missing_records"`
	RejectedRecords                  int                               `json:"rejected_records"`
	DuplicateRecords                 int                               `json:"duplicate_records"`
}

// ConfidenceAgentMetrics is the comparable per-agent subset of the aggregate report.
type ConfidenceAgentMetrics struct {
	Distribution                     map[string]int `json:"distribution"`
	ClaimKinds                       map[string]int `json:"claim_kinds"`
	AssertionEvidenceCoverage        float64        `json:"assertion_evidence_coverage"`
	UnsupportedAssertionRate         float64        `json:"unsupported_assertion_rate"`
	QuestionPreservationRate         float64        `json:"question_preservation_rate"`
	CorrectedHighConfidenceClaimRate float64        `json:"corrected_high_confidence_claim_rate"`
	SampleSize                       int            `json:"sample_size"`
	GroundTruthSampleSize            int            `json:"ground_truth_sample_size"`
	CalibrationStatus                string         `json:"calibration_status"`
}

// ConfidenceReadDiagnostics makes discarded history visible to operators.
type ConfidenceReadDiagnostics struct {
	MalformedLines  int      `json:"malformed_lines"`
	InvalidRecords  int      `json:"invalid_records"`
	DuplicateEvents int      `json:"duplicate_events"`
	Reasons         []string `json:"reasons,omitempty"`
}

// ConfidenceHistoryRelPath is the runtime-relative path for confidence history.
const ConfidenceHistoryRelPath = "memory/confidence-records.jsonl"

// Confidence producer names keep agent identity explicit while their records
// share one quality vocabulary.
const (
	ConfidenceAgentScout            = "scout"
	ConfidenceAgentRanger           = "ranger"
	ConfidenceAgentArchivist        = "archivist"
	ConfidenceAgentCritic           = "response_critic"
	ConfidenceAgentMissionQuality   = "mission_quality"
	ConfidenceAgentHandoffChallenge = "handoff_challenge"
	ConfidenceAgentSniper           = "sniper"
)

// ConfidenceHistoryPath returns the confidence history path under a runtime root.
func ConfidenceHistoryPath(strategistRoot string) string {
	return filepath.Join(strategistRoot, filepath.FromSlash(ConfidenceHistoryRelPath))
}

// AppendConfidenceClaim validates and persists a claim at a lifecycle boundary.
func AppendConfidenceClaim(path, agent, missionID, timestamp string, claim domain.ConfidenceClaim, evidence []domain.Evidence) error {
	record, err := NormalizeConfidenceClaim(agent, missionID, timestamp, claim, evidence)
	if err != nil {
		return err
	}
	return AppendConfidenceRecord(path, record)
}

// AppendConfidenceObservation persists accepted claims and rejected assertions.
func AppendConfidenceObservation(path, agent, missionID, timestamp string, claim domain.ConfidenceClaim, evidence []domain.Evidence) (ConfidenceRecord, error) {
	return AppendConfidenceObservationForRun(path, agent, missionID, "", timestamp, claim, evidence)
}

// AppendConfidenceObservationForRun is AppendConfidenceObservation with an
// explicit optional run correlation. Empty run keeps legacy mission-wide
// semantics.
func AppendConfidenceObservationForRun(path, agent, missionID, run, timestamp string, claim domain.ConfidenceClaim, evidence []domain.Evidence) (ConfidenceRecord, error) {
	record, err := NormalizeConfidenceClaim(agent, missionID, timestamp, claim, evidence)
	record.Run = run
	record.EventID = ConfidenceEventID(record)
	if err == nil {
		return record, AppendConfidenceRecord(path, record)
	}
	level, levelErr := domain.ConfidenceLevelForPercent(claim.ConfidencePercent)
	if levelErr != nil {
		level = ""
	}
	rejected := ConfidenceRecord{
		EventID:           ConfidenceEventID(ConfidenceRecord{MissionID: missionID, Run: run, Agent: agent, ClaimID: claim.ID, CorrelationKey: claim.CorrelationKey, ClaimKind: claim.ClaimKind, ConfidencePercent: claim.ConfidencePercent}),
		MissionID:         missionID,
		Run:               run,
		ClaimID:           claim.ID,
		Agent:             agent,
		CorrelationKey:    claim.CorrelationKey,
		ClaimKind:         claim.ClaimKind,
		ConfidenceLevel:   level,
		ConfidencePercent: claim.ConfidencePercent,
		CoverageStatus:    ConfidenceCoverageRejected,
		Violation:         err.Error(),
		Timestamp:         timestamp,
	}
	return rejected, AppendConfidenceRecord(path, rejected)
}

// AppendMissingConfidenceRecord makes absent producer coverage explicit.
func AppendMissingConfidenceRecord(path, agent, missionID, correlationKey, reason, timestamp string) error {
	return AppendMissingConfidenceRecordForRun(path, agent, missionID, "", correlationKey, reason, timestamp)
}

// AppendMissingConfidenceRecordForRun records missing coverage for one
// explicit run without assigning legacy records to a run by inference.
func AppendMissingConfidenceRecordForRun(path, agent, missionID, run, correlationKey, reason, timestamp string) error {
	identity := []string{missionID}
	if run != "" {
		identity = append(identity, run)
	}
	identity = append(identity, agent, correlationKey, reason)
	record := ConfidenceRecord{
		EventID:        fmt.Sprintf("missing-%x", sha256.Sum256([]byte(strings.Join(identity, "\x00")))),
		MissionID:      missionID,
		Run:            run,
		Agent:          agent,
		CorrelationKey: correlationKey,
		CoverageStatus: ConfidenceCoverageMissing,
		MissingReason:  reason,
		Timestamp:      timestamp,
	}
	return AppendConfidenceRecord(path, record)
}
