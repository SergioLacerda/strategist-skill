package handoffconsumer

import (
	"fmt"
	"math"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
)

// RecordConfidence reflects a provider reading in the existing confidence
// ledger, under the handoff producer, so the confidence review, the metrics and
// the gate rows include it with no parallel number.
//
// An answered reading is one claim: an assertion, or a question when the
// provider's confidence is low (a low-confidence claim must be a question). Its
// evidence is the provider's own report, which is explicit; the claim stays
// uncalibrated until it is compared with the deterministic outcome. An enabled
// provider that did not answer leaves an explicit missing record with the reason.
// A disabled integration writes nothing, a shadow reading writes nothing (it must not reach the gate rows), and a reuse writes nothing: the
// original reading already recorded it once.
func RecordConfidence(path string, request Request, report Report, run string) error {
	if report.Status == StatusDisabled || report.Status == StatusShadow || report.Reused {
		return nil
	}
	producer, err := telemetry.NewConfidenceProducerAdapter(path, telemetry.ConfidenceAgentHandoffChallenge, request.MissionID)
	if err != nil {
		return fmt.Errorf("provider confidence: %w", err)
	}
	producer = producer.WithRun(run)
	key := "provider-conformance-" + request.Transition
	if report.Status == StatusFallback {
		return recordMissing(producer, key, report)
	}
	claim, evidence := providerClaim(request, report, key)
	if _, err := producer.RecordClaim(claim, evidence); err != nil {
		return fmt.Errorf("provider confidence: %w", err)
	}
	return nil
}

func recordMissing(producer telemetry.ConfidenceProducerAdapter, key string, report Report) error {
	reason := string(report.Reason)
	if reason == "" {
		reason = "provider_not_answered"
	}
	if err := producer.RecordMissing(key, reason); err != nil {
		return fmt.Errorf("provider confidence: %w", err)
	}
	return nil
}

func providerClaim(request Request, report Report, key string) (domain.ConfidenceClaim, []domain.Evidence) {
	percent := int(math.Round(report.Confidence * 100))
	level, err := domain.ConfidenceLevelForPercent(percent)
	if err != nil {
		level = domain.ConfidenceLow
	}
	kind := domain.ClaimKindAssertion
	if level == domain.ConfidenceLow {
		kind = domain.ClaimKindQuestion
	}
	evidenceID := "provider-report-" + request.Transition
	evidence := domain.Evidence{
		ID: evidenceID, SourceRef: "memory/integration-calls.jsonl#" + request.Subject, Class: domain.EvidenceClassExplicit,
		Confidence: level, ConfidencePercent: &percent,
	}
	claim := domain.ConfidenceClaim{
		ID:                key,
		Statement:         fmt.Sprintf("Provider-reported confidence that the %s handoff conforms to its contract criteria%s.", request.Transition, calibrationSuffix(report.Calibration)),
		Agent:             telemetry.ConfidenceAgentHandoffChallenge,
		CorrelationKey:    key,
		ClaimKind:         kind,
		ConfidencePercent: percent,
		ConfidenceLevel:   level,
		EvidenceIDs:       []string{evidenceID},
		EvidenceClasses:   []string{domain.EvidenceClassExplicit},
		CalibrationStatus: domain.CalibrationNoSample,
	}
	return withCalibration(claim, []domain.Evidence{evidence}, report.Calibration, level)
}

// withCalibration moves the claim beyond `no_sample` only with evidence: a valid
// pinned calibration is cited as a second explicit source and its sample size
// becomes the claim's. Without one the claim stays uncalibrated.
func withCalibration(claim domain.ConfidenceClaim, evidence []domain.Evidence, use CalibrationUse, level string) (domain.ConfidenceClaim, []domain.Evidence) {
	if !use.Applied {
		return claim, evidence
	}
	id := "provider-calibration-" + use.ID
	percent := claim.ConfidencePercent
	evidence = append(evidence, domain.Evidence{
		ID: id, SourceRef: "dojo/calibrations/" + use.ID + ".json", Class: domain.EvidenceClassExplicit,
		Confidence: level, ConfidencePercent: &percent,
	})
	claim.EvidenceIDs = append(claim.EvidenceIDs, id)
	claim.EvidenceClasses = append(claim.EvidenceClasses, domain.EvidenceClassExplicit)
	claim.CalibrationStatus, claim.SampleSize = domain.CalibrationObserved, use.Samples
	return claim, evidence
}

func calibrationSuffix(use CalibrationUse) string {
	if use.Applied {
		return " (calibration " + use.ID + ")"
	}
	return " (uncalibrated)"
}
