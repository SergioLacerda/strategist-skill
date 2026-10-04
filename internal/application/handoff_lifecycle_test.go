package application_test

import (
	"errors"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRecordHandoffLifecycleOwnsLockLoadEvaluateSaveOrder(t *testing.T) {
	t.Parallel()

	var order []string
	status := domain.MissionEngineStatus{MissionID: "m-1"}
	err := application.RecordHandoffLifecycle("root", "base", "m-1", application.HandoffLifecyclePorts{
		Lock: func(_, _ string, fn func() error) error {
			order = append(order, "lock")
			return fn()
		},
		Load: func(_, _ string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			order = append(order, "load")
			return &domain.MissionEngine{}, status, nil
		},
		Evaluate: func(_, _ string, _ *domain.MissionEngine) (domain.MissionEngineStatus, bool, error) {
			order = append(order, "evaluate")
			return status, true, nil
		},
		Save: func(_ string, _ domain.MissionEngineStatus) error {
			order = append(order, "save")
			return nil
		},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"lock", "load", "evaluate", "save"}, order)
}

func TestRecordHandoffLifecycleSavesChangedStateBeforeReturningEvaluationError(t *testing.T) {
	t.Parallel()

	evaluationErr := errors.New("evaluation failed")
	saveCalled := false
	err := application.RecordHandoffLifecycle("root", "base", "m-1", application.HandoffLifecyclePorts{
		Lock: func(_, _ string, fn func() error) error { return fn() },
		Load: func(_, _ string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			return &domain.MissionEngine{}, domain.MissionEngineStatus{MissionID: "m-1"}, nil
		},
		Evaluate: func(_, _ string, _ *domain.MissionEngine) (domain.MissionEngineStatus, bool, error) {
			return domain.MissionEngineStatus{MissionID: "m-1"}, true, evaluationErr
		},
		Save: func(_ string, _ domain.MissionEngineStatus) error {
			saveCalled = true
			return nil
		},
	})

	require.ErrorIs(t, err, evaluationErr)
	require.True(t, saveCalled)
}

func TestRecordHandoffLifecycleJoinsEvaluationAndSaveErrors(t *testing.T) {
	t.Parallel()

	evaluationErr := errors.New("evaluation failed")
	saveErr := errors.New("save failed")
	err := application.RecordHandoffLifecycle("root", "base", "m-1", application.HandoffLifecyclePorts{
		Lock: func(_, _ string, fn func() error) error { return fn() },
		Load: func(_, _ string) (*domain.MissionEngine, domain.MissionEngineStatus, error) {
			return &domain.MissionEngine{}, domain.MissionEngineStatus{}, nil
		},
		Evaluate: func(_, _ string, _ *domain.MissionEngine) (domain.MissionEngineStatus, bool, error) {
			return domain.MissionEngineStatus{}, true, evaluationErr
		},
		Save: func(_ string, _ domain.MissionEngineStatus) error { return saveErr },
	})

	require.ErrorIs(t, err, evaluationErr)
	require.ErrorIs(t, err, saveErr)
	require.Contains(t, err.Error(), "handoff_state_persist_failed")
}

func TestRecordHandoffLifecycleRejectsMissingPort(t *testing.T) {
	t.Parallel()

	err := application.RecordHandoffLifecycle("root", "base", "m-1", application.HandoffLifecyclePorts{})
	require.EqualError(t, err, "handoff lifecycle ports are required")
}
