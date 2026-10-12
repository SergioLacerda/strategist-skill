package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/dojo/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
	"github.com/SergioLacerda/strategist-skill/internal/integration/probe"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	providerpkg "github.com/SergioLacerda/strategist-skill/internal/provider"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoverageRangerBudgetHelpers(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		decision leveling.DelegationDecision
		want     leveling.BudgetEventKind
	}{
		{leveling.DelegationDecision{Granted: true}, leveling.BudgetEventGrant},
		{leveling.DelegationDecision{Reason: leveling.ReasonLevelingBudgetExhausted}, leveling.BudgetEventExhaustion},
		{leveling.DelegationDecision{Reason: "other"}, leveling.BudgetEventDenial},
	} {
		assert.Equal(t, tc.want, rangerBudgetEvent(tc.decision))
	}
	budget := domain.EffortBudget{}
	normalizeRangerBudget(&budget, "mission-1")
	assert.Equal(t, domain.EffortBudget{MissionID: "mission-1", RunID: "mission-1", Role: "ranger"}, budget)
	assert.Equal(t, "unavailable", providerEvidence(handoffconsumer.Report{}))
	assert.Equal(t, "unverified", providerEvidence(handoffconsumer.Report{Attempted: true}))
	report := rangerBudgetUnavailableReport()
	assert.Equal(t, handoffconsumer.StatusFallback, report.Status)
	assert.Equal(t, leveling.ReasonLevelingBudgetUnavailable, report.BudgetReason)
}

func TestCoverageDelegationPresentationBranches(t *testing.T) {
	t.Parallel()

	approved := handoffconsumer.Report{
		Status: handoffconsumer.StatusApproved, Confidence: .91, Threshold: .8,
		Delegation:  &handoff.Delegation{Provider: "jev", Model: "jev-1"},
		BudgetState: "granted", BudgetReason: "bounded",
	}
	signal := handoffconsumer.Report{Status: handoffconsumer.StatusSignal, Confidence: .4, Threshold: .8, Hints: []handoffconsumer.Hint{{Criterion: "scope"}, {Criterion: "tests"}}}
	fallback := handoffconsumer.Report{Status: handoffconsumer.StatusFallback, Reason: integration.StateUnavailable}
	assert.Contains(t, delegationDetails(approved), "provider=jev model=jev-1")
	assert.Contains(t, delegationDetails(signal), "hints=scope,tests")
	assert.Contains(t, delegationDetails(fallback), "reason=unavailable")
	assert.Equal(t, "scope,tests", hintList(signal.Hints))

	var line strings.Builder
	approved.Decision.Suspect = true
	approved.Reused = true
	appendDelegationMetadata(&line, approved)
	assert.Contains(t, line.String(), "integrity=suspect")
	assert.Contains(t, line.String(), "reused=true")
	assert.Contains(t, line.String(), "budget=granted")

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	printDelegation(cmd, approved)
	printDelegation(cmd, signal)
	printDelegation(func() *cobra.Command { c := &cobra.Command{}; c.SetOut(failingWriter{}); return c }(), signal)
	before := out.Len()
	printDelegation(cmd, handoffconsumer.Report{Status: handoffconsumer.StatusDisabled})
	assert.Greater(t, out.Len(), before-1)
	assert.NotNil(t, commandContext(cmd))
	cmd.SetContext(context.Background())
	assert.NotNil(t, commandContext(cmd))
	assert.NotNil(t, commandContext(&cobra.Command{}))
}

func TestCoverageIntegrationsAndDoctorHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "second", firstNonEmpty("", "second", "third"))
	assert.Empty(t, firstNonEmpty("", ""))
	workspace := t.TempDir()
	path, ok := dotenvPath("dotenv:.env#TOKEN", workspace)
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(workspace, ".env"), path)
	assert.False(t, func() bool { _, ok := dotenvPath("env:TOKEN", workspace); return ok }())
	absolute := filepath.Join(workspace, "secrets", ".env")
	path, ok = dotenvPath("dotenv:"+absolute+"#TOKEN", workspace)
	assert.True(t, ok)
	assert.Equal(t, absolute, path)

	report := doctorReport{Provider: "jev", Declared: true, Bound: true, Enabled: true, Compatible: true, Available: "yes", ObservedAt: "now", ValidUntil: "later", PinnedModel: "pinned", ResolvedModel: "resolved", Endpoint: "https://example.test", Credential: "pending", Calibration: "não calibrado", Findings: []string{"missing"}}
	lines := doctorLines(report)
	assert.Contains(t, strings.Join(lines, "\n"), "findings: missing")
	assert.Equal(t, "not_configured", func() string { state, _ := credentialState(workspace, config.Provider{}); return state }())
	declared := buildDoctorReport(workspace, config.File{}, false, probeRecordZero(), false, time.Now())
	assert.False(t, declared.Declared)

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, printDoctorJSON(cmd, report))
	assert.Contains(t, out.String(), `"provider": "jev"`)
	require.NoError(t, printLines(cmd, "one", "two"))
	assert.Error(t, printLines(func() *cobra.Command { c := &cobra.Command{}; c.SetOut(failingWriter{}); return c }(), "fail"))
}

func TestCoverageCalibrationHelpers(t *testing.T) {
	t.Parallel()

	assert.Empty(t, firstArg(nil))
	assert.Equal(t, "id", firstArg([]string{"id"}))
	provider := config.Provider{Model: "jev-1"}
	threshold := .8
	record := calibration.Record{ID: "cal-20261010-12345678", Status: calibration.StatusRecommended, Model: "other", Profile: calibration.ProfileV1, Threshold: &threshold}
	require.ErrorContains(t, pinnable(record, provider), "pinned model")
	record.Status = calibration.StatusNotFit
	require.ErrorContains(t, pinnable(record, provider), "carries no threshold")
	record.Status = calibration.StatusRecommended
	record.Model = provider.Model
	record.Profile = "other-profile"
	require.ErrorContains(t, pinnable(record, provider), "question profile")
	assert.False(t, profileCurrent(record, ""))
	record.ProfileVersion = handoffconsumer.ProfileVersionOf(calibration.ProfileV1, calibration.BoundaryPackage)
	assert.True(t, profileCurrent(record, ""))
}

func TestCoverageIntegrationCommandErrorsAndCalibrationBoundaries(t *testing.T) {
	t.Parallel()

	fileRoot := filepath.Join(t.TempDir(), "root-file")
	require.NoError(t, os.WriteFile(fileRoot, []byte("not a directory"), 0o600))
	_, err := integrationsRoot(integrationsOptions{Root: fileRoot})
	require.Error(t, err)
	_, _, err = loadIntegrations(fileRoot)
	require.Error(t, err)
	_, err = decide(fileRoot, setup.ChoiceEnable, integrationsOptions{})
	require.Error(t, err)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	require.Error(t, runIntegrationsEnable(cmd, integrationsOptions{Root: fileRoot}))
	require.Error(t, runIntegrationsDisable(cmd, integrationsOptions{Root: fileRoot}))
	require.Error(t, runCalibrationList(cmd, integrationsOptions{Root: fileRoot}))
	require.Error(t, runCalibrationShow(cmd, integrationsOptions{Root: fileRoot}, ""))
	require.Error(t, runCalibrationPin(cmd, calibrationOptions{integrationsOptions: integrationsOptions{Root: fileRoot}}, "missing"))
	require.Error(t, runCalibrationUnpin(cmd, integrationsOptions{Root: fileRoot}))

	root := t.TempDir()
	provider := config.Provider{Enabled: true, CredentialRef: "env:NOT_CONFIGURED"}
	state, findings := credentialState(filepath.Join(root, ".strategist"), provider)
	assert.Equal(t, "pending", state)
	assert.NotEmpty(t, findings)
	report := buildDoctorReport(filepath.Join(root, ".strategist"), config.File{Providers: map[string]config.Provider{integrationProvider: provider}}, true, probe.Record{}, false, time.Now())
	assert.True(t, report.Declared)
	assert.Equal(t, "pending", report.Credential)

	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("leveling: ["), 0o600))
	if _, err := rangerBudgetLedgerPath(root); err == nil {
		t.Fatal("malformed active leveling config should produce a warning")
	}
	assert.False(t, func() bool { _, ok := loadRangerBudget(root, "missing"); return ok }())
	_, granted := admitRangerBudget(context.Background(), root, "m")
	assert.False(t, granted)
	plainRoot := t.TempDir()
	_, granted = admitRangerBudget(context.Background(), plainRoot, "m")
	assert.False(t, granted)
	invalidBudget := rangerBudget("m", "m", 1)
	invalidBudget.Budget.MaxDepth = -1
	writeRangerBudget(t, plainRoot, invalidBudget)
	_, granted = admitRangerBudget(context.Background(), plainRoot, "m")
	assert.False(t, granted)
	finishRangerBudget(context.Background(), rangerBudgetAdmission{}, handoffconsumer.Report{})
	blocker := filepath.Join(root, "blocker")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))
	assert.False(t, appendRangerBudgetEvent(context.Background(), filepath.Join(blocker, "ledger"), unavailableRangerBudget("m"), leveling.BudgetEventDenial, "reason", "unavailable", 0, 0))
	_, err = probeTarget(plainRoot)
	require.Error(t, err)
	probeConfig := filepath.Join(plainRoot, config.FileName)
	require.NoError(t, config.Save(probeConfig, config.File{Providers: map[string]config.Provider{integrationProvider: {}}}))
	_, err = probeTarget(plainRoot)
	require.Error(t, err)
	disabled := config.Provider{Endpoint: "https://example.test", Model: "jev-1"}
	require.NoError(t, config.Save(probeConfig, config.File{Providers: map[string]config.Provider{integrationProvider: disabled}}))
	_, err = probeTarget(plainRoot)
	require.Error(t, err)
}

func TestCoverageCalibrationLinesAndDoctorCredentialForms(t *testing.T) {
	t.Parallel()

	threshold := .8
	record := calibration.Record{ID: "cal-20261010-12345678", Status: calibration.StatusRecommended, Provider: "jev", Model: "jev-1", Profile: calibration.ProfileV1, ProfileVersion: "profile-version", Threshold: &threshold, CreatedAt: "2026-10-10T00:00:00Z", CorpusDigest: "corpus", CatalogVersion: "catalog", Gates: []calibration.Gate{{Name: "gate", Status: "passed", Value: .1, Limit: .2}}, Notes: []string{"note"}, FailingGate: "gate"}
	assert.Greater(t, len(calibrationLines(record)), 5)
	record.HeldOut = &calibration.SetResult{Defects: 2, Positives: 3, FalseApproval: calibration.Rate{Value: .1, Low: .05, High: .2}, FalseRejection: calibration.Rate{Value: .2}}
	assert.Contains(t, strings.Join(calibrationLines(record), "\n"), "recommended threshold")
	path, ok := dotenvPath("dotenv:.env#TOKEN", t.TempDir())
	assert.True(t, ok)
	assert.NotEmpty(t, path)
}

func TestCoverageHostBridgeAndInvocationContracts(t *testing.T) {
	t.Parallel()

	args := codexExecArgs("/tmp/state", "/tmp/state/result.md", "/workspace")
	assert.Equal(t, "exec", args[0])
	assert.Equal(t, "-", args[len(args)-1])
	assert.Equal(t, "/custom/codex", codexHomeDir([]string{"CODEX_HOME=/custom/codex"}))
	assert.Contains(t, codexBridgeEnv([]string{"PATH=/bin", "CODEX_HOME=/old"}, "/tmp/state"), "CODEX_HOME=/tmp/state")
	require.NoError(t, linkCodexAuth([]string{"CODEX_HOME=" + t.TempDir()}, t.TempDir()))

	for _, tc := range []struct {
		role, slot, want string
	}{
		{"ranger", "discovery", "discovery"},
		{"archivist", string(domain.SlotRefinement), "archivist"},
		{"sniper", string(domain.SlotExecution), "sniper"},
		{"other", "other", ""},
	} {
		execution, output := invocationContracts(tc.role, tc.slot)
		if tc.want == "" {
			assert.Empty(t, execution)
			assert.Empty(t, output)
		} else {
			assert.NotEmpty(t, execution)
			assert.NotEmpty(t, output)
		}
	}
	assert.Equal(t, "corr", criticalHitStageCorrelation(domain.MissionEngineStatus{Stage: domain.StageShort, StageFeat: "critical_hit", StageCorrelationID: "corr"}))
	assert.Empty(t, criticalHitStageCorrelation(domain.MissionEngineStatus{}))
	id, err := newInvocationID()
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestCoverageProviderOutputRejectedFormats(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.SetOut(failingWriter{})
	require.Error(t, printProviderJSON(cmd, map[string]string{"x": "y"}))
	require.Error(t, printProviderYAML(cmd, map[string]string{"x": "y"}))
	require.Error(t, printProviderHuman(cmd, "unsupported"))
	require.Error(t, printProviderOutput(cmd, "unsupported", "xml"))
	require.Error(t, runProviderAdd(cmd, []string{"missing"}, t.TempDir(), "", providerOutputOptions{}))
	var reportOut bytes.Buffer
	require.NoError(t, printProviderReport(&reportOut, providerpkg.Report{Validated: true}))
	assert.Error(t, printProviderReport(failingWriter{}, providerpkg.Report{Reasons: []providerpkg.Reason{{Code: "bad", Detail: "reason"}}}))
}

type failAfterFirstWriter struct{ writes int }

func (w *failAfterFirstWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > 1 {
		return 0, os.ErrClosed
	}
	return len(p), nil
}

type yamlErrorValue struct{}

func (yamlErrorValue) MarshalYAML() (any, error) { return nil, errors.New("yaml test error") }

func TestCoverageAdditionalProviderAndCompletionBranches(t *testing.T) {
	// These branches are intentionally exercised without changing process-wide
	// command state, so this test may run alongside the package's parallel tests.
	t.Parallel()

	var yamlCmd cobra.Command
	yamlCmd.SetOut(&bytes.Buffer{})
	require.Error(t, printProviderYAML(&yamlCmd, yamlErrorValue{}))
	failingAfterFirst := &failAfterFirstWriter{}
	require.Error(t, printProviderReport(failingAfterFirst, providerpkg.Report{Reasons: []providerpkg.Reason{{Code: "bad", Detail: "reason"}}}))

	assert.Equal(t, filepath.Join(os.Getenv("HOME"), ".codex"), codexHomeDir(nil))
	_, err := loadCompletableInvocation(missionruntime.InvocationStore{}, missionadapter.InvocationCompleteInput{RequestID: "missing-request"})
	require.Error(t, err)

	criticalRoot := filepath.Join(t.TempDir(), ".strategist")
	criticalBase := filepath.Join(filepath.Dir(criticalRoot), ".analysis")
	criticalInput := missionadapter.InvocationCompleteInput{
		Root: criticalRoot, BasePath: criticalBase,
		Completion: domain.MissionInvocationCompletion{Result: "sniper: done | report_path: .analysis/archived/m-report.md | mission_status: documentation_applied"},
	}
	criticalRecord := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{MissionID: "m"}}
	_, _, err = verifyCriticalHitCompletion(criticalInput, criticalRecord)
	require.ErrorContains(t, err, "report is missing")

	rangerRecord := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{MissionID: "m", Role: "ranger", Slot: string(domain.SlotDiscovery)}}
	require.Error(t, recordRangerHandoffAfterPublish(context.Background(), missionadapter.InvocationCompleteInput{Root: criticalRoot, BasePath: criticalBase}, rangerRecord))

	_, _, err = resolveActiveBasePathForMission(filepath.Join(t.TempDir(), "missing-root"))
	require.Error(t, err)
	_, err = adrCanonicalPath(filepath.Join(t.TempDir(), "missing-root"))
	require.Error(t, err)

	badTasks := filepath.Join(t.TempDir(), "tasks.md")
	require.NoError(t, os.WriteFile(badTasks, []byte("not valid frontmatter: [\n"), 0o600))
	require.Error(t, requireSniperTargets(badTasks))
	require.Error(t, requireSniperTargets(filepath.Join(t.TempDir(), "missing-tasks.md")))
	_, _, err = readMissionInvocationPayload(domain.RoleWeaponBinding{ConnectorID: "strategist-native-role", WeaponID: "missing-role"})
	require.Error(t, err)
	_, _, err = readMissionInvocationPayload(domain.RoleWeaponBinding{WeaponID: "missing-weapon", WeaponVersion: "0.0.0"})
	require.Error(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = buildMissionInvocation(ctx, missionadapter.InvocationBuildInput{})
	require.ErrorContains(t, err, "context")

	brokenRoot := filepath.Join(t.TempDir(), ".strategist")
	assert.NoError(t, os.MkdirAll(brokenRoot, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(brokenRoot, "roles"), []byte("not a directory"), 0o600))
	_, warning := loadRoleRegistry(brokenRoot)
	assert.NotEmpty(t, warning)
}

func TestCoverageRunbookAndCalibrationResidualBranches(t *testing.T) {
	t.Parallel()

	root := bareRoot(t)
	configPath := filepath.Join(root, config.FileName)
	validProvider := config.Provider{Enabled: true, Endpoint: "https://example.test", CredentialRef: "env:KEY", Model: "jev-1", AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate}, DataPolicy: config.DataPolicy{AllowedFields: handoffconsumer.Fields(handoff.TransitionRangerToArchivist), MaxStateBytes: 16384}}
	require.NoError(t, config.Save(configPath, config.File{Providers: map[string]config.Provider{integrationProvider: validProvider}}))
	require.Error(t, prepareRunbookOpportunity(root, filepath.Join(filepath.Dir(root), ".analysis"), "m-1", "sha256:missing", "OA-RUNBOOK-m-1"))

	basePath := filepath.Join(filepath.Dir(root), ".analysis")
	calibrationDir := filepath.Join(basePath, "dojo", "calibrations")
	require.NoError(t, os.MkdirAll(calibrationDir, 0o755))
	badID := "cal-20261010-12345678"
	require.NoError(t, os.WriteFile(filepath.Join(calibrationDir, badID+".json"), []byte("not-json"), 0o600))
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	assert.NoError(t, runCalibrationList(cmd, integrationsOptions{Root: root}))
}

func TestCoverageCommandAuthorizationAndListingErrors(t *testing.T) {
	require.Error(t, authorizeCriticalHitEntry(domain.MissionEngineStatus{State: domain.StateExecution}))
	require.Error(t, authorizeCriticalHitEntry(domain.MissionEngineStatus{State: domain.StateExecution, Stage: domain.StageShort, StageFeat: "critical_hit", StageGateApproved: true, ApprovalGatePackageDigest: "stale"}))
	require.Error(t, requireSniperGateDigest(missionadapter.InvocationBuildInput{Root: t.TempDir(), BasePath: t.TempDir(), MissionID: "m"}))
	_, err := listMissionInvocationRequests("", "m")
	require.Error(t, err)

	root := bareRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, config.FileName), []byte("providers: ["), 0o600))
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	require.Error(t, runCalibrationUnpin(cmd, integrationsOptions{Root: root}))
	require.Error(t, runCalibrationShow(cmd, integrationsOptions{Root: root}, ""))

	probeConfig := filepath.Join(root, config.FileName)
	provider := config.Provider{Enabled: true, Endpoint: "https://example.test", CredentialRef: "env:PROBE_KEY", Model: "jev-1", AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate}, DataPolicy: config.DataPolicy{AllowedFields: handoffconsumer.Fields(handoff.TransitionRangerToArchivist), MaxStateBytes: 16384}}
	require.NoError(t, config.Save(probeConfig, config.File{Providers: map[string]config.Provider{integrationProvider: provider}}))
	t.Setenv("PROBE_KEY", "secret")
	require.Error(t, runIntegrationsProbe(cmd, integrationsOptions{Root: root}))
	probeRootFile := filepath.Join(t.TempDir(), "root-file")
	require.NoError(t, os.WriteFile(probeRootFile, []byte("not a directory"), 0o600))
	_, err = probeTarget(probeRootFile)
	require.Error(t, err)
	provider.Endpoint = "https://127.0.0.1:1"
	require.NoError(t, config.Save(probeConfig, config.File{Providers: map[string]config.Provider{integrationProvider: provider}}))
	require.NoError(t, os.RemoveAll(filepath.Join(root, "memory")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory"), []byte("blocked"), 0o600))
	assert.Error(t, runIntegrationsProbe(cmd, integrationsOptions{Root: root}))
}

func TestCoverageAdditionalInvocationAndProviderWrappers(t *testing.T) {
	require.Error(t, providerValidateCmd.RunE(providerValidateCmd, []string{filepath.Join(t.TempDir(), "missing-provider")}))
	require.Error(t, providerAddCmd.RunE(providerAddCmd, []string{"missing-provider"}))

	_, _, err := newMissionInvocationRequest(context.Background(), missionadapter.InvocationBuildInput{Root: t.TempDir(), MissionID: "missing", Role: "sniper", Slot: string(domain.SlotExecution)}, domain.RoleWeaponBinding{}, domain.CompiledWeapon{}, nil, "", nil)
	require.Error(t, err)

	root, basePath := sniperInvocationWorkspace(t)
	request, err := buildMissionInvocation(context.Background(), missionadapter.InvocationBuildInput{Root: root, BasePath: basePath, MissionID: flowMissionID, Role: "sniper", Slot: string(domain.SlotExecution)})
	require.NoError(t, err)
	record, err := missionruntime.NewInvocationStore(root).Get(request.RequestID)
	require.NoError(t, err)
	_, _, err = verifySniperCompletion(missionadapter.InvocationCompleteInput{Root: root, BasePath: basePath, Completion: domain.MissionInvocationCompletion{Result: "sniper: done | report_path: .analysis/archived/flow-mission-report.md | mission_status: documentation_applied"}}, record)
	require.Error(t, err)

	status := domain.MissionEngineStatus{MissionID: flowMissionID, Phase: domain.PhaseExecution, State: domain.StateExecution}
	require.NoError(t, os.WriteFile(filepath.Join(root, "missions", flowMissionID+".json"), mustJSON(status), 0o600))
	record.Request.BindingDigest = "sha256:different"
	assert.ErrorContains(t, verifyInvocationStillCurrent(root, record), "binding changed")
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func TestCoverageLevelingAndCodexFailureBranches(t *testing.T) {
	t.Parallel()

	emitRoleLevel(context.Background(), "m", "run", leveling.Level{Role: "ranger", Provider: "jev", Capability: "handoff", PolicyVersion: 1, PolicyDigest: "digest"}, "reason")
	_, warning := readActiveLevelingConfig(filepath.Join(t.TempDir(), "root"))
	assert.Empty(t, warning)
	_, err := readEmbeddedLevelingDefaults()
	require.NoError(t, err)
	_, err = runCodexBinary(context.Background(), "missing-codex-binary", t.TempDir(), "prompt", 100)
	require.Error(t, err)
	err = runCalibrationShow(&cobra.Command{}, integrationsOptions{Root: bareRoot(t)}, "missing")
	require.Error(t, err)
}

func TestCoverageIntegrationPersistenceAndDoctorErrors(t *testing.T) {
	t.Parallel()

	root := bareRoot(t)
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	require.Error(t, func() error {
		_, err := decide(root, setup.ChoiceEnable, integrationsOptions{Endpoint: "not-an-https-endpoint"})
		return err
	}())
	badConfig := filepath.Join(root, config.FileName)
	require.NoError(t, os.WriteFile(badConfig, []byte("providers: ["), 0o600))
	require.Error(t, runIntegrationsDoctor(cmd, integrationsOptions{Root: root}))

	require.NoError(t, os.Remove(badConfig))
	basePath := filepath.Join(filepath.Dir(root), ".analysis")
	require.NoError(t, os.MkdirAll(filepath.Join(basePath, "dojo"), 0o755))
	calibrationsPath := filepath.Join(basePath, "dojo", "calibrations")
	require.NoError(t, os.WriteFile(calibrationsPath, nil, 0o600))
	require.Error(t, runCalibrationList(cmd, integrationsOptions{Root: root}))
	require.Error(t, runCalibrationShow(cmd, integrationsOptions{Root: root}, "missing"))
	require.NoError(t, os.MkdirAll(filepath.Dir(probePath(root)), 0o755))
	require.NoError(t, os.WriteFile(probePath(root), []byte("not-json"), 0o600))
	assert.Error(t, runIntegrationsDoctor(cmd, integrationsOptions{Root: root}))
}

func TestCoverageRegisteredIntegrationCommandWrappers(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root-file")
	require.NoError(t, os.WriteFile(root, []byte("not a directory"), 0o600))
	for _, tc := range []struct {
		parent *cobra.Command
		args   []string
	}{
		{integrationsCalibrationCmd, []string{"list"}},
		{integrationsCalibrationCmd, []string{"show"}},
		{integrationsCalibrationCmd, []string{"pin", "missing"}},
		{integrationsCalibrationCmd, []string{"unpin"}},
		{integrationsCmd, []string{"doctor"}},
		{integrationsCmd, []string{"probe"}},
		{integrationsCmd, []string{"enable"}},
		{integrationsCmd, []string{"disable"}},
	} {
		command, _, err := tc.parent.Find(tc.args)
		require.NoError(t, err)
		require.NotNil(t, command)
		if flag := command.Flags().Lookup("root"); flag != nil {
			require.NoError(t, flag.Value.Set(root))
		}
		callArgs := tc.args[1:]
		if tc.parent == integrationsCmd {
			callArgs = tc.args[1:]
		}
		require.Error(t, command.RunE(command, callArgs))
	}
	resolver := calibrationResolver(root)
	_, err := resolver("missing")
	assert.Error(t, err)
}

func TestCoverageUnreadableAndContextFallbacks(t *testing.T) {
	t.Parallel()

	report := unreadableReport()
	assert.Equal(t, handoffconsumer.StatusDisabled, report.Status)
	assert.Equal(t, integration.StateDisabled, report.Reason)
	root := t.TempDir()
	assert.Equal(t, handoffconsumer.StatusDisabled, rangerArtifactPrecheck(context.Background(), root, root, "missing").Status)
	assert.Equal(t, handoffconsumer.StatusDisabled, archivistPrecheck(context.Background(), root, root, "missing").Status)
	assert.Equal(t, handoffconsumer.StatusDisabled, gatePrecheck(context.Background(), root, root, "missing").Status)
	assert.Nil(t, rangerDelegate(context.Background(), root, root, "missing")("ignored"))
	assert.NoFileExists(t, filepath.Join(root, "memory", "integration-calls.jsonl"))
	badPending := filepath.Join(root, "pending")
	require.NoError(t, os.MkdirAll(badPending, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(badPending, "bad-analysis.md"), []byte("---\n: [\n---\nbody"), 0o600))
	assert.Equal(t, handoffconsumer.StatusDisabled, rangerArtifactPrecheck(context.Background(), root, root, "bad").Status)
}

func TestCoverageRunbookAndIntegrationConstructionFallbacks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := runbookCanonicalPath(root); err == nil {
		t.Fatal("missing active config should make the canonical Runbook path unavailable")
	}
	if err := prepareRunbookOpportunity(root, t.TempDir(), "m-1", "sha256:missing", "OA-RUNBOOK-m-1"); err == nil {
		t.Fatal("missing refined package should reject Runbook preparation")
	}
	consumer, err := handoffConsumer(root)
	require.NoError(t, err)
	assert.Nil(t, consumer)
	runbook, err := runbookConsumer(root)
	require.NoError(t, err)
	assert.Nil(t, runbook)
	assert.Equal(t, handoffconsumer.StatusDisabled, precheckHandoff(context.Background(), root, "m-1", handoff.TransitionRangerToArchivist, "subject", nil).Status)

	badConfig := filepath.Join(root, config.FileName)
	require.NoError(t, os.WriteFile(badConfig, []byte("providers: ["), 0o600))
	if _, err := handoffConsumer(root); err == nil {
		t.Fatal("malformed integration config should be reported")
	}
	if _, err := runbookConsumer(root); err == nil {
		t.Fatal("malformed Runbook integration config should be reported")
	}
	if err := prepareRunbookOpportunity(root, t.TempDir(), "m-1", "sha256:missing", "OA-RUNBOOK-m-1"); err == nil {
		t.Fatal("Runbook preparation should reject the malformed integration setup")
	}
	provider := config.Provider{Endpoint: "://invalid", Model: "jev-1"}
	if _, _, err := newJEVAdapter(root, provider); err == nil {
		t.Fatal("invalid provider endpoint should be rejected")
	}
	validProvider := config.Provider{Enabled: true, Endpoint: "https://example.test", CredentialRef: "env:KEY", Model: "jev-1", AllowedCapabilities: []integration.Capability{integration.CapHandoffValidate}, DataPolicy: config.DataPolicy{AllowedFields: handoffconsumer.Fields(handoff.TransitionRangerToArchivist), MaxStateBytes: 16384}}
	require.NoError(t, config.Save(badConfig, config.File{Providers: map[string]config.Provider{integrationProvider: validProvider}}))
	handoff, err := handoffConsumer(root)
	require.NoError(t, err)
	assert.NotNil(t, handoff)
	runbook, err = runbookConsumer(root)
	require.NoError(t, err)
	assert.NotNil(t, runbook)
	recordProviderConfidence(root, handoffconsumer.Request{}, handoffconsumer.Report{})
}

func TestCoverageMissionCompletionBoundaries(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "docs/file.md", func() string {
		got, _ := workspaceRelative("/workspace/.strategist", "/workspace/docs/file.md")
		return got
	}())
	_, err := workspaceRelative("/workspace/.strategist", "/outside/file.md")
	require.ErrorContains(t, err, "escapes workspace")

	input := missionadapter.InvocationCompleteInput{}
	record := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{Role: "unknown", Slot: "unknown"}}
	_, err = dispatchCompletion(context.Background(), missionruntime.InvocationStore{}, input, record)
	require.ErrorContains(t, err, "completion normalization is not registered")
	_, err = completeDiscoveryInvocation(context.Background(), missionruntime.InvocationStore{}, input, record)
	require.ErrorContains(t, err, "telemetry sink")
	assert.NoError(t, recordRangerHandoffAfterPublish(context.Background(), input, record))

	assert.NoError(t, verifySniperCompletionSignal("Return: sniper: done | report_path: docs/report.md | mission_status: documentation_applied", "docs/report.md"))
	require.Error(t, verifySniperCompletionSignal("wrong", "docs/report.md"))
	assert.True(t, documentationTargetCompleted([]byte("- [x] write `docs/report.md`"), "docs/report.md"))
	assert.False(t, documentationTargetCompleted([]byte("- [ ] write `docs/report.md`"), "docs/report.md"))
	require.Error(t, requireCompletedDocumentationTargets(filepath.Join(t.TempDir(), "missing"), []string{"docs/report.md"}))
	criticalInput := missionadapter.InvocationCompleteInput{Root: t.TempDir(), BasePath: t.TempDir(), Completion: domain.MissionInvocationCompletion{Result: "Return: sniper: done | report_path: missing/archived/m-report.md | mission_status: documentation_applied"}}
	criticalRecord := domain.MissionInvocationRecord{Request: domain.MissionInvocationRequest{MissionID: "m", Role: "sniper", Slot: string(domain.SlotExecution)}}
	_, _, err = verifyCriticalHitCompletion(criticalInput, criticalRecord)
	require.Error(t, err)
	assert.False(t, isCriticalHitInvocation(t.TempDir(), "m"))
	assert.NoError(t, completeCriticalHitMission(t.TempDir(), "m"))
	assert.NoError(t, completeCriticalHitMissionLocked(t.TempDir(), nil, domain.MissionEngineStatus{State: domain.StateDoneDelivery}))
	require.Error(t, completeCriticalHitMissionLocked(t.TempDir(), nil, domain.MissionEngineStatus{State: domain.StateExecution}))

	ports := archivistPorts(t.TempDir(), "m")
	require.Error(t, ports.RecordConfidence(domain.ConfidenceClaim{}, nil))
	require.Error(t, ports.RecordPublication(application.NormalizeOpenSpecPublication{}))
	_, err = completeArchivistInvocation(missionruntime.InvocationStore{}, input, criticalRecord)
	require.Error(t, err)
	assert.Error(t, verifySniperAuthorization(input, criticalRecord))
}

func probeRecordZero() probe.Record { return probe.Record{} }
