//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestE2E_CLI_CleanForceInstallIsHermeticAndReady(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	env := withHostOpenSpec(t)
	strategistDir := filepath.Join(workspace, ".strategist")
	install := runStrategistCLIWithEnv(t, workspace, env, "install", "--target", workspace, "--force", "--silent", "--no-shim")
	require.Equal(t, 0, install.exitCode, install.output())

	for _, path := range []string{
		filepath.Join(strategistDir, "active.yaml"),
		filepath.Join(strategistDir, ".config.lock"),
		filepath.Join(strategistDir, ".compiled", ".config.gz"),
		filepath.Join(strategistDir, ".compiled", ".manifest.gz"),
		filepath.Join(strategistDir, domainInstallManifestName),
	} {
		assert.FileExists(t, path)
	}

	active, err := os.ReadFile(filepath.Join(strategistDir, "active.yaml"))
	require.NoError(t, err)
	var config struct {
		Language map[string]string `yaml:"language"`
	}
	require.NoError(t, yaml.Unmarshal(active, &config))
	assert.Equal(t, map[string]string{
		"ui": "pt-BR", "docs": "en", "chat": "pt-BR", "code": "en",
	}, config.Language)

	check := runStrategistCLIWithEnv(t, workspace, env, "check", "--root", strategistDir, "--json")
	require.Equal(t, 0, check.exitCode, check.output())
	assert.Contains(t, check.stdout, `"status": "ready"`)
	assert.NoDirExists(t, filepath.Join(workspace, "openspec"))
	assert.NoDirExists(t, filepath.Join(workspace, ".openspec"))
}

func TestE2E_CLI_CleanInstallsDoNotShareRuntimeState(t *testing.T) {
	t.Parallel()

	first := t.TempDir()
	second := t.TempDir()
	firstInstall := runStrategistCLI(t, first, "install", "--target", first, "--force", "--silent", "--no-shim")
	secondInstall := runStrategistCLI(t, second, "install", "--target", second, "--force", "--silent", "--no-shim")
	require.Equal(t, 0, firstInstall.exitCode, firstInstall.output())
	require.Equal(t, 0, secondInstall.exitCode, secondInstall.output())

	firstActive, err := os.ReadFile(filepath.Join(first, ".strategist", "active.yaml"))
	require.NoError(t, err)
	secondActive, err := os.ReadFile(filepath.Join(second, ".strategist", "active.yaml"))
	require.NoError(t, err)
	assert.Equal(t, firstActive, secondActive)
	assert.NotContains(t, string(secondActive), first)
	assert.NotContains(t, string(secondActive), first+string(os.PathSeparator))
}

func TestE2E_CLI_HermeticEnvironmentIgnoresHostDecoys(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "must-not-leak")
	t.Setenv("OPEN_SPEC_CONFIG", "/host/foreign/config.yaml")
	env := hermeticEnv(map[string]string{"HOME": t.TempDir()})
	joined := strings.Join(env, "\n")
	assert.NotContains(t, joined, "OPENAI_API_KEY")
	assert.NotContains(t, joined, "OPEN_SPEC_CONFIG")
	assert.Contains(t, joined, "HOME=")
}

func TestE2E_CLI_ForceInstallPreservesHostDecoys(t *testing.T) {
	workspace := t.TempDir()
	home := t.TempDir()
	xdgConfig := t.TempDir()
	xdgCache := t.TempDir()
	xdgData := t.TempDir()
	decoy := filepath.Join(home, "host-decoy.txt")
	decoyContent := []byte("must remain untouched\n")
	require.NoError(t, os.WriteFile(decoy, decoyContent, 0o600))
	env := map[string]string{
		"HOME":             home,
		"USERPROFILE":      home,
		"XDG_CONFIG_HOME":  xdgConfig,
		"XDG_CACHE_HOME":   xdgCache,
		"XDG_DATA_HOME":    xdgData,
		"OPEN_SPEC_CONFIG": filepath.Join(home, "host-config.yaml"),
	}
	install := runStrategistCLIWithEnv(t, workspace, env, "install", "--target", workspace, "--force", "--silent", "--no-shim")
	require.Equal(t, 0, install.exitCode, install.output())
	got, err := os.ReadFile(decoy)
	require.NoError(t, err)
	assert.Equal(t, decoyContent, got)
	assert.Equal(t, []string{"host-decoy.txt"}, directoryEntries(t, home, "runtime state outside target"))
	assert.Empty(t, directoryEntries(t, xdgConfig, "runtime state outside target"))
	assert.Empty(t, directoryEntries(t, xdgCache, "runtime state outside target"))
	assert.Empty(t, directoryEntries(t, xdgData, "runtime state outside target"))
}

func TestE2E_CLI_WizardRankedInstallBootstrapsContainedRuntime(t *testing.T) {
	workspace := t.TempDir()
	foreignConfig := filepath.Join(t.TempDir(), "foreign-config.yaml")
	foreignBefore := []byte("foreign: untouched\n")
	require.NoError(t, os.WriteFile(foreignConfig, foreignBefore, 0o644))
	env := map[string]string{
		"PATH":             os.Getenv("PATH"),
		"OPEN_SPEC_CONFIG": foreignConfig,
	}
	input := strings.Join([]string{
		"en", "en", "en", "en", "epic", ".analysis",
		"brainstorming::ranked", "openspec-propose::ranked", "sniper::ranked", "", "",
	}, "\n")
	install := runStrategistCLIWithInput(t, workspace, env, input, "install", "--target", workspace, "--wizard", "--no-shim")
	require.Equal(t, 0, install.exitCode, install.output())

	strategist := filepath.Join(workspace, ".strategist")
	require.FileExists(t, filepath.Join(strategist, "openspec", "config.yaml"))
	require.FileExists(t, filepath.Join(strategist, "ranked-runtimes.yaml"))
	require.NoDirExists(t, filepath.Join(strategist, "openspec", "openspec"))
	require.NoDirExists(t, filepath.Join(workspace, "openspec"))

	lock, err := os.ReadFile(filepath.Join(strategist, "plugins.lock"))
	require.NoError(t, err)
	assert.Contains(t, string(lock), "mode: ranked")
	assert.Contains(t, string(lock), "installed_instance_id: openspec-propose")

	runtimeState, err := os.ReadFile(filepath.Join(strategist, "ranked-runtimes.yaml"))
	require.NoError(t, err)
	var state struct {
		Entries []struct {
			Slot           string `yaml:"slot"`
			Provider       string `yaml:"provider"`
			ContractDigest string `yaml:"contract_digest"`
			Root           string `yaml:"root"`
			Kind           string `yaml:"kind"`
			Runtime        struct {
				Mode   string `yaml:"mode"`
				Node   string `yaml:"node"`
				Script string `yaml:"script"`
			} `yaml:"runtime"`
		} `yaml:"entries"`
	}
	require.NoError(t, yaml.Unmarshal(runtimeState, &state))
	require.Len(t, state.Entries, 1)
	assert.Equal(t, "refinement", state.Entries[0].Slot)
	assert.Equal(t, "openspec-propose", state.Entries[0].Provider)
	assert.NotEmpty(t, state.Entries[0].ContractDigest)
	assert.Equal(t, ".strategist/openspec", state.Entries[0].Root)
	assert.Equal(t, "openspec_root", state.Entries[0].Kind)
	assert.True(t, filepath.IsAbs(state.Entries[0].Runtime.Node))
	assert.Contains(t, state.Entries[0].Runtime.Script, "weapon-runtime/openspec-propose/openspec/")
	assert.FileExists(t, filepath.Join(strategist, filepath.FromSlash(state.Entries[0].Runtime.Script)))

	check := runStrategistCLIWithEnv(t, workspace, env, "check", "--root", strategist, "--json")
	require.Equal(t, 0, check.exitCode, check.output())
	assert.Contains(t, check.stdout, `"status": "ready"`)

	foreignAfter, err := os.ReadFile(foreignConfig)
	require.NoError(t, err)
	assert.Equal(t, foreignBefore, foreignAfter)
}

func directoryEntries(t *testing.T, dir, message string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err, message)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

const domainInstallManifestName = ".install-manifest.json"
