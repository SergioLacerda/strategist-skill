package telemetry

import (
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
	for _, id := range domain.DefaultRoleRegistry().IDs() {
		if !isConfidenceAgent(id) {
			t.Errorf("registered role %q must be a confidence agent", id)
		}
	}
	for _, agent := range []string{ConfidenceAgentCritic, ConfidenceAgentMissionQuality, ConfidenceAgentHandoffChallenge} {
		if !isConfidenceAgent(agent) {
			t.Errorf("non-role producer %q must stay accepted", agent)
		}
	}
	for _, agent := range []string{"gate", "Ranger", "", "transport"} {
		if isConfidenceAgent(agent) {
			t.Errorf("%q must not be a confidence agent", agent)
		}
	}
}
