//go:build integration

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestE2E_PrivateOpenSpecLauncherAvoidsNestedRoot(t *testing.T) {
	workspace := t.TempDir()
	foreignConfig := filepath.Join(t.TempDir(), "foreign-config.yaml")
	foreignBefore := []byte("foreign: untouched\n")
	require.NoError(t, os.WriteFile(foreignConfig, foreignBefore, 0o644))
	input := strings.Join([]string{
		"en", "en", "en", "en", "epic", ".analysis",
		"brainstorming::ranked", "openspec-propose::ranked", "sniper::ranked", "", "",
	}, "\n")
	install := runStrategistCLIWithInput(t, workspace, map[string]string{
		"PATH":             os.Getenv("PATH"),
		"OPEN_SPEC_CONFIG": foreignConfig,
	}, input, "install", "--target", workspace, "--wizard", "--no-shim")
	require.Equal(t, 0, install.exitCode, install.output())

	strategist := filepath.Join(workspace, ".strategist")
	runtimeState, err := os.ReadFile(filepath.Join(strategist, "ranked-runtimes.yaml"))
	require.NoError(t, err)
	var state struct {
		Entries []struct {
			Provider string `yaml:"provider"`
			Runtime  struct {
				Node   string `yaml:"node"`
				Script string `yaml:"script"`
			} `yaml:"runtime"`
		} `yaml:"entries"`
	}
	require.NoError(t, yaml.Unmarshal(runtimeState, &state))
	require.Len(t, state.Entries, 1)
	require.Equal(t, "openspec-propose", state.Entries[0].Provider)
	require.True(t, filepath.IsAbs(state.Entries[0].Runtime.Node))
	require.NotEmpty(t, state.Entries[0].Runtime.Script)
	require.NoDirExists(t, filepath.Join(strategist, "openspec", "openspec"))

	launcherHome := t.TempDir()
	launcher := exec.Command(
		state.Entries[0].Runtime.Node,
		filepath.Join(strategist, filepath.FromSlash(state.Entries[0].Runtime.Script)),
		"validate", "--changes", "--json",
	)
	launcher.Dir = filepath.Join(strategist, "openspec")
	launcher.Env = hermeticEnv(map[string]string{
		"HOME":             launcherHome,
		"USERPROFILE":      launcherHome,
		"XDG_CONFIG_HOME":  t.TempDir(),
		"XDG_CACHE_HOME":   t.TempDir(),
		"XDG_DATA_HOME":    t.TempDir(),
		"OPEN_SPEC_CONFIG": filepath.Join(launcherHome, "openspec.json"),
	})
	launcherOutput, err := launcher.CombinedOutput()
	require.NoError(t, err, "private launcher: %s", launcherOutput)
	assert.NotContains(t, string(launcherOutput), "openspec/openspec")
	assert.NotContains(t, string(launcherOutput), "deprecated")

	foreignAfter, err := os.ReadFile(foreignConfig)
	require.NoError(t, err)
	assert.Equal(t, foreignBefore, foreignAfter)
}
