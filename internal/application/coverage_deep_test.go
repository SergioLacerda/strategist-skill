package application_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	leveling "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

func TestStartMissionFailureBranches(t *testing.T) {
	_, err := application.StartMission("root", "m-1", application.MissionStartPorts{})
	require.Error(t, err)
	_, err = application.StartMission("root", "m-1", application.MissionStartPorts{
		RequireNoExisting: func(string, string) error { return errors.New("already exists") },
		Save:              func(string, domain.MissionEngineStatus) error { return nil },
	})
	require.Error(t, err)
	ports := application.MissionStartPorts{
		RequireNoExisting: func(string, string) error { return nil },
		Save:              func(string, domain.MissionEngineStatus) error { return nil },
	}
	_, err = application.StartMission("root", "", ports)
	require.Error(t, err)
	ports.Save = func(string, domain.MissionEngineStatus) error { return errors.New("disk full") }
	_, err = application.StartMission("root", "m-1", ports)
	require.Error(t, err)
	ports.Save = func(string, domain.MissionEngineStatus) error { return nil }
	ports.InitiativeStart = func(string, string) error { return errors.New("initiative unavailable") }
	_, err = application.StartMission("root", "m-1", ports)
	require.Error(t, err)
}

func TestLoadStatusAndRouteFailureBranches(t *testing.T) {
	_, err := application.LoadMissionStatus(nil, "root", "m-1")
	require.Error(t, err)
	loaderErr := func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
		return nil, domain.MissionEngineStatus{}, errors.New("load failed")
	}
	_, err = application.LoadMissionStatus(loaderErr, "root", "m-1")
	require.Error(t, err)
	_, err = application.LoadMissionStatus(func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
		return nil, domain.MissionEngineStatus{}, nil
	}, "root", "m-1")
	require.Error(t, err)

	_, err = application.RecordRoute(context.Background(), application.RecordRouteRequest{}, nil)
	require.Error(t, err)
	_, err = application.RecordRoute(context.Background(), application.RecordRouteRequest{Root: "root"}, func(context.Context, string, string, []byte) (bool, error) {
		return false, nil
	})
	require.Error(t, err)
	_, err = application.RecordRoute(context.Background(), application.RecordRouteRequest{Root: "root", MissionID: "m-1"}, func(context.Context, string, string, []byte) (bool, error) {
		return false, errors.New("record failed")
	})
	require.Error(t, err)
}

func TestInvocationBoundaryFailureBranches(t *testing.T) {
	_, err := application.BuildInvocation(context.Background(), application.InvocationBuildRequest{}, nil)
	require.Error(t, err)
	_, err = application.CompleteInvocation(context.Background(), application.InvocationCompletionRequest{}, nil)
	require.Error(t, err)
	_, err = application.CompleteInvocation(context.Background(), application.InvocationCompletionRequest{}, func(context.Context, application.InvocationCompletionRequest) (domain.MissionInvocationOutcome, error) {
		return domain.MissionInvocationOutcome{}, nil
	})
	require.Error(t, err)

	require.NoError(t, application.ValidateInvocationDispatch("", "", string(domain.SlotExecution)))
	require.Error(t, application.ValidateInvocationDispatch("codex", "", string(domain.SlotDiscovery)))
	require.Error(t, application.ValidateInvocationDispatch("codex", "ctx", string(domain.SlotExecution)))
	require.Error(t, application.ValidateInvocationDispatch("future", "ctx", string(domain.SlotDiscovery)))
	require.NoError(t, application.ValidateInvocationDispatch("codex", "ctx", string(domain.SlotDiscovery)))

	require.Error(t, application.PrepareInvocation("root", "m-1", string(domain.SlotDiscovery), nil))
	require.Error(t, application.PrepareInvocation("root", "m-1", string(domain.SlotDiscovery), func(string, string) (domain.MissionEngineStatus, error) {
		return domain.MissionEngineStatus{}, errors.New("load failed")
	}))
	require.Error(t, application.PrepareInvocation("root", "m-1", "unknown", func(string, string) (domain.MissionEngineStatus, error) {
		return domain.MissionEngineStatus{Phase: domain.PhaseDiscovery}, nil
	}))
	require.NoError(t, application.PrepareInvocation("root", "m-1", string(domain.SlotDiscovery), func(string, string) (domain.MissionEngineStatus, error) {
		return domain.MissionEngineStatus{Phase: domain.PhaseDiscovery}, nil
	}))
}

func TestLevelingFailureAndReuseBranches(t *testing.T) {
	root := t.TempDir()
	ledger := filepath.Join(root, "levels.jsonl")
	_, err := application.ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return leveling.Policy{}, nil
	}, domain.LevelingConfig{}, ledger, application.LevelingInput{Mission: "m-1", Role: "ranger"}, 1)
	require.NoError(t, err)
	_, err = application.ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) {
		return leveling.Policy{}, errors.New("policy unavailable")
	}, domain.LevelingConfig{}, filepath.Join(root, "broken", "levels.jsonl"), application.LevelingInput{Role: "ranger", Provider: "p"}, 1)
	require.NoError(t, err)
	badLedger := filepath.Join(root, "bad")
	require.NoError(t, os.WriteFile(badLedger, []byte("not-json\n"), 0o600))
	_, err = application.ResolveLevel(domain.DefaultRoleRegistry(), func() (leveling.Policy, error) { return leveling.Policy{}, nil }, domain.LevelingConfig{}, badLedger, application.LevelingInput{Mission: "m-1", Role: "ranger"}, 1)
	require.NoError(t, err)
}

type failingConfidenceRecorder struct{}

func (failingConfidenceRecorder) RecordMissing(string, string) error {
	return errors.New("telemetry unavailable")
}

func TestApplicationAdapterFailureBranches(t *testing.T) {
	_, err := application.PlanInstall(application.InstallPlanInput{Stage: domain.StageFull})
	require.Error(t, err)

	require.Error(t, application.RecordMissingConfidence(nil, application.MissingConfidenceInput{}))
	require.Error(t, application.RecordMissingConfidence(failingConfidenceRecorder{}, application.MissingConfidenceInput{
		CorrelationKey: "boundary", Reason: "not produced",
	}))

	_, err = application.ResolveDigestInput("", "skill.md", nil)
	require.Error(t, err)
	_, err = application.ResolveDigestInput("", "skill.md", func(string) (string, error) {
		return "", errors.New("hash failed")
	})
	require.Error(t, err)
	got, err := application.ResolveDigestInput("sha256:flag", "  ", nil)
	require.NoError(t, err)
	require.Equal(t, "sha256:flag", got)
	_, err = application.CompareResolvedDigest("provider", "digest", nil)
	require.Error(t, err)

	_, err = application.RecordMissionTokenUsage("root", "base", application.MissionTokenUsageInput{}, time.Time{}, application.MissionTokenUsagePorts{})
	require.Error(t, err)
	_, err = application.RecordMissionTokenUsage("root", "base", application.MissionTokenUsageInput{MissionID: "m-1", TokensIn: -1}, time.Time{}, application.MissionTokenUsagePorts{})
	require.Error(t, err)

	loadErr := errors.New("load failed")
	err = application.RecordHandoffLifecycle("root", "base", "m-1", application.HandoffLifecyclePorts{
		Lock: func(_, _ string, fn func() error) error { return fn() },
		Load: func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			return nil, domain.MissionEngineStatus{}, loadErr
		},
		Save: func(string, domain.MissionEngineStatus) error { return nil },
		Evaluate: func(string, string, *domain.MissionEngine) (domain.MissionEngineStatus, bool, error) {
			return domain.MissionEngineStatus{}, false, nil
		},
	})
	require.ErrorIs(t, err, loadErr)

	labelsErr := errors.New("labels unavailable")
	_, err = application.ReportHandoffMetrics("root", application.HandoffMetricsReportPorts{
		ReadChallenges: func(string) ([]application.HandoffChallengeMetric, error) { return nil, nil },
		ReadLabels:     func(string) ([]application.HandoffApplicationGroundTruth, error) { return nil, labelsErr },
		Compute: func([]application.HandoffChallengeMetric, []application.HandoffApplicationGroundTruth) application.HandoffMetricsReport {
			return application.HandoffMetricsReport{}
		},
	})
	require.ErrorIs(t, err, labelsErr)
}

func TestValidateWorkspaceChecksInvalidKnowledgeIndex(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("invalid: [yaml\n"), 0o600))
	report := application.ValidateWorkspace(root)
	require.NotEmpty(t, report.Errors)
	require.GreaterOrEqual(t, report.Checks, 2)
}
