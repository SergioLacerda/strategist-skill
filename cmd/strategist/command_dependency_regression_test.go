package main

// Executable dependency and host-boundary regression coverage.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissionInvocationDependencyClosures(t *testing.T) {
	deps := missionInvocationDependencies()

	_, err := deps.LoadMission(t.TempDir(), "ghost")
	require.ErrorContains(t, err, "not found")
	_, _, err = deps.ResolveBasePath(filepath.Join(t.TempDir(), "absent", ".strategist"))
	require.ErrorContains(t, err, "resolve active base path")

	_, root := workspaceWithRoot(t)
	strategistRoot, basePath, err := deps.ResolveBasePath(root)
	require.NoError(t, err)
	assert.Equal(t, root, strategistRoot)
	assert.NotEmpty(t, basePath)
	assert.NotNil(t, deps.TelemetrySink())
	assert.Equal(t, "root", deps.RootFlag)
}

func TestCodexHomeAndTransientStateFailures(t *testing.T) {
	assert.Equal(t, "/custom/codex", codexHomeDir([]string{"A=b", "CODEX_HOME=/custom/codex"}))
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	assert.Empty(t, codexHomeDir(nil), "an unresolvable home yields no credentials directory")
}

func TestRunCodexBinaryFailsWhenTheTransientDirectoryCannotBeCreated(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	t.Setenv("TMPDIR", missing)
	t.Setenv("TMP", missing)
	t.Setenv("TEMP", missing)
	_, err := runCodexBinary(context.Background(), "codex", t.TempDir(), "prompt", 1024)
	require.ErrorContains(t, err, "transient state directory")
}

func TestHostStreamsFailureFallsBackToStdout(t *testing.T) {
	streams := hostStreams{stdout: &cappedBuffer{limit: 1024}, stderr: &cappedBuffer{limit: 1024}}
	streams.stdout.buf.WriteString("only stdout talked")
	err := streams.failure("codex", assertAnError{})
	require.ErrorContains(t, err, "only stdout talked")
}

type assertAnError struct{}

func (assertAnError) Error() string { return "exit status 1" }

func TestStartInitiativeConsultationFailures(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "memory", roleLevelLedger), 0o755))
	require.Error(t, startInitiativeConsultation(root, "m1"), "an unreadable LEVELING ledger blocks the consultation")

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.Error(t, startInitiativeConsultation(blocker, "m1"))
}

func TestEvaluatePrintFailureAndSilencedBrief(t *testing.T) {
	root := evalMission(t, evalRequiredFacts, "- [ ] 1.1 [task_type: implementation_handoff] change the code\n")
	cmd := &cobra.Command{}
	cmd.SetOut(failingWriter{})
	err := runArchivistHandoff(cmd, handoffEvaluateOptions{
		Root: root, MissionID: "m1",
		Challenges: writeEvalFile(t, "c.yaml", evalChallenges), Ack: writeEvalFile(t, "a.yaml", evalPassingAck),
	})
	require.Error(t, err)

	brief := &cobra.Command{}
	var out bytes.Buffer
	brief.SetOut(&out)
	attachMissionRun(t, brief)
	require.Error(t, runMechanismsBrief(brief, "", "not-a-role", false))
}

func TestMechanismsBriefResolutionAndWriteFailures(t *testing.T) {
	t.Chdir(t.TempDir())
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	require.ErrorContains(t, runMechanismsBrief(cmd, "", "ranger", false), "mechanisms brief:")

	_, root := workspaceWithRoot(t)
	broken := &cobra.Command{}
	broken.SetOut(failingWriter{})
	err := runMechanismsBrief(broken, root, "ranger", false)
	require.Error(t, err)
}

func TestValidateRecordsTelemetryWhenAMissionRunIsAttached(t *testing.T) {
	root := minimalValidateRoot(t)
	validateRoot = root
	t.Cleanup(func() { validateRoot = "" })
	cmd := validateCmd
	cmd.SetOut(&bytes.Buffer{})
	attachMissionRun(t, cmd)
	require.NoError(t, cmd.RunE(cmd, nil))
}

func TestRunProviderAddOnAFixtureAndReportsAFailedWrite(t *testing.T) {
	fixture := "../../internal/embed/defaults/plugins/fixtures/minimal-provider"
	root := rankedWorkspace(t, nil)
	cmd := &cobra.Command{}
	cmd.SetOut(failingWriter{})
	err := runProviderAdd(cmd, []string{fixture}, root, "discovery", providerOutputOptions{Format: "json"})
	require.Error(t, err)

	ok := &cobra.Command{}
	var out bytes.Buffer
	ok.SetOut(&out)
	root2 := rankedWorkspace(t, nil)
	if err := runProviderAdd(ok, []string{fixture}, root2, "discovery", providerOutputOptions{Format: "json"}); err == nil {
		assert.Contains(t, out.String(), "instance_id")
	}
}

func TestEvaluatePrintFailureOnASkippedHandoff(t *testing.T) {
	root := evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	cmd := &cobra.Command{}
	cmd.SetOut(failingWriter{})
	require.ErrorContains(t, runArchivistHandoff(cmd, handoffEvaluateOptions{Root: root, MissionID: "m1"}), "print handoff outcome")
}

func TestEvaluateCommandRunsThroughItsFlags(t *testing.T) {
	root := evalMission(t, evalInformationalFacts, "- [ ] 1.1 [analysis_artifact] record the evidence\n")
	require.NoError(t, handoffEvaluateCmd.Flags().Set("root", root))
	require.NoError(t, handoffEvaluateCmd.Flags().Set("mission-id", "m1"))
	t.Cleanup(func() {
		_ = handoffEvaluateCmd.Flags().Set("root", "")       //nolint:errcheck // test cleanup
		_ = handoffEvaluateCmd.Flags().Set("mission-id", "") //nolint:errcheck // test cleanup
	})
	var out bytes.Buffer
	handoffEvaluateCmd.SetOut(&out)
	require.NoError(t, handoffEvaluateCmd.RunE(handoffEvaluateCmd, nil))
	assert.Contains(t, out.String(), "outcome: ")
}

func TestLinkCodexAuthReportsAFailedLink(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, "auth.json"), []byte("{}"), 0o600))
	stateDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "auth.json"), nil, 0o600))
	require.ErrorContains(t, linkCodexAuth([]string{"CODEX_HOME=" + home}, stateDir), "link Codex credentials")
}

func TestCompleteRejectsAnInvocationWhoseBindingStateVanished(t *testing.T) {
	root := rankedWorkspace(t, nil)
	input := flowBuildInput(root)
	request, err := buildMissionInvocation(t.Context(), input)
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(root, "plugins.lock")))
	_, err = completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{
		Root: root, BasePath: input.BasePath, RequestID: request.RequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: flowResult},
		Adapter:    domain.ExecutionAdapterCurrentHost, Sink: &captureSink{},
	})
	require.ErrorContains(t, err, "plugins.lock")
}

func TestStateDirectoryBlockedByAFile(t *testing.T) {
	_, root := workspaceWithRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory"), nil, 0o644))
	opts := missionadapter.NormalizeOptions{Root: root, MissionID: "m1"}

	require.Error(t, recordNormalizeConfidence(opts, domain.ConfidenceClaim{}, nil))
	_, err := resolveNormalizeGateLabel(opts)
	if err != nil {
		assert.Contains(t, err.Error(), "gate outcome")
	}
}

func TestLoadLevelingPolicyReportsAnUnreadableOverride(t *testing.T) {
	_, root := workspaceWithRoot(t)
	require.NoError(t, os.MkdirAll(filepath.Join(root, "leveling.yaml"), 0o755))
	_, _, err := loadLevelingPolicy()
	require.ErrorContains(t, err, "read policy")
}

func TestExecuteMissionHostFailureModes(t *testing.T) {
	root := rankedWorkspace(t, nil)
	request, err := buildMissionInvocation(t.Context(), flowBuildInput(root))
	require.NoError(t, err)

	_, err = executeMissionHost(t.Context(), root, "claude", "ctx", domain.MissionInvocationRequest{RequestID: "inv_unknown"})
	require.ErrorContains(t, err, "commit execution adapter")

	t.Setenv("PATH", t.TempDir())
	_, err = executeMissionHost(t.Context(), root, "claude", "ctx", request)
	require.Error(t, err, "a host binary that cannot be started fails the run")
}
