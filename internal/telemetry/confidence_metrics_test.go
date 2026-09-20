package telemetry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func confidenceRecordFixture(id, agent, kind, level string, percent int) ConfidenceRecord {
	return ConfidenceRecord{
		ClaimID: id, Agent: agent, ClaimKind: kind, ConfidenceLevel: level,
		ConfidencePercent: percent, EvidenceProvided: kind == domain.ClaimKindAssertion,
		QuestionPreserved: kind == domain.ClaimKindQuestion,
	}
}

func TestComputeConfidenceMetricsEmptyHistoryIsNoSample(t *testing.T) {
	t.Parallel()
	m := ComputeConfidenceMetrics(nil)
	if m.SampleSize != 0 || m.CalibrationStatus != domain.CalibrationNoSample {
		t.Fatalf("empty metrics = %+v, want sample=0 and no_sample", m)
	}
	if m.CorrectedHighConfidenceClaimRate != 0 {
		t.Fatalf("empty corrected rate = %v, want 0", m.CorrectedHighConfidenceClaimRate)
	}
}

func TestComputeConfidenceMetricsSeparatesKindsEvidenceAndGroundTruth(t *testing.T) {
	t.Parallel()
	high := confidenceRecordFixture("a-1", "archivist", domain.ClaimKindAssertion, domain.ConfidenceHigh, 90)
	high.GroundTruthRef = "review-1"
	high.GroundTruthKind = domain.GroundTruthUserRevision
	question := confidenceRecordFixture("q-1", "ranger", domain.ClaimKindQuestion, domain.ConfidenceLow, 30)
	records := []ConfidenceRecord{high, question,
		confidenceRecordFixture("a-2", "scout", domain.ClaimKindAssertion, domain.ConfidenceMedium, 70),
	}
	m := ComputeConfidenceMetrics(records)
	if m.SampleSize != 3 || m.ClaimKinds[domain.ClaimKindQuestion] != 1 || m.ClaimKinds[domain.ClaimKindAssertion] != 2 {
		t.Fatalf("unexpected counts: %+v", m)
	}
	if m.AssertionEvidenceCoverage != 1 || m.UnsupportedAssertionRate != 0 {
		t.Fatalf("evidence metrics = %v/%v, want 1/0", m.AssertionEvidenceCoverage, m.UnsupportedAssertionRate)
	}
	if m.QuestionPreservationRate != 1 || m.GroundTruthSampleSize != 1 || m.CalibrationStatus != domain.CalibrationObserved {
		t.Fatalf("calibration/question metrics = %+v", m)
	}
}

func TestValidateConfidenceRecordRequiresGroundTruthForCorrections(t *testing.T) {
	t.Parallel()
	record := confidenceRecordFixture("a-1", "critic", domain.ClaimKindAssertion, domain.ConfidenceHigh, 95)
	record.Corrected = true
	if err := ValidateConfidenceRecord(record); err == nil {
		t.Fatal("expected corrected record without ground truth to be rejected")
	}
}

func TestNormalizeConfidenceClaimPreservesProducerAndQuestionKind(t *testing.T) {
	t.Parallel()
	claim := domain.ConfidenceClaim{
		ID: "q-1", Statement: "Is the handoff complete?", ClaimKind: domain.ClaimKindQuestion,
		ConfidencePercent: 40,
	}
	record, err := NormalizeConfidenceClaim(ConfidenceAgentRanger, "m-1", "2026-09-20T00:00:00Z", claim, nil)
	if err != nil {
		t.Fatalf("normalize question: %v", err)
	}
	if record.Agent != ConfidenceAgentRanger || record.ClaimKind != domain.ClaimKindQuestion || !record.QuestionPreserved {
		t.Fatalf("normalized record lost provenance or question kind: %+v", record)
	}
}

func TestConfidenceHistoryRoundTripAndMissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ConfidenceHistoryRelPath)
	record := confidenceRecordFixture("q-1", "ranger", domain.ClaimKindQuestion, domain.ConfidenceLow, 10)
	if err := AppendConfidenceRecord(path, record); err != nil {
		t.Fatalf("append: %v", err)
	}
	got, err := ReadConfidenceRecords(path)
	if err != nil || len(got) != 1 || got[0].ClaimID != record.ClaimID {
		t.Fatalf("round trip got=%+v err=%v", got, err)
	}
	missing, err := ReadConfidenceRecords(filepath.Join(dir, "missing.jsonl"))
	if err != nil || len(missing) != 0 {
		t.Fatalf("missing history got=%+v err=%v", missing, err)
	}
}

func TestConfidenceHistoryDeduplicatesStableEventID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ConfidenceHistoryRelPath)
	record := confidenceRecordFixture("a-1", "ranger", domain.ClaimKindQuestion, domain.ConfidenceLow, 10)
	first, err := AppendConfidenceRecordOnce(path, record)
	if err != nil || !first {
		t.Fatalf("first append = %v, %v", first, err)
	}
	second, err := AppendConfidenceRecordOnce(path, record)
	if err != nil || second {
		t.Fatalf("duplicate append = %v, %v", second, err)
	}
	got, diagnostics, err := ReadConfidenceRecordsWithDiagnostics(path)
	if err != nil || len(got) != 1 || diagnostics.DuplicateEvents != 0 {
		t.Fatalf("deduplicated history got=%+v diagnostics=%+v err=%v", got, diagnostics, err)
	}
}

func TestConfidenceHistoryConcurrentAppendsRetainAllEvents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ConfidenceHistoryRelPath)
	const count = 24
	var wg sync.WaitGroup
	errCh := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			record := confidenceRecordFixture("q-"+string(rune('a'+i)), "ranger", domain.ClaimKindQuestion, domain.ConfidenceLow, 10)
			errCh <- AppendConfidenceRecord(path, record)
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent append: %v", err)
		}
	}
	got, err := ReadConfidenceRecords(path)
	if err != nil || len(got) != count {
		t.Fatalf("concurrent history len=%d err=%v, want %d", len(got), err, count)
	}
}

func TestReadConfidenceRecordsWithDiagnosticsReportsMalformedAndInvalidLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ConfidenceHistoryRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data := "not-json\n" +
		`{"claim_id":"bad","agent":"ranger","claim_kind":"assertion","confidence_level":"high","confidence_percent":90}` + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	got, diagnostics, err := ReadConfidenceRecordsWithDiagnostics(path)
	if err != nil || len(got) != 0 || diagnostics.MalformedLines != 1 || diagnostics.InvalidRecords != 1 {
		t.Fatalf("records=%+v diagnostics=%+v err=%v", got, diagnostics, err)
	}
}

func TestComputeConfidenceMetricsProvidesComparableAgentMetrics(t *testing.T) {
	t.Parallel()
	first := confidenceRecordFixture("q-1", "ranger", domain.ClaimKindQuestion, domain.ConfidenceLow, 20)
	second := confidenceRecordFixture("q-2", "archivist", domain.ClaimKindQuestion, domain.ConfidenceLow, 20)
	m := ComputeConfidenceMetrics([]ConfidenceRecord{first, second})
	if len(m.AgentMetrics) != 2 || m.AgentMetrics["ranger"].SampleSize != 1 || m.AgentMetrics["archivist"].SampleSize != 1 {
		t.Fatalf("unexpected per-agent metrics: %+v", m.AgentMetrics)
	}
}

func TestComputeConfidenceMetricsDoesNotSelfDeclareCalibration(t *testing.T) {
	t.Parallel()
	record := confidenceRecordFixture("a-1", "critic", domain.ClaimKindAssertion, domain.ConfidenceHigh, 90)
	record.GroundTruthRef = "review-1"
	record.GroundTruthKind = domain.GroundTruthUserRevision
	record.CalibrationStatus = domain.CalibrationCalibrated
	record.Reviewed = false
	record.SampleSize = domain.CalibrationMinimumSample
	m := ComputeConfidenceMetrics([]ConfidenceRecord{record})
	if m.CalibrationStatus == domain.CalibrationCalibrated {
		t.Fatal("self-declared calibration must not become calibrated")
	}
}

func TestComputeConfidenceMetricsAcceptsReviewedCalibrationMinimum(t *testing.T) {
	t.Parallel()
	record := confidenceRecordFixture("a-1", "critic", domain.ClaimKindAssertion, domain.ConfidenceHigh, 90)
	record.GroundTruthRef = "review-1"
	record.GroundTruthKind = domain.GroundTruthUserRevision
	record.GroundTruthOutcome = domain.GroundTruthCorrect
	record.CalibrationStatus = domain.CalibrationCalibrated
	record.Reviewed = true
	record.SampleSize = domain.CalibrationMinimumSample
	if err := ValidateConfidenceRecord(record); err != nil {
		t.Fatalf("valid calibrated record rejected: %v", err)
	}
	m := ComputeConfidenceMetrics([]ConfidenceRecord{record})
	if m.CalibrationStatus != domain.CalibrationCalibrated {
		t.Fatalf("expected calibrated metrics, got %+v", m)
	}
}

func TestAppendConfidenceObservationKeepsRejectedAssertionVisible(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ConfidenceHistoryRelPath)
	claim := domain.ConfidenceClaim{ID: "bad-1", Statement: "unsupported", ClaimKind: domain.ClaimKindAssertion, ConfidencePercent: 70}
	record, err := AppendConfidenceObservation(path, "ranger", "m-1", "2026-09-20T00:00:00Z", claim, nil)
	if err != nil {
		t.Fatalf("rejected observation should persist: %v", err)
	}
	if record.CoverageStatus != ConfidenceCoverageRejected || record.Violation == "" {
		t.Fatalf("unexpected rejected record: %+v", record)
	}
	got, diagnostics, err := ReadConfidenceRecordsWithDiagnostics(path)
	if err != nil || len(got) != 1 || diagnostics.InvalidRecords != 0 {
		t.Fatalf("rejected record got=%+v diagnostics=%+v err=%v", got, diagnostics, err)
	}
	metrics := ComputeConfidenceMetrics(got)
	if metrics.UnsupportedAssertionRate != 1 || metrics.RejectedRecords != 1 {
		t.Fatalf("rejected assertion metrics = %+v", metrics)
	}
}
