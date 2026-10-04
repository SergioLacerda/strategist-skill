package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/mechanisms"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/refinement"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissionCompletionJSONValidationBranches(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte("[]"),
		[]byte(`{"request_id":"a"`),
		[]byte(`{"request_id":}`),
	} {
		require.Error(t, validateCompletionObject(raw))
	}
	require.NoError(t, validateCompletionObject([]byte(`{"request_id":"a","result":"ok","unknown":true}`)))
}

func TestRangerHandoffInputAndOutputBranches(t *testing.T) {
	tmp := t.TempDir()
	challenges := filepath.Join(tmp, "challenges.yaml")
	ack := filepath.Join(tmp, "ack.yaml")
	require.NoError(t, os.WriteFile(challenges, []byte("challenges:\n  - id: c1\n    type: evidence\n    source_refs: [finding.md]\n    critical: true\n"), 0o600))
	require.NoError(t, os.WriteFile(ack, []byte("challenge_refs: [c1]\nunderstood_refs: [c1]\nclassifications: {c1: confirmed}\n"), 0o600))
	input, err := rangerHandoffInput(rangerHandoffEvaluateOptions{Challenges: challenges, Ack: ack})
	require.NoError(t, err)
	require.Len(t, input.Challenges, 1)
	require.Equal(t, []string{"c1"}, input.Ack.ChallengeRefs)
	_, err = rangerHandoffInput(rangerHandoffEvaluateOptions{Challenges: filepath.Join(tmp, "missing")})
	require.Error(t, err)
	_, err = rangerHandoffInput(rangerHandoffEvaluateOptions{Challenges: challenges, Ack: filepath.Join(tmp, "missing")})
	require.Error(t, err)
	require.Error(t, runRangerHandoffEvaluate(&cobra.Command{}, rangerHandoffEvaluateOptions{}))
	_, root := workspaceWithRoot(t)
	missing := filepath.Join(tmp, "missing.yaml")
	require.Error(t, runRangerHandoffEvaluate(&cobra.Command{}, rangerHandoffEvaluateOptions{Root: root, MissionID: "m-1", Challenges: missing, Ack: missing}))
	require.Error(t, runRangerHandoffEvaluate(&cobra.Command{}, rangerHandoffEvaluateOptions{Root: root, MissionID: "m-1"}))
	_, err = evaluateRangerLocked(tmp, tmp, "m-1", missionruntime.RangerHandoffInput{})
	require.Error(t, err)

	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)
	require.NoError(t, printRangerHandoffEvaluation(cmd, missionruntime.RangerHandoffResult{
		Outcome: handoff.Outcome{Required: true, Result: handoff.OutcomePassed, Attempt: 1, ArtifactDigest: "sha256:artifact"},
	}))
	assert.Contains(t, output.String(), "status:")
}

func TestSniperCompletionSupportBranches(t *testing.T) {
	expected := ".analysis/archived/m-1-report.md"
	require.NoError(t, verifySniperCompletionSignal("Return: sniper: done | report_path: "+expected+" | mission_status: documentation_applied", expected))
	require.Error(t, verifySniperCompletionSignal("sniper: done", expected))

	raw := []byte("- [x] 1.1 [documentation_target] Write `docs/guide.md`\n- [X] 1.2 [documentation_target] Write `docs/other.md`\n")
	require.True(t, documentationTargetCompleted(raw, "docs/guide.md"))
	require.True(t, documentationTargetCompleted(raw, "docs/other.md"))
	require.False(t, documentationTargetCompleted(raw, "docs/missing.md"))
	tasks := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(tasks, raw, 0o600))
	require.NoError(t, requireCompletedDocumentationTargets(tasks, []string{"docs/guide.md", "docs/other.md"}))
	require.Error(t, requireCompletedDocumentationTargets(tasks, []string{"docs/missing.md"}))
	require.Error(t, requireCompletedDocumentationTargets(filepath.Join(t.TempDir(), "missing.md"), []string{"docs/guide.md"}))
	refined := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("---\nmission_id: m-1\nmission_status: documentation_applied\nclaimed_by: sniper\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(refined, "tasks.md"), raw, 0o600))
	targets, err := verifySniperLifecycleAndTasks(refined, "m-1")
	require.NoError(t, err)
	require.Equal(t, []string{"docs/guide.md", "docs/other.md"}, targets)
	_, err = verifySniperLifecycleAndTasks(filepath.Join(t.TempDir(), "missing"), "m-1")
	require.Error(t, err)
	incomplete := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(incomplete, "analysis.md"), []byte("---\nmission_id: wrong\nmission_status: pending\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(incomplete, "tasks.md"), raw, 0o600))
	_, err = verifySniperLifecycleAndTasks(incomplete, "m-1")
	require.Error(t, err)
	noTargets := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(noTargets, "analysis.md"), []byte("---\nmission_id: m-1\nmission_status: documentation_applied\nclaimed_by: sniper\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(noTargets, "tasks.md"), []byte("- [ ] 1.1 implementation\n"), 0o600))
	_, err = verifySniperLifecycleAndTasks(noTargets, "m-1")
	require.Error(t, err)
	incompleteTargets := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(incompleteTargets, "analysis.md"), []byte("---\nmission_id: m-1\nmission_status: documentation_applied\nclaimed_by: sniper\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(incompleteTargets, "tasks.md"), []byte("- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n"), 0o600))
	_, err = verifySniperLifecycleAndTasks(incompleteTargets, "m-1")
	require.Error(t, err)

	root := t.TempDir()
	validTasks := filepath.Join(root, "tasks.md")
	require.NoError(t, os.WriteFile(validTasks, []byte("- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n"), 0o600))
	require.NoError(t, requireSniperTargets(validTasks))
	require.Error(t, requireSniperTargets(filepath.Join(root, "absent.md")))
	require.Error(t, verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCodexChild}, domain.MissionInvocationRecord{}))
	require.Error(t, verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost, Root: filepath.Join(root, ".strategist"), BasePath: filepath.Join(root, ".analysis"), Completion: domain.MissionInvocationCompletion{Result: "bad"}}, domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{MissionID: "m-1"}}))
	require.Error(t, requireSniperMaterializations(root, recordForSniper("m-1"), []string{"docs/guide.md"}))
	require.NoError(t, authorizeSniperEntry(missionadapter.InvocationBuildInput{Role: "ranger", Slot: string(domain.SlotDiscovery)}))
	require.Error(t, authorizeArchivistEntry(missionadapter.InvocationBuildInput{Role: "archivist", Slot: string(domain.SlotRefinement)}))

	record := domain.MissionInvocationRecord{
		CreatedAt: time.Unix(100, 0),
		Request:   domain.MissionInvocationRequest{MissionID: "m-1", Input: map[string]any{"approval_gate_package_digest": "sha256:pkg"}},
	}
	assert.Equal(t, "sha256:pkg", requestedPackageDigest(record))
	record.Request.Input["approval_gate_package_digest"] = 7
	assert.Empty(t, requestedPackageDigest(record))
	matching := telemetry.SniperMaterializationRecord{MissionID: "m-1", TargetPath: "docs/guide.md", PackageDigest: "sha256:pkg", MaterializedAt: time.Unix(101, 0)}
	assert.True(t, materializationMatchesRequest(matching, record, "sha256:pkg"))
	matching.MissionID = "other"
	assert.False(t, materializationMatchesRequest(matching, record, "sha256:pkg"))
	matching.MissionID, matching.MaterializedAt = "m-1", time.Unix(99, 0)
	assert.False(t, materializationMatchesRequest(matching, record, "sha256:pkg"))
	seen := sniperMaterializationTargets([]telemetry.SniperMaterializationRecord{{MissionID: "m-1", TargetPath: "docs/guide.md", MaterializedAt: time.Unix(101, 0)}}, record)
	assert.True(t, seen["docs/guide.md"])

	relative, err := sniperReportRelativePath(filepath.Join(root, ".strategist"), filepath.Join(root, ".analysis"), "m-1")
	require.NoError(t, err)
	assert.Equal(t, ".analysis/archived/m-1-report.md", relative)
	_, err = sniperReportRelativePath(filepath.Join(root, ".strategist"), filepath.Join(t.TempDir(), ".analysis"), "m-1")
	require.Error(t, err)
}

func recordForSniper(missionID string) domain.MissionInvocationRecord {
	return domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{MissionID: missionID}}
}

func TestVerifyExecutionAdapterBranches(t *testing.T) {
	record := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{RequestID: "inv-1"}}
	for _, adapter := range []domain.MissionExecutionAdapter{"", "future", domain.ExecutionAdapterCodexChild} {
		require.Error(t, verifyExecutionAdapter(record, missionadapter.InvocationCompleteInput{Adapter: adapter}))
	}
	record.ExecutionAdapter = domain.ExecutionAdapterCodexChild
	require.Error(t, verifyExecutionAdapter(record, missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost}))
	require.Error(t, verifyExecutionAdapter(record, missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCodexChild}))
	record.ChildPolicyID = "policy-v1"
	require.NoError(t, verifyExecutionAdapter(record, missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCodexChild}))
	record.ExecutionAdapter = domain.ExecutionAdapterCurrentHostUnverified
	require.NoError(t, verifyExecutionAdapter(record, missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost}))
}

func TestMissionIDValidationBranches(t *testing.T) {
	for _, id := range []string{"", "Upper", "has_underscore", "../escape", "-starts"} {
		require.Error(t, validateMissionID(id))
		require.Error(t, requireMissionID(id))
	}
	require.NoError(t, validateMissionID("20261003-valid-mission"))
	require.NoError(t, requireMissionID("valid"))
}

func TestMechanismsBriefRenderingBranches(t *testing.T) {
	registry := mechanisms.Registry{Rows: []mechanisms.Row{{ID: "tool", Family: mechanisms.FamilyTool, Summary: "summary", HowToInvoke: "invoke", InvokedBy: []string{mechanisms.AllRoles}, PhaseScope: []string{"execution"}}}}
	brief, err := renderMechanismsBrief(registry, "ranger", "", false)
	require.NoError(t, err)
	require.NotEmpty(t, brief)
	brief, err = renderMechanismsBrief(registry, "ranger", "full", true)
	require.NoError(t, err)
	require.NotEmpty(t, brief)
	brief, err = renderMechanismsBrief(registry, "ranger", "short", false)
	require.NoError(t, err)
	require.NotEmpty(t, brief)
	_, err = renderMechanismsBrief(registry, "ranger", "future", true)
	require.Error(t, err)
}

func TestMissionInvocationAuthorizationAndPayloadFailures(t *testing.T) {
	_, _, _, _, err := resolveMissionInvocationWeapon(missionadapter.InvocationBuildInput{Root: t.TempDir()})
	require.Error(t, err)

	_, _, err = readMissionInvocationPayload(domain.RoleWeaponBinding{ConnectorID: "strategist-native-role", WeaponID: "missing"})
	require.Error(t, err)
	_, _, err = readMissionInvocationPayload(domain.RoleWeaponBinding{WeaponID: "missing", WeaponVersion: "1.0.0"})
	require.Error(t, err)
	_, _, err = resolveEmbeddedInvocationBinding(domain.ActiveConfig{}, domain.PluginLockFile{}, domain.CompiledRegistry{}, missionadapter.InvocationBuildInput{Role: "ranger", Slot: "discovery"})
	require.Error(t, err)

	root := t.TempDir()
	tasks := filepath.Join(root, "refined", "m-1", "tasks.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(tasks), 0o755))
	require.NoError(t, os.WriteFile(tasks, []byte("- [ ] 1.1 [documentation_target] missing path\n"), 0o600))
	require.Error(t, authorizeSniperEntry(missionadapter.InvocationBuildInput{Root: root, BasePath: root, MissionID: "m-1", Role: "sniper", Slot: string(domain.SlotExecution)}))
	require.NoError(t, authorizeSniperEntry(missionadapter.InvocationBuildInput{Role: "ranger", Slot: string(domain.SlotDiscovery)}))
	badTasks := filepath.Join(root, "refined", "m-2", "tasks.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(badTasks), 0o755))
	require.NoError(t, os.WriteFile(badTasks, []byte("- [ ] 1.1 [documentation_target] Write the guide\n"), 0o600))
	require.Error(t, requireSniperTargets(badTasks))
}

func TestMissionNormalizeAndRouteDependencyFailures(t *testing.T) {
	broken := missionadapter.NormalizeOptions{Root: filepath.Join(t.TempDir(), "absent"), MissionID: "m-1"}
	_, err := loadNormalizeMission(broken)
	require.Error(t, err)
	_, root := workspaceWithRoot(t)
	_, err = loadNormalizeMission(missionadapter.NormalizeOptions{Root: root, MissionID: "missing"})
	require.Error(t, err)
	_, err = adrCanonicalPath(t.TempDir())
	require.Error(t, err)
	err = recordNormalizeConfidence(missionadapter.NormalizeOptions{Root: root, MissionID: ""}, domain.ConfidenceClaim{}, nil)
	require.Error(t, err)
	err = recordNormalizePublication(missionadapter.NormalizeOptions{Root: root}, refinement.PackagePublication{})
	require.Error(t, err)
	err = recordNormalizePublication(missionadapter.NormalizeOptions{Root: filepath.Join(t.TempDir(), "missing")}, refinement.PackagePublication{})
	require.Error(t, err)

	raw := []byte(`{"mission_id":"m-1","request_category":"general","selected_route":"full_pipeline","route_reason":"r","route_confidence":0.9,"evidence_state":"explicit","fallback_route":"full_pipeline"}`)
	rootFile := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.WriteFile(rootFile, []byte("not a directory"), 0o600))
	_, err = recordMissionRoute(t.Context(), rootFile, "m-1", raw)
	require.Error(t, err)
}

func TestRangerHandoffEvaluationSuccessAndOutputFailures(t *testing.T) {
	root := evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	basePath := filepath.Join(filepath.Dir(root), ".analysis")
	require.NoError(t, os.MkdirAll(filepath.Join(basePath, "pending"), 0o755))
	rangerFacts := "ranger_handoff_policy_facts:\n  schema_version: strategist-ranger-handoff-policy-facts/v1\n  require_recall: false\n  require_boundary: false\n  require_classification: false\n  require_verdict: false\n  informational_only: true\n"
	artifact := "---\nmission_id: m1\nmission_status: ranger_pending\nsources_consulted: []\n" + rangerFacts + "discovery_subtype: evaluation\nevaluation_verdict: partially_implemented\n---\n\n## mission_objective\nbody\n## known_facts\n- id: F-1\n  statement: fact\n## confidence_summary\nsummary\n## handoff\nhandoff\n"
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "pending", "m1-analysis.md"), []byte(artifact), 0o600))
	cmd := &cobra.Command{}
	var output bytes.Buffer
	cmd.SetOut(&output)
	require.NoError(t, runRangerHandoffEvaluate(cmd, rangerHandoffEvaluateOptions{Root: root, MissionID: "m1"}))
	assert.Contains(t, output.String(), "outcome:")

	failedOutput := &cobra.Command{}
	failedOutput.SetOut(failingWriter{})
	require.Error(t, printRangerHandoffEvaluation(failedOutput, missionruntime.RangerHandoffResult{
		Outcome: handoff.Outcome{Required: true},
		Result:  handoff.Result{Status: "passed"},
	}))
	plainFailedOutput := &cobra.Command{}
	plainFailedOutput.SetOut(failingWriter{})
	require.Error(t, printRangerHandoffEvaluation(plainFailedOutput, missionruntime.RangerHandoffResult{}))

	requiredRoot := evalMission(t, evalRequiredFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	requiredBase := filepath.Join(filepath.Dir(requiredRoot), ".analysis")
	require.NoError(t, os.MkdirAll(filepath.Join(requiredBase, "pending"), 0o755))
	requiredArtifact := strings.ReplaceAll(artifact, "require_recall: false", "require_recall: true")
	requiredArtifact = strings.ReplaceAll(requiredArtifact, "require_boundary: false", "require_boundary: true")
	requiredArtifact = strings.ReplaceAll(requiredArtifact, "require_classification: false", "require_classification: true")
	requiredArtifact = strings.ReplaceAll(requiredArtifact, "require_verdict: false", "require_verdict: true")
	requiredArtifact = strings.ReplaceAll(requiredArtifact, "informational_only: true", "informational_only: false")
	require.NoError(t, os.WriteFile(filepath.Join(requiredBase, "pending", "m1-analysis.md"), []byte(requiredArtifact), 0o600))
	require.Error(t, runRangerHandoffEvaluate(&cobra.Command{}, rangerHandoffEvaluateOptions{Root: requiredRoot, MissionID: "m1"}))
}

func TestSniperAuthorizationAndMaterializationFailures(t *testing.T) {
	root, basePath := sniperInvocationWorkspace(t)
	request, err := buildMissionInvocation(t.Context(), missionadapter.InvocationBuildInput{Root: root, BasePath: basePath, MissionID: flowMissionID, Role: "sniper", Slot: "execution"})
	require.NoError(t, err)

	canonical := "sniper: done | report_path: .analysis/archived/flow-mission-report.md | mission_status: documentation_applied"
	err = verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost, Root: root, BasePath: basePath, Completion: domain.MissionInvocationCompletion{Result: canonical}}, domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{MissionID: flowMissionID}})
	require.Error(t, err)

	record, err := missionruntime.NewInvocationStore(root).Get(request.RequestID)
	require.NoError(t, err)
	record.Request.Input = map[string]any{}
	require.Error(t, verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost, Root: root, BasePath: basePath, Completion: domain.MissionInvocationCompletion{Result: canonical}}, record))
	record.Request.Input["approval_gate_package_digest"] = "sha256:wrong"
	require.Error(t, verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost, Root: root, BasePath: basePath, Completion: domain.MissionInvocationCompletion{Result: canonical}}, record))

	outsideRoot := filepath.Join(t.TempDir(), ".strategist")
	err = verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost, Root: outsideRoot, BasePath: basePath, Completion: domain.MissionInvocationCompletion{Result: canonical}}, record)
	require.Error(t, err)
	missingMissionRoot := filepath.Join(t.TempDir(), ".strategist")
	err = verifySniperAuthorization(missionadapter.InvocationCompleteInput{Adapter: domain.ExecutionAdapterCurrentHost, Root: missingMissionRoot, BasePath: filepath.Join(filepath.Dir(missingMissionRoot), ".analysis"), Completion: domain.MissionInvocationCompletion{Result: canonical}}, record)
	require.Error(t, err)

	materializationRoot := t.TempDir()
	history := telemetry.SniperMaterializationHistoryPath(materializationRoot)
	require.NoError(t, os.MkdirAll(history, 0o755))
	require.Error(t, requireSniperMaterializations(materializationRoot, recordForSniper(flowMissionID), []string{"docs/guide.md"}))
}

func TestSniperCompletionFailureBranches(t *testing.T) {
	root, basePath := sniperInvocationWorkspace(t)
	request, err := buildMissionInvocation(t.Context(), missionadapter.InvocationBuildInput{Root: root, BasePath: basePath, MissionID: flowMissionID, Role: "sniper", Slot: "execution"})
	require.NoError(t, err)
	input := missionadapter.InvocationCompleteInput{Root: root, BasePath: basePath, RequestID: request.RequestID, Adapter: domain.ExecutionAdapterCurrentHost, Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: "bad"}}
	record, err := missionruntime.NewInvocationStore(root).Get(request.RequestID)
	require.NoError(t, err)
	_, err = completeSniperInvocation(missionruntime.NewInvocationStore(root), input, record)
	require.Error(t, err)

	refined := filepath.Join(basePath, "refined", flowMissionID)
	require.NoError(t, os.WriteFile(filepath.Join(refined, "analysis.md"), []byte("---\nmission_id: "+flowMissionID+"\nmission_status: documentation_applied\nclaimed_by: test\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(refined, "tasks.md"), []byte("- [x] 1.1 [documentation_target] Write `docs/guide.md`.\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(filepath.Dir(root), "docs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), "docs", "guide.md"), []byte("guide\n"), 0o600))
	require.NoError(t, telemetry.AppendSniperMaterialization(telemetry.SniperMaterializationHistoryPath(root), telemetry.SniperMaterializationRecord{MissionID: flowMissionID, BasePath: basePath, TargetPath: "docs/guide.md", MaterializedAt: time.Now().UTC()}))
	input.Completion.Result = "sniper: done | report_path: .analysis/archived/flow-mission-report.md | mission_status: documentation_applied"
	_, err = completeSniperInvocation(missionruntime.NewInvocationStore(root), input, record)
	require.Error(t, err)

	require.NoError(t, os.MkdirAll(filepath.Join(basePath, "archived"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "archived", flowMissionID+"-report.md"), []byte("report\n"), 0o600))
	store := missionruntime.NewInvocationStore(root)
	require.NoError(t, store.BeginProcessing(request.RequestID, "other.md", "sha256:other"))
	record, err = store.Get(request.RequestID)
	require.NoError(t, err)
	_, err = completeSniperInvocation(store, input, record)
	require.Error(t, err)
}

func TestMechanismsAndCompletionValidationFailures(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(failingWriter{})
	_, root := workspaceWithRoot(t)
	require.Error(t, runMechanismsBriefAtStage(cmd, root, "ranger", "", false))
	_, err := renderMechanismsBrief(mechanisms.Registry{}, "ranger", "future", false)
	require.Error(t, err)
	for _, raw := range [][]byte{
		[]byte{},
		[]byte(`{"request_id":"a","request_id":"b"}`),
		[]byte(`{"request_id":}`),
		[]byte(`{"request_id":"a"`),
		[]byte(`{"request_id":"a"]`),
	} {
		require.Error(t, validateCompletionObject(raw))
	}
	completionCmd := &cobra.Command{}
	completionCmd.SetIn(bytes.NewBufferString(`{"request_id":7}`))
	_, err = readMissionCompletion(completionCmd)
	require.Error(t, err)
}

func TestDirectJSONAndAuthorizationParserFailures(t *testing.T) {
	decoder := json.NewDecoder(strings.NewReader("[1]"))
	_, err := readCompletionField(decoder, map[string]bool{})
	require.Error(t, err)
	endDecoder := json.NewDecoder(strings.NewReader("[1]"))
	_, tokenErr := endDecoder.Token()
	require.NoError(t, tokenErr)
	require.Error(t, validateCompletionEnd(endDecoder))

	root, basePath := sniperInvocationWorkspace(t)
	tasks := filepath.Join(basePath, "refined", flowMissionID, "tasks.md")
	require.NoError(t, os.WriteFile(tasks, []byte("- [ ] 1.1 [documentation_target] Write the guide\n"), 0o600))
	require.Error(t, authorizeSniperEntry(missionadapter.InvocationBuildInput{Root: root, BasePath: basePath, MissionID: flowMissionID, Role: "sniper", Slot: "execution"}))
	require.Error(t, requireSniperGateDigest(missionadapter.InvocationBuildInput{Root: root, BasePath: filepath.Join(t.TempDir(), "missing"), MissionID: flowMissionID, Role: "sniper", Slot: "execution"}))
	statusRaw, err := json.Marshal(domain.MissionEngineStatus{MissionID: flowMissionID, Phase: domain.PhaseExecution, State: domain.StateExecution, ApprovalGatePackageDigest: "sha256:wrong"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", flowMissionID+".json"), statusRaw, 0o600))
	require.NoError(t, os.WriteFile(tasks, []byte("- [ ] 1.1 [documentation_target] Write `docs/guide.md`\n"), 0o600))
	require.Error(t, authorizeSniperEntry(missionadapter.InvocationBuildInput{Root: root, BasePath: basePath, MissionID: flowMissionID, Role: "sniper", Slot: "execution"}))

	providerRoot := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.WriteFile(providerRoot, []byte("file"), 0o600))
	require.Error(t, runProviderAdd(&cobra.Command{}, []string{"source"}, providerRoot, "discovery", providerOutputOptions{}))
	routeRoot := filepath.Join(t.TempDir(), "root")
	require.NoError(t, os.WriteFile(routeRoot, []byte("file"), 0o600))
	err = runRangerHandoffEvaluate(&cobra.Command{}, rangerHandoffEvaluateOptions{Root: routeRoot, MissionID: "m-1"})
	require.Error(t, err)
}

func TestAdditionalCommandBoundaryFailures(t *testing.T) {
	_, err := createCodexOutputFile(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
	require.Error(t, runProviderAdd(&cobra.Command{}, []string{"source"}, filepath.Join(t.TempDir(), "missing"), "discovery", providerOutputOptions{}))

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("slots: {}\n"), 0o600))
	_, _, _, err = loadMissionInvocationState(root)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("{}\n"), 0o600))
	_, _, _, err = loadMissionInvocationState(root)
	require.Error(t, err)
}
