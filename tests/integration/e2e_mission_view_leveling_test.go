//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type e2eMissionView struct {
	Confidence struct {
		Availability string `json:"availability"`
	} `json:"confidence"`
	Leveling struct {
		Availability string `json:"availability"`
		Roles        []struct {
			Role      string `json:"role"`
			Effective *struct {
				Provider string `json:"provider"`
				Effort   string `json:"effort"`
			} `json:"effective"`
		} `json:"roles"`
	} `json:"leveling"`
}

func missionView(t *testing.T, workspace, missionID string) e2eMissionView {
	t.Helper()
	res := runStrategistCLI(t, workspace, "mission", "view", "--mission-id", missionID, "--json")
	require.Equal(t, 0, res.exitCode, res.output())
	var v e2eMissionView
	require.NoError(t, json.Unmarshal([]byte(res.stdout), &v), res.output())
	return v
}

// A fresh mission shows explicit absence; the Ranger on_start hook, run the
// way an agent runs it (host model known, effort placeholder unreplaced),
// resolves a policy-completed level in automatic mode; the mission view then
// reports it.
func TestE2E_CLI_MissionViewAndLevelingActivation(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())
	start := runStrategistCLI(t, workspace, "mission", "start", "--mission-id", "m-e2e")
	require.Equal(t, 0, start.exitCode, start.output())

	fresh := missionView(t, workspace, "m-e2e")
	assert.Equal(t, "no_sample", fresh.Confidence.Availability)
	assert.Equal(t, "unknown", fresh.Leveling.Availability)

	label := runStrategistCLI(t, workspace, "leveling", "label", "--role", "ranger", "--mission", "m-e2e",
		"--host-model", "claude-opus-5", "--host-effort", "<your-effort>", "--json")
	require.Equal(t, 0, label.exitCode, label.output())
	assert.Contains(t, label.stderr, "placeholder", "the unreplaced effort is reported, not recorded")

	after := missionView(t, workspace, "m-e2e")
	assert.Equal(t, "available", after.Leveling.Availability)
	var ranger *struct {
		Provider string `json:"provider"`
		Effort   string `json:"effort"`
	}
	for _, role := range after.Leveling.Roles {
		if role.Role == "ranger" {
			ranger = role.Effective
		}
	}
	require.NotNil(t, ranger, "the recorded level reaches the mission view")
	assert.Equal(t, "CLAUDE", ranger.Provider)
	assert.NotEmpty(t, ranger.Effort, "automatic mode completed the effort from the policy")
}

// With a tampered install authority, LEVELING activation fails closed with a
// cataloged reason code instead of silently applying the policy, and labelling
// still never blocks the mission.
func TestE2E_CLI_LevelingActivationFailsClosedOnStaleAuthority(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())

	manifestPath := filepath.Join(workspace, ".strategist", ".install-manifest.json")
	raw, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	var manifest map[string]any
	require.NoError(t, json.Unmarshal(raw, &manifest))
	manifest["leveling_policy_digest"] = "stale-digest"
	tampered, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, tampered, 0o600))

	label := runStrategistCLI(t, workspace, "leveling", "label", "--role", "archivist", "--host-model", "claude-opus-5", "--json")
	require.Equal(t, 0, label.exitCode, "labelling never blocks a mission: %s", label.output())
	assert.Contains(t, label.stderr, "leveling_policy_stale")
	assert.NotContains(t, label.stdout, `"level_source":"policy"`, "a stale authority is never applied")
}
