package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

func confidenceAgentMetrics(records []ConfidenceRecord, agents map[string]int) map[string]ConfidenceAgentMetrics {
	metrics := make(map[string]ConfidenceAgentMetrics, len(agents))
	byAgent := make(map[string][]ConfidenceRecord, len(agents))
	for _, record := range records {
		byAgent[record.Agent] = append(byAgent[record.Agent], record)
	}
	for agent := range agents {
		metrics[agent] = computeAgentMetrics(byAgent[agent])
	}
	return metrics
}

func computeAgentMetrics(records []ConfidenceRecord) ConfidenceAgentMetrics {
	m := ConfidenceAgentMetrics{
		Distribution:      map[string]int{domain.ConfidenceLow: 0, domain.ConfidenceMedium: 0, domain.ConfidenceHigh: 0},
		ClaimKinds:        map[string]int{domain.ClaimKindQuestion: 0, domain.ClaimKindAssertion: 0},
		CalibrationStatus: domain.CalibrationUncalibrated,
		SampleSize:        len(records),
	}
	assertions, supported, questions, preserved := 0, 0, 0, 0
	groundTruth := 0
	corrected := 0
	for _, record := range records {
		m.Distribution[record.ConfidenceLevel]++
		m.ClaimKinds[record.ClaimKind]++
		assertions, supported, questions, preserved, groundTruth, corrected = accumulateAgentRecord(record, assertions, supported, questions, preserved, groundTruth, corrected, &m)
	}
	if len(records) == 0 {
		m.CalibrationStatus = domain.CalibrationNoSample
	}
	m.AssertionEvidenceCoverage = safeRate(supported, assertions)
	m.UnsupportedAssertionRate = safeRate(assertions-supported, assertions)
	m.QuestionPreservationRate = safeRate(preserved, questions)
	m.GroundTruthSampleSize = groundTruth
	m.CorrectedHighConfidenceClaimRate = safeRate(corrected, groundTruth)
	return m
}

func accumulateAgentRecord(record ConfidenceRecord, assertions, supported, questions, preserved, groundTruth, corrected int, metrics *ConfidenceAgentMetrics) (int, int, int, int, int, int) {
	assertions, supported, questions, preserved, groundTruth, corrected = accumulateAgentClaim(record, assertions, supported, questions, preserved, groundTruth, corrected)
	updateAgentCalibration(record, metrics)
	return assertions, supported, questions, preserved, groundTruth, corrected
}

func accumulateAgentClaim(record ConfidenceRecord, assertions, supported, questions, preserved, groundTruth, corrected int) (int, int, int, int, int, int) {
	switch record.ClaimKind {
	case domain.ClaimKindAssertion:
		assertions, supported, groundTruth, corrected = accumulateAgentAssertion(record, assertions, supported, groundTruth, corrected)
	case domain.ClaimKindQuestion:
		questions++
		if record.QuestionPreserved {
			preserved++
		}
	}
	return assertions, supported, questions, preserved, groundTruth, corrected
}

func accumulateAgentAssertion(record ConfidenceRecord, assertions, supported, groundTruth, corrected int) (int, int, int, int) {
	assertions++
	if record.EvidenceProvided || len(record.EvidenceIDs) > 0 {
		supported++
	}
	if record.ConfidenceLevel == domain.ConfidenceHigh && record.GroundTruthRef != "" && record.Reviewed {
		groundTruth += maxInt(record.SampleSize, 1)
		if record.Corrected {
			corrected++
		}
	}
	return assertions, supported, groundTruth, corrected
}

func updateAgentCalibration(record ConfidenceRecord, metrics *ConfidenceAgentMetrics) {
	if record.GroundTruthRef != "" {
		metrics.CalibrationStatus = domain.CalibrationObserved
	}
	if record.CalibrationStatus == domain.CalibrationCalibrated && record.Reviewed && record.SampleSize >= domain.CalibrationMinimumSample {
		metrics.CalibrationStatus = domain.CalibrationCalibrated
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
