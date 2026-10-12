package telemetry

import (
	"path/filepath"
	"testing"
)

func TestRecordArchivistAdvisoriesRecordsExplicitMissingBoundaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfidenceHistoryRelPath)
	input := ArchivistAdvisoryInput{
		MissionID: "m-advisory", PackageDigest: "sha256:package", Analysis: []byte("---\nmission_id: m-advisory\n---\nbody\n"),
	}
	recordMissingBoundaries(t, path, input)
	assertAdvisoryReplayIsIdempotent(t, path, input)
}

func recordMissingBoundaries(t *testing.T, path string, input ArchivistAdvisoryInput) {
	t.Helper()
	if err := RecordArchivistAdvisories(path, input); err != nil {
		t.Fatal(err)
	}
	records, err := ReadConfidenceRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records=%+v", records)
	}
	if records[0].Agent != ConfidenceAgentCritic || records[0].CoverageStatus != ConfidenceCoverageMissing || records[0].MissingReason != "no_rubric" {
		t.Fatalf("critic record=%+v", records[0])
	}
	if records[1].Agent != ConfidenceAgentMissionQuality || records[1].CoverageStatus != ConfidenceCoverageMissing || records[1].MissingReason != "no_decisions_or_evidence" {
		t.Fatalf("mission-quality record=%+v", records[1])
	}
}

func assertAdvisoryReplayIsIdempotent(t *testing.T, path string, input ArchivistAdvisoryInput) {
	t.Helper()
	if err := RecordArchivistAdvisories(path, input); err != nil {
		t.Fatal(err)
	}
	records, err := ReadConfidenceRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("replayed records=%+v", records)
	}
}

func TestRecordArchivistAdvisoriesEvaluatesMissionQuality(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfidenceHistoryRelPath)
	analysis := []byte(`---
mission_id: m-quality
decisions:
  - id: DEC-001
    statement: "The package is bounded."
    status: approved
    evidence: [EVD-001]
    confidence: high
evidence:
  - id: EVD-001
    source_ref: docs/design.md
    class: explicit
    confidence: high
acceptance_criteria: ["the package is bounded"]
---
body
`)
	if err := RecordArchivistAdvisories(path, ArchivistAdvisoryInput{MissionID: "m-quality", PackageDigest: "sha256:q", Analysis: analysis}); err != nil {
		t.Fatal(err)
	}
	records, err := ReadConfidenceRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[1].Agent != ConfidenceAgentMissionQuality {
		t.Fatalf("records=%+v", records)
	}
	if records[1].Result != "mission_quality=passed" || records[1].ClaimKind != "assertion" || records[1].ConfidencePercent != 100 {
		t.Fatalf("quality record=%+v", records[1])
	}
}

func TestRecordArchivistAdvisoriesRejectsMalformedQualityInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), ConfidenceHistoryRelPath)
	err := RecordArchivistAdvisories(path, ArchivistAdvisoryInput{
		MissionID: "m-invalid", PackageDigest: "sha256:invalid", Analysis: []byte("---\ndecisions: [bad]\n---\nbody\n"),
	})
	if err == nil {
		t.Fatal("expected malformed decision error")
	}
}
