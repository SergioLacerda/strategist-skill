package mission_test

import (
	"bytes"
	"errors"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runWith(t *testing.T, deps mission.LifecycleDependencies, newCmd func(mission.LifecycleDependencies) *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newCmd(deps)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestLifecycle_StartRecordsTheBindingsTheMissionStartedWith(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	deps := lifecycleDeps(t)
	var gotRoot, gotMission string
	deps.RecordBindings = func(r, id string) error { gotRoot, gotMission = r, id; return nil }
	_, _, err := runWith(t, deps, mission.NewStart, "--root", root, "--mission-id", "m-bind")
	require.NoError(t, err)
	assert.Equal(t, root, gotRoot)
	assert.Equal(t, "m-bind", gotMission)
}

func TestLifecycle_AFailedBindingRecordNeverFailsStart(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	deps := lifecycleDeps(t)
	deps.RecordBindings = func(string, string) error { return errors.New("disk full") }
	var warned error
	deps.LifecycleWarn = func(_ string, err error) { warned = err }
	_, _, err := runWith(t, deps, mission.NewStart, "--root", root, "--mission-id", "m-warn")
	require.NoError(t, err)
	require.Error(t, warned, "the failure must be observable")
}

func TestLifecycle_StatusReportsAChangedBindingOnStderrAndKeepsStdout(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	deps := lifecycleDeps(t)
	_, _, err := runWith(t, deps, mission.NewStart, "--root", root, "--mission-id", "m-drift")
	require.NoError(t, err)

	plain, plainErr, err := runWith(t, deps, mission.NewStatus, "--root", root, "--mission-id", "m-drift")
	require.NoError(t, err)
	assert.Empty(t, plainErr, "an unchanged workspace produces no extra output")

	deps.BindingChanges = func(string, string) []string {
		return []string{"binding_changed slot=refinement started_with=writing-plans now=openspec-propose"}
	}
	drifted, driftErr, err := runWith(t, deps, mission.NewStatus, "--root", root, "--mission-id", "m-drift")
	require.NoError(t, err)
	assert.Equal(t, plain, drifted, "stdout (the status JSON) is unchanged")
	assert.Contains(t, driftErr, "binding_changed slot=refinement")
}
