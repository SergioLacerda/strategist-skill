package application

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	leveling "github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/stretchr/testify/require"
)

func TestCommitMissionExecutionFailureBranches(t *testing.T) {
	request := SubmitExecutionCommitRequest{Root: "root", BasePath: "base", MissionID: "m-1", Event: domain.MissionEventGateApproved}
	require.NoError(t, CommitMissionExecution(request, SubmitExecutionCommitPorts{
		RecordSniperClaims: func(string, string, string, domain.MissionEngineEvent) error { return nil },
	}))

	request.Outcome = &handoff.Outcome{}
	require.ErrorContains(t, CommitMissionExecution(request, SubmitExecutionCommitPorts{}), "outcome persistence port")

	wantErr := errors.New("persist failed")
	require.ErrorIs(t, CommitMissionExecution(request, SubmitExecutionCommitPorts{
		PersistOutcome: func(string, handoff.Outcome) error { return wantErr },
	}), wantErr)

	request = SubmitExecutionCommitRequest{
		Root: "root", BasePath: "base", MissionID: "m-1", Event: domain.MissionEventHandoffSatisfied,
		EntryPresent: true,
	}
	require.ErrorContains(t, CommitMissionExecution(request, SubmitExecutionCommitPorts{}), "execution-entry claims port")
	require.ErrorIs(t, CommitMissionExecution(request, SubmitExecutionCommitPorts{
		RecordEntryClaims: func(string, string, string) error { return wantErr },
	}), wantErr)

	request.EntryPresent = false
	require.ErrorContains(t, CommitMissionExecution(request, SubmitExecutionCommitPorts{}), "Sniper claims port")
}

func TestRecoverMissionExecutionFailureBranches(t *testing.T) {
	request := RecoverExecutionRequest{
		Root: "root", BasePath: "base", MissionID: "m-1",
		PersistedStatus: domain.MissionEngineStatus{MissionID: "m-1", State: domain.StateExecution},
	}
	require.ErrorContains(t, RecoverMissionExecution(request, RecoverExecutionPorts{}), "recovery ports")

	wantErr := errors.New("load failed")
	ports := RecoverExecutionPorts{
		Load: func(string, string) (ExecutionRecoverySnapshot, bool, error) {
			return ExecutionRecoverySnapshot{}, false, wantErr
		},
		MarkStateSaved: func(string, ExecutionRecoverySnapshot) error { return nil },
		RecoverOutcome: func(string, handoff.Outcome) error { return nil },
		CompleteClaims: func(string, string, string, ExecutionRecoverySnapshot) error { return nil },
	}
	require.ErrorIs(t, RecoverMissionExecution(request, ports), wantErr)

	ports.Load = func(string, string) (ExecutionRecoverySnapshot, bool, error) {
		return ExecutionRecoverySnapshot{}, false, nil
	}
	require.NoError(t, RecoverMissionExecution(request, ports))

	ports.Load = func(string, string) (ExecutionRecoverySnapshot, bool, error) {
		return ExecutionRecoverySnapshot{DesiredStatus: domain.MissionEngineStatus{MissionID: "other"}}, true, nil
	}
	require.NoError(t, RecoverMissionExecution(request, ports))

	ports.Load = func(string, string) (ExecutionRecoverySnapshot, bool, error) {
		return ExecutionRecoverySnapshot{DesiredStatus: request.PersistedStatus}, true, nil
	}
	ports.MarkStateSaved = func(string, ExecutionRecoverySnapshot) error { return wantErr }
	require.ErrorIs(t, RecoverMissionExecution(request, ports), wantErr)

	ports.Load = func(string, string) (ExecutionRecoverySnapshot, bool, error) {
		return ExecutionRecoverySnapshot{StateSaved: true}, true, nil
	}
	ports.RecoverOutcome = func(string, handoff.Outcome) error { return wantErr }
	require.ErrorIs(t, RecoverMissionExecution(request, ports), wantErr)

	ports.RecoverOutcome = func(string, handoff.Outcome) error { return nil }
	ports.CompleteClaims = func(string, string, string, ExecutionRecoverySnapshot) error { return wantErr }
	require.ErrorIs(t, RecoverMissionExecution(request, ports), wantErr)

	ports.CompleteClaims = func(string, string, string, ExecutionRecoverySnapshot) error { return nil }
	require.NoError(t, RecoverMissionExecution(request, ports))
}

func TestPreflightSubmitFailureBranches(t *testing.T) {
	request := SubmitPreflightRequest{Root: "root", BasePath: "base", MissionID: "m-1", Event: domain.MissionEventHandoffSatisfied}
	require.ErrorContains(t, func() error {
		_, err := PreflightSubmit(request, SubmitPreflightPorts{})
		return err
	}(), "preflight ports")

	wantErr := errors.New("preflight failed")
	guards := []func(*SubmitPreflightPorts){
		func(ports *SubmitPreflightPorts) {
			ports.RequireNoAcceptedSideQuest = func(string, string, domain.MissionEngineEvent) error { return wantErr }
		},
		func(ports *SubmitPreflightPorts) {
			ports.RequireAuthoredPackageRepair = func(string, string, domain.MissionEngineEvent) error { return wantErr }
		},
		func(ports *SubmitPreflightPorts) {
			ports.ValidateArtifacts = func(string, string, domain.MissionEngineEvent) error { return wantErr }
		},
	}
	for _, configure := range guards {
		ports := preflightCoveragePorts()
		configure(&ports)
		_, err := PreflightSubmit(request, ports)
		require.ErrorIs(t, err, wantErr)
	}

	evidence := []func(*SubmitPreflightPorts){
		func(ports *SubmitPreflightPorts) {
			ports.ApprovalGatePackageDigest = func(string, domain.MissionEngineStatus, domain.MissionEngineEvent) (string, error) {
				return "", wantErr
			}
		},
		func(ports *SubmitPreflightPorts) {
			ports.RequireExecutionEvidence = func(string, string, domain.MissionEngineStatus, domain.MissionEngineEvent) (*handoff.Outcome, error) {
				return nil, wantErr
			}
		},
		func(ports *SubmitPreflightPorts) {
			ports.RepairPackageDigest = func(string, string, domain.MissionEngineEvent) (string, error) { return "", wantErr }
		},
		func(ports *SubmitPreflightPorts) {
			ports.PreflightSniperClaims = func(string, string, domain.MissionEngineEvent) ([]string, error) { return nil, wantErr }
		},
	}
	for _, configure := range evidence {
		ports := preflightCoveragePorts()
		configure(&ports)
		_, err := PreflightSubmit(request, ports)
		require.ErrorIs(t, err, wantErr)
	}
}

func preflightCoveragePorts() SubmitPreflightPorts {
	return SubmitPreflightPorts{
		RequireNoAcceptedSideQuest:   func(string, string, domain.MissionEngineEvent) error { return nil },
		RequireAuthoredPackageRepair: func(string, string, domain.MissionEngineEvent) error { return nil },
		ValidateArtifacts:            func(string, string, domain.MissionEngineEvent) error { return nil },
		ApprovalGatePackageDigest: func(string, domain.MissionEngineStatus, domain.MissionEngineEvent) (string, error) {
			return "gate", nil
		},
		RequireExecutionEvidence: func(string, string, domain.MissionEngineStatus, domain.MissionEngineEvent) (*handoff.Outcome, error) {
			return nil, nil
		},
		RepairPackageDigest:   func(string, string, domain.MissionEngineEvent) (string, error) { return "package", nil },
		PreflightSniperClaims: func(string, string, domain.MissionEngineEvent) ([]string, error) { return nil, nil },
	}
}

func TestPersistMissionSubmitStateFailureBranches(t *testing.T) {
	require.ErrorContains(t, PersistMissionSubmitState("root", domain.MissionEngineStatus{}, "", nil, false, MissionSubmitPersistencePorts{}), "state persistence port")

	wantErr := errors.New("save failed")
	require.ErrorIs(t, PersistMissionSubmitState("root", domain.MissionEngineStatus{}, "", nil, false, MissionSubmitPersistencePorts{
		Save: func(string, domain.MissionEngineStatus) error { return wantErr },
	}), wantErr)

	restoreErr := errors.New("restore failed")
	require.ErrorIs(t, PersistMissionSubmitState("root", domain.MissionEngineStatus{}, "analysis.md", []byte("before"), true, MissionSubmitPersistencePorts{
		Save:            func(string, domain.MissionEngineStatus) error { return wantErr },
		RestoreAnalysis: func(string, []byte) error { return restoreErr },
	}), restoreErr)
}

func TestSubmitMissionRejectsTransitionAndDigestFailure(t *testing.T) {
	engine, _, err := domain.StartMission(domain.MissionStartRequest{MissionID: "m-1"})
	require.NoError(t, err)
	_, err = SubmitMission(engine, SubmitMissionRequest{Event: domain.MissionEventGateApproved})
	require.Error(t, err)

	engine, _, err = domain.StartMission(domain.MissionStartRequest{MissionID: "m-2"})
	require.NoError(t, err)
	_, err = SubmitMission(engine, SubmitMissionRequest{Event: domain.MissionEventBootstrapDone, GateDigest: "sha256:gate"})
	require.ErrorContains(t, err, "approval gate package digest")
}

type failingApplicationContextReader struct{}

func (failingApplicationContextReader) ReadFile(string) ([]byte, error) {
	return nil, errors.New("context read failed")
}

func TestContextAndMissionStatusFailureBranches(t *testing.T) {
	_, err := MaterializeMissionContext(
		func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			return nil, domain.MissionEngineStatus{}, nil
		},
		"root", "m-1", failingApplicationContextReader{}, []domain.ContextReference{{Ref: "brief.md"}}, 1, 100,
	)
	require.ErrorContains(t, err, "materialize mission context")

	status, err := LoadMissionStatus(func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
		return nil, domain.MissionEngineStatus{MissionID: "m-1", Phase: domain.PhaseRefinement, State: domain.StateRefinement}, nil
	}, "root", "m-1")
	require.NoError(t, err)
	require.Equal(t, "m-1", status.MissionID)
}

func TestSideQuestFailureBranches(t *testing.T) {
	require.Error(t, func() error {
		_, err := DecideSideQuest(SideQuestDecisionRequest{}, nil, nil)
		return err
	}())
	require.Error(t, func() error {
		_, err := DecideSideQuest(SideQuestDecisionRequest{}, func(string, string) (domain.MissionState, error) {
			return "", errors.New("load failed")
		}, func(SideQuestDecisionRequest) (SideQuestDecisionResult, error) { return SideQuestDecisionResult{}, nil })
		return err
	}())
	require.Error(t, func() error {
		_, err := DecideSideQuest(SideQuestDecisionRequest{}, func(string, string) (domain.MissionState, error) { return domain.StateApprovalGate, nil }, func(SideQuestDecisionRequest) (SideQuestDecisionResult, error) {
			return SideQuestDecisionResult{}, errors.New("decision failed")
		})
		return err
	}())

	require.Error(t, func() error {
		_, err := ReserveADRTarget(ADRTargetRequest{}, nil, nil)
		return err
	}())
	require.Error(t, func() error {
		_, err := ReserveADRTarget(ADRTargetRequest{}, func(string, string) (domain.MissionState, error) {
			return "", errors.New("load failed")
		}, func(ADRTargetRequest, domain.MissionState) (string, error) { return "", nil })
		return err
	}())
	require.Error(t, func() error {
		_, err := ReserveADRTarget(ADRTargetRequest{}, func(string, string) (domain.MissionState, error) { return domain.StateExecution, nil }, func(ADRTargetRequest, domain.MissionState) (string, error) {
			return "", errors.New("reserve failed")
		})
		return err
	}())
}

func TestResolveLevelCoversReuseAndHostPassthrough(t *testing.T) {
	root := t.TempDir()
	ledger := root + "/levels.jsonl"
	loader := func() (leveling.Policy, error) {
		return leveling.Policy{}, errors.New("policy must not load")
	}

	first, err := ResolveLevel(domain.DefaultRoleRegistry(), loader, domain.LevelingConfig{Mode: domain.LevelingModeManual}, ledger, LevelingInput{
		Mission: "m-1", Role: "RANGER", HostModel: "claude-sonnet-5", HostEffort: "high",
	}, 0)
	require.NoError(t, err)
	require.True(t, first.Recorded)

	reused, err := ResolveLevel(domain.DefaultRoleRegistry(), loader, domain.LevelingConfig{Mode: domain.LevelingModeManual}, ledger, LevelingInput{
		Mission: "m-1", Role: "ranger",
	}, 1)
	require.NoError(t, err)
	require.True(t, reused.Reused)

	forced, err := ResolveLevel(domain.DefaultRoleRegistry(), loader, domain.LevelingConfig{Mode: domain.LevelingModeManual}, ledger, LevelingInput{
		Mission: "m-1", Role: "ranger", Reason: "escalated",
	}, 1)
	require.NoError(t, err)
	require.True(t, forced.Recorded)

	gate, err := ResolveLevel(domain.DefaultRoleRegistry(), loader, domain.LevelingConfig{Mode: domain.LevelingModeManual}, ledger, LevelingInput{
		Mission: "m-2", Role: "gate",
	}, 1)
	require.NoError(t, err)
	require.False(t, gate.Recorded)
}
