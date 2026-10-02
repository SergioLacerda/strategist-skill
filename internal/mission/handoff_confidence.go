package mission

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// persistHandoffConfidence records the Archivist-to-Sniper handoff confidence
// for one evaluation attempt under the handoff_challenge producer: the declared
// claims and open questions, or an explicit missing-record when none was
// supplied. A failure is returned so the caller persists no authorizing outcome.
func persistHandoffConfidence(strategistRoot, missionID string, attempt int, summary *domain.ConfidenceSummary) error {
	path := telemetry.ConfidenceHistoryPath(strategistRoot)
	producer, err := telemetry.NewConfidenceProducerAdapter(path, telemetry.ConfidenceAgentHandoffChallenge, missionID)
	if err != nil {
		return fmt.Errorf("create confidence producer: %w", err)
	}
	producer = producer.WithRun(fmt.Sprintf("handoff-attempt-%d", attempt))
	if summary == nil {
		return recordMissingHandoffConfidence(producer, "confidence_summary_not_supplied")
	}
	if err := domain.ValidateConfidenceSummary(*summary); err != nil {
		return fmt.Errorf("validate confidence summary: %w", err)
	}
	if summary.MissingRecord {
		return recordMissingHandoffConfidence(producer, "confidence_summary_missing_record")
	}
	return recordHandoffClaims(producer, *summary)
}

func recordMissingHandoffConfidence(producer telemetry.ConfidenceProducerAdapter, reason string) error {
	if err := producer.RecordMissing("archivist_to_sniper", reason); err != nil {
		return fmt.Errorf("record missing confidence summary: %w", err)
	}
	return nil
}

func recordHandoffClaims(producer telemetry.ConfidenceProducerAdapter, summary domain.ConfidenceSummary) error {
	for _, claim := range confidenceSummaryClaims(summary) {
		if _, err := producer.RecordClaim(claim, summary.Evidence); err != nil {
			return fmt.Errorf("record confidence claim %q: %w", claim.ID, err)
		}
	}
	return nil
}

func confidenceSummaryClaims(summary domain.ConfidenceSummary) []domain.ConfidenceClaim {
	claims := make([]domain.ConfidenceClaim, 0, len(summary.Claims)+len(summary.OpenQuestions))
	claims = append(claims, summary.Claims...)
	return append(claims, summary.OpenQuestions...)
}
