package mission_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	mission "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lifecycleDeps injects hermetic fakes over the on-disk layout that
// cmd/strategist/mission_persistence.go uses (missions/<id>.json).
func lifecycleDeps(t *testing.T) mission.LifecycleDependencies {
	t.Helper()
	return mission.LifecycleDependencies{
		RootFlag: cliutil.FlagRoot,
		RequireMissionID: func(id string) error {
			if id == "" {
				return errors.New("mission: --mission-id is required")
			}
			return nil
		},
		ResolveBasePath: cliutil.ResolveActiveBasePath,
		RequireNoExisting: func(root, id string) error {
			if _, err := os.Stat(filepath.Join(root, "missions", id+".json")); err == nil {
				return fmt.Errorf("mission start: mission %q already exists", id)
			}
			return nil
		},
		Save: func(root string, status domain.MissionEngineStatus) error {
			writeMissionState(t, root, status)
			return nil
		},
		Load: testLoadMission,
		WriteResult: func(cmd *cobra.Command, _ bool, value any) error {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(value)
		},
	}
}

func runLifecycle(t *testing.T, newCmd func(mission.LifecycleDependencies) *cobra.Command, args ...string) (string, error) {
	t.Helper()
	cmd := newCmd(lifecycleDeps(t))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func decodeStatus(t *testing.T, out string) domain.MissionEngineStatus {
	t.Helper()
	var status domain.MissionEngineStatus
	require.NoError(t, json.Unmarshal([]byte(out), &status))
	return status
}

func TestLifecycle_StartStatusSubmitRoundTrip(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})

	out, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-life")
	require.NoError(t, err)
	started := decodeStatus(t, out)
	assert.Equal(t, "m-life", started.MissionID)
	assert.FileExists(t, filepath.Join(root, "missions", "m-life.json"))

	out, err = runLifecycle(t, mission.NewStatus, "--root", root, "--mission-id", "m-life")
	require.NoError(t, err)
	assert.Equal(t, started, decodeStatus(t, out))

	out, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-life", "--event", string(domain.MissionEventBootstrapDone))
	require.NoError(t, err)
	assert.NotEqual(t, started.Phase, decodeStatus(t, out).Phase)
}

func TestLifecycle_RejectsMissingMissionID(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	for name, newCmd := range map[string]func(mission.LifecycleDependencies) *cobra.Command{
		"start": mission.NewStart, "status": mission.NewStatus, "submit": mission.NewSubmit, "context": mission.NewContext,
	} {
		_, err := runLifecycle(t, newCmd, "--root", root)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "--mission-id is required", name)
	}
}

func TestLifecycle_StartRejectsDuplicate(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-dup")
	require.NoError(t, err)

	_, err = runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLifecycle_UnknownMission(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	for name, args := range map[string][]string{
		"status":  {"--root", root, "--mission-id", "m-missing"},
		"submit":  {"--root", root, "--mission-id", "m-missing", "--event", string(domain.MissionEventBootstrapDone)},
		"context": {"--root", root, "--mission-id", "m-missing"},
	} {
		newCmd := map[string]func(mission.LifecycleDependencies) *cobra.Command{"status": mission.NewStatus, "submit": mission.NewSubmit, "context": mission.NewContext}[name]
		_, err := runLifecycle(t, newCmd, args...)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "mission "+name+": ", name)
		assert.Contains(t, err.Error(), "not found", name)
	}
}

func TestLifecycle_SubmitRequiresEventAndRejectsInvalidTransition(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-event")
	require.NoError(t, err)

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-event")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission submit: --event is required")

	_, err = runLifecycle(t, mission.NewSubmit, "--root", root, "--mission-id", "m-event", "--event", string(domain.MissionEventSniperDone))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission submit: rejected:")
}

func TestLifecycle_ContextMaterializesAndChecksDigest(t *testing.T) {
	root := setupViewRoot(t, domain.MissionEngineStatus{})
	_, err := runLifecycle(t, mission.NewStart, "--root", root, "--mission-id", "m-ctx")
	require.NoError(t, err)
	body := []byte("context body\n")
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), "notes.md"), body, 0o600))
	sum := sha256.Sum256(body)

	out, err := runLifecycle(t, mission.NewContext, "--root", root, "--mission-id", "m-ctx", "--ref", "notes.md", "--digest", "sha256:"+hex.EncodeToString(sum[:]))
	require.NoError(t, err)
	assert.Contains(t, out, "notes.md")

	_, err = runLifecycle(t, mission.NewContext, "--root", root, "--mission-id", "m-ctx", "--ref", "notes.md", "--digest", "sha256:"+hex.EncodeToString(make([]byte, 32)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission context:")
}

func TestNew_ComposesIsolatedCompleteTree(t *testing.T) {
	build := func() *cobra.Command {
		return mission.New(lifecycleDeps(t), testDeps(), mission.NormalizeDependencies{RootFlag: cliutil.FlagRoot}, mission.ReportUsageDependencies{RootFlag: cliutil.FlagRoot})
	}
	first, second := build(), build()
	assert.NotSame(t, first, second)
	require.Len(t, first.Commands(), 7)
	for _, name := range []string{"start", "status", "submit", "context", "view", "normalize-openspec", "report-usage"} {
		a, _, err := first.Find([]string{name})
		require.NoError(t, err)
		b, _, err := second.Find([]string{name})
		require.NoError(t, err)
		assert.Equal(t, name, a.Name())
		assert.NotSame(t, a, b, name)
	}
}
