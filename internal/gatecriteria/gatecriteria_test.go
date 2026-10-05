package gatecriteria

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/stretchr/testify/require"
)

const mission = "m1"

type fixture struct {
	root, basePath string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	return fixture{root: filepath.Join(dir, ".strategist"), basePath: filepath.Join(dir, ".analysis")}
}

func (f fixture) input() Input {
	return Input{Root: f.root, BasePath: f.basePath, MissionID: mission}
}

func (f fixture) claim(t *testing.T, agent, id string, percent int, kind string) {
	t.Helper()
	producer, err := telemetry.NewConfidenceProducerAdapter(telemetry.ConfidenceHistoryPath(f.root), agent, mission)
	require.NoError(t, err)
	level, err := domain.ConfidenceLevelForPercent(percent)
	require.NoError(t, err)
	evidence := domain.Evidence{ID: "ev-" + id, SourceRef: "x.md", Class: domain.EvidenceClassExplicit, Confidence: level, ConfidencePercent: &percent}
	_, err = producer.RecordClaim(domain.ConfidenceClaim{
		ID: id, Statement: "s " + id, Agent: agent, CorrelationKey: id, ClaimKind: kind, ConfidencePercent: percent, ConfidenceLevel: level,
		EvidenceIDs: []string{"ev-" + id}, EvidenceClasses: []string{domain.EvidenceClassExplicit}, CalibrationStatus: domain.CalibrationNoSample,
	}, []domain.Evidence{evidence})
	require.NoError(t, err)
}

func (f fixture) route(t *testing.T, confidence float64) {
	t.Helper()
	path := filepath.Join(f.root, "memory", "route-decisions.jsonl")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	line, err := json.Marshal(map[string]any{"mission_id": mission, "selected_route": "full_pipeline", "route_confidence": confidence, "evidence_state": "requires_discovery"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(line, '\n'), 0o644))
}

func (f fixture) outcome(t *testing.T, transition, result string, delegation *handoff.Delegation) {
	t.Helper()
	outcome := handoff.Outcome{
		MissionID: mission, Transition: transition, Attempt: 1, Result: result, PolicyID: "p", Required: result != handoff.OutcomeSkipped,
		GateObserved: "ranger_normalized", ChallengeStatus: "passed", Delegation: delegation,
	}
	if transition == handoff.TransitionRangerToArchivist {
		outcome.ArtifactDigest = "sha256:artifact"
	} else {
		outcome.PackageDigest = "sha256:package"
	}
	_, err := handoff.NewOutcomeStore(f.root).Append(outcome)
	require.NoError(t, err)
}

func (f fixture) ledger(t *testing.T, record handoffconsumer.Record) {
	t.Helper()
	record.MissionID = mission
	require.NoError(t, handoffconsumer.Ledger{Path: filepath.Join(f.root, "memory", "integration-calls.jsonl")}.Append(record))
}

func (f fixture) enableIntegration(t *testing.T) {
	t.Helper()
	require.NoError(t, config.Save(filepath.Join(f.root, config.FileName), config.File{SchemaVersion: config.SchemaVersion, Providers: map[string]config.Provider{"jev": {
		Enabled: true, Endpoint: "https://api.example.test/v1", CredentialRef: "env:K", Model: "jev-1.13.0",
		AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate},
		DataPolicy:          config.DataPolicy{AllowedFields: []string{"objective"}, MaxStateBytes: 100},
	}}}))
}

func row(t *testing.T, report Report, criterion string) Row {
	t.Helper()
	for _, r := range report.Rows {
		if r.Criterion == criterion {
			return r
		}
	}
	t.Fatalf("no row %q in %v", criterion, report.Rows)
	return Row{}
}

func TestEveryRowIsAlwaysPresentAndFilledEvenWithNoRecordsAtAll(t *testing.T) {
	report := Build(newFixture(t).input())

	require.Equal(t, Criteria(), criteriaOf(report), "every criterion appears, in a stable order")
	for _, r := range report.Rows {
		require.NotEmpty(t, r.Criterion)
		require.NotEmpty(t, r.Value, "%s: a value is never blank", r.Criterion)
		require.NotEmpty(t, r.Source, "%s", r.Criterion)
		require.NotEmpty(t, r.Detail, "%s", r.Criterion)
		require.Contains(t, []string{StateEvaluated, StateNotEvaluated, StateUnknown}, r.State, r.Criterion)
		require.Equal(t, StateNotEvaluated, r.State, "%s: with nothing recorded, the default is not evaluated", r.Criterion)
		require.Equal(t, StateNotEvaluated, r.Value, "%s: the default fills the value", r.Criterion)
	}
}

func criteriaOf(report Report) []string {
	names := make([]string, 0, len(report.Rows))
	for _, r := range report.Rows {
		names = append(names, r.Criterion)
	}
	return names
}

func TestExistingRecordsAreReusedNotRecomputed(t *testing.T) {
	f := newFixture(t)
	f.route(t, 0.85)
	f.claim(t, telemetry.ConfidenceAgentScout, "scout-route", 85, domain.ClaimKindAssertion)
	f.claim(t, telemetry.ConfidenceAgentRanger, "ranger-1", 82, domain.ClaimKindAssertion)
	f.claim(t, telemetry.ConfidenceAgentArchivist, "archivist-1", 100, domain.ClaimKindAssertion)
	f.claim(t, telemetry.ConfidenceAgentCritic, "critic-1", 78, domain.ClaimKindAssertion)
	f.outcome(t, handoff.TransitionRangerToArchivist, handoff.OutcomePassed, nil)

	report := Build(f.input())

	require.Equal(t, "85%", row(t, report, "route_confidence").Value)
	require.Equal(t, StateEvaluated, row(t, report, "route_confidence").State)
	require.Contains(t, row(t, report, "claim_confidence.ranger").Value, "82%")
	require.Contains(t, row(t, report, "claim_confidence.archivist").Value, "100%")
	require.Contains(t, row(t, report, "critic").Value, "78%")
	require.Equal(t, StateNotEvaluated, row(t, report, "mission_quality").State, "no record, so not evaluated, never presumed")
	require.Contains(t, row(t, report, "handoff.ranger_to_archivist").Value, "passed")
	require.Contains(t, row(t, report, "handoff.ranger_to_archivist").Value, "attempt 1")
}

func TestAnUncalibratedOrAbsentReviewIsUnknownNotAGuess(t *testing.T) {
	absent := Build(newFixture(t).input())
	require.Equal(t, StateNotEvaluated, row(t, absent, "confidence_review").State)

	f := newFixture(t)
	f.claim(t, telemetry.ConfidenceAgentRanger, "ranger-1", 82, domain.ClaimKindAssertion)
	review := row(t, Build(f.input()), "confidence_review")
	require.Equal(t, StateEvaluated, review.State)
	require.Contains(t, review.Value, "review_required=")
	require.Contains(t, review.Value, "calibration=")
}

func TestAnAnsweredProviderDelegationShowsItsConfidenceAgainstTheThreshold(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	delegation := &handoff.Delegation{
		Provider: "jev", Model: "jev-1.13.0", BindingDigest: "sha256:b", Capability: "handoff.validate", Criterion: "handoff_acceptable",
		Subject: "sha256:artifact", Checks: handoff.DelegableChecks(), Confidence: 0.94, Threshold: 0.90,
	}
	f.outcome(t, handoff.TransitionRangerToArchivist, handoff.OutcomePassed, delegation)

	provider := row(t, Build(f.input()), "provider.ranger_to_archivist")
	require.Equal(t, StateEvaluated, provider.State)
	require.Equal(t, "JEV confiança: 94% (limiar 90%) — aprovado, não calibrado", provider.Value)
}

func TestASignalFromTheLedgerIsShownBelowTheThresholdOnTheMainPath(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	f.ledger(t, handoffconsumer.Record{Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:artifact", Provider: "jev", Attempted: true, Answered: true, Confidence: 0.71, Threshold: 0.90})

	provider := row(t, Build(f.input()), "provider.ranger_to_archivist")
	require.Equal(t, StateEvaluated, provider.State)
	require.Equal(t, "JEV confiança: 71% (limiar 90%) — abaixo do limiar, caminho main, não calibrado", provider.Value)
}

func TestAProviderThatDidNotAnswerIsNotEvaluatedWithTheReason(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	f.ledger(t, handoffconsumer.Record{Transition: handoff.TransitionRangerToArchivist, Subject: "sha256:artifact", Provider: "jev", Attempted: true, Reason: "timeout", FellBack: true})

	provider := row(t, Build(f.input()), "provider.ranger_to_archivist")
	require.Equal(t, StateNotEvaluated, provider.State)
	require.Contains(t, provider.Detail, "timeout")
}

func TestWithoutAnEnabledIntegrationTheProviderRowsSaySo(t *testing.T) {
	report := Build(newFixture(t).input())
	for _, criterion := range []string{"provider.ranger_to_archivist", "provider.archivist_to_gate", "provider.archivist_to_sniper"} {
		r := row(t, report, criterion)
		require.Equal(t, StateNotEvaluated, r.State)
	}
	require.Contains(t, row(t, report, "provider.archivist_to_gate").Detail, "integração não habilitada")
	require.Contains(t, row(t, report, "handoff.archivist_to_sniper").Detail, "só existe depois do gate")
}

func TestAGateReadingOfAnotherRevisionIsUnknown(t *testing.T) {
	f := newFixture(t)
	f.enableIntegration(t)
	pkg := filepath.Join(f.basePath, "refined", mission)
	require.NoError(t, os.MkdirAll(pkg, 0o755))
	for name, content := range map[string]string{
		"analysis.md": "---\nmission_id: m1\nhandoff_policy_facts: {}\n---\n\nbody\n", "proposal.md": "p", "design.md": "d", "tasks.md": "t",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(pkg, name), []byte(content), 0o644))
	}
	digest, err := handoff.PackageDigest(pkg)
	require.NoError(t, err)

	f.ledger(t, handoffconsumer.Record{Transition: handoffconsumer.TransitionArchivistToGate, Subject: "sha256:older", Provider: "jev", Attempted: true, Answered: true, Confidence: 0.96, Threshold: 0.90})
	stale := row(t, Build(f.input()), "provider.archivist_to_gate")
	require.Equal(t, StateUnknown, stale.State, "a reading of another revision cannot support a statement about this one")

	f.ledger(t, handoffconsumer.Record{Transition: handoffconsumer.TransitionArchivistToGate, Subject: digest, Provider: "jev", Attempted: true, Answered: true, Confidence: 0.96, Threshold: 0.90})
	current := row(t, Build(f.input()), "provider.archivist_to_gate")
	require.Equal(t, StateEvaluated, current.State)
	require.Contains(t, current.Value, "96%")
}

func TestTheReportIsJSONWithNoEmptyField(t *testing.T) {
	raw, err := json.Marshal(Build(newFixture(t).input()))
	require.NoError(t, err)
	var parsed struct {
		Rows []map[string]any `json:"rows"`
	}
	require.NoError(t, json.Unmarshal(raw, &parsed))
	require.NotEmpty(t, parsed.Rows)
	for _, r := range parsed.Rows {
		for field, value := range r {
			require.NotEmpty(t, value, "row %v: field %s is never empty", r["criterion"], field)
		}
	}
}
