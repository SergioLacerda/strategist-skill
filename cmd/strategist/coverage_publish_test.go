package main

import (
	"os"
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResumeCommittedCompletionBranches(t *testing.T) {
	store := missionruntime.NewInvocationStore(t.TempDir())
	pending := domain.MissionInvocationRecord{}
	resumed, err := resumeCommittedCompletion(store, pending, filepath.Join(t.TempDir(), "x.md"))
	require.NoError(t, err)
	assert.False(t, resumed, "a pending request has nothing to resume")

	processing := domain.MissionInvocationRecord{State: domain.InvocationStateProcessing, Request: domain.MissionInvocationRequest{RequestID: "r1"}}
	resumed, err = resumeCommittedCompletion(store, processing, filepath.Join(t.TempDir(), "absent.md"))
	require.NoError(t, err)
	assert.False(t, resumed, "an absent artifact means publication is retried")

	_, err = resumeCommittedCompletion(store, processing, t.TempDir())
	require.ErrorContains(t, err, "inspect existing discovery artifact")

	foreign := filepath.Join(t.TempDir(), "foreign.md")
	require.NoError(t, os.WriteFile(foreign, []byte("not an artifact"), 0o644))
	_, err = resumeCommittedCompletion(store, processing, foreign)
	require.ErrorContains(t, err, "invocation_artifact_conflict")
}

func TestCompleteRejectsABasePathOutsideTheWorkspace(t *testing.T) {
	root := rankedWorkspace(t, nil)
	input := flowBuildInput(root)
	request, err := buildMissionInvocation(t.Context(), input)
	require.NoError(t, err)
	_, err = completeMissionInvocation(t.Context(), missionadapter.InvocationCompleteInput{
		Root: root, BasePath: t.TempDir(), RequestID: request.RequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: flowResult},
		Adapter:    domain.ExecutionAdapterCurrentHost, Sink: &captureSink{},
	})
	require.ErrorContains(t, err, "escapes workspace")
}
