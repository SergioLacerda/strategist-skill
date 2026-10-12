package mission

import (
	"fmt"
	"math"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

const (
	rangerClaimKey     = "ranger-discovery"
	maxRangerEvidences = 8
)

// RecordRangerConfidence makes the artifact's own `confidence_score` the single
// source of the Ranger confidence claim: the claim's percent is that score, so
// the number exists once and the artifact and the ledger cannot disagree. It is
// idempotent. An absent or out-of-range score, or a score with no supporting
// evidence, is an explicit missing record with its reason, never a guess and
// never an unsupported assertion.
func RecordRangerConfidence(strategistRoot, artifactPath, missionID string) error {
	declared, err := handoff.ReadRangerConfidence(artifactPath)
	if err != nil {
		return fmt.Errorf("ranger confidence: %w", err)
	}
	path := telemetry.ConfidenceHistoryPath(strategistRoot)
	producer, err := telemetry.NewConfidenceProducerAdapter(path, telemetry.ConfidenceAgentRanger, missionID)
	if err != nil {
		return fmt.Errorf("ranger confidence: %w", err)
	}
	if recorded, err := rangerClaimRecorded(path, missionID); err != nil || recorded {
		return err
	}
	if reason := unusableScore(declared); reason != "" {
		return recordRangerMissing(producer, reason)
	}
	claim, evidence := rangerClaim(declared, missionID)
	if len(evidence) == 0 {
		return recordRangerMissing(producer, "no_supporting_evidence_for_confidence_score")
	}
	if _, err := producer.RecordClaim(claim, evidence); err != nil {
		return recordRangerMissing(producer, "confidence_claim_rejected: "+err.Error())
	}
	return nil
}

func unusableScore(declared handoff.RangerConfidence) string {
	switch {
	case declared.Score == nil:
		return "confidence_score_absent"
	case *declared.Score < 0 || *declared.Score > 1:
		return "confidence_score_invalid"
	}
	return ""
}

func recordRangerMissing(producer telemetry.ConfidenceProducerAdapter, reason string) error {
	if err := producer.RecordMissing(rangerClaimKey, reason); err != nil {
		return fmt.Errorf("ranger confidence: %w", err)
	}
	return nil
}

func rangerClaimRecorded(path, missionID string) (bool, error) {
	records, err := telemetry.ReadConfidenceRecords(path)
	if err != nil {
		return false, nil //nolint:nilerr // no readable ledger yet means nothing was recorded
	}
	for _, record := range records {
		if record.MissionID == missionID && record.Agent == telemetry.ConfidenceAgentRanger && record.CorrelationKey == rangerClaimKey {
			return true, nil
		}
	}
	return false, nil
}

func rangerClaim(declared handoff.RangerConfidence, missionID string) (domain.ConfidenceClaim, []domain.Evidence) {
	percent := int(math.Round(*declared.Score * 100))
	level, err := domain.ConfidenceLevelForPercent(percent)
	if err != nil {
		level = domain.ConfidenceLow
	}
	kind := domain.ClaimKindAssertion
	if level == domain.ConfidenceLow {
		kind = domain.ClaimKindQuestion
	}
	evidence, classes, ids := rangerEvidence(declared.Sources, level, percent)
	subject := declared.Subtype
	if declared.Verdict != "" {
		subject += " verdict " + declared.Verdict
	}
	return domain.ConfidenceClaim{
		ID: rangerClaimKey, Agent: telemetry.ConfidenceAgentRanger, CorrelationKey: rangerClaimKey, ClaimKind: kind,
		Statement:         fmt.Sprintf("Ranger %s for mission %q, as declared by the artifact confidence_score.", subject, missionID),
		ConfidencePercent: percent, ConfidenceLevel: level, EvidenceIDs: ids, EvidenceClasses: classes,
		CalibrationStatus: domain.CalibrationNoSample,
	}, evidence
}

// rangerEvidence turns the sources the Ranger read into evidence: a fully read
// source is explicit, a partially read one is a corroborated inference.
func rangerEvidence(sources []handoff.RangerSource, level string, percent int) ([]domain.Evidence, []string, []string) {
	var evidence []domain.Evidence
	var classes, ids []string
	for index, source := range sources {
		if len(evidence) == maxRangerEvidences {
			break
		}
		class := domain.EvidenceClassCorroboratedInference
		if source.Coverage == "full" {
			class = domain.EvidenceClassExplicit
		}
		id := fmt.Sprintf("ranger-source-%d", index+1)
		evidence = append(evidence, domain.Evidence{ID: id, SourceRef: filepath.ToSlash(source.Path), Class: class, Confidence: level, ConfidencePercent: &percent})
		classes, ids = append(classes, class), append(ids, id)
	}
	return evidence, classes, ids
}
