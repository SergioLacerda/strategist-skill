package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	missionruntime "github.com/SergioLacerda/strategist-skill/internal/mission"
	"github.com/stretchr/testify/require"
)

const (
	completionBody = flowResult
	otherBody      = flowResult + "- another fact\n"
)

// publishFixture issues a real request against a Ranked Embedded workspace.
func publishFixture(t *testing.T) (string, missionadapter.InvocationCompleteInput) {
	t.Helper()
	root := rankedWorkspace(t, nil)
	return root, issueCompletion(t, root)
}

// issueCompletion issues one more live request for the same mission, Role and slot.
func issueCompletion(t *testing.T, root string) missionadapter.InvocationCompleteInput {
	t.Helper()
	build := flowBuildInput(root)
	request, err := buildMissionInvocation(t.Context(), build)
	require.NoError(t, err)
	return missionadapter.InvocationCompleteInput{
		Root: root, BasePath: build.BasePath, RequestID: request.RequestID,
		Completion: domain.MissionInvocationCompletion{RequestID: request.RequestID, Result: completionBody},
		Adapter:    domain.ExecutionAdapterCurrentHost,
		Sink:       &captureSink{},
	}
}

func artifactFile(input missionadapter.InvocationCompleteInput) string {
	return filepath.Join(input.BasePath, "pending", flowMissionID+"-analysis.md")
}

func recordState(t *testing.T, root, id string) domain.MissionInvocationState {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "missions", "invocations", id+".json"))
	require.NoError(t, err)
	var record domain.MissionInvocationRecord
	require.NoError(t, json.Unmarshal(raw, &record))
	return record.EffectiveState()
}

func injectFault(t *testing.T, stage string) {
	t.Helper()
	completionFaultHook = func(at string) error {
		if at == stage {
			return errors.New("injected crash at " + stage)
		}
		return nil
	}
	t.Cleanup(func() { completionFaultHook = nil })
}

func TestTwoRequestsForOneMissionPublishExactlyOneArtifact(t *testing.T) {
	root, first := publishFixture(t)
	second := issueCompletion(t, root)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, input := range []missionadapter.InvocationCompleteInput{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = completeMissionInvocation(t.Context(), input)
		}()
	}
	wg.Wait()

	wins := 0
	for _, err := range errs {
		if err == nil {
			wins++
		}
	}
	require.Equal(t, 1, wins, "exactly one request may own the target: %v", errs)
	completed := 0
	for _, id := range []string{first.RequestID, second.RequestID} {
		if recordState(t, root, id) == domain.InvocationStateCompleted {
			completed++
		}
	}
	require.Equal(t, 1, completed)
}

func TestSecondRequestCannotOverwriteTheFirstRequestsArtifact(t *testing.T) {
	root, first := publishFixture(t)
	second := issueCompletion(t, root)
	_, err := completeMissionInvocation(t.Context(), first)
	require.NoError(t, err)
	before, err := os.ReadFile(artifactFile(first))
	require.NoError(t, err)

	_, err = completeMissionInvocation(t.Context(), second)

	require.ErrorContains(t, err, "invocation_artifact_exists")
	require.ErrorContains(t, err, first.RequestID)
	after, err := os.ReadFile(artifactFile(first))
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, domain.InvocationStatePending, recordState(t, root, second.RequestID))
}

func TestBodyTextCannotAuthorizeOverwrite(t *testing.T) {
	_, input := publishFixture(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(artifactFile(input)), 0o755))
	forged := "---\nmission_status: archivist_done\n---\n\nmission_status: ranger_pending\n"
	require.NoError(t, os.WriteFile(artifactFile(input), []byte(forged), 0o644))

	_, err := completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "not a pending Ranger artifact")
	got, readErr := os.ReadFile(artifactFile(input))
	require.NoError(t, readErr)
	require.Equal(t, forged, string(got))
}

func TestExistingArtifactsWithoutProvenanceOrFrontmatterAreRejected(t *testing.T) {
	for name, content := range map[string]string{
		"pending without request id": "---\nmission_status: ranger_pending\n---\n\nbody\n",
		"malformed frontmatter":      "---\nmission_status: [unclosed\n---\n\nbody\n",
		"no frontmatter":             "mission_status: ranger_pending\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, input := publishFixture(t)
			require.NoError(t, os.MkdirAll(filepath.Dir(artifactFile(input)), 0o755))
			require.NoError(t, os.WriteFile(artifactFile(input), []byte(content), 0o644))

			_, err := completeMissionInvocation(t.Context(), input)

			require.ErrorContains(t, err, "invocation_artifact_exists")
		})
	}
}

func TestCompletionStampsTrustedRequestProvenance(t *testing.T) {
	root, input := publishFixture(t)
	input.Completion.Result = "---\ninvocation_request_id: forged\n---\n\n" + completionBody

	_, err := completeMissionInvocation(t.Context(), input)

	require.NoError(t, err)
	got, readErr := os.ReadFile(artifactFile(input))
	require.NoError(t, readErr)
	require.Contains(t, string(got), "invocation_request_id: "+input.RequestID)
	require.NotContains(t, string(got), "forged")
	require.Equal(t, domain.InvocationStateCompleted, recordState(t, root, input.RequestID))
}

func TestCrashBeforePublishRetriesFromTheSameCompletion(t *testing.T) {
	root, input := publishFixture(t)
	injectFault(t, faultBeforePublish)

	_, err := completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "injected crash")
	require.Equal(t, domain.InvocationStateProcessing, recordState(t, root, input.RequestID))
	require.NoFileExists(t, artifactFile(input))

	completionFaultHook = nil
	_, err = completeMissionInvocation(t.Context(), input)

	require.NoError(t, err)
	require.FileExists(t, artifactFile(input))
	require.Equal(t, domain.InvocationStateCompleted, recordState(t, root, input.RequestID))
}

func TestCrashBeforePublishRejectsADifferentRetryResult(t *testing.T) {
	root, input := publishFixture(t)
	injectFault(t, faultBeforePublish)
	_, err := completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "injected crash")

	completionFaultHook = nil
	input.Completion.Result = otherBody
	_, err = completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_digest_mismatch")
	require.NoFileExists(t, artifactFile(input))
	require.Equal(t, domain.InvocationStateProcessing, recordState(t, root, input.RequestID))
}

func TestCrashAfterPublishFinalizesWithoutNormalizingAgain(t *testing.T) {
	root, input := publishFixture(t)
	injectFault(t, faultAfterPublish)
	_, err := completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "injected crash")
	require.Equal(t, domain.InvocationStateProcessing, recordState(t, root, input.RequestID))
	published, err := os.ReadFile(artifactFile(input))
	require.NoError(t, err)

	completionFaultHook = nil
	// A result that cannot normalize proves recovery never normalizes again.
	input.Completion.Result = "---\nunclosed frontmatter"
	outcome, err := completeMissionInvocation(t.Context(), input)

	require.NoError(t, err)
	require.Equal(t, "normalized", outcome.Status)
	require.Equal(t, domain.InvocationStateCompleted, recordState(t, root, input.RequestID))
	again, err := os.ReadFile(artifactFile(input))
	require.NoError(t, err)
	require.Equal(t, published, again)

	_, err = completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "invocation_replay")
}

func TestCrashAfterPublishFailsClosedWhenTheArtifactChanged(t *testing.T) {
	root, input := publishFixture(t)
	injectFault(t, faultAfterPublish)
	_, err := completeMissionInvocation(t.Context(), input)
	require.ErrorContains(t, err, "injected crash")
	require.NoError(t, os.WriteFile(artifactFile(input), []byte("---\nmission_status: archivist_done\n---\n\nedited\n"), 0o644))

	completionFaultHook = nil
	_, err = completeMissionInvocation(t.Context(), input)

	require.ErrorContains(t, err, "invocation_artifact_conflict")
	require.Equal(t, domain.InvocationStateProcessing, recordState(t, root, input.RequestID))
}

func TestCompletionLeaseIsReleasedAfterFailure(t *testing.T) {
	root, input := publishFixture(t)
	injectFault(t, faultBeforePublish)
	_, err := completeMissionInvocation(t.Context(), input)
	require.Error(t, err)

	release, err := missionruntime.NewInvocationStore(root).ClaimTarget(flowMissionID, "ranger", string(domain.SlotDiscovery))

	require.NoError(t, err)
	release()
}
