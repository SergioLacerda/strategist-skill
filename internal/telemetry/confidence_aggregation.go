package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// ComputeConfidenceMetrics aggregates records. Empty histories are explicitly
// no_sample; they are never rendered as calibrated zero percent.
func ComputeConfidenceMetrics(records []ConfidenceRecord) ConfidenceMetrics {
	m := newConfidenceMetrics(len(records))
	valid, discarded := distinctConfidenceRecords(records)
	m.MissingRecords = discarded.missing
	m.RejectedRecords = discarded.rejected
	m.DuplicateRecords = discarded.duplicates
	m.SampleSize = len(valid)
	aggregate := summarizeConfidenceRecords(valid)
	m.Distribution = aggregate.distribution
	m.ClaimKinds = aggregate.claimKinds
	m.ByAgent = aggregate.byAgent
	m.AssertionEvidenceCoverage = safeRate(aggregate.supported, aggregate.assertions+discarded.rejectedAssertions)
	m.UnsupportedAssertionRate = safeRate(aggregate.assertions+discarded.rejectedAssertions-aggregate.supported, aggregate.assertions+discarded.rejectedAssertions)
	m.QuestionPreservationRate = safeRate(aggregate.preserved, aggregate.questions)
	m.GroundTruthSampleSize = aggregate.groundTruth
	if aggregate.highClaims > 0 && aggregate.groundTruth > 0 {
		m.CorrectedHighConfidenceClaimRate = safeRate(aggregate.correctedHigh, aggregate.groundTruth)
	}
	m.CalibrationStatus = aggregate.calibration
	m.AgentMetrics = confidenceAgentMetrics(valid, m.ByAgent)
	return m
}

func newConfidenceMetrics(sampleSize int) ConfidenceMetrics {
	return ConfidenceMetrics{
		PolicyVersion: domain.ConfidencePolicyVersion,
		Distribution:  map[string]int{domain.ConfidenceLow: 0, domain.ConfidenceMedium: 0, domain.ConfidenceHigh: 0},
		ClaimKinds:    map[string]int{domain.ClaimKindQuestion: 0, domain.ClaimKindAssertion: 0},
		ByAgent:       map[string]int{},
		AgentMetrics:  map[string]ConfidenceAgentMetrics{},
		SampleSize:    sampleSize,
	}
}

type discardedConfidenceRecords struct {
	missing            int
	rejected           int
	duplicates         int
	rejectedAssertions int
}

func distinctConfidenceRecords(records []ConfidenceRecord) ([]ConfidenceRecord, discardedConfidenceRecords) {
	valid := make([]ConfidenceRecord, 0, len(records))
	discarded := discardedConfidenceRecords{}
	seenEvents := map[string]struct{}{}
	for _, record := range records {
		record = normalizeConfidenceRecord(record)
		switch record.CoverageStatus {
		case ConfidenceCoverageMissing:
			discarded.missing++
			continue
		case ConfidenceCoverageRejected:
			discarded.rejectedAssertions += rejectedAssertionCount(record)
			discarded.rejected++
			continue
		}
		if confidenceRecordDuplicate(record, seenEvents) {
			discarded.duplicates++
			continue
		}
		valid = append(valid, record)
	}
	return valid, discarded
}

func rejectedAssertionCount(record ConfidenceRecord) int {
	if record.ClaimKind == domain.ClaimKindAssertion {
		return 1
	}
	return 0
}

func normalizeConfidenceRecord(record ConfidenceRecord) ConfidenceRecord {
	if record.EventID == "" && record.CoverageStatus != ConfidenceCoverageMissing {
		record.EventID = ConfidenceEventID(record)
	}
	return record
}

func confidenceRecordDuplicate(record ConfidenceRecord, seen map[string]struct{}) bool {
	if record.EventID == "" {
		return false
	}
	if _, ok := seen[record.EventID]; ok {
		return true
	}
	seen[record.EventID] = struct{}{}
	return false
}

type confidenceAggregate struct {
	distribution  map[string]int
	claimKinds    map[string]int
	byAgent       map[string]int
	assertions    int
	supported     int
	questions     int
	preserved     int
	highClaims    int
	correctedHigh int
	groundTruth   int
	calibration   string
}

func summarizeConfidenceRecords(records []ConfidenceRecord) confidenceAggregate {
	aggregate := confidenceAggregate{
		distribution: map[string]int{domain.ConfidenceLow: 0, domain.ConfidenceMedium: 0, domain.ConfidenceHigh: 0},
		claimKinds:   map[string]int{domain.ClaimKindQuestion: 0, domain.ClaimKindAssertion: 0},
		byAgent:      map[string]int{},
		calibration:  domain.CalibrationUncalibrated,
	}
	for _, record := range records {
		summarizeConfidenceRecord(record, &aggregate)
	}
	if len(records) == 0 {
		aggregate.calibration = domain.CalibrationNoSample
	}
	return aggregate
}

func summarizeConfidenceRecord(record ConfidenceRecord, aggregate *confidenceAggregate) {
	aggregate.distribution[record.ConfidenceLevel]++
	aggregate.claimKinds[record.ClaimKind]++
	aggregate.byAgent[record.Agent]++
	summarizeClaimKind(record, aggregate)
	summarizeHighConfidence(record, aggregate)
	summarizeCalibration(record, aggregate)
}

func summarizeClaimKind(record ConfidenceRecord, aggregate *confidenceAggregate) {
	switch record.ClaimKind {
	case domain.ClaimKindQuestion:
		aggregate.questions++
		if record.QuestionPreserved {
			aggregate.preserved++
		}
	case domain.ClaimKindAssertion:
		aggregate.assertions++
		if record.EvidenceProvided {
			aggregate.supported++
		}
	}
}

func summarizeHighConfidence(record ConfidenceRecord, aggregate *confidenceAggregate) {
	if record.ConfidenceLevel == domain.ConfidenceHigh && record.ClaimKind == domain.ClaimKindAssertion {
		aggregate.highClaims++
		if record.GroundTruthRef != "" {
			aggregate.groundTruth += maxInt(record.SampleSize, 1)
			if record.Corrected {
				aggregate.correctedHigh++
			}
		}
	}
}

func summarizeCalibration(record ConfidenceRecord, aggregate *confidenceAggregate) {
	if record.GroundTruthRef != "" {
		aggregate.calibration = domain.CalibrationObserved
	}
	if record.CalibrationStatus == domain.CalibrationCalibrated && record.GroundTruthRef != "" && record.Reviewed && record.SampleSize >= domain.CalibrationMinimumSample {
		aggregate.calibration = domain.CalibrationCalibrated
	}
}
