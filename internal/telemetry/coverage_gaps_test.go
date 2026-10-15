package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfidenceProducerAdapter_Gaps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	historyPath := filepath.Join(dir, ConfidenceHistoryRelPath)

	// Unsupported agent
	_, err := NewConfidenceProducerAdapter(historyPath, "unsupported_agent", "m-1")
	require.ErrorContains(t, err, "unsupported agent")

	// Empty mission_id
	_, err = NewConfidenceProducerAdapter(historyPath, "ranger", "")
	require.ErrorContains(t, err, "mission_id is required")

	// Valid adapter + WithRun
	adapter, err := NewConfidenceProducerAdapter(historyPath, "ranger", "m-1")
	require.NoError(t, err)

	scoped := adapter.WithRun("run-42")
	assert.Equal(t, "run-42", scoped.Run)

	// Record missing
	err = scoped.RecordMissing("key-1", "no_claim")
	require.NoError(t, err)

	// Record claim
	claim := domain.ConfidenceClaim{
		ID:                "C-1",
		Statement:         "test claim",
		ClaimKind:         domain.ClaimKindAssertion,
		ConfidencePercent: 95,
		CorrelationKey:    "key-1",
		EvidenceIDs:       []string{"E-1"},
	}
	evidence := []domain.Evidence{
		{
			ID:         "E-1",
			SourceRef:  "docs/test.md",
			Class:      domain.EvidenceClassExplicit,
			Confidence: domain.ConfidenceHigh,
		},
	}
	rec, err := scoped.RecordClaim(claim, evidence)
	require.NoError(t, err)
	assert.Equal(t, "ranger", rec.Agent)
	assert.Equal(t, "run-42", rec.Run)
}

func TestDiscovery_Gaps(t *testing.T) {
	t.Parallel()

	event := NewDiscoveryWeaponEvent("run-1", "prov-1", "/path/to/art", DiscoveryInvocationFailed, DiscoveryNormalizationRejected, "ev", "reas", "auth", "pin", "iso")
	assert.Equal(t, SeverityError, event.SeverityNumber)

	// WithDiscoveryAdapterProvenance empty vs populated
	unchanged := WithDiscoveryAdapterProvenance(event, "", "")
	assert.Equal(t, event, unchanged)

	withAdapter := WithDiscoveryAdapterProvenance(event, "adapter-1", "child-policy-1")
	assert.Equal(t, "adapter-1", withAdapter.Attributes[AttrExecutionAdapter])
	assert.Equal(t, "child-policy-1", withAdapter.Attributes[AttrChildPolicyID])

	// WithDiscoveryRequestCorrelation
	withReq := WithDiscoveryRequestCorrelation(event, "req-123")
	assert.Equal(t, "req-123", withReq.Attributes[AttrInvocationRequestID])
}

func TestInitiativeEventHistoryPath(t *testing.T) {
	t.Parallel()

	path := InitiativeEventHistoryPath("/tmp/strategist")
	assert.Equal(t, filepath.Join("/tmp/strategist", "memory", "initiative-events.jsonl"), path)
}

func TestRefinementHandoffLine_ValidationAndRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := HandoffMetricsPath(dir)
	assert.Equal(t, filepath.Join(dir, "memory", "handoff-metrics.jsonl"), path)

	// Validation errors
	line := RefinementHandoffLine{}
	require.ErrorContains(t, line.Validate(), "mission_id is required")

	neg := int64(-1)
	line.MissionID = "m-1"
	line.DiscoveryTokens = &neg
	require.ErrorContains(t, line.Validate(), "discovery_tokens must be >= 0")

	line.DiscoveryTokens = nil
	line.BriefTokens = &neg
	require.ErrorContains(t, line.Validate(), "brief_tokens must be >= 0")

	negInt := -1
	line.BriefTokens = nil
	line.RefinementReopens = &negInt
	require.ErrorContains(t, line.Validate(), "refinement_reopens must be >= 0")

	line.RefinementReopens = nil
	line.Revision = &negInt
	require.ErrorContains(t, line.Validate(), "revision must be >= 1")

	negFloat := -0.5
	line.Revision = nil
	line.BriefCompressionRatio = &negFloat
	require.ErrorContains(t, line.Validate(), "brief_compression_ratio must be >= 0")

	line.BriefCompressionRatio = nil
	line.EvidenceCoverageRatio = &negFloat
	require.ErrorContains(t, line.Validate(), "evidence_coverage_ratio must be >= 0")

	// Append & Read lines
	lineValid := RefinementHandoffLine{
		MissionID: "m-1",
	}
	appended, err := AppendRefinementHandoffLine(path, lineValid)
	require.NoError(t, err)
	assert.True(t, appended)

	// Append duplicate base line
	appended, err = AppendRefinementHandoffLine(path, lineValid)
	require.NoError(t, err)
	assert.False(t, appended)

	// Read lines
	lines, err := ReadRefinementHandoffLines(path)
	require.NoError(t, err)
	require.Len(t, lines, 1)
	assert.Equal(t, "m-1", lines[0].MissionID)

	// Read non-existent file
	none, err := ReadRefinementHandoffLines(filepath.Join(dir, "nonexistent.jsonl"))
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestTokenLedgerComparison_Contradictory(t *testing.T) {
	t.Parallel()

	// Contradictory reported mission usage
	rec1 := MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 100, TokensOut: 50}
	rec2 := MissionTokenUsageRecord{MissionID: "m-1", TokensIn: 200, TokensOut: 50}
	comp := CompareTokenLedgers([]MissionTokenUsageRecord{rec1, rec2}, nil)
	assert.Equal(t, TokenLedgerInconsistent, comp.Status)
	require.Len(t, comp.Inconsistencies, 1)

	// Contradictory handoff ledger
	tok1 := int64(100)
	tok2 := int64(200)
	line1 := RefinementHandoffLine{MissionID: "m-1", DiscoveryTokens: &tok1}
	line2 := RefinementHandoffLine{MissionID: "m-1", DiscoveryTokens: &tok2}
	comp = CompareTokenLedgers(nil, []RefinementHandoffLine{line1, line2})
	assert.Equal(t, TokenLedgerInconsistent, comp.Status)
	require.Len(t, comp.Inconsistencies, 1)
}

func TestGroundTruthMetrics_BoolToInt(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, boolToInt(true))
	assert.Equal(t, 0, boolToInt(false))
}

func TestNewLevelingBudgetEventCoversClosedEventShapes(t *testing.T) {
	t.Parallel()

	budget := domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion,
		Provider:      "jev",
		Model:         "jev-1",
		ContextBudget: "medium",
		EvidenceDepth: "standard",
		PolicyVersion: 3,
		PolicyDigest:  "policy-digest",
	}
	input := LevelingBudgetEventInput{MissionID: "m-1", Role: "ranger", ResolutionID: "res-1", Remaining: 2, Active: 1}
	shapes := []struct {
		kind     LevelingBudgetEventKind
		name     string
		severity SeverityNumber
	}{
		{LevelingBudgetRecommended, LevelingBudgetRecommendedEvent, SeverityInfo},
		{LevelingBudgetGranted, LevelingBudgetGrantedEvent, SeverityInfo},
		{LevelingBudgetConsumed, LevelingBudgetConsumedEvent, SeverityInfo},
		{LevelingBudgetDenied, LevelingBudgetDeniedEvent, SeverityWarn},
		{LevelingBudgetExhausted, LevelingBudgetExhaustedEvent, SeverityWarn},
		{LevelingBudgetEscalated, LevelingBudgetEscalatedEvent, SeverityInfo},
		{LevelingProviderUsage, LevelingProviderUsageEvent, SeverityInfo},
		{"future", "strategist.leveling.budget.unknown", SeverityWarn},
	}
	for _, shape := range shapes {
		event := NewLevelingBudgetEvent(shape.kind, "run-1", budget, input)
		if event.Name != shape.name || event.SeverityNumber != shape.severity {
			t.Fatalf("kind %q produced name=%q severity=%v", shape.kind, event.Name, event.SeverityNumber)
		}
		if event.Attributes[AttrLevelingProviderEvidence] != "unverified" {
			t.Fatalf("kind %q evidence=%v, want unverified", shape.kind, event.Attributes[AttrLevelingProviderEvidence])
		}
	}
	input.ProviderEvidence = "verified"
	event := NewLevelingBudgetEvent(LevelingProviderUsage, "run-1", budget, input)
	if event.Attributes[AttrLevelingProviderEvidence] != "verified" {
		t.Fatalf("explicit provider evidence was lost: %v", event.Attributes[AttrLevelingProviderEvidence])
	}
}

func TestValidateConfidenceRecordCoversCoverageStatesAndInvariants(t *testing.T) {
	t.Parallel()

	valid := func() ConfidenceRecord {
		return ConfidenceRecord{
			ClaimID: "claim-1", Agent: "ranger", ClaimKind: domain.ClaimKindAssertion,
			ConfidenceLevel: domain.ConfidenceHigh, ConfidencePercent: 90,
			EvidenceIDs: []string{"e-1"},
		}
	}
	validMissing := func() ConfidenceRecord {
		return ConfidenceRecord{MissionID: "m-1", Agent: "ranger", CorrelationKey: "k-1", MissingReason: "not supplied", EventID: "event-1", Timestamp: "2026-10-10T00:00:00Z", CoverageStatus: ConfidenceCoverageMissing}
	}
	validRejected := func() ConfidenceRecord {
		return ConfidenceRecord{ClaimID: "claim-1", Agent: "ranger", Violation: "invalid claim", CoverageStatus: ConfidenceCoverageRejected}
	}

	require.NoError(t, ValidateConfidenceRecord(valid()))
	require.NoError(t, ValidateConfidenceRecord(validMissing()))
	require.NoError(t, ValidateConfidenceRecord(validRejected()))

	cases := []struct {
		name   string
		record ConfidenceRecord
		want   string
	}{
		{"unknown status", ConfidenceRecord{CoverageStatus: "unknown"}, "coverage_status"},
		{"missing timestamp", ConfidenceRecord{CoverageStatus: ConfidenceCoverageMissing, MissionID: "m", Agent: "a", CorrelationKey: "k", MissingReason: "r", EventID: "e"}, "timestamp"},
		{"invalid missing timestamp", func() ConfidenceRecord { r := validMissing(); r.Timestamp = "yesterday"; return r }(), "RFC3339"},
		{"rejected claim id", ConfidenceRecord{Agent: "a", Violation: "v", CoverageStatus: ConfidenceCoverageRejected}, "claim_id"},
		{"rejected agent", ConfidenceRecord{ClaimID: "c", Violation: "v", CoverageStatus: ConfidenceCoverageRejected}, "agent"},
		{"rejected violation", ConfidenceRecord{ClaimID: "c", Agent: "a", CoverageStatus: ConfidenceCoverageRejected}, "violation"},
		{"reported identity", ConfidenceRecord{}, "claim_id"},
		{"reported invalid kind", func() ConfidenceRecord { r := valid(); r.ClaimKind = "other"; return r }(), "claim_kind"},
		{"reported unsupported assertion", func() ConfidenceRecord { r := valid(); r.EvidenceIDs = nil; return r }(), "requires evidence"},
		{"reported empty evidence class", func() ConfidenceRecord { r := valid(); r.EvidenceClasses = []string{""}; return r }(), "evidence_classes"},
		{"reported level mismatch", func() ConfidenceRecord { r := valid(); r.ConfidenceLevel = domain.ConfidenceLow; return r }(), "conflicts"},
		{"corrected without reference", func() ConfidenceRecord { r := valid(); r.Corrected = true; return r }(), "ground_truth_ref"},
		{"reference without kind", func() ConfidenceRecord { r := valid(); r.GroundTruthRef = "ref"; return r }(), "ground_truth_kind"},
		{"kind without reference", func() ConfidenceRecord { r := valid(); r.GroundTruthKind = domain.GroundTruthHandoff; return r }(), "ground_truth_ref"},
		{"outcome without reference", func() ConfidenceRecord { r := valid(); r.GroundTruthOutcome = domain.GroundTruthCorrect; return r }(), "ground_truth_ref"},
		{"invalid outcome", func() ConfidenceRecord {
			r := valid()
			r.GroundTruthRef = "ref"
			r.GroundTruthOutcome = "unknown"
			return r
		}(), "ground_truth_outcome"},
		{"invalid ground truth kind", func() ConfidenceRecord { r := valid(); r.GroundTruthKind = "other"; return r }(), "ground_truth_kind"},
		{"invalid calibration status", func() ConfidenceRecord { r := valid(); r.CalibrationStatus = "other"; return r }(), "calibration"},
		{"calibrated incomplete", func() ConfidenceRecord { r := valid(); r.CalibrationStatus = domain.CalibrationCalibrated; return r }(), "calibrated"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, ValidateConfidenceRecord(tc.record), tc.want)
		})
	}
}

func TestRefinedPackagePublicationSkipsMalformedHistory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory", "publications.jsonl")
	record := RefinedPackagePublicationRecord{MissionID: "m-1", ProviderChangeID: "change-1", SourceDigest: "source", PackageDigest: "package", PublishedAt: time.Unix(1, 0).UTC()}
	if err := AppendRefinedPackagePublication(path, record); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("not-json\n")
	_ = f.Close()
	got, err := ReadRefinedPackagePublications(path)
	if err != nil || len(got) != 1 || got[0].MissionID != record.MissionID {
		t.Fatalf("records=%+v err=%v", got, err)
	}
	missing, err := ReadRefinedPackagePublications(filepath.Join(t.TempDir(), "missing.jsonl"))
	if err != nil || missing != nil {
		t.Fatalf("missing history=%+v err=%v", missing, err)
	}
}

func TestCompareTokenLedgersReportsEveryNonContradictoryState(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		report  []MissionTokenUsageRecord
		handoff []RefinementHandoffLine
		want    string
	}{
		{"no data", nil, nil, TokenLedgerNoData},
		{"handoff only", nil, []RefinementHandoffLine{{MissionID: "m"}}, TokenLedgerHandoffOnly},
		{"reported only", []MissionTokenUsageRecord{{MissionID: "m", TokensIn: 1}}, nil, TokenLedgerReportedOnly},
		{"both not comparable", []MissionTokenUsageRecord{{MissionID: "m", TokensIn: 1}}, []RefinementHandoffLine{{MissionID: "m"}}, TokenLedgerNotComparable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CompareTokenLedgers(tc.report, tc.handoff).Status; got != tc.want {
				t.Fatalf("status=%q, want %q", got, tc.want)
			}
		})
	}
	if got := CompareTokenLedgers(
		[]MissionTokenUsageRecord{{MissionID: "m", TokensIn: 1, TokensOut: 1}, {MissionID: "m", TokensIn: 1, TokensOut: 1}},
		[]RefinementHandoffLine{{MissionID: "m", DiscoveryTokens: i64(1)}, {MissionID: "m", DiscoveryTokens: i64(1)}},
	); got.Status != TokenLedgerNotComparable {
		t.Fatalf("matching ledgers status=%q, want %q", got.Status, TokenLedgerNotComparable)
	}
}

func TestBuildConfidenceGateReviewReportsDiagnosticsAndRecordFlags(t *testing.T) {
	t.Parallel()

	review := BuildConfidenceGateReview([]ConfidenceRecord{
		{ClaimID: "low", Agent: "ranger", ClaimKind: domain.ClaimKindQuestion, ConfidenceLevel: domain.ConfidenceLow, ConfidencePercent: 20},
		{ClaimID: "bad", Agent: "ranger", ClaimKind: domain.ClaimKindAssertion, CoverageStatus: ConfidenceCoverageRejected, Violation: "unsupported"},
	}, ConfidenceReadDiagnostics{MalformedLines: 1, InvalidRecords: 2, DuplicateEvents: 1})
	if !review.ReviewRequired || review.Unavailable {
		t.Fatalf("diagnostic review=%+v", review)
	}
	for _, reason := range []string{"invalid_confidence_records", "duplicate_confidence_records", "unsupported_assertions", "calibration_not_established"} {
		if !containsCoverageReason(review.Reasons, reason) {
			t.Fatalf("reasons=%v, missing %q", review.Reasons, reason)
		}
	}
	var nilReview *ConfidenceGateReview
	nilReview.RequiredCalibrationReason()
}

func TestEventJSONLAppendsAndRejectsInvalidEvents(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "memory", "events.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	event := NewEvent("strategist.test.complete", SeverityInfo, "run-1", true)
	event.Body = "done"
	event.Attributes = map[string]any{AttrEventAuthority: AuthorityStrategistLocal}
	if err := AppendEventLine(path, event); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if line, err := os.ReadFile(path); err != nil || len(line) == 0 {
		t.Fatalf("event history bytes=%q err=%v", line, err)
	}
	if err := AppendEventLine(filepath.Join(dir, "missing", "events"), Event{}); err == nil {
		t.Fatal("invalid event should be rejected")
	}
	critical := NewEvent("strategist.test.critical", SeverityError, "run-1", true)
	if _, err := MarshalEventLine(critical); err == nil {
		t.Fatal("critical event without contract id should be rejected")
	}
}

func TestMissionTokenUsageValidationAndMissingHistory(t *testing.T) {
	t.Parallel()

	err := ValidateMissionTokenUsage(MissionTokenUsageRecord{TokensIn: -1, TokensOut: -2})
	require.Error(t, err)
	for _, want := range []string{"mission_id", "tokens_in", "tokens_out", "source", "reported_at"} {
		require.Contains(t, err.Error(), want)
	}
	missing, err := ReadMissionTokenUsage(filepath.Join(t.TempDir(), "missing.jsonl"))
	require.NoError(t, err)
	require.Nil(t, missing)
	path := filepath.Join(t.TempDir(), "usage.jsonl")
	record := MissionTokenUsageRecord{MissionID: "m", TokensIn: 1, TokensOut: 2, Source: MissionUsageSourceAgentReport, ReportedAt: "2026-10-10T00:00:00Z"}
	require.NoError(t, AppendMissionTokenUsage(path, record))
	got, err := ReadMissionTokenUsage(path)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "m", got[0].MissionID)
}

func containsCoverageReason(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestCoverageTelemetryHistoryErrorPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	confidencePath := ConfidenceHistoryPath(root)
	claim := domain.ConfidenceClaim{ID: "claim-coverage", Statement: "supported", ClaimKind: domain.ClaimKindAssertion, ConfidencePercent: 90, CorrelationKey: "coverage", EvidenceIDs: []string{"e-coverage"}}
	evidence := []domain.Evidence{{ID: "e-coverage", SourceRef: "docs/evidence.md", Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh}}
	require.NoError(t, AppendConfidenceClaim(confidencePath, "ranger", "mission-coverage", "2026-10-10T00:00:00Z", claim, evidence))
	_, records, err := LoadConfidenceGateReviewWithRecords(root, "mission-coverage")
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, ConfidenceCoverageReported, records[0].CoverageStatus)

	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("blocked"), 0o600))
	validRecord := ConfidenceRecord{EventID: "event-coverage", MissionID: "m", ClaimID: "c", Agent: ConfidenceAgentRanger, ClaimKind: domain.ClaimKindQuestion, ConfidenceLevel: domain.ConfidenceLow, ConfidencePercent: 20, Timestamp: "2026-10-10T00:00:00Z"}
	require.Error(t, AppendConfidenceRecord(filepath.Join(blocker, "memory", "records.jsonl"), validRecord))
	_, _, err = ReadConfidenceRecordsWithDiagnostics(filepath.Join(t.TempDir(), "records\x00.jsonl"))
	require.Error(t, err)

	_, err = AppendRefinementHandoffLine(filepath.Join(blocker, "handoff.jsonl"), RefinementHandoffLine{MissionID: "m"})
	require.Error(t, err)
	handoffDir := filepath.Join(t.TempDir(), "handoff-dir")
	require.NoError(t, os.MkdirAll(handoffDir, 0o755))
	_, err = ReadRefinementHandoffLines(handoffDir)
	require.Error(t, err)

	claimPath := filepath.Join(blocker, "claims.jsonl")
	require.Error(t, AppendSniperClaim(claimPath, SniperClaimRecord{MissionID: "m", TargetPath: "docs/a.md", ClaimedAt: time.Now()}))
	claimDir := filepath.Join(t.TempDir(), "claims-dir")
	require.NoError(t, os.MkdirAll(claimDir, 0o755))
	require.Error(t, AppendSniperClaim(claimDir, SniperClaimRecord{MissionID: "m", TargetPath: "docs/a.md", ClaimedAt: time.Now()}))
	_, err = ReadRecentSniperClaims(filepath.Join(t.TempDir(), "claims\x00.jsonl"), time.Now(), SniperClaimWindow)
	require.Error(t, err)

	label := gtLabel("m", GroundTruthSubjectRoute, RouteLabelConfirmed)
	require.Error(t, func() error {
		_, err := AppendGroundTruthLabel(filepath.Join(blocker, "labels.jsonl"), label)
		return err
	}())
	_, err = ReadGroundTruthLabels(filepath.Join(t.TempDir(), "labels\x00.jsonl"), "")
	require.Error(t, err)

	usage := MissionTokenUsageRecord{MissionID: "m", TokensIn: 1, TokensOut: 2, Source: MissionUsageSourceAgentReport, ReportedAt: "2026-10-10T00:00:00Z"}
	require.Error(t, AppendMissionTokenUsage(filepath.Join(blocker, "usage.jsonl"), usage))
	_, err = ReadMissionTokenUsage(filepath.Join(t.TempDir(), "usage\x00.jsonl"))
	require.Error(t, err)

	publication := RefinedPackagePublicationRecord{MissionID: "m", ProviderChangeID: "change", SourceDigest: "source", PackageDigest: "package", PublishedAt: time.Now().UTC()}
	require.Error(t, AppendRefinedPackagePublication(filepath.Join(blocker, "publications.jsonl"), publication))
	_, err = ReadRefinedPackagePublications(filepath.Join(t.TempDir(), "publications\x00.jsonl"))
	require.Error(t, err)

	event := NewEvent("strategist.coverage", SeverityInfo, "run", true)
	require.Error(t, AppendEventLine(filepath.Join(blocker, "events.jsonl"), event))

	checkpoint := &MissionCheckpoint{MissionID: "m"}
	require.Error(t, SaveCheckpoint(filepath.Join(blocker, "checkpoint.json"), checkpoint))
	_, err = LoadCheckpoint(filepath.Join(t.TempDir(), "checkpoint\x00.json"))
	assert.Error(t, err)
}

func coverageConfidenceRecord() ConfidenceRecord {
	return ConfidenceRecord{
		EventID: "event-coverage", MissionID: "m", ClaimID: "c", Agent: ConfidenceAgentRanger,
		ClaimKind: domain.ClaimKindAssertion, ConfidenceLevel: domain.ConfidenceHigh, ConfidencePercent: 90,
		EvidenceIDs: []string{"e"}, EvidenceProvided: true, Timestamp: "2026-10-10T00:00:00Z",
	}
}

func TestCoverageConfidenceAndEventBranches(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "memory", "confidence.jsonl")
	record := coverageConfidenceRecord()
	appended, err := AppendConfidenceRecordOnce(path, record)
	require.NoError(t, err)
	assert.True(t, appended)
	appended, err = AppendConfidenceRecordOnce(path, record)
	require.NoError(t, err)
	assert.False(t, appended)
	require.Error(t, AppendConfidenceRecord(path, ConfidenceRecord{}))
	_, err = confidenceEventExists(filepath.Join(t.TempDir(), "missing.jsonl"), "event")
	require.Error(t, err)

	invalidClaim := domain.ConfidenceClaim{ID: "bad", ClaimKind: domain.ClaimKindAssertion, ConfidencePercent: 200}
	_, err = NormalizeConfidenceClaim("ranger", "m", "2026-10-10T00:00:00Z", invalidClaim, nil)
	require.Error(t, err)
	claim := domain.ConfidenceClaim{ID: "declared", Statement: "supported", ClaimKind: domain.ClaimKindAssertion, ConfidencePercent: 90, ConfidenceLevel: domain.ConfidenceHigh, CorrelationKey: "k", EvidenceIDs: []string{"e"}, EvidenceClasses: []string{domain.EvidenceClassExplicit}}
	_, err = NormalizeConfidenceClaim("ranger", "m", "2026-10-10T00:00:00Z", claim, []domain.Evidence{{ID: "e", SourceRef: "docs/e.md", Class: domain.EvidenceClassExplicit, Confidence: domain.ConfidenceHigh}})
	require.NoError(t, err)

	validJSON, err := json.Marshal(record)
	require.NoError(t, err)
	history := filepath.Join(t.TempDir(), "records.jsonl")
	raw := append([]byte("\nnot-json\n{\"claim_id\":\"bad\"}\n"), validJSON...)
	raw = append(raw, '\n')
	raw = append(raw, validJSON...)
	raw = append(raw, '\n')
	require.NoError(t, os.WriteFile(history, raw, 0o600))
	_, diagnostics, err := ReadConfidenceRecordsWithDiagnostics(history)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, diagnostics.MalformedLines, 1)
	assert.GreaterOrEqual(t, diagnostics.InvalidRecords, 1)
	assert.GreaterOrEqual(t, diagnostics.DuplicateEvents, 1)

	corrected := record
	corrected.EventID = "corrected"
	corrected.GroundTruthRef = "review"
	corrected.GroundTruthKind = domain.GroundTruthUserRevision
	corrected.GroundTruthOutcome = domain.GroundTruthIncorrect
	corrected.Corrected = true
	corrected.Reviewed = true
	corrected.SampleSize = domain.CalibrationMinimumSample
	metrics := ComputeConfidenceMetrics([]ConfidenceRecord{record, corrected, record})
	assert.Equal(t, 1, metrics.DuplicateRecords)
	assert.Equal(t, domain.CalibrationMinimumSample, metrics.GroundTruthSampleSize)

	mainCriterion := WithHandoffCriterion(Event{}, HandoffCriterion{})
	assert.Equal(t, CriterionSourceMain, mainCriterion.Attributes[AttrHandoffCriterionSource])
	failed := NewArchivistToSniperEvent("run", "failed", true, 1, "blocked", 1, "reason")
	assert.Equal(t, SeverityError, failed.SeverityNumber)
	missing := NewConfidenceReportEvent("run", domain.NewMissingConfidenceReport("missing"))
	assert.Equal(t, SeverityWarn, missing.SeverityNumber)
}
