package mission

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/stretchr/testify/require"
)

func sealedExecutionOutcome(t *testing.T, missionID string) handoff.Outcome {
	t.Helper()
	outcome, err := (handoff.Outcome{MissionID: missionID, Transition: handoff.TransitionArchivistToSniper, PackageDigest: "package-a", Attempt: 1, Result: handoff.OutcomePassed, PolicyID: "policy", GateObserved: "gate_analysis_accepted"}).Seal(time.Now().UTC())
	require.NoError(t, err)
	return outcome
}

func TestPrepareExecutionEntryRejectsDifferentRevisionWhileIncomplete(t *testing.T) {
	root := t.TempDir()
	status := domain.MissionEngineStatus{MissionID: "m1", Phase: domain.PhaseExecution}
	outcome := handoff.Outcome{MissionID: "m1", Integrity: "outcome-a"}
	entry, err := prepareExecutionEntry(root, "m1", "package-a", []string{"docs/a.md"}, &outcome, status)
	require.NoError(t, err)
	require.False(t, entry.Completed)
	other := outcome
	other.Integrity = "outcome-b"
	_, err = prepareExecutionEntry(root, "m1", "package-b", []string{"docs/b.md"}, &other, status)
	require.ErrorContains(t, err, "identity mismatch")
}

func TestPrepareExecutionEntryRotatesCompletedRevision(t *testing.T) {
	root := t.TempDir()
	status := domain.MissionEngineStatus{MissionID: "m1", Phase: domain.PhaseExecution}
	a := handoff.Outcome{MissionID: "m1", Integrity: "outcome-a"}
	entry, err := prepareExecutionEntry(root, "m1", "package-a", nil, &a, status)
	require.NoError(t, err)
	entry.Completed = true
	require.NoError(t, saveExecutionEntry(root, entry))
	b := handoff.Outcome{MissionID: "m1", Integrity: "outcome-b"}
	next, err := prepareExecutionEntry(root, "m1", "package-b", nil, &b, status)
	require.NoError(t, err)
	require.Equal(t, "package-b", next.PackageDigest)
	require.False(t, next.Completed)
}

func TestSaveNewExecutionEntryLinkFailureLeavesNoPublishedEntry(t *testing.T) {
	root := t.TempDir()
	path := executionEntryPath(root, "m1")
	old := executionEntryLink
	executionEntryLink = func(string, string) error { return errors.New("link failed") }
	t.Cleanup(func() { executionEntryLink = old })
	err := saveNewExecutionEntry(path, executionEntry{ID: "entry", MissionID: "m1"})
	require.ErrorContains(t, err, "link failed")
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestSaveExecutionEntryRenameFailurePreservesPreviousJournal(t *testing.T) {
	root := t.TempDir()
	entry := executionEntry{ID: "entry", MissionID: "m1"}
	require.NoError(t, saveNewExecutionEntry(executionEntryPath(root, "m1"), entry))
	old := executionEntryRename
	executionEntryRename = func(string, string) error { return errors.New("rename failed") }
	t.Cleanup(func() { executionEntryRename = old })
	entry.Completed = true
	require.ErrorContains(t, saveExecutionEntry(root, &entry), "rename failed")
	stored, err := loadExecutionEntry(filepath.Join(root, "missions", "execution-entry", "m1.json"))
	require.NoError(t, err)
	require.False(t, stored.Completed)
}

func TestRecoverExecutionEntryConsumesAndCompletesOnePreparedEntry(t *testing.T) {
	root := t.TempDir()
	outcome := sealedExecutionOutcome(t, "m1")
	_, err := handoff.NewOutcomeStore(root).Append(outcome)
	require.NoError(t, err)
	status := domain.MissionEngineStatus{MissionID: "m1", Phase: domain.PhaseExecution}
	entry := executionEntry{ID: "entry", MissionID: "m1", PackageDigest: "package-a", Outcome: outcome, DesiredStatus: status, StateSaved: true}
	require.NoError(t, saveNewExecutionEntry(executionEntryPath(root, "m1"), entry))
	require.NoError(t, recoverExecutionEntry(root, filepath.Join(root, "analysis"), "m1", status))
	stored, err := loadExecutionEntry(executionEntryPath(root, "m1"))
	require.NoError(t, err)
	require.True(t, stored.Completed)
	consumed, err := handoff.NewOutcomeStore(root).Consumed(outcome)
	require.NoError(t, err)
	require.True(t, consumed)
}

func TestRecoverExecutionEntryRetriesAfterClaimProjectionFailure(t *testing.T) {
	root := t.TempDir()
	outcome := sealedExecutionOutcome(t, "m1")
	_, err := handoff.NewOutcomeStore(root).Append(outcome)
	require.NoError(t, err)
	status := domain.MissionEngineStatus{MissionID: "m1", Phase: domain.PhaseExecution}
	entry := executionEntry{ID: "entry", MissionID: "m1", PackageDigest: "package-a", Outcome: outcome, DesiredStatus: status, StateSaved: true}
	require.NoError(t, saveNewExecutionEntry(executionEntryPath(root, "m1"), entry))
	old := executionEntryRecordClaims
	executionEntryRecordClaims = func(string, string, string, string, []string, time.Time) (int, error) {
		return 0, errors.New("claims failed")
	}
	require.ErrorContains(t, recoverExecutionEntry(root, filepath.Join(root, "analysis"), "m1", status), "claims failed")
	executionEntryRecordClaims = old
	t.Cleanup(func() { executionEntryRecordClaims = old })
	require.NoError(t, recoverExecutionEntry(root, filepath.Join(root, "analysis"), "m1", status))
	stored, err := loadExecutionEntry(executionEntryPath(root, "m1"))
	require.NoError(t, err)
	require.True(t, stored.Completed)
}
