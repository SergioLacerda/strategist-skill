package install

import (
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveCustomSkillAvailabilityUnavailableForUnknownID(t *testing.T) {
	t.Parallel()

	availability := resolveCustomSkillAvailability("definitely-not-a-real-installed-skill-id-xyz")
	assert.False(t, availability.Available)
	assert.NotEmpty(t, availability.Reason)
}

func TestResolveCustomSkillAvailabilityAvailableWhenInstalledUnderHome(t *testing.T) {
	homeDir := t.TempDir()
	testutil.SetHome(t, homeDir)
	skillDir := filepath.Join(homeDir, claudeDirName, installedProvidersDirName, "my-team-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: my-team-skill\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))

	availability := resolveCustomSkillAvailability("my-team-skill")
	assert.True(t, availability.Available)
	assert.Empty(t, availability.Reason)
}

func TestResolveCustomSkillAvailabilityUnavailableWhenHomeDirUnresolvable(t *testing.T) {
	testutil.SetHome(t, "")

	availability := resolveCustomSkillAvailability("whatever")
	assert.False(t, availability.Available)
	assert.Contains(t, availability.Reason, "cannot resolve home directory")
}

func TestCheckCustomSkillAvailabilitySkipsEmptySlotValue(t *testing.T) {
	t.Parallel()

	err := checkCustomSkillAvailability(map[string]string{}, map[string]string{
		"discovery": "",
	})
	require.NoError(t, err)
}

func TestCheckCustomSkillAvailabilitySkipsRegistryKnownEntries(t *testing.T) {
	t.Parallel()

	err := checkCustomSkillAvailability(map[string]string{"brainstorming": "write_analysis"}, map[string]string{
		"discovery": "brainstorming",
	})
	assert.NoError(t, err)
}

func TestCheckCustomSkillAvailabilityPausesOnUnresolvableCustomID(t *testing.T) {
	t.Parallel()

	err := checkCustomSkillAvailability(map[string]string{"brainstorming": "write_analysis"}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "definitely-not-a-real-installed-skill-id-xyz",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configured_unverified")
	assert.Contains(t, err.Error(), "definitely-not-a-real-installed-skill-id-xyz")
	assert.Contains(t, err.Error(), "refinement")
}

func TestCheckCustomSkillAvailabilityAllowsResolvableWorkspaceSkill(t *testing.T) {
	homeDir := t.TempDir()
	testutil.SetHome(t, homeDir)
	skillDir := filepath.Join(homeDir, claudeDirName, installedProvidersDirName, "my-team-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: my-team-skill\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))

	err := checkCustomSkillAvailability(map[string]string{"brainstorming": "write_analysis"}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "my-team-skill",
	})
	assert.NoError(t, err)
}
