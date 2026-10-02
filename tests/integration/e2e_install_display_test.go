//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wizardInputEN = "en\nen\nen\nen\nepic\n.analysis\nbrainstorming\narchivist\nsniper\n\n"
	wizardInputPT = "pt-BR\npt-BR\npt-BR\nen\nepic\n.analysis\nbrainstorming\narchivist\nsniper\n\n"
)

func TestE2E_CLI_InstallWizardIsQuietByDefault(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()

	res := runStrategistCLIWithInput(t, workspace, nil, wizardInputPT, "install", "--wizard", "--no-shim", "--target", workspace)

	require.Equal(t, 0, res.exitCode, res.output())
	out := res.output()
	assert.NotContains(t, out, " INFO ", "no INFO log line is displayed")
	assert.NotContains(t, out, "Plugins de slot - qual skill preenche cada papel da missão")
	assert.NotContains(t, out, "Baú do tesouro — base de conhecimento offline opcional para todos os slots:")
	assert.NotContains(t, out, "role/provider migration preview")
	assert.Contains(t, out, "Ranger / plugin de descoberta", "the prompts are still shown")
	assert.Contains(t, out, "Node.js >=", "the Ranked Node prerequisite note is still shown")
	assert.Contains(t, out, "install complete", "the result banner is still shown")
	assert.FileExists(t, filepath.Join(workspace, ".strategist", "active.yaml"))
}

func TestE2E_CLI_InstallWizardVerboseRestoresTheDisplay(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()

	res := runStrategistCLIWithInput(t, workspace, nil, wizardInputPT, "install", "--wizard", "--verbose", "--no-shim", "--target", workspace)

	require.Equal(t, 0, res.exitCode, res.output())
	out := res.output()
	assert.Contains(t, out, "INFO [Strategist] install starting")
	assert.Contains(t, out, "INFO [Strategist] install complete")
	assert.Contains(t, out, "Plugins de slot - qual skill preenche cada papel da missão")
	assert.Contains(t, out, "Baú do tesouro — base de conhecimento offline opcional para todos os slots:")
	assert.Contains(t, out, "role/provider migration preview")
}

func TestE2E_CLI_InstallWizardEnglishHeadersFollowTheSameSwitch(t *testing.T) {
	t.Parallel()
	quiet, verbose := t.TempDir(), t.TempDir()

	hidden := runStrategistCLIWithInput(t, quiet, nil, wizardInputEN, "install", "--wizard", "--no-shim", "--target", quiet)
	shown := runStrategistCLIWithInput(t, verbose, nil, wizardInputEN, "install", "--wizard", "--verbose", "--no-shim", "--target", verbose)

	require.Equal(t, 0, hidden.exitCode, hidden.output())
	require.Equal(t, 0, shown.exitCode, shown.output())
	assert.NotContains(t, hidden.output(), "Slot plugins - which skill fills each mission role:")
	assert.Contains(t, shown.output(), "Slot plugins - which skill fills each mission role:")
	assert.Contains(t, shown.output(), "Treasure chest — optional offline knowledge source for all slots:")
}

func TestE2E_CLI_InstallSilentIsQuietByDefaultAndVerboseShowsInfo(t *testing.T) {
	t.Parallel()
	quiet, verbose := t.TempDir(), t.TempDir()

	hidden := runStrategistCLI(t, quiet, "install", "--silent", "--no-shim", "--target", quiet)
	shown := runStrategistCLI(t, verbose, "install", "--silent", "--verbose", "--no-shim", "--target", verbose)

	require.Equal(t, 0, hidden.exitCode, hidden.output())
	require.Equal(t, 0, shown.exitCode, shown.output())
	assert.NotContains(t, hidden.output(), " INFO ")
	assert.Contains(t, hidden.output(), "install complete")
	assert.Contains(t, shown.output(), "INFO [Strategist] install complete")
}

func TestE2E_CLI_InstallWarningsStayVisibleByDefault(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	first := runStrategistCLI(t, workspace, "install", "--silent", "--no-shim", "--target", workspace)
	require.Equal(t, 0, first.exitCode, first.output())
	active := filepath.Join(workspace, ".strategist", "active.yaml")
	raw, err := os.ReadFile(active)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(active, append(raw, []byte("# edited\n")...), 0o644))

	forced := runStrategistCLI(t, workspace, "install", "--silent", "--force", "--no-shim", "--target", workspace)

	require.Equal(t, 0, forced.exitCode, forced.output())
	assert.Contains(t, forced.output(), "WARN [Strategist] install force-overwriting user-owned config", "a destructive notice is not hidden")
	assert.NotContains(t, forced.output(), " INFO ")
}

func TestE2E_CLI_OtherCommandsKeepTheirInfoDisplay(t *testing.T) {
	t.Parallel()
	workspace := t.TempDir()
	install := runStrategistCLI(t, workspace, "install", "--silent", "--no-shim", "--target", workspace)
	require.Equal(t, 0, install.exitCode, install.output())

	// `upgrade` is not annotated: its display level is unchanged.
	upgrade := runStrategistCLI(t, workspace, "upgrade", "--dry-run")

	require.Equal(t, 0, upgrade.exitCode, upgrade.output())
	assert.Contains(t, upgrade.output(), " INFO ", "commands that did not opt in keep the INFO display")
}
