package telemetry

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func TestConfidenceProducerAdapterPreservesAgentAndRecordsMissingCoverage(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), ConfidenceHistoryRelPath)
	producer, err := NewConfidenceProducerAdapter(path, ConfidenceAgentRanger, "m-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := producer.RecordMissing("question-1", "producer_unavailable"); err != nil {
		t.Fatal(err)
	}
	records, err := ReadConfidenceRecords(path)
	if err != nil || len(records) != 1 || records[0].Agent != ConfidenceAgentRanger || records[0].CoverageStatus != ConfidenceCoverageMissing {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}

func TestConfidenceProducerAdapterRejectsUnknownAgent(t *testing.T) {
	t.Parallel()
	if _, err := NewConfidenceProducerAdapter("/tmp/confidence.jsonl", "unknown", "m-1"); err == nil {
		t.Fatal("expected unknown agent to be rejected")
	}
}

func TestConfidenceProducerAdapterRecordsSupportedClaim(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), ConfidenceHistoryRelPath)
	producer, err := NewConfidenceProducerAdapter(path, ConfidenceAgentArchivist, "m-2")
	if err != nil {
		t.Fatal(err)
	}
	evidence := []domain.Evidence{{ID: "E-1", SourceRef: "docs/finding.md", Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh}}
	claim := domain.ConfidenceClaim{ID: "A-1", Statement: "supported", ClaimKind: domain.ClaimKindAssertion, ConfidencePercent: 90, CorrelationKey: "q-1", EvidenceIDs: []string{"E-1"}}
	record, err := producer.RecordClaim(claim, evidence)
	if err != nil || record.Agent != ConfidenceAgentArchivist || record.EventID == "" {
		t.Fatalf("record=%+v err=%v", record, err)
	}
}

func TestIsConfidenceAgentFollowsTheRoleRegistry(t *testing.T) {
	expectConfidenceAgents(t, true, domain.DefaultRoleRegistry().IDs()...)
	expectConfidenceAgents(t, true, ConfidenceAgentCritic, ConfidenceAgentMissionQuality, ConfidenceAgentHandoffChallenge)
	expectConfidenceAgents(t, false, "gate", "Ranger", "", "transport")
}

func TestConfidenceEventIDPreservesLegacyIdentityAndSeparatesRuns(t *testing.T) {
	record := ConfidenceRecord{MissionID: "m1", Agent: "ranger", ClaimID: "c1", CorrelationKey: "k", ClaimKind: domain.ClaimKindAssertion, ConfidencePercent: 80, EvidenceIDs: []string{"e2", "e1"}}
	legacySeed := "m1\x00ranger\x00c1\x00k\x00assertion\x0080\x00e1,e2\x00\x00\x00"
	if want, got := fmt.Sprintf("ce-%x", sha256.Sum256([]byte(legacySeed))), ConfidenceEventID(record); want != got {
		t.Fatalf("legacy event id = %q, want %q", got, want)
	}
	runRecord := record
	runRecord.Run = "revision-2"
	if ConfidenceEventID(record) == ConfidenceEventID(runRecord) {
		t.Fatal("run-scoped event id must differ")
	}
}

func expectConfidenceAgents(t *testing.T, want bool, agents ...string) {
	t.Helper()
	for _, agent := range agents {
		if isConfidenceAgent(agent) != want {
			t.Errorf("isConfidenceAgent(%q) = %t, want %t", agent, !want, want)
		}
	}
}
